# Tinkerdown Testing Strategy: Architecture-Specific

> **Status**: Proposal
> **Date**: 2025-12-31
> **Goal**: Deterministic black-box testing mapped to tinkerdown's actual components

---

## Tinkerdown Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         TINKERDOWN DATA FLOW                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  INPUT                    PROCESSING                      OUTPUT            │
│  ─────                    ──────────                      ──────            │
│                                                                              │
│  app.md ──────┐                                                              │
│               │                                                              │
│  tinkerdown   │     ┌──────────────┐    ┌──────────────┐                    │
│  .yaml ───────┼────▶│   Parser     │───▶│   Server     │                    │
│               │     │  (parser.go) │    │ (server.go)  │                    │
│  frontmatter ─┘     └──────┬───────┘    └──────┬───────┘                    │
│                            │                   │                             │
│                            ▼                   ▼                             │
│                     ┌──────────────┐    ┌──────────────┐    ┌────────────┐  │
│                     │   Sources    │    │  WebSocket   │───▶│   Client   │  │
│                     │ (source/*.go)│◀───│(websocket.go)│    │    (JS)    │  │
│                     └──────────────┘    └──────────────┘    └────────────┘  │
│                            │                   │                             │
│                            ▼                   ▼                             │
│                     ┌──────────────┐    ┌──────────────┐                    │
│                     │   Runtime    │    │  HTTP API    │                    │
│                     │(runtime/*.go)│    │  (api.go)    │                    │
│                     └──────────────┘    └──────────────┘                    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Component-Specific Testing Strategy

### 1. Parser Layer (`parser.go`, `internal/markdown/`)

**What it does:** Transforms markdown → HTML + extracts frontmatter, code blocks, sources

**Current tests:** `parser_test.go` - table-driven unit tests

**Deterministic outputs to verify:**

| Input | Output | Test Type |
|-------|--------|-----------|
| `# Heading` | `<h1>Heading</h1>` | Golden file |
| Frontmatter YAML | Parsed `Frontmatter` struct | Unit test |
| Code block with flags | `CodeBlock` with metadata | Unit test |
| `## Tasks` + list | Auto-detected source | Golden file |

**Recommended approach:**

```go
// parser_golden_test.go
func TestParseMarkdownGolden(t *testing.T) {
    tests := []string{
        "simple-heading",
        "task-list",
        "table-with-frontmatter",
        "code-blocks-server",
        "lvt-interactive",
    }

    for _, name := range tests {
        t.Run(name, func(t *testing.T) {
            input := readFixture(t, name+".md")
            _, _, html, _ := tinkerdown.ParseMarkdown(input)
            assertGolden(t, name+".html", html)
        })
    }
}
```

**Fixtures needed:**
```
testdata/
├── parser/
│   ├── fixtures/
│   │   ├── simple-heading.md
│   │   ├── task-list.md
│   │   ├── table-with-frontmatter.md
│   │   ├── code-blocks-server.md
│   │   └── lvt-interactive.md
│   └── golden/
│       ├── simple-heading.html
│       ├── task-list.html
│       └── ...
```

---

### 2. Source Layer (`internal/source/`)

**What it does:** Fetches data from 8 source types

**Source types:**
- `exec` - Shell commands
- `pg` - PostgreSQL
- `rest` - HTTP APIs
- `json` - JSON files
- `csv` - CSV files
- `markdown` - Markdown tables/lists
- `sqlite` - SQLite databases
- `wasm` - WebAssembly modules

**Current tests:** `markdown_test.go` only

**Deterministic outputs to verify:**

| Source | Input | Output | Test Type |
|--------|-------|--------|-----------|
| json | `users.json` | `[]map[string]interface{}` | Golden JSON |
| csv | `products.csv` | `[]map[string]interface{}` | Golden JSON |
| markdown | `## Tasks` section | Parsed rows | Golden JSON |
| sqlite | `test.db` table | Query results | Golden JSON |
| exec | Deterministic script | Script output | Golden JSON |

**Recommended approach:**

```go
// source_golden_test.go
func TestJSONSource(t *testing.T) {
    src, _ := source.NewJSONFileSource("users", "testdata/sources/users.json", ".")
    data, _ := src.Fetch(context.Background())

    // Serialize to JSON for comparison
    output, _ := json.MarshalIndent(data, "", "  ")
    assertGolden(t, "json-users.json", output)
}

func TestMarkdownSource(t *testing.T) {
    src, _ := source.NewMarkdownSource(
        "tasks",
        "testdata/sources/tasks.md",
        "tasks",  // anchor
        ".", "",
        true,  // readonly
    )
    data, _ := src.Fetch(context.Background())

    output, _ := json.MarshalIndent(data, "", "  ")
    assertGolden(t, "markdown-tasks.json", output)
}

func TestExecSource(t *testing.T) {
    // Use a deterministic script
    src, _ := source.NewExecSource("data", "./testdata/sources/echo-json.sh", ".")
    data, _ := src.Fetch(context.Background())

    output, _ := json.MarshalIndent(data, "", "  ")
    assertGolden(t, "exec-data.json", output)
}
```

**Fixtures needed:**
```
testdata/
├── sources/
│   ├── fixtures/
│   │   ├── users.json
│   │   ├── products.csv
│   │   ├── tasks.md
│   │   ├── test.db
│   │   └── echo-json.sh  # Deterministic script
│   └── golden/
│       ├── json-users.json
│       ├── csv-products.json
│       ├── markdown-tasks.json
│       ├── sqlite-users.json
│       └── exec-data.json
```

---

### 3. WebSocket Protocol (`internal/server/websocket.go`)

**What it does:** Sends `MessageEnvelope` with tree updates, handles actions

**Message format:**
```go
type MessageEnvelope struct {
    BlockID  string          `json:"blockID"`
    Action   string          `json:"action"`      // "tree", action names
    Data     json.RawMessage `json:"data"`        // Tree JSON or action data
    ExecMeta *ExecMeta       `json:"execMeta"`    // Optional exec metadata
}
```

**Current tests:** None direct - tested via chromedp

**Deterministic outputs to verify:**

| Scenario | Input | Output | Test Type |
|----------|-------|--------|-----------|
| Initial state | Connect | `{action:"tree", data:...}` | Golden JSON |
| Action | `{action:"Add", data:{...}}` | Updated tree | Golden JSON |
| Refresh | `{action:"Refresh"}` | Re-fetched tree | Golden JSON |

**Recommended approach - test WebSocket directly without browser:**

```go
// websocket_protocol_test.go
func TestWebSocketInitialState(t *testing.T) {
    // Setup test server
    srv := setupTestServer(t, "testdata/ws/simple-counter.md")

    // Connect WebSocket directly (no browser!)
    wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
    conn, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
    defer conn.Close()

    // Read initial message
    _, msg, _ := conn.ReadMessage()

    // Scrub dynamic values and compare
    scrubbed := scrubWebSocketMessage(msg)
    assertGolden(t, "ws-initial-state.json", scrubbed)
}

func TestWebSocketAction(t *testing.T) {
    srv := setupTestServer(t, "testdata/ws/counter.md")
    conn := connectWebSocket(t, srv)
    defer conn.Close()

    // Skip initial state
    conn.ReadMessage()

    // Send action
    action := `{"blockID":"lvt-0","action":"increment","data":{}}`
    conn.WriteMessage(websocket.TextMessage, []byte(action))

    // Read response
    _, msg, _ := conn.ReadMessage()

    scrubbed := scrubWebSocketMessage(msg)
    assertGolden(t, "ws-after-increment.json", scrubbed)
}
```

**Scrubbers for WebSocket:**
```go
func scrubWebSocketMessage(msg []byte) []byte {
    // Remove dynamic block IDs like "lvt-0-abc123"
    msg = regexp.MustCompile(`"blockID":"[^"]+"`).
        ReplaceAll(msg, []byte(`"blockID":"BLOCK_ID"`))

    // Remove timestamps
    msg = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`).
        ReplaceAll(msg, []byte("TIMESTAMP"))

    // Remove exec durations
    msg = regexp.MustCompile(`"duration":\d+`).
        ReplaceAll(msg, []byte(`"duration":0`))

    return msg
}
```

**Speed improvement:** ~50ms vs ~5s for chromedp

---

### 4. HTTP API Layer (`internal/server/`)

**What it does:** Serves pages, handles API requests

**Endpoints:**
- `GET /` - Rendered HTML page
- `GET /page.md` - Specific page
- `GET /_sources/{name}` - Source data (if exposed)
- WebSocket `/ws` - Real-time updates

**Deterministic outputs to verify:**

| Endpoint | Input | Output | Test Type |
|----------|-------|--------|-----------|
| `GET /` | Simple app | Full HTML | Golden file |
| `GET /page.md` | With sources | HTML + attributes | Golden file |

**Recommended approach:**

```go
// http_golden_test.go
func TestServeHomepage(t *testing.T) {
    srv := setupTestServer(t, "testdata/http/simple.md")

    resp, _ := http.Get(srv.URL + "/")
    body, _ := io.ReadAll(resp.Body)

    // Scrub dynamic content
    scrubbed := scrubHTML(body)
    assertGolden(t, "http-simple-page.html", scrubbed)
}

func scrubHTML(html []byte) []byte {
    // Remove cache-busting query params
    html = regexp.MustCompile(`\?v=\d+`).
        ReplaceAll(html, []byte("?v=VERSION"))

    // Remove nonce values
    html = regexp.MustCompile(`nonce="[^"]+"`).
        ReplaceAll(html, []byte(`nonce="NONCE"`))

    return html
}
```

---

### 5. Runtime Layer (`internal/runtime/`)

**What it does:** Handles actions, manages state

**Key interface:**
```go
type Store interface {
    HandleAction(action string, data map[string]interface{}) error
    Close() error
}
```

**Deterministic outputs to verify:**

| Action | Input State | Action Data | Output State |
|--------|-------------|-------------|--------------|
| Add | Empty table | `{text:"Buy milk"}` | 1 row table |
| Toggle | Task unchecked | `{id:1}` | Task checked |
| Delete | 3 rows | `{id:2}` | 2 rows |

**Recommended approach:**

```go
// runtime_test.go
func TestGenericStateActions(t *testing.T) {
    // Setup state with known initial data
    cfg := config.SourceConfig{Type: "json", File: "testdata/runtime/tasks.json"}
    state, _ := runtime.NewGenericState("tasks", cfg, ".", "")

    // Get initial state
    initial := stateToJSON(state)
    assertGolden(t, "runtime-initial.json", initial)

    // Execute Add action
    state.HandleAction("Add", map[string]interface{}{
        "text": "New task",
        "done": false,
    })

    afterAdd := stateToJSON(state)
    assertGolden(t, "runtime-after-add.json", afterAdd)

    // Execute Toggle action
    state.HandleAction("Toggle", map[string]interface{}{
        "id": 1,
    })

    afterToggle := stateToJSON(state)
    assertGolden(t, "runtime-after-toggle.json", afterToggle)
}
```

---

### 6. CLI Commands (`cmd/tinkerdown/commands/`)

**Commands:**
- `serve` - Start server
- `validate` - Validate markdown
- `fix` - Auto-fix issues
- `blocks` - List code blocks
- `new` - Create new app

**Recommended approach - testscript:**

```txtar
# testdata/cli/validate.txtar

# Test validate command with valid file
exec tinkerdown validate input.md
stdout 'valid'
! stderr .

# Test validate with invalid file
! exec tinkerdown validate invalid.md
stderr 'error'

-- input.md --
# Valid App

## Tasks
- [ ] Test task

-- invalid.md --
---
sources:
  bad: { type: unknown }
---
# Invalid
```

```go
// cli_test.go
func TestCLI(t *testing.T) {
    testscript.Run(t, testscript.Params{
        Dir: "testdata/cli",
        Setup: func(env *testscript.Env) error {
            // Build tinkerdown binary
            cmd := exec.Command("go", "build", "-o",
                filepath.Join(env.WorkDir, "tinkerdown"),
                "./cmd/tinkerdown")
            return cmd.Run()
        },
    })
}
```

---

## Testing Pyramid for Tinkerdown

```
                    ▲
                   ╱ ╲
                  ╱   ╲     Browser E2E (chromedp)
                 ╱     ╲    • 3-5 critical user flows
                ╱───────╲   • Login, data loading, actions
               ╱         ╲
              ╱           ╲   WebSocket Protocol Tests
             ╱             ╲  • Direct WS connection (no browser)
            ╱               ╲ • Message format verification
           ╱─────────────────╲
          ╱                   ╲   HTTP API + Source Tests
         ╱                     ╲  • Golden file responses
        ╱                       ╲ • Source output verification
       ╱─────────────────────────╲
      ╱                           ╲   Parser + Runtime Unit Tests
     ╱                             ╲  • Golden file HTML output
    ╱                               ╲ • State transition verification
   ╱─────────────────────────────────╲

   Fast, Many ◄────────────────────────► Slow, Few
```

---

## Test File Organization

```
testdata/
├── parser/
│   ├── fixtures/           # Input .md files
│   │   ├── simple.md
│   │   ├── frontmatter.md
│   │   ├── code-blocks.md
│   │   └── lvt-source.md
│   └── golden/             # Expected HTML output
│       ├── simple.html
│       └── ...
│
├── sources/
│   ├── fixtures/           # Source data files
│   │   ├── users.json
│   │   ├── products.csv
│   │   ├── tasks.md
│   │   ├── test.db
│   │   └── echo-json.sh
│   └── golden/             # Expected parsed data
│       ├── json-users.json
│       ├── csv-products.json
│       └── ...
│
├── websocket/
│   ├── fixtures/           # Test apps
│   │   ├── counter.md
│   │   └── todo.md
│   └── golden/             # Expected WS messages
│       ├── initial-state.json
│       └── after-action.json
│
├── http/
│   ├── fixtures/           # Test apps
│   └── golden/             # Expected HTML responses
│
├── runtime/
│   ├── fixtures/           # Initial state files
│   └── golden/             # State after actions
│
└── cli/
    ├── validate.txtar
    ├── serve.txtar
    └── new.txtar
```

---

## Scrubber Registry

Centralized scrubbers for tinkerdown-specific patterns:

```go
// internal/testutil/scrub.go
package testutil

var Scrubbers = map[string]Scrubber{
    // Block IDs: "lvt-0", "auto-persist-lvt-0"
    "blockID": regexp.MustCompile(`(lvt|auto-persist-lvt)-\d+`),

    // Timestamps: "2024-01-15T10:30:00"
    "timestamp": regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`),

    // Exec durations: "duration":1234
    "duration": regexp.MustCompile(`"duration":\d+`),

    // Cache busters: "?v=1234567890"
    "cacheBust": regexp.MustCompile(`\?v=\d+`),

    // Nonces: nonce="abc123"
    "nonce": regexp.MustCompile(`nonce="[^"]+"`),

    // WebSocket conn IDs
    "wsConn": regexp.MustCompile(`ws-conn-[a-zA-Z0-9]+`),
}

func ScrubAll(data []byte) []byte {
    for _, pattern := range Scrubbers {
        data = pattern.ReplaceAll(data, []byte("SCRUBBED"))
    }
    return data
}
```

---

## Browser Tests: When to Keep Chromedp

Keep chromedp for testing things that **require a real browser:**

| Keep Chromedp | Don't Need Chromedp |
|---------------|---------------------|
| `lvt-click` button works | WebSocket message format |
| Form submit triggers action | HTML output structure |
| Real-time updates display | Source data parsing |
| CSS renders correctly | Action state changes |
| JavaScript executes | API response format |

**Reduced chromedp test count: 16 → 5**

Essential browser tests:
1. `TestLvtClickAction` - Button click triggers WebSocket action
2. `TestLvtSourceRendersData` - Data appears in DOM after WS message
3. `TestFormSubmitAction` - Form submission works
4. `TestRealTimeUpdate` - State change reflects in UI
5. `TestNavigationWorks` - Page navigation functional

---

## Implementation Priority

| Phase | Component | Test Type | Effort | Value |
|-------|-----------|-----------|--------|-------|
| 1 | Parser | Golden files | Low | High |
| 1 | Sources | Golden JSON | Low | High |
| 2 | WebSocket | Protocol tests | Medium | High |
| 2 | Runtime | State tests | Low | Medium |
| 3 | HTTP | Golden HTML | Low | Medium |
| 3 | CLI | testscript | Medium | Medium |
| 4 | Browser | Reduce chromedp | Low | High (speed) |

**Start with Phase 1** - Parser and Sources golden tests provide highest coverage with lowest effort.

---

## Summary

| Layer | Tool | Tests | Speed |
|-------|------|-------|-------|
| Parser | Golden files | Markdown → HTML | ~1ms |
| Sources | Golden JSON | Data fetching | ~10ms |
| WebSocket | Direct connection | Protocol format | ~50ms |
| Runtime | Unit tests | State transitions | ~1ms |
| HTTP | Golden HTML | Page rendering | ~10ms |
| CLI | testscript | Command behavior | ~100ms |
| Browser | Chromedp (reduced) | Critical flows | ~5s |

**Total test time reduction: ~80 seconds → ~10 seconds**
