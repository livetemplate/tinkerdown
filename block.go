package livepage

import "github.com/livetemplate/livetemplate"

// ServerBlock represents author-written server-side code.
// This code is trusted, pre-compiled, and powers interactive blocks.
type ServerBlock struct {
	ID       string
	Language string // Currently only "go"
	Content  string
	Metadata map[string]string
}

// WasmBlock represents student-editable code that runs in the browser.
// This code is untrusted and never sent to the server.
type WasmBlock struct {
	ID            string
	Language      string // Currently only "go"
	DefaultCode   string
	ShowRunButton bool
	Metadata      map[string]string
}

// InteractiveBlock represents a live UI component powered by server state.
// Each block is a mini livetemplate instance.
type InteractiveBlock struct {
	ID       string
	StateRef string // References a ServerBlock ID
	Template *livetemplate.Template
	Store    interface{} // State object with action methods (uses method dispatch)
	Content  string      // Template content
	Metadata map[string]string
}
