# Multi-Interface Tinkerdown Design

**Date:** 2025-12-14
**Status:** Design Complete

## Overview

Extend Tinkerdown so the same markdown file can power multiple interfaces beyond the web browser: CLI, HTTP API, and future chat bots (Slack, Discord, GitHub, Telegram).

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
  tinkerdown.yaml       # Config: sources, auth, etc. (optional)
```

### With interface overrides (optional)

```
myapp/
  app.md              # State + Actions + Web View
  app.cli.md          # CLI-specific output formatting (optional)
  app.slack.md        # Slack Block Kit templates (optional)
  tinkerdown.yaml
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

When no override exists, Tinkerdown **auto-generates** CLI output using conventions.

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
# tinkerdown.yaml
api:
  enabled: true
  auth: none | token | custom
  token: ${LIVEMDTOOLS_API_TOKEN}
```

---

## CLI Design

### Invocation modes

```bash
# One-shot commands (transient - default)
livemdtools cli app.md todos              # List state
livemdtools cli app.md add "Buy milk"     # Execute action
livemdtools cli app.md delete --id=1      # Action with named arg

# One-shot connected to running server
livemdtools cli app.md todos --server     # Fetches from localhost:8080
livemdtools cli app.md todos --server=:9000  # Custom port

# Interactive REPL
livemdtools cli app.md --interactive
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
$ livemdtools cli app.md --help
Usage: livemdtools cli app.md [command] [args]

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
livemdtools cli app.md add "Buy milk"

1. Parse app.md
2. Load state from SQLite (app.db)
3. Execute Add action
4. Persist state to SQLite
5. Print result
6. Exit
```

### Server mode (web UI, API, connected CLI)

```
tinkerdown serve app.md

