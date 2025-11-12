# Livepage Design Document

**Date**: 2025-11-12
**Status**: Draft
**Author**: Design session with Claude

## Executive Summary

Livepage is a CLI tool for creating interactive documentation - tutorials, guides, reference docs, and playgrounds - using markdown files with embedded executable code blocks. It builds on the livetemplate library to provide real-time reactivity and is designed for simplicity: authors write markdown, run `livepage serve`, and get a fully interactive tutorial website.

**Core Innovation**: Dual execution model where trusted author code runs on the server powering interactive demos, while untrusted student code runs client-side in WASM sandboxes.

## Goals

### Primary Goals
1. **Easy authoring**: Write tutorials in markdown with special code blocks
2. **Zero config**: `livepage serve` on a directory of .md files just works
3. **Interactive demos**: Embed working apps (counters, forms, etc.) powered by server-side state
4. **Student playgrounds**: Editable Go code that compiles and runs in the browser
5. **Multi-session support**: Progress tracking and state persistence

### Non-Goals
- Not a general-purpose documentation generator (use Hugo/Docusaurus for static docs)
- Not a full IDE (simple textarea editor, not VS Code)
- Not multi-language initially (Go WASM only, extensible later)
- Not a library API (CLI tool first, library extraction if needed later)

## User Personas

### Tutorial Author (Primary)
- Wants to create interactive Go tutorials
- Knows livetemplate and Go
- Values simplicity over flexibility
- Wants to focus on content, not tooling

### Tutorial Student (Secondary)
- Learning Go or livetemplate
- Needs to see working examples AND try code themselves
- May be on slow connections (WASM bundle size matters)
- Expects modern UX (syntax highlighting, instant feedback)

## Use Cases

1. **Step-by-step tutorials**: "Build a Counter in 5 Steps"
2. **Interactive API references**: Documentation with runnable examples
3. **Playgrounds**: Free-form coding environment with starter code
4. **Guided exercises**: Problems with test cases to validate solutions

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────┐
│                  Tutorial Author                     │
│              (Writes Markdown Files)                 │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │   livepage serve      │
         │   (CLI Tool)          │
         └───────────┬───────────┘
                     │
                     ▼
┌────────────────────────────────────────────────────┐
│              Livepage Server                        │
│  ┌──────────────────────────────────────────────┐  │
│  │  Markdown Parser                             │  │
│  │  • Extracts code blocks                      │  │
│  │  • Identifies types (server/wasm/lvt)        │  │
│  │  • Converts prose to static HTML             │  │
│  └─────────────────┬────────────────────────────┘  │
│                    │                                │
│  ┌─────────────────▼────────────────────────────┐  │
│  │  Page Manager                                │  │
│  │  • ServerBlocks (author code, pre-compiled)  │  │
│  │  • WasmBlocks (metadata only)                │  │
│  │  • InteractiveBlocks (mini livetemplate)     │  │
│  └─────────────────┬────────────────────────────┘  │
│                    │                                │
│  ┌─────────────────▼────────────────────────────┐  │
│  │  WebSocket Multiplexer                       │  │
│  │  • Single connection per page                │  │
│  │  • Routes by blockID                         │  │
│  └──────────────────────────────────────────────┘  │
└────────────────────┬───────────────────────────────┘
                     │
                     │ WebSocket (multiplexed)
                     │
                     ▼
┌────────────────────────────────────────────────────┐
│              Browser (Student)                      │
│  ┌──────────────────────────────────────────────┐  │
│  │  Livepage Client                             │  │
│  │  • Message router (by blockID)               │  │
│  │  • Persistence manager (localStorage)        │  │
│  └────────┬─────────────────────┬─────────────┬─┘  │
│           │                     │             │    │
│  ┌────────▼─────────┐  ┌────────▼──────┐  ┌──▼───────┐
│  │ Interactive      │  │ WASM Executor │  │ Static   │
│  │ Blocks           │  │ • TinyGo      │  │ Content  │
│  │ • livetemplate   │  │ • Compile     │  │ (cached) │
│  │ • Real-time UI   │  │ • Execute     │  │          │
│  └──────────────────┘  └───────────────┘  └──────────┘
└────────────────────────────────────────────────────┘
```

### Component Architecture

#### 1. Markdown Parser

**Responsibility**: Parse .md files into structured Page objects

**Key Features**:
- Frontmatter parsing (YAML)
- Code block extraction with metadata
- Static HTML generation for prose
- Block type identification

**Code Block Syntax**:

```markdown
```go server readonly id="counter-state"
type CounterState struct { Counter int }
```

