package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/livetemplate/components/base"
	"github.com/livetemplate/components/datatable"
	"github.com/livetemplate/livetemplate"
	"github.com/livetemplate/livepage"
	"github.com/livetemplate/livepage/internal/compiler"
	"github.com/livetemplate/livepage/internal/config"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// MessageEnvelope represents a multiplexed WebSocket message.
type MessageEnvelope struct {
	BlockID  string          `json:"blockID"`
	Action   string          `json:"action"`
	Data     json.RawMessage `json:"data"`
	ExecMeta *ExecMeta       `json:"execMeta,omitempty"` // Optional exec source metadata
}

// ExecMeta contains execution state for exec source blocks
type ExecMeta struct {
	Status   string `json:"status"`
	Duration int64  `json:"duration,omitempty"`
	Output   string `json:"output,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	Command  string `json:"command,omitempty"`
}

// WebSocketHandler handles WebSocket connections for interactive blocks.
type WebSocketHandler struct {
	page       *livepage.Page
	mu         sync.RWMutex
	instances  map[string]*BlockInstance // blockID -> instance
	debug      bool
	server     *Server // Reference to server for connection tracking
	compiler   *compiler.ServerBlockCompiler
	stateFactories map[string]func() compiler.Store // Compiled state factories
	rootDir    string // Site root directory for database path
	config     *config.Config // Site configuration with sources
}

// BlockInstance represents a running LiveTemplate instance for an interactive block.
type BlockInstance struct {
	blockID  string
	state    compiler.Store
	template *livetemplate.Template
	conn     *websocket.Conn
	mu       sync.Mutex
}

// NewWebSocketHandler creates a new WebSocket handler for a page.
func NewWebSocketHandler(page *livepage.Page, server *Server, debug bool, rootDir string, cfg *config.Config) *WebSocketHandler {
	if debug {
		log.Printf("[WS] Creating WebSocket handler for page: %s", page.ID)
		log.Printf("[WS] Page has %d server blocks", len(page.ServerBlocks))
		log.Printf("[WS] Page has %d interactive blocks", len(page.InteractiveBlocks))
	}

	h := &WebSocketHandler{
		page:           page,
		instances:      make(map[string]*BlockInstance),
		debug:          debug,
		server:         server,
		compiler:       compiler.NewServerBlockCompiler(debug),
		stateFactories: make(map[string]func() compiler.Store),
		rootDir:        rootDir,
		config:         cfg,
	}

	// Compile all server blocks
	h.compileServerBlocks()

	return h
}

// Close cleans up all resources including plugin processes and build artifacts
func (h *WebSocketHandler) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.debug {
		log.Printf("[WS] Closing WebSocket handler, cleaning up %d instances", len(h.instances))
	}

	// Close all state instances (kills plugin processes)
	for blockID, instance := range h.instances {
		if instance.state != nil {
			if err := instance.state.Close(); err != nil && h.debug {
				log.Printf("[WS] Error closing state for block %s: %v", blockID, err)
			}
		}
	}

	// Clear instances map
	h.instances = make(map[string]*BlockInstance)

	// Cleanup compiler build artifacts
	if h.compiler != nil {
		h.compiler.Cleanup()
	}
}

// compileServerBlocks compiles all server blocks into loadable plugins
func (h *WebSocketHandler) compileServerBlocks() {
	// Debug: list all server block IDs first
	if h.debug {
		var blockIDs []string
		for id := range h.page.ServerBlocks {
			blockIDs = append(blockIDs, id)
		}
		log.Printf("[WS] All server block IDs: %v", blockIDs)
	}

	for blockID, block := range h.page.ServerBlocks {
		if h.debug {
			log.Printf("[WS] Compiling server block: %s (metadata: %v)", blockID, block.Metadata)
		}

		var factory func() compiler.Store
		var err error

		// Check if this is an lvt-source block (takes precedence if both are present)
		if sourceName := block.Metadata["lvt-source"]; sourceName != "" {
			// lvt-source block - generate code to fetch from source
			// Check page-level sources first (from frontmatter), then site-level (from livepage.yaml)
			sourceCfg, found := h.getEffectiveSource(sourceName)
			if !found {
				log.Printf("[WS] Source %q not found (checked frontmatter and livepage.yaml) for block %s", sourceName, blockID)
				continue
			}
			if h.debug {
				log.Printf("[WS] Compiling lvt-source block: %s (source: %s, type: %s)", blockID, sourceName, sourceCfg.Type)
			}
			factory, err = h.compiler.CompileLvtSource(blockID, sourceName, sourceCfg, h.rootDir, block.Metadata)
		} else if block.Metadata["auto-persist"] == "true" {
			// Auto-persist block - generate code from form fields
			dbPath := filepath.Join(h.rootDir, "site.sqlite")
			if h.debug {
				log.Printf("[WS] Compiling auto-persist block: %s (db: %s)", blockID, dbPath)
			}
			factory, err = h.compiler.CompileAutoPersist(blockID, block.Content, dbPath)
		} else {
			// Regular server block
			factory, err = h.compiler.CompileServerBlock(block)
		}

		if err != nil {
			log.Printf("[WS] Failed to compile block %s: %v", blockID, err)
			continue
		}

		h.stateFactories[blockID] = factory

		if h.debug {
			log.Printf("[WS] Successfully compiled block: %s", blockID)
		}
	}

	// Debug: list all compiled state factories
	if h.debug {
		var factoryIDs []string
		for id := range h.stateFactories {
			factoryIDs = append(factoryIDs, id)
		}
		log.Printf("[WS] Compiled state factories: %v", factoryIDs)
	}
}

// getEffectiveSource looks up a source by name, checking page-level sources first
// (from frontmatter), then falling back to site-level sources (from livepage.yaml).
func (h *WebSocketHandler) getEffectiveSource(name string) (config.SourceConfig, bool) {
	// Check page-level sources first (from frontmatter)
	if h.page != nil && h.page.Config.Sources != nil {
		if src, ok := h.page.Config.Sources[name]; ok {
			// Convert livepage.SourceConfig to config.SourceConfig
			return config.SourceConfig{
				Type:    src.Type,
				Cmd:     src.Cmd,
				Query:   src.Query,
				URL:     src.URL,
				File:    src.File,
				Options: src.Options,
				Manual:  src.Manual,
			}, true
		}
	}

	// Fall back to site-level sources (from livepage.yaml)
	if h.config != nil && h.config.Sources != nil {
		if src, ok := h.config.Sources[name]; ok {
			return src, true
		}
	}

	return config.SourceConfig{}, false
}

// ServeHTTP handles WebSocket upgrade and message routing.
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Failed to upgrade connection: %v", err)
		return
	}
	defer func() {
		// Unregister connection
		if h.server != nil {
			h.server.UnregisterConnection(conn)
		}
		conn.Close()
	}()

	// Register connection for reload broadcasts
	if h.server != nil {
		h.server.RegisterConnection(conn)
	}

	if h.debug {
		log.Printf("[WS] Client connected: %s", conn.RemoteAddr())
	}

	// Initialize instances for all interactive blocks
	h.initializeInstances(conn)

	// Handle messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Unexpected close: %v", err)
			}
			break
		}

		if h.debug {
			log.Printf("[WS] Received: %s", message)
		}

		h.handleMessage(conn, message)
	}

	if h.debug {
		log.Printf("[WS] Client disconnected: %s", conn.RemoteAddr())
	}
}

// initializeInstances creates LiveTemplate instances for each interactive block.
func (h *WebSocketHandler) initializeInstances(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for blockID, block := range h.page.InteractiveBlocks {
		// Get the state from the server block it references
		stateBlock, ok := h.page.ServerBlocks[block.StateRef]
		if !ok {
			log.Printf("[WS] Interactive block %s references unknown state %s", blockID, block.StateRef)
			continue
		}

		// Get the compiled state factory
		factory, ok := h.stateFactories[block.StateRef]
		if !ok {
			log.Printf("[WS] No compiled factory for state %s", block.StateRef)
			continue
		}

		// Create state instance using the compiled factory
		state := factory()

		// Create template from inline content
		// Since livetemplate.New() requires template files, we use a workaround:
		// Write content to a temp file, parse it, then delete
		tmpFile := fmt.Sprintf("/tmp/lvt-%s.tmpl", blockID)
		if h.debug {
			log.Printf("[WS] Block %s template content:\n%s", blockID, block.Content)
		}
		if err := os.WriteFile(tmpFile, []byte(block.Content), 0644); err != nil {
			log.Printf("[WS] Failed to write temp template for block %s: %v", blockID, err)
			continue
		}
		defer os.Remove(tmpFile)

		tmpl, err := livetemplate.New(blockID,
			livetemplate.WithComponentTemplates(getComponentTemplates()...),
			livetemplate.WithParseFiles(tmpFile))
		if err != nil {
			log.Printf("[WS] Failed to create template for block %s: %v", blockID, err)
			continue
		}

		// Register component-specific template functions for tree generation
		// These are needed because WithComponentTemplates adds funcs to t.tmpl but not t.funcs
		tmpl.Funcs(getComponentFuncs())

		instance := &BlockInstance{
			blockID:  blockID,
			state:    state,
			template: tmpl,
			conn:     conn,
		}

		h.instances[blockID] = instance

		// Send initial state
		h.sendInitialState(instance)

		if h.debug {
			log.Printf("[WS] Initialized block: %s (state ref: %s)", blockID, block.StateRef)
		}

		_ = stateBlock // Mark as used
	}
}

// sendInitialState sends the initial tree update to the client.
func (h *WebSocketHandler) sendInitialState(instance *BlockInstance) {
	instance.mu.Lock()
	defer instance.mu.Unlock()

	// Get state data for rendering
	var stateData interface{}

	// Check if this is an RPC adapter with GetStateAsInterface method
	type StateGetter interface {
		GetStateAsInterface() (interface{}, error)
	}

	if getter, ok := instance.state.(StateGetter); ok {
		// RPC plugin - fetch state via RPC
		var err error
		stateData, err = getter.GetStateAsInterface()
		if err != nil {
			log.Printf("[WS] Failed to get state for %s: %v", instance.blockID, err)
			return
		}
		// Hydrate datatable structs so template methods work
		stateData = hydrateDataTableState(stateData)
		if h.debug {
			log.Printf("[WS] RPC state for %s: %+v (type: %T)", instance.blockID, stateData, stateData)
		}
	} else {
		// Regular in-process state
		stateData = instance.state
		if h.debug {
			log.Printf("[WS] Direct state for %s: %+v (type: %T)", instance.blockID, stateData, stateData)
		}
	}

	// Render tree update using ExecuteUpdates (follows LiveTemplate tree-update specification)
	var buf bytes.Buffer
	if err := instance.template.ExecuteUpdates(&buf, stateData); err != nil {
		log.Printf("[WS] Failed to render initial state for %s: %v", instance.blockID, err)
		return
	}

	// The buffer contains the tree JSON directly
	response := MessageEnvelope{
		BlockID:  instance.blockID,
		Action:   "tree",
		Data:     json.RawMessage(buf.Bytes()),
		ExecMeta: extractExecMeta(stateData),
	}

	h.sendMessage(instance.conn, response)
}

// handleMessage routes incoming messages to the appropriate block instance.
func (h *WebSocketHandler) handleMessage(conn *websocket.Conn, message []byte) {
	var envelope MessageEnvelope
	if err := json.Unmarshal(message, &envelope); err != nil {
		log.Printf("[WS] Failed to parse message: %v", err)
		return
	}

	h.mu.RLock()
	instance, ok := h.instances[envelope.BlockID]
	h.mu.RUnlock()

	if !ok {
		log.Printf("[WS] Unknown block ID: %s", envelope.BlockID)
		return
	}

	// Handle action
	if err := h.handleAction(instance, envelope.Action, envelope.Data); err != nil {
		log.Printf("[WS] Error handling action: %v", err)
		return
	}

	// Re-render and send update
	h.sendUpdate(instance)
}

// handleAction executes an action on the state.
func (h *WebSocketHandler) handleAction(instance *BlockInstance, action string, data json.RawMessage) error {
	instance.mu.Lock()
	defer instance.mu.Unlock()

	// Parse data into map
	var dataMap map[string]interface{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &dataMap); err != nil {
			return fmt.Errorf("failed to parse action data: %w", err)
		}
	} else {
		dataMap = make(map[string]interface{})
	}

	// Execute action via HandleAction method
	if err := instance.state.HandleAction(action, dataMap); err != nil {
		return fmt.Errorf("action failed: %w", err)
	}

	if h.debug {
		log.Printf("[WS] Executed action %s on block %s", action, instance.blockID)
	}

	return nil
}

// sendUpdate re-renders the state and sends a tree update to the client.
func (h *WebSocketHandler) sendUpdate(instance *BlockInstance) {
	instance.mu.Lock()
	defer instance.mu.Unlock()

	// Get state data for rendering
	var stateData interface{}

	// Check if this is an RPC adapter with GetStateAsInterface method
	type StateGetter interface {
		GetStateAsInterface() (interface{}, error)
	}

	if getter, ok := instance.state.(StateGetter); ok {
		// RPC plugin - fetch state via RPC
		var err error
		stateData, err = getter.GetStateAsInterface()
		if err != nil {
			log.Printf("[WS] Failed to get state for %s: %v", instance.blockID, err)
			return
		}
		// Hydrate datatable structs so template methods work
		stateData = hydrateDataTableState(stateData)
	} else {
		// Regular in-process state
		stateData = instance.state
	}

	// Render tree update using ExecuteUpdates (follows LiveTemplate tree-update specification)
	// ExecuteUpdates returns only changed dynamics after the first render
	var buf bytes.Buffer
	if err := instance.template.ExecuteUpdates(&buf, stateData); err != nil {
		log.Printf("[WS] Failed to render update for %s: %v", instance.blockID, err)
		return
	}

	// The buffer contains the tree JSON directly
	response := MessageEnvelope{
		BlockID:  instance.blockID,
		Action:   "tree",
		Data:     json.RawMessage(buf.Bytes()),
		ExecMeta: extractExecMeta(stateData),
	}

	h.sendMessage(instance.conn, response)
}

// sendMessage sends a message envelope over WebSocket.
func (h *WebSocketHandler) sendMessage(conn *websocket.Conn, envelope MessageEnvelope) {
	data, err := json.Marshal(envelope)
	if err != nil {
		log.Printf("[WS] Failed to marshal response: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("[WS] Failed to send message: %v", err)
		return
	}

	if h.debug {
		log.Printf("[WS] Sent: %s", data)
	}
}

// extractExecMeta extracts exec state metadata from a state object.
// Returns nil if the state doesn't contain exec metadata fields (Status).
func extractExecMeta(stateData interface{}) *ExecMeta {
	stateMap, ok := stateData.(map[string]interface{})
	if !ok {
		return nil
	}

	// Check if this state has exec metadata by looking for Status field
	status, hasStatus := stateMap["Status"].(string)
	if !hasStatus {
		return nil
	}

	meta := &ExecMeta{
		Status: status,
	}

	// Extract other optional fields
	if duration, ok := stateMap["Duration"].(int); ok {
		meta.Duration = int64(duration)
	} else if duration, ok := stateMap["Duration"].(int64); ok {
		meta.Duration = duration
	} else if duration, ok := stateMap["Duration"].(float64); ok {
		meta.Duration = int64(duration)
	}

	if output, ok := stateMap["Output"].(string); ok {
		meta.Output = output
	}

	if stderr, ok := stateMap["Stderr"].(string); ok {
		meta.Stderr = stderr
	}

	if command, ok := stateMap["Command"].(string); ok {
		meta.Command = command
	}

	return meta
}

// Note: State types are now dynamically compiled from server blocks
// No need for hardcoded placeholder states

// hydrateDataTableState converts JSON map data back to typed datatable structs.
// When state is transmitted via RPC as JSON, structs become maps and lose their methods.
// This function reconstructs datatable.DataTable so template methods work correctly.
func hydrateDataTableState(stateData interface{}) interface{} {
	stateMap, ok := stateData.(map[string]interface{})
	if !ok {
		return stateData
	}

	// Check for Table field with datatable data (used by lvt-source datatables)
	// Try lowercase first (json:"table"), then uppercase (json:"Table")
	for _, key := range []string{"table", "Table"} {
		if tableData, ok := stateMap[key].(map[string]interface{}); ok {
			if hydratedTable := hydrateDataTable(tableData); hydratedTable != nil {
				stateMap[key] = hydratedTable
			}
			break
		}
	}

	return stateData
}

// hydrateDataTable converts a map[string]interface{} to a *datatable.DataTable.
func hydrateDataTable(tableData map[string]interface{}) *datatable.DataTable {
	// Re-serialize and deserialize to get proper types
	tableJSON, err := json.Marshal(tableData)
	if err != nil {
		log.Printf("[WS] hydrateDataTable: marshal error: %v", err)
		return nil
	}

	var dt datatable.DataTable
	if err := json.Unmarshal(tableJSON, &dt); err != nil {
		log.Printf("[WS] hydrateDataTable: unmarshal error: %v", err)
		return nil
	}

	log.Printf("[WS] hydrateDataTable: success, ID=%s, Columns=%d, Rows=%d",
		dt.ID(), len(dt.Columns), len(dt.Rows))

	return &dt
}

// convertTemplateSet converts a base.TemplateSet to a livetemplate.TemplateSet.
// This is needed because the components library uses base.TemplateSet while
// livetemplate.WithComponentTemplates expects livetemplate.TemplateSet.
// The types are structurally identical so we just copy the fields.
func convertTemplateSet(bs *base.TemplateSet) *livetemplate.TemplateSet {
	return &livetemplate.TemplateSet{
		FS:        bs.FS,
		Pattern:   bs.Pattern,
		Namespace: bs.Namespace,
		Funcs:     bs.Funcs,
	}
}

// getComponentTemplates returns livetemplate.TemplateSet versions of all component templates
func getComponentTemplates() []*livetemplate.TemplateSet {
	return []*livetemplate.TemplateSet{
		convertTemplateSet(datatable.Templates()),
	}
}

// getComponentFuncs returns all component-specific template functions
// These need to be registered with tmpl.Funcs() for tree generation to work
func getComponentFuncs() template.FuncMap {
	funcs := template.FuncMap{}
	// Merge all component funcs
	for _, set := range getComponentTemplates() {
		for name, fn := range set.Funcs {
			funcs[name] = fn
		}
	}
	return funcs
}
