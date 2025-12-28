# Tinkerdown Roadmap: Micro Apps & Tooling Platform

## Vision

Transform Tinkerdown into a **full-featured platform for building micro apps and internal tooling** using markdown as the primary interface.

**Target Users:**
- Developers building internal tools, dashboards, and admin panels
- Technical writers creating interactive documentation
- Teams needing quick data-driven apps without full-stack complexity
- AI systems generating functional apps from natural language

---

## Priority Framework

| Level | Criteria | Focus |
|-------|----------|-------|
| **P0** | Blocks core functionality or causes data loss | Immediate |
| **P1** | High-impact features enabling new use cases | Near-term |
| **P2** | Developer experience and productivity | Medium-term |
| **P3** | Polish, optimization, and edge cases | Ongoing |

---

## Migration Guide (v0.x → v1.0)

If you're upgrading from an earlier version of Tinkerdown that used Go plugin compilation, this guide covers the key changes.

### Architecture Changes

**Before (v0.x):** Custom logic required compiling Go plugins at runtime.

**After (v1.0):** GenericState handles all standard CRUD operations; custom logic uses WASM modules.

### Migration Steps

1. **Remove plugin compilation**
   - Delete `*.so` files and plugin source directories
   - Remove `plugin:` references from source configurations

2. **Convert custom sources to WASM**
   - Rewrite custom Go plugins as TinyGo WASM modules
   - Update source configuration:
     ```yaml
     # Before
     sources:
       custom:
         type: plugin
         path: ./plugins/custom.so

     # After
     sources:
       custom:
         type: wasm
         module: ./wasm/custom.wasm
     ```
   - See `docs/wasm-sources.md` for WASM module interface

3. **Update SQLite sources**
   - SQLite source is now built-in with full CRUD support
   - Remove custom plugins that only wrapped SQLite:
     ```yaml
     sources:
       tasks:
         type: sqlite
         db: ./tasks.db
         table: tasks
         readonly: false  # Enables Add, Update, Delete, Toggle
     ```

4. **Verify standard actions work**
   - `Add`, `Update`, `Delete`, `Toggle`, `Refresh` are built-in
   - Remove custom action handlers for these operations

### Breaking Changes

| Feature | v0.x | v1.0 |
|---------|------|------|
| Custom logic | Go plugins (`.so`) | WASM modules (`.wasm`) |
| Plugin compilation | Runtime | Build-time (TinyGo) |
| SQLite support | Plugin required | Built-in |
| Cross-platform | Limited (CGO) | Full (pure Go + wazero) |

### Benefits of Migration

- **Cross-platform builds:** No CGO dependency, works on all platforms
- **Faster startup:** No plugin compilation at runtime
- **Simpler deployment:** Single binary with embedded WASM
- **Better sandboxing:** WASM modules run in isolated environment

---

## Phase 1: Zero-Template Development (P0)

### Value Proposition
> "Build interactive apps without learning Go templates"

The biggest barrier to adoption is requiring users to learn Go's `html/template` syntax. This phase introduces `lvt-*` attributes as **compile-time sugar** that transforms to Go templates during parsing. Server-side rendering remains the core—the client stays simple.

### Implementation Approach

**Where:** Core LiveTemplate library (not Tinkerdown-specific)

This ensures the same `lvt-*` attributes work for both:
- **Full apps** built directly with LiveTemplate
- **Micro apps** built with Tinkerdown

```
lvt-* attributes in HTML
        ↓ (LiveTemplate core: parse time)
Transform to Go templates
        ↓
Server-side rendering (unchanged)
        ↓
HTML to client
```

**Benefits:**
- Single implementation in core library
- Consistent syntax across full and micro apps
- No new rendering engine
- Client remains lightweight
- SSR advantages preserved (security, performance, no JS required)

### Current Attribute Ownership

**Already in `@livetemplate/client` (core):**

*Event Handling:*
- `lvt-click`, `lvt-submit`, `lvt-change` - Event handlers
- `lvt-click-away` - Click outside detection
- `lvt-key` - Keyboard key filtering
- `lvt-throttle`, `lvt-debounce` - Rate limiting
- `lvt-window-{event}` - Window-level events

*Lifecycle & Reactive:*
- `lvt-{action}-on:{event}` - Lifecycle hooks (reset, addClass, disable, etc.)
- `lvt-confirm` - Confirmation dialogs
- `lvt-data-*`, `lvt-value-*` - Data extraction
- `lvt-preserve` - Preserve form fields during DOM updates
- `lvt-disable-with` - Button text during submit

*UI Directives:*
- `lvt-scroll` - Scroll behavior (bottom, top, sticky)
- `lvt-highlight` - Flash highlight animation
- `lvt-animate` - Entry animations (fade, slide, scale)
- `lvt-autofocus` - Auto-focus on visibility
- `lvt-focus-trap` - Modal focus trapping
- `lvt-modal-open`, `lvt-modal-close` - Modal controls

**New for Core (server-side template sugar):**
- `lvt-for`, `lvt-if`, `lvt-text` - Loop/conditional/text binding
- `lvt-checked`, `lvt-disabled`, `lvt-selected` - Boolean attributes
- `lvt-class` - Dynamic class binding

**Tinkerdown-specific (micro-app data binding):**
- `lvt-source` - Data source binding
- `lvt-columns`, `lvt-actions` - Auto-table generation
- `lvt-value`, `lvt-label` - Select field mapping (for lvt-source)
- `lvt-template`, `lvt-empty` - Custom rendering

**⚠️ Cleanup: Remove duplicates from Tinkerdown client**
- `lvt-click`, `lvt-submit`, `lvt-change` handlers already in core

---

### 1.1 Auto-Rendering Components

**Problem:** Users must write `{{range .Data}}...{{end}}` loops manually.

**Solution:** HTML elements that auto-render from data sources:

```html
<!-- Instead of writing Go template loops -->
<table lvt-source="tasks" lvt-columns="done,text,priority">
  <!-- Auto-generates thead, tbody, and rows -->
</table>

<ul lvt-source="items" lvt-template="card">
  <!-- Auto-generates list items -->
</ul>

<select lvt-source="categories" lvt-value="id" lvt-label="name">
  <!-- Auto-populates options -->
</select>
```

**Work Required:**
- [ ] `<table lvt-source>` auto-generates full table from data
- [ ] `<ul/ol lvt-source>` auto-generates list items
- [ ] `<select lvt-source>` already works - document it
- [ ] `lvt-template` attribute for card/row/custom layouts
- [ ] `lvt-columns` for table column selection and ordering
- [ ] `lvt-empty` for empty state message

**Impact:** 80% of apps need no template knowledge

---

### 1.2 Declarative Attributes (Alpine.js-style)

**Problem:** Go template syntax `{{if .Done}}checked{{end}}` is unfamiliar.

**Solution:** HTML attributes that express logic declaratively:

```html
<!-- Instead of Go templates -->
<tr lvt-for="item in tasks">
  <td lvt-text="item.text"></td>
  <td lvt-if="item.done">✓</td>
  <td lvt-class="{'completed': item.done}"></td>
  <input type="checkbox" lvt-checked="item.done">
  <button lvt-click="Delete" lvt-data-id="item.id">Delete</button>
</tr>

<!-- Conditionals -->
<div lvt-if="error" class="error" lvt-text="error"></div>
<div lvt-if="!data.length">No items yet</div>
```

**Transformations:**
```
lvt-for="item in tasks"     → {{range .tasks}}...{{end}}
lvt-text="item.text"        → {{.text}}
lvt-if="item.done"          → {{if .done}}...{{end}}
lvt-checked="item.done"     → {{if .done}}checked{{end}}
lvt-class="done: item.done" → class="{{if .done}}done{{end}}"
```

