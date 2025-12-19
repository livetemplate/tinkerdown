# Multi-Interface LivePage Design

**Date:** 2025-12-14
**Status:** Design Complete

## Overview

Extend LivePage so the same markdown file can power multiple interfaces beyond the web browser: CLI, HTTP API, and future chat bots (Slack, Discord, GitHub, Telegram).

```
┌─────────────────────────────────────────────────────────┐
│                      app.md                             │
│  ┌─────────┐  ┌─────────┐  ┌──────────────────────┐    │
│  │  State  │  │ Actions │  │  View (HTML default) │    │
│  └─────────┘  └─────────┘  └──────────────────────┘    │
└─────────────────────────────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        ▼                 ▼                 ▼
   ┌─────────┐      ┌──────────┐      ┌─────────┐
   │ Web UI  │      │ HTTP API │      │   CLI   │
   │ :8080   │      │ /api/*   │      │ REPL/   │
   └─────────┘      └──────────┘      │ one-shot│
                          │           └─────────┘
                          ▼
                 ┌─────────────────┐
                 │  Future: Bots   │
                 │ Slack/Discord/  │
                 │ GitHub/Telegram │
                 └─────────────────┘
```

**Key principle:** State and Actions are the **shared core**. Views are **interface-specific** with convention-based defaults.

---

## Design Decisions

| Decision | Choice |
|----------|--------|
| **File model** | Single file + optional interface overrides |
| **Rendering** | Convention-based (infer CLI from `lvt-*` attributes) |
| **Priority interfaces** | CLI + HTTP API (foundation for everything else) |
| **CLI runtime** | Both transient (default) and server-connected modes |
| **API design** | Full state + field-level access, configurable auth |
| **CLI experience** | Grouped help, one-shot commands, optional REPL |
| **Distribution** | Subcommands for dev, compiled binary for distribution |
| **Future bots** | Self-hosted adapters calling HTTP API |

---

## File Structure & Overrides

### Default: Single file works everywhere

```
myapp/
  app.md              # State + Actions + Web View (required)
  livepage.yaml       # Config: sources, auth, etc. (optional)
```

### With interface overrides (optional)

```
myapp/
  app.md              # State + Actions + Web View
  app.cli.md          # CLI-specific output formatting (optional)
  app.slack.md        # Slack Block Kit templates (optional)
  livepage.yaml
```

Override files contain **only view logic**, not state/actions:

```markdown
<!-- app.cli.md -->
## Todos View
{{range .Todos}}
  {{.ID | printf "%3d"}}. {{.Title}}{{if .Done}} ✓{{end}}
{{end}}

## Help
  todos              List all todos
  add <title>        Add a new todo
  delete <id>        Delete a todo
```

When no override exists, LivePage **auto-generates** CLI output using conventions.

---

## HTTP API Design

### Auto-generated endpoints

Given State + Actions:

```go
type State struct {
    Todos  []Todo
    Filter string
}

func (s *State) Add(ctx Context) { ... }
func (s *State) Delete(ctx Context) { ... }
```

Generates:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/state` | GET | Full state JSON |
| `/api/state/Todos` | GET | Just the Todos slice |
| `/api/state/Filter` | GET | Just the Filter value |
| `/api/action/Add` | POST | Execute Add action |
| `/api/action/Delete` | POST | Execute Delete action |

### Request/Response examples

```bash
# Get full state
curl http://localhost:8080/api/state
{"Todos": [{"ID": 1, "Title": "Buy milk"}], "Filter": "all"}

# Get specific field
curl http://localhost:8080/api/state/Todos
[{"ID": 1, "Title": "Buy milk"}]

# Execute action
curl -X POST http://localhost:8080/api/action/Add \
  -d '{"NewTitle": "Call mom"}'
{"ok": true, "state": {...}}

# With auth enabled
curl -H "Authorization: Bearer <token>" ...
```

### Configuration

```yaml
# livepage.yaml
api:
  enabled: true
  auth: none | token | custom
  token: ${LIVEPAGE_API_TOKEN}
```

---

## CLI Design

### Invocation modes

```bash
# One-shot commands (transient - default)
livepage cli app.md todos              # List state
livepage cli app.md add "Buy milk"     # Execute action
livepage cli app.md delete --id=1      # Action with named arg

# One-shot connected to running server
livepage cli app.md todos --server     # Fetches from localhost:8080
livepage cli app.md todos --server=:9000  # Custom port

# Interactive REPL
livepage cli app.md --interactive
> todos
  ID  Title       Done
  1   Buy milk    [ ]
  2   Call mom    [✓]
> add "New todo"
  Added: New todo
> help
> quit
```

### Help output (grouped)

```bash
$ livepage cli app.md --help
Usage: livepage cli app.md [command] [args]

State (read-only):
  todos              List all todos
  filter             Show current filter

Actions:
  add <title>        Add a new todo
  delete <id>        Delete a todo
  set-filter <val>   Set filter (all|active|done)

Flags:
  --interactive, -i  Start REPL mode
  --server[=addr]    Connect to running server
  --format=json      Output as JSON (default: table)
  --help, -h         Show this help
```

---

## Runtime Modes

### Transient mode (default for CLI)

```
livepage cli app.md add "Buy milk"

1. Parse app.md
2. Load state from SQLite (app.db)
3. Execute Add action
4. Persist state to SQLite
5. Print result
6. Exit
```

### Server mode (web UI, API, connected CLI)

```
livepage serve app.md

