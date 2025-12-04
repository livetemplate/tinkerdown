package livepage

import (
	"encoding/json"
	"fmt"
	"sync"
)

// PageState manages the runtime state of a livepage session.
type PageState struct {
	mu sync.RWMutex

	// Page reference
	page *Page

	// Current step for multi-step tutorials
	CurrentStep int

	// Interactive block states (stored as generic interfaces)
	// State objects use method dispatch for action handling
	InteractiveStates map[string]interface{}

	// Code edits for WASM blocks (blockID -> code)
	CodeEdits map[string]string

	// Completed steps tracking
	CompletedSteps []int
}

// NewPageState creates a new page state for a session.
func NewPageState(page *Page) *PageState {
	return &PageState{
		page:              page,
		CurrentStep:       0,
		InteractiveStates: make(map[string]interface{}),
		CodeEdits:         make(map[string]string),
		CompletedSteps:    make([]int, 0),
	}
}

// HandleAction processes page-level actions.
func (ps *PageState) HandleAction(action string, data map[string]interface{}) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	switch action {
	case "nextStep":
		return ps.handleNextStep()
	case "prevStep":
		return ps.handlePrevStep()
	case "saveCodeEdit":
		return ps.handleSaveCodeEdit(data)
	default:
		return fmt.Errorf("unknown page action: %s", action)
	}
}

func (ps *PageState) handleNextStep() error {
	if !ps.page.Config.MultiStep {
		return fmt.Errorf("page is not multi-step")
	}

	if ps.CurrentStep < ps.page.Config.StepCount-1 {
		ps.CurrentStep++
		if !contains(ps.CompletedSteps, ps.CurrentStep-1) {
			ps.CompletedSteps = append(ps.CompletedSteps, ps.CurrentStep-1)
		}
	}

	return nil
}

func (ps *PageState) handlePrevStep() error {
	if !ps.page.Config.MultiStep {
		return fmt.Errorf("page is not multi-step")
	}

	if ps.CurrentStep > 0 {
		ps.CurrentStep--
	}

	return nil
}

func (ps *PageState) handleSaveCodeEdit(data map[string]interface{}) error {
	blockID, ok := data["blockID"].(string)
	if !ok {
		return fmt.Errorf("missing blockID")
	}

	code, ok := data["code"].(string)
	if !ok {
		return fmt.Errorf("missing code")
	}

	ps.CodeEdits[blockID] = code
	return nil
}

// MessageEnvelope wraps messages with block ID routing information.
type MessageEnvelope struct {
	BlockID string          `json:"blockID"`
	Action  string          `json:"action"`
	Data    json.RawMessage `json:"data"`
}

// ResponseEnvelope wraps responses with block ID routing information.
type ResponseEnvelope struct {
	BlockID string                 `json:"blockID"`
	Tree    map[string]interface{} `json:"tree,omitempty"`
	Meta    map[string]interface{} `json:"meta"`
}

// MessageRouter routes messages to appropriate blocks based on blockID.
type MessageRouter struct {
	pageState *PageState
}

// NewMessageRouter creates a new message router for a page state.
func NewMessageRouter(ps *PageState) *MessageRouter {
	return &MessageRouter{
		pageState: ps,
	}
}

// Route routes an incoming message to the appropriate handler.
func (mr *MessageRouter) Route(envelope *MessageEnvelope) (*ResponseEnvelope, error) {
	// Special case: page-level actions
	if envelope.BlockID == "_page" {
		return mr.routePageAction(envelope)
	}

	// Interactive block actions (TODO: implement with livetemplate integration)
	if _, ok := mr.pageState.InteractiveStates[envelope.BlockID]; ok {
		return mr.routeInteractiveBlock(envelope)
	}

	return nil, fmt.Errorf("unknown block: %s", envelope.BlockID)
}

func (mr *MessageRouter) routePageAction(envelope *MessageEnvelope) (*ResponseEnvelope, error) {
	// Parse data
	var dataMap map[string]interface{}
	if err := json.Unmarshal(envelope.Data, &dataMap); err != nil {
		dataMap = make(map[string]interface{})
	}

	// Execute page action
	err := mr.pageState.HandleAction(envelope.Action, dataMap)
	if err != nil {
		return &ResponseEnvelope{
			BlockID: "_page",
			Meta: map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			},
		}, nil
	}

	return &ResponseEnvelope{
		BlockID: "_page",
		Meta: map[string]interface{}{
			"success": true,
		},
	}, nil
}

func (mr *MessageRouter) routeInteractiveBlock(envelope *MessageEnvelope) (*ResponseEnvelope, error) {
	// TODO: Implement livetemplate integration
	// For now, return success with empty tree
	return &ResponseEnvelope{
		BlockID: envelope.BlockID,
		Tree:    map[string]interface{}{},
		Meta: map[string]interface{}{
			"success": true,
		},
	}, nil
}

func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