**Work Required (in LiveTemplate core):**
- [ ] `lvt-for="item in source"` - Loop over data
- [ ] `lvt-text="field"` - Set text content
- [ ] `lvt-if="condition"` - Conditional rendering
- [ ] `lvt-checked`, `lvt-disabled`, `lvt-selected` - Boolean attributes
- [ ] `lvt-class="name: condition"` - Dynamic classes
- [ ] Attribute transformer in LiveTemplate's template processing

**Impact:** Familiar syntax for frontend developers, zero runtime overhead, works in both full and micro apps

---

### 1.3 Form Auto-Binding

**Problem:** Forms require manual wiring of inputs to actions.

**Solution:** Smart form components that infer behavior:

```html
<!-- Auto-generates Add form from source schema -->
<form lvt-source="tasks" lvt-action="Add">
  <!-- Auto-creates inputs based on table columns -->
</form>

<!-- Or explicit but simple -->
<form lvt-submit="Add" lvt-source="tasks">
  <input lvt-field="text" placeholder="Task...">
  <select lvt-field="priority" lvt-options="low,medium,high">
  <button type="submit">Add</button>
</form>
```

**Work Required:**
- [ ] `lvt-field` auto-binds input name and validation
- [ ] `lvt-options` for simple select options
- [ ] Form schema inference from SQLite tables
- [ ] Built-in validation display

**Impact:** CRUD apps in minutes without template code

---

### 1.4 Markdown-Native Data Binding

**Problem:** Users want to stay in markdown, not write HTML.

**Solution:** Extend markdown syntax for data display:

```markdown
## Tasks

<!-- Markdown table that binds to source -->
| Done | Task | Priority |
|------|------|----------|
{tasks}

<!-- Or a simple list -->
- {tasks: text} ({tasks: priority})
```

**Work Required:**
- [ ] `{source}` syntax in markdown tables
- [ ] `{source: field}` for inline field access
- [ ] Automatic table generation from source
- [ ] List binding syntax

**Impact:** True markdown-first development

---

## Phase 2: Stability & Performance (P0)

### Value Proposition
> "Make what exists work reliably in production"

### 2.1 Data Source Error Handling
**Files:** `internal/source/*.go`, `internal/runtime/state.go`

**Current State:** Errors can crash or hang; no retry logic; silent failures possible.

**Work Required:**
- [ ] Unified error types for all sources
- [ ] Retry with exponential backoff for transient failures
- [ ] Circuit breaker for repeatedly failing sources
- [ ] User-friendly error messages in templates (`.Error` field rendering)
- [ ] Timeout configuration per source

**Impact:** Production reliability for data-driven apps

---

### 1.2 Source Caching Layer
**Current State:** Every page view refetches all data.

**Work Required:**
- [ ] Cache configuration per source:
  ```yaml
  sources:
    users:
      type: rest
      url: https://api.example.com/users
      cache:
        ttl: 5m
        strategy: stale-while-revalidate
  ```
- [ ] In-memory cache with TTL
- [ ] Cache invalidation on write operations
- [ ] Manual cache clear via `Refresh` action

**Impact:** 10x faster page loads; reduced API costs; better UX

---

### 1.3 Multi-Page WebSocket Support
**Files:** `internal/server/server.go`

**Current State:** WebSocket handler only serves first route - multi-page sites limited.

**Work Required:**
- [ ] Accept page identifier via query param or path
- [ ] Route WebSocket messages to correct page's state
- [ ] Handle page transitions gracefully
- [ ] Clean up state on page navigation

**Impact:** Enables documentation sites with interactive examples on every page

---

## Phase 3: Developer Experience (P1)

### Value Proposition
> "Reduce time from idea to working app by 10x"

### 2.1 Enhanced CLI Scaffolding
**Files:** `cmd/tinkerdown/commands/new.go`

**Current State:** `new` command creates minimal template only.

**Work Required:**
- [ ] Add `--template` flag with options:
  - `basic` - Minimal with one source
  - `todo` - SQLite CRUD with toggle/delete
  - `dashboard` - Multi-source data display
  - `form` - Contact form with SQLite persistence
  - `api-explorer` - REST source with refresh
  - `cli-wrapper` - Exec source with argument form
  - `wasm-source` - Template for building custom WASM sources
- [ ] Generate sample data files for each template
- [ ] Include inline documentation comments

**Impact:** 5-minute start to working prototype

---

### 2.2 Expanded Validation
**Files:** `cmd/tinkerdown/commands/validate.go`

**Current State:** Only validates markdown parsing and Mermaid syntax.

**Work Required:**
- [ ] Validate source references exist in config
- [ ] Check `lvt-*` attributes have valid values
- [ ] Verify source types are valid (exec, pg, rest, json, csv, markdown, sqlite, wasm)
- [ ] Warn on unused source definitions
- [ ] Validate WASM module paths exist
- [ ] Type-check common template patterns

**Impact:** Catch errors at write-time, not runtime

---

### 2.3 Debug Mode & Logging
**Files:** `internal/server/server.go`

**Work Required:**
- [ ] `--debug` / `--verbose` CLI flags
- [ ] Structured JSON logging option
- [ ] Request correlation IDs
- [ ] WebSocket message logging (with sensitive data redaction)
- [ ] State change logging
- [ ] Source fetch timing

**Impact:** 10x faster debugging of production issues

---

### 2.4 Hot Reload for Configuration
**Current State:** Config changes require server restart.

**Work Required:**
- [ ] Watch `tinkerdown.yaml` for changes
- [ ] Reload sources without dropping WebSocket connections
- [ ] Notify connected clients of config reload
- [ ] Support frontmatter changes via file watcher

**Impact:** Faster iteration on source configuration

---

### 2.5 WASM Source Development Kit
**New Feature** - Critical for ecosystem growth

**Work Required:**
- [ ] `tinkerdown wasm init <name>` - Scaffold new WASM source
- [ ] `tinkerdown wasm build` - Compile TinyGo source to WASM
- [ ] `tinkerdown wasm test` - Test WASM module locally
- [ ] Documentation for WASM interface contract
- [ ] Example sources: GitHub API, Notion, Airtable

**Impact:** Enable community source contributions

---

## Phase 4: Data Ecosystem (P1)

### Value Proposition
> "Connect to any data source in minutes, not days"

### 3.1 GraphQL Source
**Work Required:**
- [ ] New source type: `graphql`
- [ ] Config: `url`, `query`, `variables`
- [ ] Authentication headers
- [ ] Auto-flatten nested response
- [ ] Support for mutations via write operations

**Impact:** Modern API ecosystem support

---

### 3.2 MongoDB Source
**Work Required:**
- [ ] New source type: `mongodb`
- [ ] Config: `uri`, `database`, `collection`, `filter`
- [ ] Pure Go driver (no cgo)
- [ ] CRUD operations support

**Impact:** NoSQL database support

---

### 3.3 Source Composition
**New Feature**

**Work Required:**
- [ ] Computed sources that transform other sources:
  ```yaml
  sources:
    users:
      type: rest
      url: https://api.example.com/users
    active_users:
      type: computed
      from: users
      filter: "status == 'active'"
      sort: "name asc"
  ```
- [ ] Join sources on common fields
- [ ] Aggregation (count, sum, avg)

**Impact:** Complex data apps without custom code

---

### 3.4 Webhook Source
**Work Required:**
- [ ] Source that receives HTTP POST
- [ ] Store latest N events
- [ ] Trigger UI update on new data
- [ ] Optional signature verification (Stripe, GitHub)

**Impact:** Real-time integrations (webhooks, events)

---

### 3.5 S3/Cloud Storage Source
**Work Required:**
- [ ] New source type: `s3`
- [ ] List objects, read JSON/CSV files
- [ ] Support for GCS, Azure Blob (compatible APIs)
- [ ] Credentials via environment variables

**Impact:** Cloud-native data access

---

## Phase 5: Production Readiness (P1)

### Value Proposition
> "Deploy with confidence to real users"

### 4.1 Authentication Middleware
**Work Required:**
- [ ] Built-in auth strategies:
  - API key (header-based)
  - Basic auth
  - OAuth2 (Google, GitHub)
  - Custom JWT validation