```go wasm editable
package main
func main() { fmt.Println("Hello") }
```

```lvt interactive state="counter-state"
<button lvt-click="increment">{{.Counter}}</button>
```
```

**Implementation**: Use goldmark (extensible, widely adopted)

#### 2. Block Types

**ServerBlock** (`go server readonly`):
- Author-written backend code
- Pre-compiled, trusted
- Displayed with syntax highlighting
- Powers interactive blocks
- Example: State structs, Change() methods

**WasmBlock** (`go wasm editable`):
- Student-written playground code
- Runs in browser only (never sent to server)
- Compiled to WASM client-side via TinyGo
- Has: editor, run button, output panel
- Example: Learning exercises, experiments

**InteractiveBlock** (`lvt interactive`):
- Live UI powered by server state
- Each block is a mini livetemplate instance
- References a ServerBlock for state
- Updates via WebSocket
- Example: Buttons, forms, real-time counters

#### 3. Page State Management

```go
type Page struct {
    ID                string
    Title             string
    Config            PageConfig

    // Static content (cached)
    StaticHTML        string

    // Code blocks
    ServerBlocks      map[string]*ServerBlock
    WasmBlocks        map[string]*WasmBlock
    InteractiveBlocks map[string]*InteractiveBlock
}

type PageState struct {
    // Managed by server
    CurrentStep       int
    InteractiveStates map[string]livetemplate.Store

    // Synchronized for persistence
    CodeEdits         map[string]string  // blockID -> code
    CompletedSteps    []int
}
```

#### 4. WebSocket Multiplexing

**Single connection per page**, messages tagged by block ID:

```typescript
// Client → Server
{
  blockID: "lvt-counter",
  action: "increment",
  data: {}
}

// Server → Client
{
  blockID: "lvt-counter",
  tree: { /* livetemplate tree diff */ },
  meta: { success: true }
}

// Special: Page-level actions
{
  blockID: "_page",
  action: "nextStep",
  data: {}
}
```

**Routing**:
- `blockID: "_page"` → Page.Change()
- `blockID: "lvt-*"` → InteractiveBlock.Store.Change()
- `blockID: "wasm-*"` → No server handling (client-only)

#### 5. Interactive Block Implementation

Each interactive block is a **mini livetemplate instance**:

```go
type InteractiveBlock struct {
    ID       string
    StateRef string  // References ServerBlock ID
    Template *livetemplate.Template
    Store    livetemplate.Store
}

// During page parsing
func (p *Page) parseInteractiveBlock(md *MarkdownBlock) error {
    // Find referenced state
    serverBlock := p.ServerBlocks[md.Metadata["state"]]

    // Instantiate the state struct (reflection)
    store := serverBlock.NewInstance()

    // Create mini livetemplate
    tmpl := livetemplate.New(md.ID)
    tmpl.Parse(md.Content)

    block := &InteractiveBlock{
        ID:       md.ID,
        StateRef: md.Metadata["state"],
        Template: tmpl,
        Store:    store,
    }

    p.InteractiveBlocks[md.ID] = block
    return nil
}
```

### Execution Model

**Two execution contexts**:

#### Author Code (Server-Side)
- Written in `server readonly` blocks
- Pre-compiled with the tutorial
- Trusted code
- Runs on server
- Powers interactive blocks
- Zero compilation cost at runtime

**Flow**:
1. Author writes CounterState in `server readonly` block
2. Livepage compiles it on startup
3. Student clicks button in `lvt interactive` block
4. Server executes CounterState.Change()
5. Server sends tree diff back
6. Client updates DOM

#### Student Code (Client-Side WASM)
- Written in `wasm editable` blocks
- Untrusted code
- Never sent to server
- Compiled in browser via TinyGo WASM
- Executed in WASM sandbox

**Flow**:
1. Student modifies code in textarea
2. Student clicks "Run"
3. **Client compiles** Go → WASM (TinyGo in browser)
4. **Client executes** WASM
5. **Client captures** stdout/stderr
6. **Client displays** output
7. No server communication for execution!
8. Optional: Send "saveEdit" action to persist code text

**Security**: Student code never reaches server, preventing code injection attacks.

### CLI Tool

#### Commands

```bash
# Primary command: serve directory
livepage serve [directory]

# Create new tutorial
livepage new <name>

# Version
livepage version
```

#### Auto-Discovery

```
tutorials/
├── index.md              → /
├── counter.md            → /counter
├── chat.md               → /chat
└── advanced/
    └── state.md          → /advanced/state
