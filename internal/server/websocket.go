package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/livetemplate/livetemplate"
	"github.com/livetemplate/livepage"
	"github.com/livetemplate/livepage/internal/compiler"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// MessageEnvelope represents a multiplexed WebSocket message.
type MessageEnvelope struct {
	BlockID string          `json:"blockID"`
	Action  string          `json:"action"`
	Data    json.RawMessage `json:"data"`
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
func NewWebSocketHandler(page *livepage.Page, server *Server, debug bool) *WebSocketHandler {
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
	}

	// Compile all server blocks
	h.compileServerBlocks()

	return h
}

// compileServerBlocks compiles all server blocks into loadable plugins
func (h *WebSocketHandler) compileServerBlocks() {
	for blockID, block := range h.page.ServerBlocks {
		if h.debug {
			log.Printf("[WS] Compiling server block: %s", blockID)
		}

		factory, err := h.compiler.CompileServerBlock(block)
		if err != nil {
			log.Printf("[WS] Failed to compile block %s: %v", blockID, err)
			continue
		}

		h.stateFactories[blockID] = factory

		if h.debug {
			log.Printf("[WS] Successfully compiled block: %s", blockID)
		}
	}
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
		if err := os.WriteFile(tmpFile, []byte(block.Content), 0644); err != nil {
			log.Printf("[WS] Failed to write temp template for block %s: %v", blockID, err)
			continue
		}
		defer os.Remove(tmpFile)

		tmpl, err := livetemplate.New(blockID, livetemplate.WithParseFiles(tmpFile))
		if err != nil {
			log.Printf("[WS] Failed to create template for block %s: %v", blockID, err)
			continue
		}

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

// sendInitialState sends the initial rendered HTML to the client.
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

	// Render initial HTML using Execute
	var buf bytes.Buffer
	if err := instance.template.Execute(&buf, stateData); err != nil {
		log.Printf("[WS] Failed to render initial state for %s: %v", instance.blockID, err)
		return
	}

	html := buf.String()

	// Send as update message with properly encoded JSON
	data := map[string]interface{}{
		"html": html,
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("[WS] Failed to marshal data: %v", err)
		return
	}

	response := MessageEnvelope{
		BlockID: instance.blockID,
		Action:  "update",
		Data:    json.RawMessage(dataJSON),
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

// sendUpdate re-renders the state and sends an update to the client.
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
	} else {
		// Regular in-process state
		stateData = instance.state
	}

	// For now, use Execute to get full HTML (ExecuteUpdates not yet implemented)
	// TODO: Use ExecuteUpdates when tree diffing is available
	var buf bytes.Buffer
	if err := instance.template.Execute(&buf, stateData); err != nil {
		log.Printf("[WS] Failed to render update for %s: %v", instance.blockID, err)
		return
	}

	html := buf.String()

	// Send as update message with properly encoded JSON
	data := map[string]interface{}{
		"html": html,
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("[WS] Failed to marshal data: %v", err)
		return
	}

	response := MessageEnvelope{
		BlockID: instance.blockID,
		Action:  "update",
		Data:    json.RawMessage(dataJSON),
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

// Note: State types are now dynamically compiled from server blocks
// No need for hardcoded placeholder states
