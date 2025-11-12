# Livepage Implementation Progress

**Last Updated**: 2025-11-12

## Project Overview

Livepage is a CLI tool for creating interactive documentation (tutorials, guides, playgrounds) using markdown files with embedded executable code blocks, powered by livetemplate.

## Design Document

See [docs/plans/2025-11-12-livepage-design.md](docs/plans/2025-11-12-livepage-design.md) for complete design.

## Implementation Status

### Phase 1: Project Setup ✅
- [x] Create GitHub repository (livetemplate/livepage)
- [x] Initialize Go module
- [x] Create directory structure
- [x] Write progress tracker
- [x] Write design document
- [x] Create README.md
- [x] Initial commit

### Phase 2: Core Library 🚧
- [x] Markdown parser with code block extraction
  - [x] Parse frontmatter (title, type, persist)
  - [x] Extract code blocks with metadata
  - [x] Identify block types (server, wasm, lvt)
  - [x] Parse block attributes (id, state reference)
- [x] Block type registry
  - [x] ServerBlock implementation
  - [x] WasmBlock implementation
  - [x] InteractiveBlock implementation
- [x] Page builder (ParseFile)
  - [x] Convert code blocks to typed blocks
  - [x] Block ID generation (explicit and auto)
  - [x] Reference validation
- [ ] Page state management
  - [ ] PageState struct
  - [ ] Code block state tracking
  - [ ] Interactive block state tracking
- [ ] Message multiplexing
  - [ ] Envelope format (blockID, action, data)
  - [ ] Message router
  - [ ] Block-specific action handling
- [ ] Integration with livetemplate
  - [ ] Mini template instances per interactive block
  - [ ] State struct instantiation
  - [ ] WebSocket multiplexing

### Phase 3: CLI Tool 📝
- [ ] Command structure
  - [ ] `serve` command
  - [ ] `new` command (scaffold generator)
  - [ ] `version` command
- [ ] Dev server
  - [ ] Auto-discovery (scan .md files)
  - [ ] Route generation
  - [ ] Static markdown → HTML conversion
  - [ ] Hot reload (file watcher)
  - [ ] Built-in theme serving
- [ ] Config file support
  - [ ] Parse livepage.yaml
  - [ ] Apply configuration
- [ ] Error handling
  - [ ] Friendly error pages
  - [ ] File location in errors
  - [ ] Helpful suggestions

### Phase 4: Client Runtime 🖥️
- [ ] Core client
  - [ ] Extend livetemplate-client
  - [ ] Message router (multiplex by blockID)
  - [ ] Persistence manager (localStorage)
- [ ] Code blocks
  - [ ] Code block manager
  - [ ] Simple editor (textarea with syntax highlighting)
  - [ ] Output panel component
- [ ] WASM execution
  - [ ] TinyGo WASM compiler integration
  - [ ] Client-side compilation
  - [ ] Execution sandbox
  - [ ] Output capture (stdout/stderr)
- [ ] Interactive blocks
  - [ ] Block connector (WebSocket multiplexing)
  - [ ] DOM update coordination

### Phase 5: Testing 🧪
- [ ] E2E browser tests (chromedp)
  - [ ] Counter tutorial test
  - [ ] Interactive block updates
  - [ ] WASM block execution
  - [ ] Step navigation
  - [ ] Capture browser console logs
  - [ ] Capture server logs
  - [ ] Capture WebSocket messages
  - [ ] Capture rendered HTML
- [ ] Unit tests
  - [ ] Markdown parser tests
  - [ ] Block registry tests
  - [ ] Message router tests
  - [ ] State management tests
- [ ] Integration tests
  - [ ] Full page lifecycle
  - [ ] Multi-block interactions
  - [ ] Persistence tests

### Phase 6: Documentation & Examples 📚
- [ ] Example tutorials
  - [ ] Counter tutorial (basic)
  - [ ] Todo app (CRUD operations)
  - [ ] Chat app (broadcasting)
- [ ] Documentation
  - [ ] README with quick start
  - [ ] Tutorial authoring guide
  - [ ] Code block reference
  - [ ] Configuration reference
  - [ ] Deployment guide

## Current Session Goals

**Session 1 (2025-11-12)**: Project setup and design documentation
- [x] Brainstorm and refine design
- [x] Create GitHub repository
- [x] Initialize project structure
- [ ] Write complete design document
- [ ] Create initial commit

## Next Steps

1. Complete design document (docs/plans/2025-11-12-livepage-design.md)
2. Write README.md with project overview
3. Create initial commit and push to GitHub
4. Begin Phase 2: Start with markdown parser implementation

## Notes

### Key Design Decisions
- **Dual execution model**: Author code runs on server (trusted), student code in browser (WASM, sandboxed)
- **Multiplexed WebSocket**: Single connection for all blocks, messages tagged by blockID
- **CLI-focused**: Not a library - `livepage serve` is the primary interface
- **Zero config**: Built-in theme, auto-discovery, works out of the box
- **Hybrid architecture**: Static markdown cached as HTML, code blocks are dynamic livetemplate components

### Dependencies
- `github.com/livetemplate/livetemplate` - Core reactivity engine
- Markdown parser (goldmark or similar)
- Syntax highlighter
- TinyGo for WASM compilation (client-side)

### Open Questions
- [ ] Which markdown parser? (goldmark is popular and extensible)
- [ ] Syntax highlighter library? (chroma for server-side, prism.js for client?)
- [ ] How to bundle TinyGo WASM compiler for browser? (need to research)