Persistent process:
- Web UI on :8080
- HTTP API on :8080/api/*
- WebSocket for real-time updates
- State in memory + persisted to SQLite

Connected clients share the same live state.
```

### Headless API-only mode

```bash
livepage api app.md          # HTTP API without web UI
livepage api app.md --port=9000
```

---

## Compiled Binary

### Build standalone executable

```bash
livepage build app.md -o myapp
# Produces: ./myapp (or myapp.exe on Windows)
```

The compiled binary embeds:
- The app.md file (and any override files)
- livepage.yaml config
- LivePage runtime

### Usage after build

```bash
./myapp                    # Starts web UI (default)
./myapp serve              # Explicit: web UI
./myapp api                # Headless API only
./myapp todos              # CLI one-shot (actions as top-level commands)
./myapp -i                 # CLI REPL
./myapp --help             # Shows app-specific help
```

### Distribution story

> "Build a tool, send the binary. They double-click, it opens in browser. No installation."

```bash
# Cross-compile for colleague on Windows
GOOS=windows livepage build app.md -o myapp.exe
```

---

## Convention-Based CLI Inference

### How CLI commands are auto-generated from HTML

Given this view:

```html
<ul>
{{range .Todos}}
  <li>
    {{.Title}}
    <button lvt-click="Delete" lvt-data-id="{{.ID}}">×</button>
    <button lvt-click="ToggleDone" lvt-data-id="{{.ID}}">✓</button>
  </li>
{{end}}
</ul>
<input lvt-model="NewTitle" placeholder="New todo">
<button lvt-click="Add">Add</button>
```

### Inferred CLI commands

| HTML Pattern | Inferred CLI | Reasoning |
|--------------|--------------|-----------|
| `lvt-click="Delete"` + `lvt-data-id` | `delete --id=<id>` | Action + data attribute → named arg |
| `lvt-click="ToggleDone"` + `lvt-data-id` | `toggle-done --id=<id>` | CamelCase → kebab-case |
| `lvt-click="Add"` + `lvt-model="NewTitle"` | `add <title>` | Action + nearby model → positional arg |
| `lvt-model="Filter"` | (state field) | Read via `filter` command |

### Inference rules

1. `lvt-click="ActionName"` → command `action-name`
2. `lvt-data-*` on same element → `--name=value` arguments
3. `lvt-model` near action → positional argument
4. State fields → read-only commands
5. CamelCase → kebab-case for CLI ergonomics

---

## Future: Chat Bot Adapters

Once CLI + HTTP API are solid, chat bots become thin adapters:

```
Slack/Discord/Telegram message
         │
         ▼
┌─────────────────┐
│  Bot Adapter    │  Parses: /myapp add "Buy milk"
│  (self-hosted)  │  Translates to HTTP API call
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  HTTP API       │  POST /api/action/Add
│  livepage api   │  {"NewTitle": "Buy milk"}
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Response       │  Format for platform:
│  Formatter      │  Slack → Block Kit
│                 │  Discord → Embeds
└─────────────────┘  Telegram → Markdown
```

### Configuration

```yaml
# livepage.yaml
bots:
  slack:
    enabled: true
    token: ${SLACK_BOT_TOKEN}
    signing_secret: ${SLACK_SIGNING_SECRET}
    commands:
      /todo: app.md
```

### Run with bot enabled

```bash
livepage serve app.md --bots    # Web UI + API + Slack bot
livepage bot app.md             # Bot only (no web UI)
```

---

## Implementation Phases

| Phase | Scope | Priority |
|-------|-------|----------|
| **1. HTTP API** | Auto-generate `/api/state/*` and `/api/action/*` endpoints | High |
| **2. CLI Core** | One-shot commands, transient mode, state persistence | High |
| **3. CLI Polish** | REPL mode, `--server` flag, grouped help | Medium |
| **4. Convention Inference** | Auto-generate CLI from `lvt-*` attributes | Medium |
| **5. Build Command** | `livepage build` to produce standalone binary | Medium |
| **6. Bot Adapters** | Slack, Discord, GitHub, Telegram | Future |

---

## Progress Tracker

| Task | Status | Notes |
|------|--------|-------|
| **Phase 1: HTTP API** | | |
| Design API endpoints | ✅ Done | This document |
| Implement `/api/state` | ⏳ TODO | |
| Implement `/api/state/<field>` | ⏳ TODO | |
| Implement `/api/action/<name>` | ⏳ TODO | |
| Add configurable auth | ⏳ TODO | |
| **Phase 2: CLI Core** | | |
| Add `livepage cli` subcommand | ⏳ TODO | |
| Implement transient mode | ⏳ TODO | |
| State persistence (SQLite) | ⏳ TODO | |
| **Phase 3: CLI Polish** | | |
| REPL mode (`--interactive`) | ⏳ TODO | |
| Server mode (`--server`) | ⏳ TODO | |
| Grouped help output | ⏳ TODO | |
| **Phase 4: Convention Inference** | | |
| Parse `lvt-*` for CLI hints | ⏳ TODO | |
| Generate help from HTML | ⏳ TODO | |
| **Phase 5: Build Command** | | |
| `livepage build` command | ⏳ TODO | |
| Embed app.md in binary | ⏳ TODO | |
| Cross-compilation support | ⏳ TODO | |