- [ ] Per-page auth requirements in frontmatter:
  ```yaml
  auth: required
  # or
  auth:
    provider: github
    allowed_orgs: [mycompany]
  ```
- [ ] User context available in templates (`{{.User.Email}}`)

**Impact:** Secure internal tools; multi-user apps

---

### 4.2 Request Rate Limiting
**Work Required:**
- [ ] Per-IP rate limiting
- [ ] Per-source rate limiting (protect external APIs)
- [ ] Configurable limits per endpoint
- [ ] Graceful 429 responses with retry-after

**Impact:** Protection against abuse; resource management

---

### 4.3 Health & Metrics Endpoints
**Work Required:**
- [ ] `/health` - Basic liveness check
- [ ] `/ready` - Readiness including source connectivity
- [ ] `/metrics` - Prometheus-compatible metrics:
  - Request count/latency by route
  - WebSocket connection count
  - Source fetch latency/error rates
  - WASM execution time

**Impact:** Kubernetes-ready deployment; observability

---

### 4.4 Graceful Shutdown
**Work Required:**
- [ ] Track in-flight requests
- [ ] Drain WebSocket connections
- [ ] Complete pending source operations
- [ ] Close WASM runtimes cleanly
- [ ] Configurable shutdown timeout

**Impact:** Zero-downtime deployments

---

### 4.5 Single Binary Distribution
**Work Required:**
- [ ] Embed client assets in Go binary
- [ ] `tinkerdown build <dir>` command producing standalone binary
- [ ] Cross-compilation support (linux, darwin, windows)
- [ ] Docker image generation

**Impact:** Simple deployment; Docker images <50MB

---

### 4.6 Security Hardening

**Work Required:**

**CSRF Protection:**
- [ ] Generate CSRF tokens for all forms
- [ ] Validate tokens on POST/PUT/DELETE requests
- [ ] Auto-inject token into `lvt-submit` forms
- [ ] Cookie-based double-submit pattern for WebSocket actions

**Content Security Policy (CSP):**
- [ ] Default restrictive CSP headers
- [ ] Configurable CSP via `tinkerdown.yaml`:
  ```yaml
  security:
    csp:
      default-src: "'self'"
      script-src: "'self' 'unsafe-inline'"  # Required for inline event handlers
      style-src: "'self' 'unsafe-inline'"
      connect-src: "'self' ws: wss:"
  ```
- [ ] Nonce-based script loading for stricter environments

**Input Sanitization:**
- [ ] HTML escape all template output by default (Go templates do this)
- [ ] Validate source configuration values at startup
- [ ] Sanitize file paths in `exec` source to prevent path traversal
- [ ] SQL parameterization enforced in SQLite source (already implemented)
- [ ] Validate and sanitize REST API URL templates

**WASM Resource Limits:**
- [ ] Memory limits per WASM module (default: 64MB)
- [ ] CPU time limits per execution (default: 30s)
- [ ] Configuration via source definition:
  ```yaml
  sources:
    custom_source:
      type: wasm
      module: ./custom.wasm
      limits:
        memory: 128MB
        timeout: 60s
  ```
- [ ] Graceful termination on limit exceeded
- [ ] Logging of resource usage for debugging

**Impact:** Secure deployment for internal tools; defense in depth

---

## Phase 6: UI & Components (P2)

### Value Proposition
> "Beautiful apps without CSS expertise"

### 5.1 Component Library
**Work Required:**
- [ ] Chart component (line, bar, pie via Chart.js or similar)
- [ ] Modal/dialog component with `lvt-modal`
- [ ] Toast notifications for action feedback
- [ ] File upload with drag-and-drop
- [ ] Tree view for hierarchical data
- [ ] Tabs component
- [ ] Accordion/collapsible sections

**Impact:** Rich UIs without custom code

---

### 5.2 Built-in Pagination
**Current State:** Must render all data; large datasets slow.

**Work Required:**
- [ ] `lvt-paginate="20"` attribute on containers
- [ ] Auto-generate prev/next controls
- [ ] Server-side pagination for sources
- [ ] URL-based page state for bookmarkability

**Impact:** Apps handling 10k+ records

---

### 5.3 Built-in Sorting & Filtering
**Work Required:**
- [ ] `lvt-sortable` attribute on tables
- [ ] `lvt-filter="field"` for search input
- [ ] Client-side for small datasets (<1000 rows)
- [ ] Server-side for large datasets

**Impact:** Usable data tables out of the box

---

### 5.4 Theme System Expansion
**Current State:** Only "clean" theme fully implemented.

**Work Required:**
- [ ] Complete "dark" and "minimal" themes
- [ ] Custom theme via CSS variables:
  ```yaml
  styling:
    theme: custom
    primary_color: "#007bff"
    background: "#1a1a2e"
  ```
- [ ] Dark mode toggle component
- [ ] Per-page theme override in frontmatter

**Impact:** Brand customization; accessibility

---

### 5.5 UX Improvements

**Navigation Enhancements:**
- [ ] Sticky table of contents in right sidebar (auto-generated from headings)
- [ ] Previous/Next page navigation buttons at bottom of content
- [ ] Sidebar collapse/expand toggle for small screens
- [ ] Active page indicator with visual highlighting
- [ ] Breadcrumb navigation for deep hierarchies

**Code Block Improvements:**
- [ ] Copy-to-clipboard button on all code blocks
- [ ] Syntax highlighting for 20+ languages
- [ ] Line numbers (optional, configurable)
- [ ] Code block titles/filenames

**Loading & Feedback:**
- [ ] Loading skeleton screens during data fetch
- [ ] Smooth transitions between page states
- [ ] Toast notifications for action success/failure
- [ ] Progress indicators for long operations

**Search Enhancements:**
- [ ] Extended search result previews (120+ characters)
- [ ] Keyboard navigation in search results (arrow keys, Enter)
- [ ] Search result highlighting in content
- [ ] Recent searches history

**Mobile Experience:**
- [ ] Responsive sidebar with swipe gestures
- [ ] Touch-friendly interactive elements
- [ ] Optimized table scrolling on mobile

**Impact:** Professional documentation sites with excellent user experience

---

## Phase 7: Advanced Features (P2)

### Value Proposition
> "Handle complex real-world scenarios"

### 6.1 Multi-User State Broadcasting
**Work Required:**
- [ ] Shared state mode for collaborative apps:
  ```yaml
  sources:
    tasks:
      type: sqlite
      broadcast: true  # Sync across all connected clients
  ```
