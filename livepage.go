// Package livepage provides the core library for building interactive documentation
// with markdown files and embedded executable code blocks.
package livepage

// Page represents a parsed livepage tutorial/guide/playground.
type Page struct {
	ID                string
	Title             string
	Type              string // tutorial, guide, reference, playground
	SourceFile        string // Absolute path to source .md file (for error messages)
	Config            PageConfig
	StaticHTML        string
	ServerBlocks      map[string]*ServerBlock
	WasmBlocks        map[string]*WasmBlock
	InteractiveBlocks map[string]*InteractiveBlock
}

// PageConfig contains configuration for a page.
type PageConfig struct {
	Persist   PersistMode
	MultiStep bool
	StepCount int
}

// PersistMode determines how tutorial state is persisted.
type PersistMode string

const (
	PersistNone         PersistMode = "none"
	PersistLocalStorage PersistMode = "localstorage"
	PersistServer       PersistMode = "server"
)

// New creates a new Page with the given ID.
func New(id string) *Page {
	return &Page{
		ID:                id,
		ServerBlocks:      make(map[string]*ServerBlock),
		WasmBlocks:        make(map[string]*WasmBlock),
		InteractiveBlocks: make(map[string]*InteractiveBlock),
		Config: PageConfig{
			Persist: PersistLocalStorage,
		},
	}
}
