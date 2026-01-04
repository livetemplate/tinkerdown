// Package tinkerdown provides the core library for building interactive documentation
// with markdown files and embedded executable code blocks.
package tinkerdown

// Page represents a parsed tinkerdown tutorial/guide/playground.
type Page struct {
	ID                string
	Title             string
	Type              string // tutorial, guide, reference, playground
	SourceFile        string // Absolute path to source .md file (for error messages)
	Sidebar           *bool  // nil = use default, true/false = explicit override
	Config            PageConfig
	StaticHTML        string
	ServerBlocks      map[string]*ServerBlock
	WasmBlocks        map[string]*WasmBlock
	InteractiveBlocks map[string]*InteractiveBlock
	// Expressions maps expression IDs to expression strings (e.g., "expr-0" -> "count(tasks where done)")
	// These are computed expressions found in inline code spans like `=count(tasks where done)`
	Expressions map[string]string
}

// PageConfig contains configuration for a page.
type PageConfig struct {
	// Page behavior
	Persist   PersistMode
	MultiStep bool
	StepCount int

	// Effective config (merged from frontmatter + site config)
	Sources  map[string]SourceConfig
	Actions  map[string]Action
	Styling  StylingConfig
	Blocks   BlocksConfig
	Features FeaturesConfig
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