- [ ] Broadcast state changes to all connected clients
- [ ] Conflict resolution strategies (last-write-wins, merge)
- [ ] Presence indicators (who's viewing)

**Impact:** Real-time collaborative tools

---

### 6.2 Scheduled Tasks
**Work Required:**
- [ ] Cron-like syntax in config:
  ```yaml
  schedules:
    refresh_data:
      cron: "*/5 * * * *"
      source: external_api
      action: Refresh
  ```
- [ ] Background execution without user connection
- [ ] Error notifications via webhook

**Impact:** Data refresh without user interaction

---

### 6.3 API Endpoint Mode
**Work Required:**
- [ ] Expose sources as REST endpoints:
  ```yaml
  api:
    enabled: true
    prefix: /api/v1
    sources:
      - name: tasks
        methods: [GET, POST, PUT, DELETE]
  ```
- [ ] OpenAPI spec generation
- [ ] API key authentication

**Impact:** Backend services from markdown definitions

---

### 6.4 Offline Support
**Work Required:**
- [ ] Service worker for asset caching
- [ ] Offline indicator component
- [ ] Queue actions when offline
- [ ] Sync when reconnected
- [ ] Conflict resolution UI

**Impact:** Reliable mobile/field usage

---

### 6.5 WASM Source Marketplace
**Work Required:**
- [ ] Registry of community WASM sources
- [ ] `tinkerdown source add <name>` command
- [ ] Versioning and updates
- [ ] Security review process
- [ ] Documentation site for source authors

**Impact:** Rich ecosystem without core development

---

## Phase 8: Polish & Optimization (P3)

### Value Proposition
> "Production-grade performance and reliability"

### 7.1 Bundle Size Optimization
**Current State:** Client bundle includes Monaco for code blocks.

**Work Required:**
- [ ] Lazy load heavy components (Monaco, charts)
- [ ] Tree-shake unused features
- [ ] Alternative lightweight code viewer
- [ ] Target <200KB initial bundle

---

### 7.2 Accessibility Audit
**Work Required:**
- [ ] WCAG 2.1 AA compliance
- [ ] Full keyboard navigation
- [ ] Screen reader testing
- [ ] ARIA attributes for all components
- [ ] Focus management on page transitions
- [ ] Color contrast validation

---

### 7.3 Performance Profiling
**Work Required:**
- [ ] Built-in performance tracing
- [ ] Slow source warnings (>500ms)
- [ ] Memory usage monitoring for WASM
- [ ] WebSocket message size optimization
- [ ] Template rendering performance

---

### 7.4 Comprehensive Test Suite
**Work Required:**
- [ ] Unit tests for all sources (including SQLite, WASM)
- [ ] Integration tests for GenericState action dispatch
- [ ] Cross-browser E2E tests
- [ ] Performance regression tests
- [ ] WASM source contract tests

---

## Implementation Priorities

```
Priority Order (based on user impact and dependencies):

HIGH IMPACT, LOW EFFORT (Quick Wins)
├── 2.3 Debug mode CLI flag
├── 2.2 Source reference validation
├── 2.1 CLI templates (todo, dashboard)
└── 5.5 Code copy buttons

HIGH IMPACT, MEDIUM EFFORT (Core Features)
├── 1.2 Source caching
├── 1.1 Error handling improvements
├── 2.5 WASM source dev kit
├── 4.3 Health endpoints
└── 3.1 GraphQL source

HIGH IMPACT, HIGH EFFORT (Major Features)
├── 4.1 Authentication
├── 6.1 Multi-user broadcasting
├── 4.5 Single binary distribution
└── 6.3 API endpoint mode

MEDIUM IMPACT (Nice to Have)
├── 5.1-5.4 Component library
├── 3.3 Source composition
├── 6.2 Scheduled tasks
└── 7.1-7.4 Polish items
```

---

## Success Metrics

### Phase 1-2 Complete
**Functionality:**
- [ ] All 8 example apps work reliably
- [ ] Zero crashes on source errors (graceful degradation)
- [ ] New app scaffolded in <1 minute
- [ ] 90% of config errors caught at validation time

**Testing Requirements:**
- [ ] Unit tests for all new lvt-* attribute transformations
- [ ] E2E tests for auto-rendering components (table, list, select)
- [ ] Integration tests for error handling and retry logic
- [ ] Test coverage ≥70% for new code

### Phase 3-4 Complete
**Functionality:**
- [ ] Apps load <500ms with caching
- [ ] 8+ data source types available
- [ ] Apps deployable to Kubernetes with health checks
- [ ] Auth-protected internal tools working

**Testing Requirements:**
- [ ] E2E tests for each new data source type
- [ ] Integration tests for caching layer
- [ ] Security tests for authentication middleware
- [ ] Performance benchmarks for source operations
- [ ] Test coverage ≥80% for core modules

### Phase 5-6 Complete
**Functionality:**
- [ ] 10+ reusable components
- [ ] 10k row tables render smoothly
- [ ] Real-time collaborative demo working
- [ ] REST API generation from sources

**Testing Requirements:**
- [ ] E2E browser tests for all UI components
- [ ] Cross-browser testing (Chrome, Firefox, Safari)
- [ ] Accessibility tests (axe-core integration)
- [ ] Load tests for pagination and large datasets
- [ ] Test coverage ≥85% for component library

### Phase 7-8 Complete
**Functionality:**
- [ ] <200KB initial bundle size
- [ ] WCAG 2.1 AA compliant
- [ ] Production-ready security hardening

**Testing Requirements:**
- [ ] Security audit tests (CSRF, CSP validation)
- [ ] WASM resource limit tests
- [ ] Performance regression test suite
- [ ] Full accessibility audit with manual testing
- [ ] Test coverage ≥95% on core (unit + integration + E2E)

---

## Quick Wins (Start Immediately)

These high-impact, low-effort items can be tackled immediately:

**Zero-Template (Highest Priority):**
1. **Auto-table rendering** - `<table lvt-source="x" lvt-columns="a,b,c">` generates full table
2. **Document existing `<select lvt-source>`** - Already works, just undocumented
3. **`lvt-for` attribute** - Simple loop syntax without Go templates
4. **`lvt-text` attribute** - Set text content from field

**Developer Experience:**
5. **Add `--debug` flag** - Expose debug logging via CLI
6. **CLI templates** - Add todo and dashboard templates to `new` command
7. **Source reference validation** - Check sources exist in validate command

---

## Contributing

See `CONTRIBUTING.md` for development setup and guidelines.

Each feature should have:
1. Design document before implementation
2. Tests covering new functionality
3. Documentation updates
4. Example apps demonstrating usage

---

# Detailed Implementation Plans

## Phase 1: Zero-Template Development - Implementation Plan

### 1.1 Auto-Rendering Components

#### Implementation Steps

**Step 1: Enhance table auto-generation (`page.go`)**
```go
// Extend autoGenerateTableTemplate() to handle lvt-columns parsing
// Input: <table lvt-source="tasks" lvt-columns="done:Done,text:Task,priority:Priority">
// Output: Full datatable template with headers and row iteration
```

- [ ] Parse `lvt-columns="field:Label,field:Label"` format
- [ ] Generate `<thead>` with column labels
- [ ] Generate `<tbody>` with `{{range .Data}}` iteration
- [ ] Support `lvt-actions="delete:Delete,toggle:Toggle"` for action columns
- [ ] Handle empty state with `lvt-empty="No items yet"`

**Step 2: Add list auto-generation (`page.go`)**
```go
// New function: autoGenerateListTemplate()
// Input: <ul lvt-source="items">
// Output: <ul>{{range .Data}}<li>{{.}}</li>{{end}}</ul>
```

- [ ] Detect `<ul lvt-source>` and `<ol lvt-source>` patterns
- [ ] Support `lvt-template="card"` for custom item templates
- [ ] Default to simple `<li>{{.Name}}</li>` or full object display

**Step 3: Document select auto-population**
- [ ] Verify current `<select lvt-source>` implementation works
- [ ] Document `lvt-value` and `lvt-label` attributes

#### E2E Tests

**File:** `lvt_auto_render_e2e_test.go`

```go
func TestAutoTableGeneration(t *testing.T) {
    // Test: <table lvt-source="x" lvt-columns="a,b,c"> generates correct HTML
}

func TestAutoTableWithActions(t *testing.T) {
    // Test: lvt-actions="delete:Delete" adds action column with buttons
}

func TestAutoListGeneration(t *testing.T) {
    // Test: <ul lvt-source="items"> generates list items
}

func TestAutoTableEmptyState(t *testing.T) {
    // Test: lvt-empty="No data" shows when source returns empty
}

func TestAutoSelectPopulation(t *testing.T) {
    // Test: <select lvt-source> populates options from data
}
```

#### Documentation Updates

- [ ] `docs/auto-rendering.md` - New guide for zero-template components
- [ ] Update `docs/lvt-source.md` with auto-rendering examples
- [ ] Add "Zero-Template Quick Start" to README

#### Example Updates

- [ ] `examples/zero-template-table/` - Table without Go templates
- [ ] `examples/zero-template-list/` - List rendering demo
- [ ] Update `examples/lvt-source-sqlite-test/` to use auto-table

---

### 1.2 Declarative Attributes (lvt-for, lvt-if, lvt-text)

#### Implementation Steps

**Step 1: Create attribute transformer (LiveTemplate core)**

**File:** `livetemplate/template/transform.go`

```go
// TransformDeclarativeAttributes converts lvt-* attributes to Go template syntax
func TransformDeclarativeAttributes(html string) string {
    // 1. Find all lvt-for="item in source" and wrap content in {{range}}
    // 2. Find all lvt-if="condition" and wrap in {{if}}
    // 3. Replace lvt-text="field" with {{.field}}
    // 4. Replace lvt-checked="field" with {{if .field}}checked{{end}}
}
```

- [ ] Implement `lvt-for` → `{{range}}` transformation
- [ ] Implement `lvt-if` → `{{if}}` transformation (including negation `!`)
- [ ] Implement `lvt-text` → `{{.field}}` transformation
- [ ] Implement boolean attributes: `lvt-checked`, `lvt-disabled`, `lvt-selected`
- [ ] Implement `lvt-class="name: condition"` → `class="{{if .condition}}name{{end}}"`

**Step 2: Integrate transformer into template pipeline**
- [ ] Call `TransformDeclarativeAttributes()` before Go template parsing
- [ ] Ensure transformed output is valid Go template syntax
- [ ] Handle nested transformations correctly

**Step 3: Error handling**
- [ ] Provide helpful error messages for invalid syntax
- [ ] Line number references for debugging

#### E2E Tests

**File:** `lvt_declarative_attrs_e2e_test.go`

```go
func TestLvtFor(t *testing.T) {
    // Test: lvt-for="item in tasks" iterates over data
}

func TestLvtForNested(t *testing.T) {
    // Test: Nested lvt-for loops work correctly
}

func TestLvtIf(t *testing.T) {
    // Test: lvt-if="done" shows/hides content
}

func TestLvtIfNegation(t *testing.T) {
    // Test: lvt-if="!error" negation works
}

func TestLvtText(t *testing.T) {
    // Test: lvt-text="name" sets element text content
}

func TestLvtChecked(t *testing.T) {
    // Test: lvt-checked="done" adds checked attribute when true
}

func TestLvtClass(t *testing.T) {
    // Test: lvt-class="completed: done" adds class conditionally
}

func TestLvtCombined(t *testing.T) {
    // Test: Multiple lvt-* attributes on same element
}
```

#### Documentation Updates

- [ ] `docs/declarative-attributes.md` - Full reference for lvt-for/if/text
- [ ] `docs/migration-from-go-templates.md` - Side-by-side comparison
- [ ] Add examples to main README

#### Example Updates

- [ ] `examples/declarative-todo/` - Todo app using only lvt-* attributes
- [ ] `examples/declarative-dashboard/` - Dashboard without Go templates
- [ ] Convert existing examples to use declarative syntax as alternative

---

### 1.3 Form Auto-Binding

#### Implementation Steps

**Step 1: Implement `lvt-field` attribute**
- [ ] Auto-set `name` attribute from `lvt-field` value
- [ ] Add validation attributes based on SQLite column types
- [ ] Support `lvt-field="email"` → `<input name="email" type="email">`

**Step 2: Implement `lvt-options` for simple selects**
- [ ] Parse `lvt-options="low,medium,high"` format
- [ ] Generate `<option>` elements automatically

**Step 3: Form schema inference**
- [ ] Read SQLite table schema for source
- [ ] Generate form fields based on column types
- [ ] Handle `<form lvt-source="x" lvt-action="Add">` auto-generation

#### E2E Tests

**File:** `lvt_form_binding_e2e_test.go`

```go
func TestLvtField(t *testing.T) {
    // Test: lvt-field="email" sets name and infers type
}

func TestLvtOptions(t *testing.T) {
    // Test: lvt-options generates select options
}

func TestFormSchemaInference(t *testing.T) {
    // Test: Form auto-generates from SQLite schema
}

func TestFormSubmission(t *testing.T) {
    // Test: Auto-bound form submits correctly
}
```

#### Documentation Updates

- [ ] `docs/form-binding.md` - Form auto-binding guide
- [ ] Add form examples to quick start

#### Example Updates

- [ ] `examples/auto-form/` - Form generated from schema
- [ ] Update contact form example to use auto-binding

---

### 1.4 Markdown-Native Data Binding

#### Implementation Steps

**Step 1: Extend markdown parser**
- [ ] Detect `{source}` syntax in table body
- [ ] Replace with Go template iteration
- [ ] Handle `{source: field}` for inline field access

**Step 2: Table binding**
```markdown
| Name | Email |
|------|-------|
{users}
```
Transforms to Go template with `{{range .users}}` iteration.

**Step 3: List binding**
```markdown
- {tasks: text}
```
Transforms to `{{range .tasks}}- {{.text}}{{end}}`

#### E2E Tests

**File:** `markdown_binding_e2e_test.go`

```go
func TestMarkdownTableBinding(t *testing.T) {
    // Test: {source} in table generates rows
}

func TestMarkdownListBinding(t *testing.T) {
    // Test: {source: field} in list generates items
}

func TestMarkdownInlineField(t *testing.T) {
    // Test: Inline {source: field} access works
}
```

#### Documentation Updates

- [ ] `docs/markdown-binding.md` - Markdown data binding syntax
- [ ] Update getting started guide

#### Example Updates

- [ ] `examples/pure-markdown-app/` - App using only markdown syntax

---

## Phase 2: Stability & Performance - Implementation Plan

### 2.1 Data Source Error Handling

#### Implementation Steps

**Step 1: Create unified error types (`internal/source/errors.go`)**
```go
type SourceError struct {
    Source    string
    Operation string
    Err       error
    Retryable bool
}

type ConnectionError struct { ... }
type TimeoutError struct { ... }
type ValidationError struct { ... }
```

**Step 2: Implement retry logic (`internal/source/retry.go`)**
- [ ] Exponential backoff: 100ms, 200ms, 400ms, max 3 retries
- [ ] Only retry on `Retryable` errors
- [ ] Log retry attempts

**Step 3: Implement circuit breaker (`internal/source/circuit.go`)**
- [ ] Open circuit after 5 failures in 1 minute
- [ ] Half-open after 30 seconds
- [ ] Close on successful request

**Step 4: Add timeout configuration**
```yaml
sources:
  slow_api:
    type: rest
    url: https://slow.api.com
    timeout: 30s  # Default: 10s
```

#### E2E Tests

**File:** `source_error_handling_e2e_test.go`

```go
func TestSourceRetryOnTransientError(t *testing.T) {
    // Test: Source retries on network error
}

func TestSourceCircuitBreaker(t *testing.T) {
    // Test: Circuit opens after repeated failures
}

func TestSourceTimeout(t *testing.T) {
    // Test: Custom timeout is respected
}

func TestErrorDisplayInTemplate(t *testing.T) {
    // Test: {{.Error}} renders user-friendly message
}
```

#### Documentation Updates

- [ ] `docs/error-handling.md` - Error handling guide
- [ ] Document timeout and retry configuration

#### Example Updates

- [ ] `examples/error-handling/` - Demo of error states and recovery

---

### 2.2 Source Caching Layer

#### Implementation Steps

**Step 1: Create cache interface (`internal/cache/cache.go`)**
```go
type Cache interface {
    Get(key string) ([]map[string]interface{}, bool)
    Set(key string, data []map[string]interface{}, ttl time.Duration)
    Invalidate(key string)
}

type MemoryCache struct { ... }
```

**Step 2: Add cache configuration parsing**
```yaml
sources:
  users:
    type: rest
    cache:
      ttl: 5m
      strategy: stale-while-revalidate
```

**Step 3: Integrate cache into source fetching**
- [ ] Check cache before fetch
- [ ] Store result after successful fetch
- [ ] Invalidate on write operations (Add, Update, Delete)

**Step 4: Implement stale-while-revalidate**
- [ ] Return stale data immediately
- [ ] Fetch fresh data in background
- [ ] Update cache when fresh data arrives

#### E2E Tests

**File:** `source_caching_e2e_test.go`

```go
func TestCacheHit(t *testing.T) {
    // Test: Second request uses cached data
}

func TestCacheTTL(t *testing.T) {
    // Test: Cache expires after TTL
}

func TestCacheInvalidationOnWrite(t *testing.T) {
    // Test: Add/Delete invalidates cache
}

func TestStaleWhileRevalidate(t *testing.T) {
    // Test: Returns stale data while fetching fresh
}
```

#### Documentation Updates

- [ ] `docs/caching.md` - Caching configuration and strategies
- [ ] Add performance tuning section

#### Example Updates

- [ ] Update REST example with caching enabled

---

### 2.3 Multi-Page WebSocket Support

#### Implementation Steps

**Step 1: Add page identifier to WebSocket connection**
- [ ] Accept `?page=/path/to/page` query parameter
- [ ] Route messages to correct page's GenericState
- [ ] Track active connections per page

**Step 2: Handle page transitions**
- [ ] Clean up state when client disconnects
- [ ] Support client-side page navigation without reload

**Step 3: Connection management**
- [ ] Limit connections per page (configurable)
- [ ] Clean up stale connections

#### E2E Tests

**File:** `multipage_websocket_e2e_test.go`

```go
func TestMultiPageWebSocket(t *testing.T) {
    // Test: Two pages have independent WebSocket connections
}

func TestPageTransition(t *testing.T) {
    // Test: Navigating between pages maintains correct state
}

func TestConnectionCleanup(t *testing.T) {
    // Test: Disconnected clients are cleaned up
}
```

#### Documentation Updates

- [ ] Update WebSocket documentation with multi-page support
- [ ] Add architecture diagram

#### Example Updates

- [ ] `examples/docs-site/` - Verify multi-page interactive examples work

---

## Phase 3: Developer Experience - Implementation Plan

### 3.1 Enhanced CLI Scaffolding

#### Implementation Steps

**Step 1: Create template definitions (`cmd/tinkerdown/templates/`)**
```
templates/
├── basic/
│   ├── index.md
│   └── tinkerdown.yaml
├── todo/
│   ├── index.md
│   ├── _data/tasks.md
│   └── tinkerdown.yaml
├── dashboard/
│   ├── index.md
│   └── tinkerdown.yaml
├── form/
│   └── ...
├── api-explorer/
│   └── ...
└── wasm-source/
    ├── source.go
    ├── Makefile
    └── README.md
```

**Step 2: Implement `--template` flag**
```go
// cmd/tinkerdown/commands/new.go
func newCmd() *cobra.Command {
    var template string
    cmd.Flags().StringVar(&template, "template", "basic", "Template: basic, todo, dashboard, form, api-explorer, wasm-source")
}
```

**Step 3: Template file generation**
- [ ] Copy template files to target directory
- [ ] Replace placeholders with project name
- [ ] Generate sample data files

#### E2E Tests

**File:** `cli_scaffolding_e2e_test.go`

```go
func TestNewBasicTemplate(t *testing.T) {
    // Test: tinkerdown new myapp creates basic structure
}

func TestNewTodoTemplate(t *testing.T) {
    // Test: tinkerdown new myapp --template=todo creates todo app
}

func TestNewWasmSourceTemplate(t *testing.T) {
    // Test: tinkerdown new mysource --template=wasm-source creates WASM scaffolding
}

func TestTemplateRuns(t *testing.T) {
    // Test: Generated app starts without errors
}
```

#### Documentation Updates

- [ ] `docs/cli.md` - Document all CLI commands and flags
- [ ] `docs/templates.md` - Template reference
- [ ] Update getting started with template examples

#### Example Updates

- [ ] Templates become canonical examples
- [ ] Link from docs to template source

---

### 3.2 Expanded Validation

#### Implementation Steps

**Step 1: Add source reference validation**
```go
// cmd/tinkerdown/commands/validate.go
func validateSourceReferences(page *tinkerdown.Page, config *config.Config) []ValidationError {
    // Check each lvt-source="name" has matching config entry
}
```

**Step 2: Add attribute validation**
- [ ] Validate `lvt-*` attribute values are well-formed
- [ ] Check source types are valid
- [ ] Warn on unused sources

**Step 3: Add WASM path validation**
- [ ] Check WASM module files exist
- [ ] Validate WASM interface (has required exports)

#### E2E Tests

**File:** `validation_e2e_test.go`

```go
func TestValidateMissingSource(t *testing.T) {
    // Test: Error when lvt-source references undefined source
}

func TestValidateInvalidSourceType(t *testing.T) {
    // Test: Error on invalid source type
}

func TestValidateUnusedSource(t *testing.T) {
    // Test: Warning on defined but unused source
}

func TestValidateMissingWasmFile(t *testing.T) {
    // Test: Error when WASM file doesn't exist
}
```

#### Documentation Updates

- [ ] `docs/validation.md` - Validation rules reference
- [ ] Add validation to CI/CD guide

---

### 3.3 Debug Mode & Logging

#### Implementation Steps

**Step 1: Add CLI flags**
```go
// cmd/tinkerdown/commands/serve.go
var debug bool
var verbose bool
cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")
cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
```

**Step 2: Implement structured logging**
```go
// internal/server/logging.go
type Logger struct {
    level LogLevel
    json  bool
}

func (l *Logger) Info(msg string, fields ...Field) { ... }
func (l *Logger) Debug(msg string, fields ...Field) { ... }
```

**Step 3: Add request correlation IDs**
- [ ] Generate unique ID per request
- [ ] Include in all log messages for request
- [ ] Pass to WebSocket messages

#### E2E Tests

```go
func TestDebugFlag(t *testing.T) {
    // Test: --debug enables debug output
}

func TestVerboseLogging(t *testing.T) {
    // Test: --verbose shows detailed logs
}
```

#### Documentation Updates

- [ ] `docs/debugging.md` - Debugging guide
- [ ] Add troubleshooting section

---

### 3.4 Hot Reload for Configuration

#### Implementation Steps

**Step 1: Watch config files**
- [ ] Use fsnotify to watch `tinkerdown.yaml`
- [ ] Debounce rapid changes (100ms)

**Step 2: Reload sources**
- [ ] Parse new config
- [ ] Recreate affected sources
- [ ] Keep WebSocket connections alive

**Step 3: Notify clients**
- [ ] Send reload message via WebSocket
- [ ] Client triggers re-fetch of data

#### E2E Tests

```go
func TestConfigHotReload(t *testing.T) {
    // Test: Changing config reloads sources without restart
}

func TestConfigReloadKeepsConnections(t *testing.T) {
    // Test: WebSocket stays connected during reload
}
```

---

### 3.5 WASM Source Development Kit

#### Implementation Steps

**Step 1: Implement `tinkerdown wasm init`**
```bash
tinkerdown wasm init github-issues
# Creates:
# - source.go (TinyGo template)
# - Makefile
# - README.md
# - test/
```

**Step 2: Implement `tinkerdown wasm build`**
```bash
tinkerdown wasm build
# Runs: tinygo build -o source.wasm -target=wasi source.go
```

**Step 3: Implement `tinkerdown wasm test`**
```bash
tinkerdown wasm test
# Loads WASM, calls fetch(), validates output
```

#### E2E Tests

```go
func TestWasmInit(t *testing.T) {
    // Test: wasm init creates correct structure
}

func TestWasmBuild(t *testing.T) {
    // Test: wasm build produces valid WASM
}

func TestWasmTest(t *testing.T) {
    // Test: wasm test validates module
}
```

#### Documentation Updates

- [ ] `docs/wasm-sources.md` - Complete WASM source authoring guide
- [ ] `docs/wasm-interface.md` - WASM interface specification
- [ ] `docs/wasm-examples.md` - Example sources walkthrough

#### Example Updates

- [ ] `examples/wasm-github-source/` - GitHub API WASM source
- [ ] `examples/wasm-notion-source/` - Notion API WASM source

---

## Phase 4: Data Ecosystem - Implementation Plan

### 4.1 GraphQL Source

#### Implementation Steps

**Step 1: Create GraphQL source (`internal/source/graphql.go`)**
```go
type GraphQLSource struct {
    name     string
    url      string
    query    string
    variables map[string]interface{}
    headers   map[string]string
}

func (s *GraphQLSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
    // Execute GraphQL query, flatten response
}
```

**Step 2: Add configuration parsing**
```yaml
sources:
  issues:
    type: graphql
    url: https://api.github.com/graphql
    query: |
      query($owner: String!, $repo: String!) {
        repository(owner: $owner, name: $repo) {
          issues(first: 10) {
            nodes { title, state, createdAt }
          }
        }
      }
    variables:
      owner: "livetemplate"
      repo: "tinkerdown"
    headers:
      Authorization: "Bearer ${GITHUB_TOKEN}"
```

**Step 3: Response flattening**
- [ ] Extract array from nested response path
- [ ] Handle pagination (cursor-based)

#### E2E Tests

```go
func TestGraphQLSource(t *testing.T) {
    // Test with mock GraphQL server
}

func TestGraphQLVariables(t *testing.T) {
    // Test variable substitution
}

func TestGraphQLAuth(t *testing.T) {
    // Test header authentication
}
```

#### Documentation Updates

- [ ] `docs/sources/graphql.md` - GraphQL source reference

#### Example Updates

- [ ] `examples/lvt-source-graphql-test/` - GraphQL source demo

---

### 4.2 MongoDB Source

#### Implementation Steps

**Step 1: Create MongoDB source (`internal/source/mongodb.go`)**
- [ ] Use official Go driver (go.mongodb.org/mongo-driver)
- [ ] Support find, insert, update, delete operations

**Step 2: Configuration**
```yaml
sources:
  products:
    type: mongodb
    uri: ${MONGODB_URI}
    database: myapp
    collection: products
    filter: { "status": "active" }
```

#### E2E Tests

```go
func TestMongoDBSource(t *testing.T) {
    // Test with embedded MongoDB or container
}

func TestMongoDBWrite(t *testing.T) {
    // Test CRUD operations
}
```

---

### 4.3 Source Composition

#### Implementation Steps

**Step 1: Create computed source (`internal/source/computed.go`)**
```go
type ComputedSource struct {
    name   string
    from   string          // Source to compute from
    filter string          // Filter expression
    sort   string          // Sort expression
    registry *SourceRegistry
}
```

**Step 2: Expression parser**
- [ ] Parse filter expressions: `status == 'active'`
- [ ] Parse sort expressions: `name asc, createdAt desc`
- [ ] Support basic operators: `==`, `!=`, `>`, `<`, `contains`

**Step 3: Aggregation**
```yaml
sources:
  stats:
    type: computed
    from: orders
    aggregate:
      total: sum(amount)
      count: count()
      avg: avg(amount)
```

#### E2E Tests

```go
func TestComputedFilter(t *testing.T) {
    // Test: filter expression works
}

func TestComputedSort(t *testing.T) {
    // Test: sort expression works
}

func TestComputedAggregate(t *testing.T) {
    // Test: aggregation functions work
}
```

---

## Phase 5: Production Readiness - Implementation Plan

### 5.1 Authentication Middleware

#### Implementation Steps

**Step 1: Create auth middleware (`internal/server/auth.go`)**
```go
type AuthMiddleware struct {
    providers map[string]AuthProvider
}

type AuthProvider interface {
    Authenticate(r *http.Request) (*User, error)
}

// Implementations:
type APIKeyAuth struct { ... }
type BasicAuth struct { ... }
type OAuth2Auth struct { ... }
type JWTAuth struct { ... }
```

**Step 2: Frontmatter configuration**
```yaml
auth: required
# or
auth:
  provider: github
  allowed_orgs: [mycompany]
  allowed_users: [admin]
```

**Step 3: User context in templates**
- [ ] Add `User` to template data: `{{.User.Email}}`
- [ ] Support user-specific data filtering

#### E2E Tests

```go
func TestAPIKeyAuth(t *testing.T) {
    // Test: API key authentication works
}

func TestOAuth2Flow(t *testing.T) {
    // Test: OAuth2 login flow
}

func TestUnauthorizedRedirect(t *testing.T) {
    // Test: Unauthenticated users redirected to login
}

func TestUserContextInTemplate(t *testing.T) {
    // Test: {{.User.Email}} renders correctly
}
```

#### Documentation Updates

- [ ] `docs/authentication.md` - Auth configuration guide
- [ ] `docs/security.md` - Security best practices

#### Example Updates

- [ ] `examples/authenticated-app/` - App with login

---

### 5.2 Health & Metrics Endpoints

#### Implementation Steps

**Step 1: Add health endpoints**
```go
// GET /health - Liveness
// GET /ready - Readiness (includes source connectivity)
// GET /metrics - Prometheus format
```

**Step 2: Implement metrics collection**
- [ ] Request count/latency by route
- [ ] WebSocket connection count
- [ ] Source fetch latency/error rates
- [ ] WASM execution time

#### E2E Tests

```go
func TestHealthEndpoint(t *testing.T) {
    // Test: /health returns 200
}

func TestReadyEndpoint(t *testing.T) {
    // Test: /ready checks source connectivity
}

func TestMetricsEndpoint(t *testing.T) {
    // Test: /metrics returns Prometheus format
}
```

---

### 5.3 Single Binary Distribution

#### Implementation Steps

**Step 1: Embed client assets**
```go
//go:embed assets/*
var assets embed.FS
```

**Step 2: Implement `tinkerdown build` command**
```bash
tinkerdown build ./myapp -o myapp-server
# Produces standalone binary with embedded markdown and assets
```

**Step 3: Cross-compilation**
```bash
tinkerdown build ./myapp --platform=linux/amd64,darwin/arm64,windows/amd64
```

#### E2E Tests

```go
func TestBuildCommand(t *testing.T) {
    // Test: build produces working binary
}

func TestBuiltBinaryRuns(t *testing.T) {
    // Test: Built binary serves app correctly
}
```

---

### 5.4 Security Hardening

#### Implementation Steps

**Step 1: CSRF Protection (`internal/server/csrf.go`)**
```go
type CSRFMiddleware struct {
    tokenStore TokenStore
}

func (m *CSRFMiddleware) GenerateToken(sessionID string) string {
    // Generate secure random token
    // Store in session-scoped map
}

func (m *CSRFMiddleware) ValidateToken(r *http.Request) bool {
    // Check X-CSRF-Token header or form field
    // Compare with stored token for session
}
```

- [ ] Generate CSRF tokens per session
- [ ] Auto-inject token into forms via template function
- [ ] Validate on all state-changing requests
- [ ] WebSocket: Use double-submit cookie pattern

**Step 2: Content Security Policy (`internal/server/security.go`)**
```go
func CSPMiddleware(config CSPConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Content-Security-Policy", config.String())
            next.ServeHTTP(w, r)
        })
    }
}
```

- [ ] Default restrictive CSP for new projects
- [ ] Configuration via `tinkerdown.yaml`
- [ ] Nonce generation for inline scripts (optional)

**Step 3: Input Sanitization**
- [ ] Validate source configuration at startup
- [ ] Path traversal prevention in `exec` source
- [ ] URL validation in `rest` source
- [ ] Ensure SQL parameterization (verify existing implementation)

**Step 4: WASM Resource Limits (`internal/source/wasm.go`)**
```go
type WASMConfig struct {
    MemoryLimit int64         // Default: 64MB
    Timeout     time.Duration // Default: 30s
}

func (s *WASMSource) executeWithLimits(ctx context.Context) ([]byte, error) {
    ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
    defer cancel()

    // Set memory limit via wazero configuration
    // Monitor execution, cancel if exceeded
}
```

- [ ] Parse limits from source configuration
- [ ] Integrate with wazero runtime limits
- [ ] Log resource usage for debugging
- [ ] Graceful error on limit exceeded

#### E2E Tests

```go
func TestCSRFProtection(t *testing.T) {
    // Test: POST without token returns 403
}

func TestCSRFTokenAutoInjection(t *testing.T) {
    // Test: Forms automatically include CSRF token
}

func TestCSPHeaders(t *testing.T) {
    // Test: Response includes correct CSP headers
}

func TestPathTraversalPrevention(t *testing.T) {
    // Test: exec source rejects ../../../etc/passwd
}

func TestWASMMemoryLimit(t *testing.T) {
    // Test: WASM module terminated when exceeding memory
}

func TestWASMTimeout(t *testing.T) {
    // Test: WASM execution cancelled after timeout
}
```

#### Documentation Updates

- [ ] `docs/security.md` - Comprehensive security guide
- [ ] Add security checklist to deployment guide
- [ ] Document WASM sandboxing model

#### Example Updates

- [ ] `examples/secure-app/` - Demo with all security features enabled

---

## Phase 6: UI & Components - Implementation Plan

### 6.1 Component Library

#### Implementation Steps

**Step 1: Chart component**
- [ ] Integrate Chart.js or lightweight alternative
- [ ] `<chart type="line" lvt-source="metrics" x="date" y="value">`
- [ ] Support line, bar, pie charts

**Step 2: Modal component**
- [ ] `lvt-modal-open="modal-id"` and `lvt-modal-close="modal-id"`
- [ ] Keyboard support (Escape to close)
- [ ] Focus trapping

**Step 3: Toast notifications**
- [ ] Auto-show on action success/error
- [ ] `lvt-toast-on:success="Item saved!"`

**Step 4: File upload**
- [ ] Drag-and-drop zone
- [ ] Progress indicator
- [ ] Integration with sources

#### E2E Tests

```go
func TestChartComponent(t *testing.T) {
    // Test: Chart renders from source data
}

func TestModalComponent(t *testing.T) {
    // Test: Modal opens/closes correctly
}

func TestToastNotification(t *testing.T) {
    // Test: Toast appears on action
}

func TestFileUpload(t *testing.T) {
    // Test: File upload works
}
```

---

### 6.2 Built-in Pagination

#### Implementation Steps

**Step 1: Add pagination attribute**
```html
<div lvt-source="users" lvt-paginate="20">
  <!-- Auto-generates prev/next controls -->
</div>
```

**Step 2: Server-side pagination**
- [ ] Add `page` and `per_page` to source queries
- [ ] Track current page in state
- [ ] Generate pagination controls

#### E2E Tests

```go
func TestPagination(t *testing.T) {
    // Test: Pagination controls work
}

func TestPaginationStatePreserved(t *testing.T) {
    // Test: Page number survives refresh
}
```

---

## Phase 7: Advanced Features - Implementation Plan

### 7.1 Multi-User State Broadcasting

#### Implementation Steps

**Step 1: Add broadcast configuration**
```yaml
sources:
  tasks:
    type: sqlite
    broadcast: true
```

**Step 2: Implement broadcasting**
- [ ] Track all WebSocket connections per source
- [ ] On state change, broadcast to all connections
- [ ] Exclude sender from broadcast

**Step 3: Conflict resolution**
- [ ] Last-write-wins (default)
- [ ] Optional merge strategy

#### E2E Tests

```go
func TestBroadcastToOtherClients(t *testing.T) {
    // Test: Change in one client appears in another
}

func TestConflictResolution(t *testing.T) {
    // Test: Concurrent edits handled correctly
}
```

---

### 7.2 Scheduled Tasks

#### Implementation Steps

**Step 1: Add scheduler (`internal/scheduler/scheduler.go`)**
```go
type Scheduler struct {
    jobs []Job
}

type Job struct {
    Cron   string
    Source string
    Action string
}
```

**Step 2: Configuration**
```yaml
schedules:
  refresh_data:
    cron: "*/5 * * * *"
    source: external_api
    action: Refresh
```

**Step 3: Background execution**
- [ ] Run jobs without active WebSocket connection
- [ ] Log execution results
- [ ] Webhook notification on error

#### E2E Tests

```go
func TestScheduledTask(t *testing.T) {
    // Test: Task runs at scheduled time
}

func TestScheduledTaskError(t *testing.T) {
    // Test: Error notification works
}
```

---

### 7.3 API Endpoint Mode

#### Implementation Steps

**Step 1: Add API configuration**
```yaml
api:
  enabled: true
  prefix: /api/v1
  sources:
    - name: tasks
      methods: [GET, POST, PUT, DELETE]
```

**Step 2: Generate REST endpoints**
- [ ] GET /api/v1/tasks - List
- [ ] POST /api/v1/tasks - Create
- [ ] PUT /api/v1/tasks/:id - Update
- [ ] DELETE /api/v1/tasks/:id - Delete

**Step 3: OpenAPI spec generation**
- [ ] `tinkerdown api spec` generates openapi.yaml
- [ ] Swagger UI endpoint

#### E2E Tests

```go
func TestAPIEndpoints(t *testing.T) {
    // Test: REST endpoints work
}

func TestOpenAPISpec(t *testing.T) {
    // Test: Spec generation works
}
```

---

## Phase 8: Polish & Optimization - Implementation Plan

### 8.1 Bundle Size Optimization

#### Implementation Steps

- [ ] Lazy load Monaco editor only for WASM blocks
- [ ] Lazy load Chart.js only when charts present
- [ ] Tree-shake unused LiveTemplate features
- [ ] Target <200KB initial bundle

#### Metrics

```go
func TestBundleSize(t *testing.T) {
    // Test: Bundle is under 200KB
}
```

---

### 8.2 Accessibility Audit

#### Implementation Steps

- [ ] Run axe-core automated tests
- [ ] Manual screen reader testing
- [ ] Keyboard navigation testing
- [ ] Color contrast validation

#### E2E Tests

```go
func TestKeyboardNavigation(t *testing.T) {
    // Test: All interactive elements keyboard accessible
}

func TestARIAAttributes(t *testing.T) {
    // Test: Components have correct ARIA
}
```

---

### 8.3 Comprehensive Test Suite

#### Test Organization

```
tests/
├── unit/
│   ├── source/          # Source unit tests
│   ├── runtime/         # GenericState tests
│   └── parser/          # Parser tests
├── integration/
│   ├── websocket/       # WebSocket integration
│   └── sources/         # Source integration
├── e2e/
│   ├── chromedp/        # Browser E2E tests
│   └── api/             # API E2E tests
└── performance/
    ├── benchmark/       # Benchmarks
    └── regression/      # Performance regression
```

#### Coverage Targets

- [ ] Unit tests: 90% coverage
- [ ] Integration tests: All source types
- [ ] E2E tests: All examples run successfully
- [ ] Performance: No regressions >10%

---

## Testing Standards

### E2E Test Template

```go
func TestFeatureName(t *testing.T) {
    // Setup
    srv := server.New("examples/feature-test")
    go srv.Start(":0")
    defer srv.Shutdown()

    // Wait for server
    waitForServer(t, srv.Addr())

    // Run browser tests with chromedp
    ctx, cancel := chromedp.NewContext(context.Background())
    defer cancel()

    var result string
    err := chromedp.Run(ctx,
        chromedp.Navigate(srv.URL()),
        chromedp.WaitVisible(".expected-element"),
        chromedp.Text(".result", &result),
    )

    require.NoError(t, err)
    assert.Equal(t, "expected", result)
}
```

### Documentation Template

Each feature documentation should include:

1. **Overview** - What the feature does
2. **Quick Start** - Minimal working example
3. **Configuration** - All config options
4. **Examples** - Multiple use cases
5. **Troubleshooting** - Common issues

### Example Template

Each example should include:

1. `index.md` - Main page with interactive demo
2. `README.md` - Explanation of what the example demonstrates
3. `tinkerdown.yaml` - Configuration (if needed)
4. `_data/` - Sample data files (if needed)
5. Working E2E test in `*_e2e_test.go`