```

Algorithm:
1. Scan directory recursively for .md files
2. Skip `_` prefixed directories (_assets, _shared)
3. Generate routes:
   - `index.md` → `/`
   - `foo.md` → `/foo`
   - `bar/index.md` → `/bar/`
   - `bar/baz.md` → `/bar/baz`

#### Hot Reload

Watch for changes:
- .md file changes → Reparse and update page
- _shared/*.go changes → Recompile shared state structs
- livepage.yaml changes → Reload config

#### Configuration (livepage.yaml)

```yaml
# Optional - zero config by default

home: index.md

server:
  port: 8080
  host: localhost

defaults:
  persist: localstorage  # none | localstorage | server
  theme: default

routes:
  /: index.md
  /docs: tutorials/intro.md

shared:
  - _shared/*.go

assets:
  dir: _assets
  prefix: /assets

wasm:
  tinygo_url: https://cdn.example.com/tinygo.wasm
  cache: true
```

### Client Runtime

#### Architecture

```
livepage-client/
├── core/
│   ├── livepage-client.ts       # Main orchestrator
│   ├── message-router.ts        # Routes by blockID
│   └── persistence-manager.ts   # localStorage
├── blocks/
│   ├── interactive-block.ts     # Wraps livetemplate
│   ├── wasm-block.ts            # WASM execution
│   ├── editor.ts                # Simple textarea editor
│   └── output-panel.ts          # Execution results
└── wasm/
    ├── tinygo-executor.ts       # TinyGo compiler interface
    └── sandbox.ts               # WASM sandbox
```

#### WASM Execution Strategy

**Option A: TinyGo WASM Compiler in Browser** (Chosen)
- Bundle TinyGo compiler as WASM (~8-10MB gzipped)
- Compile user code entirely in browser
- Zero server load
- Secure: code never leaves browser

**Trade-offs**:
- Larger initial download
- First compile slower (~2-5s)
- Higher browser memory usage
- **Benefit**: Zero security risk, scales infinitely

**Implementation Plan**:
1. Research TinyGo WASM build
2. Create compiler WASM bundle
3. Build JavaScript API wrapper
4. Cache compiled binaries in browser
5. Progressive loading (load compiler on first "Run" click)

#### Code Editor

**Requirement**: Simple textarea with syntax highlighting and line numbers

**Not Monaco/CodeMirror**: Too heavy for embedded use

**Implementation**:
- Textarea for input
- Syntax highlighting via Prism.js (lightweight)
- Line numbers via CSS
- Tab key handling
- Auto-indent

#### Persistence

**Three Modes**:

1. **None** (`persist: none`)
   - State lives only during session
   - Lost on refresh
   - Simplest implementation

2. **LocalStorage** (`persist: localstorage`)
   - Save code edits to localStorage
   - Save completed steps
   - Restore on page load
   - No authentication needed

3. **Server** (`persist: server`)
   - Save to database
   - Requires authentication
   - Cross-device sync
   - Advanced use case

**Default**: `localstorage`

### File Format

#### Frontmatter

```yaml
---
title: "Build a Counter App"
type: tutorial        # tutorial | guide | reference | playground
persist: localstorage # none | localstorage | server
steps: 3              # For step-by-step tutorials
---
```

#### Complete Example

See [Section 8 of brainstorming session](#) for full counter tutorial example.

### Built-in Theme

**Requirement**: Professional styling out of the box

**Includes**:
- Clean typography (system fonts)
- Syntax highlighting (Prism.js)
- Responsive layout
- Code editor styling
- Button and form styling
- Loading states
- Dark mode support

**Customization** (optional):
```yaml
theme:
  primary: "#007bff"
  accent: "#28a745"

assets:
  custom_css: _assets/custom.css
```

## Testing Strategy

### E2E Tests (chromedp)

**Requirements** (from CLAUDE.md):
- Browser console logs
- Server logs
- WebSocket messages
- Rendered HTML

```go
func TestCounterTutorial(t *testing.T) {
    ctx := setupTest(t)
    srv := startLivepageServer("testdata/counter")
    defer srv.Close()

    browser := chromedp.NewBrowser(ctx)
    logs := browser.ConsoleLogs()
    ws := browser.WebSocketMessages()
    serverLogs := srv.Logs()

    // Test interactive block
    browser.Navigate(srv.URL + "/counter")
    browser.Click("button[lvt-click='increment']")
    browser.WaitForText("Count: 1")

    // Verify WebSocket
    assert.Contains(t, ws.Sent(), `"action":"increment"`)

    // Test WASM block
    browser.SetValue("textarea[data-block='wasm-1']", "...")
    browser.Click("button[data-action='run']")
    browser.WaitForText(".output-panel", "Hello")

    // Verify no server call for WASM
    assert.NotContains(t, serverLogs.Latest(), "wasm-1")

    // Capture HTML
    html := browser.HTML()
    assert.Contains(t, html, "Count: 1")
}
```

### Unit Tests

- Markdown parser
- Block registry
- Message router
- State management
- File discovery
- Route generation

### Integration Tests

- Full page lifecycle
- Multi-block interactions
- Persistence
- Hot reload

## Error Handling

**Principle**: Helpful errors with file locations

```
counter.md:15:1: state reference 'counter-state' not found in block 'lvt-demo'

   13 | ## Interactive Demo
   14 |
   15 | ```lvt interactive state="counter-state"
      | ^
   16 | <button>Click</button>
   17 | ```

Suggestion: Check that you have a `server readonly` block with id="counter-state"
```

**Dev Server Error Page**:
- File path and line number
- Code snippet with error highlighted
- Suggested fixes
- Link to documentation

## Performance Considerations

### Bundle Size

**Initial Load**:
- livepage-client.js: ~15KB gzipped (extends livetemplate-client)
- Prism.js: ~5KB gzipped
- TinyGo WASM: ~8-10MB gzipped (lazy loaded on first "Run")

**Total without WASM**: ~20KB (acceptable)
**Total with WASM**: ~10MB (large but acceptable for tutorial site)

**Optimization**: Lazy load TinyGo WASM on first code execution

### Compilation Time

**Server Code**: Pre-compiled on server startup (one-time cost)
**Student Code**: 2-5 seconds first compile, faster with caching

**UX**: Show loading indicator during compilation

### Memory

**Server**: One PageState per connection (WebSocket)
**Client**: WASM compiler + runtime in browser (~50-100MB)

## Deployment

### Development

```bash
livepage serve
```

### Production

**Option 1**: Run livepage server
```bash
livepage serve --port 8080
```

**Option 2**: Static export (future)
```bash
livepage build --output ./dist
```

Generates static HTML + JavaScript. WASM blocks still work, interactive blocks require server.

## Implementation Phases

### Phase 1: Core Library
- Markdown parser (goldmark)
- Block type registry
- Page state management
- Message multiplexing
- Livetemplate integration

**Deliverable**: Can parse .md files and create Page objects

### Phase 2: CLI Tool
- `serve` command
- Auto-discovery
- Route generation
- Hot reload
- Built-in theme

**Deliverable**: `livepage serve` works, can view static tutorials

### Phase 3: Client Runtime
- Message router
- Interactive blocks (livetemplate client)
- WASM blocks (compilation + execution)
- Simple code editor
- Output panel

**Deliverable**: Full interactivity - both interactive and WASM blocks work

### Phase 4: Testing
- E2E browser tests
- Unit tests
- Integration tests
- Test infrastructure

**Deliverable**: Comprehensive test coverage

### Phase 5: Documentation
- Example tutorials
- Authoring guide
- Deployment guide
- API reference

**Deliverable**: Ready for external users

## Open Questions

### 1. TinyGo WASM Compilation in Browser

**Question**: How feasible is running TinyGo compiler as WASM in browser?

**Research Needed**:
- TinyGo WASM build size
- Compilation time benchmarks
- Memory requirements
- Browser compatibility

**Alternatives**:
- Server-side compilation with strong sandboxing
- Pre-compiled WASM for common examples
- Hybrid: simple examples compiled, complex ones require server

### 2. Markdown Parser

**Question**: Which markdown parser?

**Options**:
- goldmark (extensible, widely used)
- blackfriday (older, simpler)
- custom (overkill)

**Recommendation**: goldmark

### 3. Syntax Highlighting

**Server-side**: chroma (Go library)
**Client-side**: Prism.js (lightweight)

Use both: server for readonly blocks, client for editable blocks

## Success Criteria

1. ✅ `livepage serve` works on directory of .md files
2. ✅ Interactive blocks update in real-time
3. ✅ WASM blocks execute entirely client-side
4. ✅ Zero config for basic use
5. ✅ Hot reload during development
6. ✅ Comprehensive e2e test coverage
7. ✅ Example tutorials demonstrate all features
8. ✅ Documentation enables external authors

## Appendix: Example Tutorial Structure

See PROGRESS.md for complete counter tutorial example.

## References

- [LiveTemplate Repository](https://github.com/livetemplate/livetemplate)
- [TinyGo WASM](https://tinygo.org/docs/guides/webassembly/)
- [goldmark](https://github.com/yuin/goldmark)
- [Prism.js](https://prismjs.com/)