Persistent process:
- Web UI on :8080
- HTTP API on :8080/api/*
- WebSocket for real-time updates
- State in memory + persisted to SQLite

Connected clients share the same live state.
```

### Headless API-only mode

```bash
livemdtools api app.md          # HTTP API without web UI
livemdtools api app.md --port=9000
```

---

## Compiled Binary

### Build standalone executable

```bash
tinkerdown build app.md -o myapp
# Produces: ./myapp (or myapp.exe on Windows)
```

The compiled binary embeds:
- The app.md file (and any override files)
- tinkerdown.yaml config
- Tinkerdown runtime

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
GOOS=windows tinkerdown build app.md -o myapp.exe
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
│  livemdtools api   │  {"NewTitle": "Buy milk"}
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
# tinkerdown.yaml
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
tinkerdown serve app.md --bots    # Web UI + API + Slack bot
livemdtools bot app.md             # Bot only (no web UI)
```

---

## Implementation Phases

| Phase | Scope | Priority |
|-------|-------|----------|
| **1. HTTP API** | Auto-generate `/api/state/*` and `/api/action/*` endpoints | High |
| **2. CLI Core** | One-shot commands, transient mode, state persistence | High |
| **3. CLI Polish** | REPL mode, `--server` flag, grouped help | Medium |
| **4. Convention Inference** | Auto-generate CLI from `lvt-*` attributes | Medium |
| **5. Build Command** | `tinkerdown build` to produce standalone binary | Medium |
| **6. Bot Adapters** | Slack, Discord, GitHub, Telegram | Future |

---

## Runbook-Specific Interface Patterns

Runbooks have unique requirements across interfaces. The same incident runbook should be usable from web, CLI, Slack, and API.

### CLI for Incident Response

When SSH'd into a production box during an incident:

```bash
# Start incident from template
cp templates/service-down.md incidents/$(date +%Y-%m-%d-%H%M)-api-outage.md
tinkerdown cli incidents/2024-01-15-1432-api-outage.md

# Check current status
tinkerdown cli incident.md status
Step 1: Check service health     [✓] 14:32
Step 2: Check database           [✓] 14:35
Step 3: Restart service          [ ] pending
Step 4: Verify recovery          [ ] pending

# Run a step
tinkerdown cli incident.md run step3
Running: Restart service...
[output captured to log]
✓ Step 3 completed at 14:38

# Add log entry
tinkerdown cli incident.md log "Restarted api-server-1, waiting for health check"

# Request approval (triggers Slack notification)
tinkerdown cli incident.md request-approval --access="prod-db-write" --reason="Need to fix corrupted data"
⏳ Approval requested. Waiting for response...
```

#### Why CLI matters for incidents

| Scenario | CLI Advantage |
|----------|---------------|
| SSH'd into prod server | Can't access web UI easily |
| Scripted remediation | Chain commands with `&&` |
| Low-bandwidth connection | Text-only, fast |
| Audit logging | Commands logged to shell history |
| Parallel execution | Run in multiple terminals |

---

### Slack for Incident Response

Operators live in Slack during incidents. Bring the runbook to them.

#### Starting an incident

```
/incident new service-down --title="API returning 500s"

🚨 Incident Started: API returning 500s
Template: service-down
Operator: @alice
Channel: #incident-2024-01-15-api

View in browser: https://runbooks.internal/incidents/2024-01-15-1432-api-outage
```

#### Running steps from Slack

```
/incident step 1

Running Step 1: Check service health
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
$ curl -s localhost:8080/health
{"status": "unhealthy", "db": "timeout"}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ Step 1 completed at 14:32

[Run Step 2] [View Full Runbook] [Add Note]
```

#### Approval workflow in Slack

```
@bob Approval needed for incident #2024-01-15-api

🔐 Access Request
━━━━━━━━━━━━━━━━━
Requester: @alice
Access: prod-db-write
Reason: Need to fix corrupted user records
Incident: API returning 500s
Expires: 2 hours after approval

[✓ Approve] [✗ Deny] [View Incident]
```

```
✅ Access Approved
@bob approved prod-db-write for @alice
Expires: 16:35 (2 hours)
```

#### Status updates

```
/incident status

📋 Incident: API returning 500s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Status: 🔴 Active
Duration: 45 minutes
Operator: @alice

Progress:
✓ Step 1: Check service health (14:32)
✓ Step 2: Check database (14:35)
⏳ Step 3: Restart service (in progress)
○ Step 4: Verify recovery

Recent Log:
14:45 | Restarted api-server-1
14:43 | DB connection pool exhausted
14:35 | Database showing high latency

[Run Next Step] [Add Note] [View Full]
```

#### Handoff between operators

```
/incident handoff @charlie

🔄 Operator Handoff
@alice → @charlie

Summary from @alice:
"DB connection pool was exhausted. Restarted api-server-1,
waiting for connections to drain. Next: verify health."

@charlie type `/incident accept` to confirm handoff.
```

---

### API for Automation & Integration

External systems trigger runbook actions via API.

#### Trigger from Alerting System

```yaml
# Alerting webhook → runbook
POST /api/incidents
{
  "template": "service-down",
  "title": "{{alert.title}}",
  "metadata": {
    "alert_id": "{{incident.id}}",
    "severity": "{{alert.severity}}",
    "service": "{{service.name}}"
  }
```

Response:
```json
{
  "incident_id": "2024-01-15-1432-api-outage",
  "url": "https://runbooks.internal/incidents/2024-01-15-1432-api-outage",
  "slack_channel": "#incident-2024-01-15-api"
}
```

#### Run step via API

```bash
POST /api/incidents/2024-01-15-1432-api-outage/steps/3/run
Authorization: Bearer <operator-token>

{
  "operator": "alice",
  "dry_run": false
}
```

Response:
```json
{
  "step": 3,
  "name": "Restart service",
  "status": "completed",
  "started_at": "2024-01-15T14:38:00Z",
  "completed_at": "2024-01-15T14:38:12Z",
  "output": "Service restarted successfully",
  "log_entry": "| 14:38 | restart | Restarted api-server via systemctl | alice |"
}
```

#### Query incident status

```bash
GET /api/incidents/2024-01-15-1432-api-outage

{
  "id": "2024-01-15-1432-api-outage",
  "title": "API returning 500s",
  "status": "active",
  "operator": "alice",
  "started_at": "2024-01-15T14:32:00Z",
  "steps": [
    {"number": 1, "name": "Check service health", "status": "completed", "completed_at": "..."},
    {"number": 2, "name": "Check database", "status": "completed", "completed_at": "..."},
    {"number": 3, "name": "Restart service", "status": "in_progress"},
    {"number": 4, "name": "Verify recovery", "status": "pending"}
  ],
  "approvals": [
    {"access": "prod-db-write", "requester": "alice", "approver": "bob", "expires_at": "..."}
  ],
  "log_entries": 12
}
```

#### Webhook notifications

```yaml
# tinkerdown.yaml
webhooks:
  - url: https://alerts.internal/webhooks/tinkerdown
    events: [incident.resolved]
  - url: https://monitoring.internal/events
    events: [step.completed, step.failed]
  - url: https://chat.internal/incidents
    events: [approval.requested, approval.granted]
```

---

### Interface Comparison for Runbooks

| Action | Web | CLI | Slack | API |
|--------|-----|-----|-------|-----|
| Start incident | Copy template | `cp` + `tinkerdown cli` | `/incident new` | `POST /api/incidents` |
| Run step | Click button | `run step3` | `/incident step 3` | `POST .../steps/3/run` |
| Add log entry | Type in form | `log "message"` | `/incident note` | `POST .../log` |
| Request approval | Click button | `request-approval` | Button in Slack | `POST .../approvals` |
| Approve | N/A (approver uses Slack) | `approve <id>` | Click Approve | `POST .../approvals/.../approve` |
| View status | See page | `status` | `/incident status` | `GET .../status` |
| Handoff | Edit operator field | `handoff @user` | `/incident handoff` | `PATCH .../operator` |

---

### State Synchronization

All interfaces update the same source of truth (the markdown file):

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│   Web    │     │   CLI    │     │  Slack   │
└────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │
     └───────────┬────┴────────────────┘
                 │
                 ▼
         ┌──────────────┐
         │  Markdown    │
         │  File (git)  │
         └──────────────┘
                 │
                 ▼
         ┌──────────────┐
         │  WebSocket   │
         │  Broadcast   │
         └──────────────┘
                 │
     ┌───────────┼────────────────┐
     │           │                │
     ▼           ▼                ▼
┌──────────┐ ┌──────────┐  ┌──────────┐
│   Web    │ │   CLI    │  │  Slack   │
│ (update) │ │ (poll)   │  │ (update) │
└──────────┘ └──────────┘  └──────────┘
```

- Web/Slack: Real-time via WebSocket
- CLI: Poll or one-shot (sees current state when run)
- API: Request/response (caller decides if polling needed)

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
| Add `livemdtools cli` subcommand | ⏳ TODO | |
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
| `tinkerdown build` command | ⏳ TODO | |
| Embed app.md in binary | ⏳ TODO | |
| Cross-compilation support | ⏳ TODO | |
