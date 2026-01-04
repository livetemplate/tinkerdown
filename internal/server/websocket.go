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
	"github.com/livetemplate/tinkerdown"
	"github.com/livetemplate/tinkerdown/internal/config"
	"github.com/livetemplate/tinkerdown/internal/runtime"
	"github.com/livetemplate/tinkerdown/internal/source"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// MessageEnvelope represents a multiplexed WebSocket message.
type MessageEnvelope struct {
	BlockID   string          `json:"blockID"`
	Action    string          `json:"action"`
	Data      json.RawMessage `json:"data"`
	ExecMeta  *ExecMeta       `json:"execMeta,omitempty"`  // Optional exec source metadata
	CacheMeta *CacheMeta      `json:"cacheMeta,omitempty"` // Optional cache metadata
}

// ExecMeta contains execution state for exec source blocks
type ExecMeta struct {
	Status   string `json:"status"`
	Duration int64  `json:"duration,omitempty"`
	Output   string `json:"output,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	Command  string `json:"command,omitempty"`
}

// CacheMeta contains cache state for cached source blocks
type CacheMeta struct {
	Cached     bool   `json:"cached"`
	Age        string `json:"age,omitempty"`
	ExpiresIn  string `json:"expires_in,omitempty"`
	Stale      bool   `json:"stale"`
	Refreshing bool   `json:"refreshing"`
}

// WebSocketHandler handles WebSocket connections for interactive blocks.
type WebSocketHandler struct {
	page           *tinkerdown.Page
	mu             sync.RWMutex
	instances      map[string]*BlockInstance      // blockID -> instance
	sourceFiles    map[string][]string            // blockID -> source file paths (for file watching)
	debug          bool
	server         *Server                        // Reference to server for connection tracking
	stateFactories map[string]func() runtime.Store // State factories for lvt-source blocks
	rootDir        string                          // Site root directory for database path
	config         *config.Config                  // Site configuration with sources
	conn           *websocket.Conn                 // Current connection for this handler
	actionSources  map[string]source.Source       // Cached sources for custom actions
}

// BlockInstance represents a running LiveTemplate instance for an interactive block.
type BlockInstance struct {
	blockID  string
	state    runtime.Store
	template *livetemplate.Template
	conn     *websocket.Conn
	mu       sync.Mutex
}

// NewWebSocketHandler creates a new WebSocket handler for a page.
func NewWebSocketHandler(page *tinkerdown.Page, server *Server, debug bool, rootDir string, cfg *config.Config) *WebSocketHandler {
	if debug {
		log.Printf("[WS] Creating WebSocket handler for page: %s", page.ID)
		log.Printf("[WS] Page has %d server blocks", len(page.ServerBlocks))
		log.Printf("[WS] Page has %d interactive blocks", len(page.InteractiveBlocks))
	}

	h := &WebSocketHandler{
		page:           page,
		instances:      make(map[string]*BlockInstance),
		sourceFiles:    make(map[string][]string),
		debug:          debug,
		server:         server,
		stateFactories: make(map[string]func() runtime.Store),
		rootDir:        rootDir,
		config:         cfg,
		actionSources:  make(map[string]source.Source),
	}

	// Initialize lvt-source blocks (no compilation needed)
	h.initializeSourceBlocks()

	return h
}

// Close cleans up all resources
func (h *WebSocketHandler) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.debug {
		log.Printf("[WS] Closing WebSocket handler, cleaning up %d instances", len(h.instances))
	}

	// Close all state instances
	for blockID, instance := range h.instances {
		if instance.state != nil {
			if err := instance.state.Close(); err != nil && h.debug {
				log.Printf("[WS] Error closing state for block %s: %v", blockID, err)
			}
		}
	}

	// Close cached action sources
	for name, src := range h.actionSources {
		if err := src.Close(); err != nil && h.debug {
			log.Printf("[WS] Error closing action source %s: %v", name, err)
		}
	}

	// Clear maps
	h.instances = make(map[string]*BlockInstance)
	h.actionSources = make(map[string]source.Source)
}

// initializeSourceBlocks initializes all lvt-source blocks with runtime state.
// Regular server blocks (Go code) are no longer supported.
func (h *WebSocketHandler) initializeSourceBlocks() {
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
			log.Printf("[WS] Processing server block: %s (metadata: %v)", blockID, block.Metadata)
		}

		// Only lvt-source blocks are supported
		sourceName := block.Metadata["lvt-source"]
		if sourceName == "" {
			// Regular server blocks (Go code) are no longer supported
			log.Printf("[WS] ERROR: Server block %s is not an lvt-source block. Go code blocks are no longer supported.", blockID)
			log.Printf("[WS] Please migrate to lvt-source by defining a source in frontmatter or tinkerdown.yaml")
			continue
		}

		// lvt-source block - use runtime.GenericState (no compilation needed!)
		// Check page-level sources first (from frontmatter), then site-level (from tinkerdown.yaml)
		sourceCfg, found := h.getEffectiveSource(sourceName)
		if !found {
			log.Printf("[WS] Source %q not found (checked frontmatter and tinkerdown.yaml) for block %s", sourceName, blockID)
			continue
		}
		if h.debug {
			log.Printf("[WS] Creating runtime state for lvt-source block: %s (source: %s, type: %s)", blockID, sourceName, sourceCfg.Type)
		}
		// Pass the current markdown file path for same-file markdown sources
		currentFile := ""
		if h.page != nil {
			currentFile = h.page.SourceFile
		}

		// Create runtime state factory
		// Capture variables for closure
		srcName, srcCfg, rootDir, curFile := sourceName, sourceCfg, h.rootDir, currentFile
		// Copy metadata for closure
		blockMeta := make(map[string]string)
		for k, v := range block.Metadata {
			blockMeta[k] = v
		}

		// Build page-level actions map (convert from parser types to config types)
		pageActions := h.getPageActions()

		factory := func() runtime.Store {
			state, err := runtime.NewGenericStateWithMetadata(srcName, srcCfg, rootDir, curFile, blockMeta)
			if err != nil {
				log.Printf("[WS] Failed to create runtime state for %s: %v", srcName, err)
				return nil
			}

			// Configure page-level settings for custom actions
			if len(pageActions) > 0 {
				state.SetPageConfig(pageActions, h.lookupSource)
			}

			return state
		}

		h.stateFactories[blockID] = factory

		// Track source files for markdown sources (for live refresh)
		if sourceCfg.Type == "markdown" {
			var sourceFilePath string
			if sourceCfg.File != "" {
				// External file - resolve relative to root or current file
				if filepath.IsAbs(sourceCfg.File) {
					sourceFilePath = sourceCfg.File
				} else {
					// Try relative to current file first, then root
					if currentFile != "" {
						sourceFilePath = filepath.Join(filepath.Dir(currentFile), sourceCfg.File)
					} else {
						sourceFilePath = filepath.Join(h.rootDir, sourceCfg.File)
					}
				}
			} else {
				// Same-file source
				sourceFilePath = currentFile
			}
			if sourceFilePath != "" {
				// Make path relative to rootDir for consistent matching with watcher events
				if relPath, err := filepath.Rel(h.rootDir, sourceFilePath); err == nil {
					sourceFilePath = relPath
				}
				h.sourceFiles[blockID] = append(h.sourceFiles[blockID], sourceFilePath)
				if h.debug {
					log.Printf("[WS] Block %s tracks source file: %s", blockID, sourceFilePath)
				}
			}
		}

		if h.debug {
			log.Printf("[WS] Successfully initialized block: %s", blockID)
		}
	}

	// Debug: list all state factories
	if h.debug {
		var factoryIDs []string
		for id := range h.stateFactories {
			factoryIDs = append(factoryIDs, id)
		}
		log.Printf("[WS] State factories: %v", factoryIDs)
	}
}

// getEffectiveSource looks up a source by name, checking page-level sources first
// (from frontmatter), then falling back to site-level sources (from tinkerdown.yaml).
func (h *WebSocketHandler) getEffectiveSource(name string) (config.SourceConfig, bool) {
	// Check page-level sources first (from frontmatter)
	if h.page != nil && h.page.Config.Sources != nil {
		if src, ok := h.page.Config.Sources[name]; ok {
			// Convert tinkerdown.SourceConfig to config.SourceConfig
			return config.SourceConfig{
				Type:        src.Type,
				Cmd:         src.Cmd,
				Query:       src.Query,
				From:        src.From,
				File:        src.File,
				Anchor:      src.Anchor,
				DB:          src.DB,
				Table:       src.Table,
				Path:        src.Path,
				Headers:     src.Headers,
				QueryParams: src.QueryParams,
				ResultPath:  src.ResultPath,
				Readonly:    src.Readonly,
				Options:     src.Options,
				Manual:      src.Manual,
			}, true
		}
	}

	// Fall back to site-level sources (from tinkerdown.yaml)
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

	// Store connection in handler for source refresh
	h.conn = conn

	defer func() {
		// Unregister connection
		if h.server != nil {
			h.server.UnregisterConnection(conn)
		}
		conn.Close()
	}()

	// Register connection for reload broadcasts (with handler for source refresh)
	if h.server != nil {
		h.server.RegisterConnection(conn, h)
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
		BlockID:   instance.blockID,
		Action:    "tree",
		Data:      json.RawMessage(buf.Bytes()),
		ExecMeta:  extractExecMeta(stateData),
		CacheMeta: extractCacheMeta(stateData),
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
		BlockID:   instance.blockID,
		Action:    "tree",
		Data:      json.RawMessage(buf.Bytes()),
		ExecMeta:  extractExecMeta(stateData),
		CacheMeta: extractCacheMeta(stateData),
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

// extractCacheMeta extracts cache state metadata from a state object.
// Returns nil if the state doesn't contain cache metadata (CacheInfo field).
func extractCacheMeta(stateData interface{}) *CacheMeta {
	stateMap, ok := stateData.(map[string]interface{})
	if !ok {
		return nil
	}

	// Check if this state has cache metadata by looking for CacheInfo or cache_info field
	var cacheInfo map[string]interface{}
	if ci, ok := stateMap["CacheInfo"].(map[string]interface{}); ok {
		cacheInfo = ci
	} else if ci, ok := stateMap["cache_info"].(map[string]interface{}); ok {
		cacheInfo = ci
	}

	if cacheInfo == nil {
		return nil
	}

	meta := &CacheMeta{}

	if cached, ok := cacheInfo["cached"].(bool); ok {
		meta.Cached = cached
	}

	if age, ok := cacheInfo["age"].(string); ok {
		meta.Age = age
	}

	if expiresIn, ok := cacheInfo["expires_in"].(string); ok {
		meta.ExpiresIn = expiresIn
	}

	if stale, ok := cacheInfo["stale"].(bool); ok {
		meta.Stale = stale
	}

	if refreshing, ok := cacheInfo["refreshing"].(bool); ok {
		meta.Refreshing = refreshing
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
	funcs := template.FuncMap{
		// Standard math functions used by components
		"mod": func(a, b int) int {
			return a % b
		},
	}
	// Merge all component funcs
	for _, set := range getComponentTemplates() {
		for name, fn := range set.Funcs {
			funcs[name] = fn
		}
	}
	return funcs
}

// RefreshSourcesForFile refreshes all sources that use the given file.
// This is called by the server when a markdown source file changes externally.
func (h *WebSocketHandler) RefreshSourcesForFile(filePath string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.debug {
		log.Printf("[WS] RefreshSourcesForFile called for: %s", filePath)
	}

	// Find all server blocks that use this file
	// Note: sourceFiles uses server block IDs (e.g., auto-persist-lvt-0)
	// but instances uses interactive block IDs (e.g., lvt-0)
	// We need to find the instance by matching StateRef
	for serverBlockID, files := range h.sourceFiles {
		for _, sourceFile := range files {
			if sourceFile == filePath {
				if h.debug {
					log.Printf("[WS] Server block %s uses file %s, looking for matching instance", serverBlockID, filePath)
				}

				// Find the instance whose StateRef matches this server block ID
				var instance *BlockInstance
				for _, block := range h.page.InteractiveBlocks {
					if block.StateRef == serverBlockID {
						instance = h.instances[block.ID]
						if h.debug && instance != nil {
							log.Printf("[WS] Found instance %s for server block %s", block.ID, serverBlockID)
						}
						break
					}
				}

				if instance == nil {
					log.Printf("[WS] No instance found for server block %s", serverBlockID)
					continue
				}

				// Trigger a Refresh action on the source
				// This re-fetches data from the markdown file
				if err := h.handleAction(instance, "Refresh", nil); err != nil {
					log.Printf("[WS] Failed to refresh block %s: %v", instance.blockID, err)
					continue
				}

				// Send the updated state to the client
				h.sendUpdate(instance)

				if h.debug {
					log.Printf("[WS] Successfully refreshed block %s", instance.blockID)
				}
				break // File matched, no need to check other files for this block
			}
		}
	}
}

// getPageActions converts page-level actions from parser types to config types.
// Returns nil if no actions are defined.
func (h *WebSocketHandler) getPageActions() map[string]*config.Action {
	if h.page == nil || h.page.Config.Actions == nil {
		return nil
	}

	result := make(map[string]*config.Action)
	for name, action := range h.page.Config.Actions {
		// Convert parser ParamDef to config ParamDef
		params := make(map[string]config.ParamDef)
		for pname, pdef := range action.Params {
			params[pname] = config.ParamDef{
				Type:     pdef.Type,
				Required: pdef.Required,
				Default:  pdef.Default,
			}
		}

		result[name] = &config.Action{
			Kind:      action.Kind,
			Source:    action.Source,
			Statement: action.Statement,
			URL:       action.URL,
			Method:    action.Method,
			Body:      action.Body,
			Cmd:       action.Cmd,
			Params:    params,
			Confirm:   action.Confirm,
		}
	}
	return result
}

// lookupSource looks up a source by name for custom SQL actions.
// It checks page-level sources first, then site-level sources.
// Sources are cached for reuse and closed when the handler is closed.
func (h *WebSocketHandler) lookupSource(name string) (source.Source, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check cache first
	if src, exists := h.actionSources[name]; exists {
		return src, true
	}

	// Get source config (checks page then site level)
	srcCfg, found := h.getEffectiveSource(name)
	if !found {
		return nil, false
	}

	// Create and cache the source
	currentFile := ""
	if h.page != nil {
		currentFile = h.page.SourceFile
	}

	src, err := createSourceForAction(name, srcCfg, h.rootDir, currentFile)
	if err != nil {
		log.Printf("[WS] Failed to create source %s for action: %v", name, err)
		return nil, false
	}

	h.actionSources[name] = src
	return src, true
}

// createSourceForAction creates a source instance for use in custom actions.
// This is similar to runtime.createSource but imported here for action execution.
func createSourceForAction(name string, cfg config.SourceConfig, siteDir, currentFile string) (source.Source, error) {
	switch cfg.Type {
	case "sqlite":
		return source.NewSQLiteSource(name, cfg.DB, cfg.Table, siteDir, cfg.IsReadonly())
	case "pg":
		return source.NewPostgresSource(name, cfg.Query, cfg.Options)
	default:
		return nil, fmt.Errorf("unsupported source type %q for action", cfg.Type)
	}
}
