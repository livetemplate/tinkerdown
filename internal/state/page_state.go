// Package state provides state management for livepage.
package state

// PageState manages the runtime state of a livepage.
type PageState struct {
	CurrentStep    int
	CodeEdits      map[string]string
	CompletedSteps []int
}

// NewPageState creates a new page state.
func NewPageState() *PageState {
	return &PageState{
		CodeEdits:      make(map[string]string),
		CompletedSteps: make([]int, 0),
	}
}
