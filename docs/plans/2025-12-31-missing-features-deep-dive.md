# Tinkerdown: Missing Features Deep Dive

**Date:** 2025-12-31
**Status:** Analysis Complete

---

## Executive Summary

Tinkerdown has a solid foundation but lacks critical features for the two main use cases:
1. **Runbooks** - Missing: auto-timestamp, snapshot capture, CLI mode, Slack integration
2. **Productivity** - Missing: aggregations, computed fields, date filtering, charts

This document catalogs all gaps with priority and implementation effort.

---

## Table of Contents

1. [Current State](#current-state)
2. [Critical Gaps](#critical-gaps)
3. [Feature Gap Matrix](#feature-gap-matrix)
4. [Detailed Gap Analysis](#detailed-gap-analysis)
5. [Priority Recommendations](#priority-recommendations)

---

## Current State

### What Exists Today

#### Data Sources (8 types)
| Source | Read | Write | Status |
|--------|------|-------|--------|
| `exec` | ✅ | ❌ | Run commands, get output |
| `rest` | ✅ | ❌ | Fetch from REST APIs |
| `json` | ✅ | ❌ | Read JSON files |
| `csv` | ✅ | ❌ | Read CSV files |
| `markdown` | ✅ | ✅ | Read/write markdown tables |
| `sqlite` | ✅ | ✅ | Full CRUD on SQLite |
| `pg` | ✅ | ❌ | Query PostgreSQL |
| `wasm` | ✅ | ❌ | Custom WASM sources |

#### Auto-Rendering Components
| Component | Status | Attributes |
|-----------|--------|------------|
| Tables | ✅ Done | `lvt-source`, `lvt-columns`, `lvt-actions`, `lvt-empty` |
| Lists | ✅ Done | `lvt-source`, `lvt-field`, `lvt-actions`, `lvt-empty` |
| Selects | ✅ Done | `lvt-source`, `lvt-value`, `lvt-label` |

#### Actions
| Action | Source Types | Status |
|--------|--------------|--------|
| Add | markdown, sqlite | ✅ |
| Delete | markdown, sqlite | ✅ |
| Toggle | markdown, sqlite | ✅ |
| Update | markdown, sqlite | ✅ |
| Run | exec | ✅ |
| Refresh | all | ✅ |
| Sort | client-side | ✅ |

#### CLI Commands
| Command | Status | Description |
|---------|--------|-------------|
| `serve` | ✅ | Start dev server |
| `new` | ✅ | Scaffold new app |
| `validate` | ✅ | Check markdown syntax |
| `fix` | ✅ | Auto-fix issues |
| `blocks` | ✅ | List code blocks |

#### Other Features
- ✅ WebSocket real-time updates
- ✅ File watching / hot reload
- ✅ Go template syntax
- ✅ Mermaid diagrams
- ✅ Monaco editor for code blocks
- ✅ Search across pages
- ✅ Multi-page site mode
- ✅ Presentation mode

---

## Critical Gaps

### Gap 1: Auto-Timestamp on Form Submit

**Impact:** HIGH (blocks runbook and productivity use cases)
**Effort:** SMALL

**Current:** Operators must type timestamps manually.
**Needed:**
```yaml
sources:
  log:
    type: markdown
    auto_fields:
      time: "{{now:15:04}}"
      who: "{{operator}}"
```

**Files to modify:**
- `internal/source/markdown.go` - Add auto_fields support
- `internal/runtime/state.go` - Evaluate templates on submit

---

### Gap 2: Operator Identity

**Impact:** HIGH (who did what?)
**Effort:** SMALL

**Current:** No concept of current user.
**Needed:**
```bash
tinkerdown serve --operator alice
# or
export TINKERDOWN_OPERATOR=alice
```

**Files to modify:**
- `cmd/tinkerdown/commands/serve.go` - Add --operator flag
- `internal/config/config.go` - Add operator field

---

### Gap 3: Snapshot Capture

**Impact:** HIGH (core runbook value prop)
**Effort:** MEDIUM

**Current:** No way to freeze exec output at a point in time.
**Needed:**
```html
<button lvt-click="snapshot" lvt-data-source="containers">
  📸 Capture
</button>
```

**Files to modify:**
- `internal/runtime/actions.go` - Add snapshot handler
- `internal/source/markdown.go` - Append to sections

---

### Gap 4: CLI Mode

**Impact:** HIGH (multi-interface, SSH access)
**Effort:** MEDIUM

**Current:** Only web UI mode.
**Needed:**
```bash
tinkerdown cli app.md status
tinkerdown cli app.md add --task="Buy milk"
tinkerdown cli app.md run step3
```

**Files to create:**
- `cmd/tinkerdown/commands/cli.go`
- `internal/cli/commands.go`
- `internal/cli/output.go`

---

### Gap 5: HTTP API

**Impact:** HIGH (automation, integrations)
**Effort:** MEDIUM

**Current:** Only WebSocket for state access.
**Needed:**
```
GET  /api/state
GET  /api/state/{field}
POST /api/action/{name}
```

**Files to create:**
- `internal/server/api.go`

---

### Gap 6: Aggregations (sum, count, avg)

**Impact:** HIGH (productivity dashboards)
**Effort:** MEDIUM

**Current:** No way to aggregate data.
**Needed:**
```html
<span lvt-source="expenses" lvt-aggregate="amount:sum"></span>
<!-- or computed source -->
sources:
  total:
    type: computed
    from: expenses
    aggregate: sum(amount)
```

---

### Gap 7: Filtering & Date Ranges

**Impact:** HIGH (productivity tools)
**Effort:** MEDIUM

**Current:** No filtering, show all data.
**Needed:**
```html
<table lvt-source="items" lvt-filter="status:active">
</table>

<table lvt-source="log" lvt-filter="date:today">
</table>
```

---

### Gap 8: Computed Fields

**Impact:** MEDIUM (budget trackers, progress bars)
**Effort:** MEDIUM

**Current:** Can't calculate derived values.
**Needed:**
```yaml
sources:
  budget:
    type: markdown
    computed:
      remaining: "budget - spent"
      percent: "(spent / budget) * 100"
```

---

### Gap 9: Source Caching

**Impact:** MEDIUM (performance)
**Effort:** MEDIUM

**Current:** Every render re-fetches data.
**Needed:**
```yaml
sources:
  users:
    type: rest
    url: https://api.example.com/users
    cache:
      ttl: 5m
```

---

### Gap 10: Error Handling

**Impact:** MEDIUM (reliability)
**Effort:** MEDIUM

**Current:** Errors can crash or hang.
**Needed:**
- Retry with exponential backoff
- Circuit breaker
- User-friendly error messages
- Timeout per source

---

## Feature Gap Matrix

### By Use Case

| Feature | Runbooks | Productivity | Status |
|---------|----------|--------------|--------|
| Auto-timestamp | ⭐⭐⭐ | ⭐⭐ | ❌ Missing |
| Operator identity | ⭐⭐⭐ | ⭐ | ❌ Missing |
| Snapshot capture | ⭐⭐⭐ | ❌ | ❌ Missing |
| Step status tracking | ⭐⭐⭐ | ❌ | ❌ Missing |
| CLI mode | ⭐⭐⭐ | ⭐ | ❌ Missing |
| HTTP API | ⭐⭐ | ⭐ | ❌ Missing |
| Slack integration | ⭐⭐⭐ | ⭐ | ❌ Missing |
| Aggregations | ⭐ | ⭐⭐⭐ | ❌ Missing |
| Filtering | ⭐ | ⭐⭐⭐ | ❌ Missing |
| Date ranges | ⭐⭐ | ⭐⭐⭐ | ❌ Missing |
| Computed fields | ⭐ | ⭐⭐⭐ | ❌ Missing |
| Charts | ❌ | ⭐⭐ | ❌ Missing |
| Caching | ⭐ | ⭐⭐ | ❌ Missing |
| Offline support | ⭐ | ⭐⭐ | ❌ Missing |

### By Roadmap Phase

| Phase | Features | Status |
|-------|----------|--------|
| **Phase 1** | Auto-rendering (tables, lists, selects) | ✅ Complete |
| **Phase 2** | Error handling, caching, multi-page WS | ❌ Not started |
| **Phase 3** | CLI templates, validation, WASM SDK | ⏳ Partial |
| **Phase 4** | GraphQL, MongoDB, source composition | ❌ Not started |
| **Phase 5** | Auth, rate limiting, health endpoints | ❌ Not started |
| **Phase 6** | Components, pagination, themes | ❌ Not started |
| **Phase 7** | Broadcasting, scheduled tasks, API mode | ❌ Not started |
| **Phase 8** | Bundle optimization, accessibility | ❌ Not started |

---

## Detailed Gap Analysis

### Category 1: Core Data Operations

#### 1.1 Auto-Fields on Write

**Status:** ❌ Not Implemented

**What's needed:**
```yaml
sources:
  log:
    type: markdown
    anchor: "#log"
    readonly: false
    auto_fields:
      timestamp: "{{now:2006-01-02 15:04:05}}"
      operator: "{{env:USER}}"
      id: "{{uuid}}"
```

**Template functions needed:**
- `{{now:FORMAT}}` - Current time in Go format
- `{{env:VAR}}` - Environment variable
- `{{uuid}}` - Generate UUID
- `{{operator}}` - Current operator name

**Implementation:**
1. Add `AutoFields map[string]string` to `SourceConfig`
2. Add template evaluation in `WriteItem`
3. Merge auto-fields with form data before write

---

#### 1.2 Aggregations

**Status:** ❌ Not Implemented

**What's needed:**
```html
<!-- Inline aggregation -->
<span lvt-source="expenses" lvt-aggregate="amount:sum"></span>

<!-- Grouped aggregation -->
<table lvt-source="expenses"
       lvt-group-by="category"
       lvt-aggregate="amount:sum">
</table>
```

**Aggregate functions:**
- `sum(field)` - Sum numeric values
- `count()` - Count rows
- `avg(field)` - Average
- `min(field)`, `max(field)` - Extremes

**Implementation:**
1. Parse `lvt-aggregate` attribute
2. Add aggregation to template context
3. Handle group-by in auto-rendering

---

#### 1.3 Filtering

**Status:** ❌ Not Implemented (server-side)

**What's needed:**
```html
<!-- Static filter -->
<table lvt-source="tasks" lvt-filter="status:active"></table>

<!-- Dynamic filter (from input) -->
<input lvt-model="search" placeholder="Search...">
<table lvt-source="tasks" lvt-filter="title:{{search}}"></table>

<!-- Date filter -->
<table lvt-source="log" lvt-filter="date:today"></table>
<table lvt-source="log" lvt-filter="date:this-week"></table>
```

**Implementation:**
1. Parse `lvt-filter` attribute
2. Apply filter before rendering
3. Support special date keywords

---

#### 1.4 Computed Fields

**Status:** ❌ Not Implemented

**What's needed:**
```yaml
sources:
  budget:
    type: markdown
    computed:
      remaining: "budget - spent"
      percent: "round((spent / budget) * 100)"
      status: "remaining < 0 ? 'over' : 'ok'"
```

**Implementation:**
1. Add expression evaluator (CEL or govaluate)
2. Compute fields after fetch
3. Make available in templates

---

### Category 2: Multi-Interface

#### 2.1 CLI Mode

**Status:** ❌ Not Implemented

**What's needed:**
```bash
# Read state
tinkerdown cli app.md status
tinkerdown cli app.md tasks

# Execute actions
tinkerdown cli app.md add --text="Buy milk"
tinkerdown cli app.md delete --id=3
tinkerdown cli app.md toggle --id=2

# Output formats
tinkerdown cli app.md tasks --format=json
tinkerdown cli app.md tasks --format=table

# Connect to running server
tinkerdown cli app.md tasks --server=:8080
```

**Implementation:**
1. Parse markdown and load sources (transient mode)
2. Or connect to running server via API (connected mode)
3. Execute action and print result
4. Support multiple output formats

---

#### 2.2 HTTP API

**Status:** ❌ Not Implemented

**What's needed:**
```
GET  /api/state                  # Full state
GET  /api/state/{source}         # Single source data
POST /api/action/{source}/{action}  # Execute action
GET  /api/sources                # List sources
```

**Response format:**
```json
{
  "ok": true,
  "data": [...],
  "error": null
}
```

**Implementation:**
1. Add API router in server.go
2. Serialize state to JSON
3. Accept action via POST body
4. Optional auth (bearer token)

---

#### 2.3 Slack Bot

**Status:** ❌ Not Implemented

**What's needed:**
```yaml
# tinkerdown.yaml
slack:
  enabled: true
  token: ${SLACK_BOT_TOKEN}
  commands:
    /todo: tasks.md
    /incident: runbook.md
```

**Commands:**
- `/todo list` - Show tasks
- `/todo add "Buy milk"` - Add task
- `/incident new db-recovery` - Start incident

**Implementation:**
1. Slack bot adapter that calls HTTP API
2. Format responses for Slack
3. Interactive buttons via Block Kit

---

### Category 3: Runbook-Specific

#### 3.1 Snapshot Capture

**Status:** ❌ Not Implemented

**What's needed:**
```html
<button lvt-click="snapshot"
        lvt-data-source="containers"
        lvt-data-label="Step 1">
  📸 Capture
</button>
```

**Behavior:**
1. Execute the exec source
2. Format output as markdown code block
3. Append to `#snapshots` section with timestamp

**Output:**
```markdown
## Snapshots {#snapshots}

### 14:32 - Step 1
```
container output here...
```
```

---

#### 3.2 Step Status Tracking

**Status:** ❌ Not Implemented

**What's needed:**
```html
<div lvt-step="1" lvt-log-source="log">
  <button lvt-click="step_start">⏳ Start</button>
  <button lvt-click="step_done">✅ Done</button>
  <button lvt-click="step_failed">❌ Failed</button>
  <button lvt-click="step_skip">⏭️ Skip</button>
</div>
```

**Behavior:**
- Each button adds log entry with timestamp
- Auto-fills operator name
- Updates visual status indicator

---

#### 3.3 Approval Workflow

**Status:** ❌ Not Implemented

**What's needed:**
```html
<button lvt-click="request_approval"
        lvt-data-access="prod-db-write"
        lvt-data-notify="@managers">
  🔐 Request Access
</button>
```

**Behavior:**
1. Add log entry: `access_request`
2. Send Slack notification
3. Approver clicks approve in Slack
4. Add log entry: `access_approved`

---

### Category 4: Productivity-Specific

#### 4.1 Charts

**Status:** ❌ Not Implemented

**What's needed:**
```html
<div lvt-chart="bar"
     lvt-source="expenses"
     lvt-x="category"
     lvt-y="amount">
</div>

<div lvt-chart="line"
     lvt-source="daily_totals"
     lvt-x="date"
     lvt-y="total">
</div>
```

**Implementation:**
- Integrate Chart.js or similar
- Auto-generate chart from source data
- Support bar, line, pie, doughnut

---

#### 4.2 Calendar View

**Status:** ❌ Not Implemented

**What's needed:**
```html
<div lvt-calendar
     lvt-source="events"
     lvt-date-field="date"
     lvt-title-field="title">
</div>
```

**Implementation:**
- Simple month view
- Show items on dates
- Click to view/edit

---

#### 4.3 Kanban Board

**Status:** ❌ Not Implemented

**What's needed:**
```html
<div lvt-kanban
     lvt-source="tasks"
     lvt-status-field="status"
     lvt-columns="todo,doing,done">
</div>
```

**Implementation:**
- Drag-and-drop columns
- Auto-update status on drop
- Use existing CRUD actions

---

### Category 5: Platform Features

#### 5.1 Source Caching

**Status:** ❌ Not Implemented

**What's needed:**
```yaml
sources:
  users:
    type: rest
    url: https://api.example.com/users
    cache:
      ttl: 5m
      strategy: stale-while-revalidate
```

**Implementation:**
1. In-memory cache per source
2. TTL-based expiration
3. Invalidate on write
4. `Refresh` action clears cache

---

#### 5.2 Error Handling

**Status:** ❌ Not Implemented (robust version)

**What's needed:**
- Retry with exponential backoff
- Circuit breaker for failing sources
- Timeout per source
- User-friendly error messages

---

#### 5.3 Build Command

**Status:** ❌ Not Implemented

**What's needed:**
```bash
tinkerdown build app.md -o myapp
./myapp                    # Starts web UI
./myapp serve              # Explicit web mode
./myapp tasks              # CLI mode
```

**Implementation:**
1. Embed markdown in Go binary
2. Embed client assets
3. Cross-compile

---

### Category 6: Distribution & Accessibility

#### 6.1 Desktop App

**Status:** ❌ Not Implemented

**Problem:** Non-technical users (PMs, designers, project managers, engineering managers) won't use CLI.

**What's needed:**

```
┌─────────────────────────────────────────────────────────────────┐
│                    Tinkerdown Desktop                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Recent Apps                              [+ New App]     │  │
│  ├──────────────────────────────────────────────────────────┤  │
│  │  📋 Team Standup          ~/team/standup.md              │  │
│  │  💰 Expense Tracker       ~/personal/expenses.md         │  │
│  │  📚 Reading List          ~/personal/reading.md          │  │
│  │  🔧 Incident Runbook      ~/work/runbooks/db-recovery.md │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  [Open Folder]  [Open from GitHub]  [Create New]                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Features:**
- Double-click .md file → opens in Tinkerdown
- File browser for markdown apps
- Recent files list
- System tray for quick access
- Auto-update mechanism
- Native OS integration (macOS, Windows, Linux)

**Technology options:**
1. **Electron/Tauri** - Bundle web UI with embedded server
2. **Wails** (Go + Web) - Smaller footprint, Go native
3. **Fyne** (Pure Go) - Cross-platform, but limited web rendering

**Recommended:** Tauri (Rust + Web) or Wails (Go + Web)
- Smaller than Electron (~10MB vs 150MB)
- Can embed tinkerdown Go server
- Native performance

**User flow:**
```
1. Download Tinkerdown.app / Tinkerdown.exe
2. Double-click to install
3. Drag markdown file onto app OR open from menu
4. App runs embedded server, opens in embedded browser
5. Edit in app OR in any text editor
```

**Implementation:**
1. Wrap tinkerdown server in Tauri/Wails
2. Add file picker UI
3. Add recent files storage
4. Add auto-update via GitHub releases
5. Code signing for macOS/Windows

**Effort:** Large (2-4 weeks)
**Impact:** Opens tinkerdown to non-developers

---

#### 6.2 Public Hosted Service (tinkerdown.dev)

**Status:** ❌ Not Implemented

**Problem:** Sharing requires recipient to install tinkerdown.

**What's needed:**

```
https://tinkerdown.dev/run?url=github.com/user/repo/blob/main/app.md
https://tinkerdown.dev/run?gist=abc123
https://tinkerdown.dev/run?url=gist.github.com/user/abc123
```

**Features:**
- Fetch any public GitHub/Gist markdown
- Run stateless (no persistence)
- Read-only mode by default
- Optional: temporary state (expires after 1 hour)
- Embed in docs via iframe
- Share button generates link

**Architecture:**
```
┌─────────────────────────────────────────────────────────────────┐
│                      tinkerdown.dev                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  User visits:                                                   │
│  tinkerdown.dev/run?url=github.com/alice/tools/expenses.md     │
│                                                                 │
│           │                                                     │
│           ▼                                                     │
│  ┌─────────────────┐                                           │
│  │  Fetch markdown │ ← GitHub Raw API / Gist API               │
│  └────────┬────────┘                                           │
│           │                                                     │
│           ▼                                                     │
│  ┌─────────────────┐                                           │
│  │  Parse & Render │ ← Tinkerdown runtime                      │
│  └────────┬────────┘                                           │
│           │                                                     │
│           ▼                                                     │
│  ┌─────────────────┐                                           │
│  │  Serve to User  │ ← WebSocket for reactivity                │
│  └─────────────────┘                                           │
│                                                                 │
│  Restrictions:                                                  │
│  • exec source: disabled (security)                            │
│  • sqlite source: in-memory only                               │
│  • markdown source: read-only OR temp state                    │
│  • rest source: allowed (user's responsibility)                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**URL patterns:**
```
# GitHub file
tinkerdown.dev/run?url=github.com/user/repo/blob/main/app.md

# GitHub raw
tinkerdown.dev/run?url=raw.githubusercontent.com/user/repo/main/app.md

# Gist
tinkerdown.dev/run?gist=abc123def456

# Short form
tinkerdown.dev/gh/user/repo/app.md
tinkerdown.dev/gist/abc123
```

**Embed support:**
```html
<iframe src="https://tinkerdown.dev/embed/gh/user/repo/app.md"
        width="100%" height="400">
</iframe>
```

**Landing page:**
```
tinkerdown.dev/

┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│           Run Markdown Apps Instantly                           │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │  Paste GitHub URL or Gist ID:                             │ │
│  │  [github.com/user/repo/blob/main/app.md    ] [▶ Run]     │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  Examples:                                                      │
│  • Expense Tracker   [Try it →]                                │
│  • Team Standup      [Try it →]                                │
│  • Reading List      [Try it →]                                │
│                                                                 │
│  ────────────────────────────────────────────────              │
│                                                                 │
│  Want to run locally?                                          │
│  brew install tinkerdown                                       │
│  # or                                                          │
│  Download Tinkerdown Desktop                                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Security considerations:**
- No exec source (arbitrary command execution)
- No file system access
- Rate limiting per IP
- Content Security Policy
- Sandbox iframes for embeds
- Report abuse mechanism

**Business model (optional):**
- Free: public repos, read-only, temp state (1 hour)
- Pro: private repos, persistent state, custom domain

**Implementation:**
1. Deploy tinkerdown as serverless function or container
2. Add GitHub/Gist URL fetcher
3. Restrict dangerous source types
4. Add landing page
5. Add embed support
6. Add rate limiting

**Effort:** Medium-Large (2-3 weeks)
**Impact:** Viral sharing, "try before install", documentation embeds

---

#### 6.3 Share Button / Export Link

**Status:** ❌ Not Implemented

**What's needed:**
When running tinkerdown locally, a "Share" button that:
1. Uploads markdown to Gist (with user's token)
2. Returns tinkerdown.dev/gist/xxx URL
3. Copies to clipboard

```html
<!-- In tinkerdown UI -->
<button id="share">📤 Share</button>

<!-- Creates -->
https://tinkerdown.dev/gist/abc123
```

**Implementation:**
1. Add Share button to UI
2. Prompt for GitHub token (store locally)
3. Create Gist via API
4. Return hosted URL

---

#### 5.4 Authentication

**Status:** ❌ Not Implemented

**What's needed:**
```yaml
auth:
  provider: basic
  users:
    - username: admin
      password: ${ADMIN_PASSWORD}
# or
auth:
  provider: oauth
  client_id: ${OAUTH_CLIENT_ID}
```

---

#### 5.5 Webhooks

**Status:** ❌ Not Implemented

**What's needed:**
```yaml
webhooks:
  - url: https://chat.example.com/api/...
    events: [incident.created, step.completed]
  - url: https://alerts.example.com/...
    events: [incident.resolved]
```

---

## Priority Recommendations

### Immediate (P0) - Enables Core Use Cases

| Feature | Effort | Impact | Dependencies |
|---------|--------|--------|--------------|
| Auto-timestamp on submit | Small | Very High | None |
| Operator identity | Small | High | None |
| Snapshot capture | Medium | Very High | None |
| Step status buttons | Medium | High | Auto-timestamp |

**Rationale:** These 4 features unlock the entire runbook use case.

### Near-Term (P1) - Multiplies Value

| Feature | Effort | Impact | Dependencies |
|---------|--------|--------|--------------|
| HTTP API | Medium | Very High | None |
| CLI mode | Medium | Very High | HTTP API |
| Filtering (lvt-filter) | Medium | High | None |
| Aggregations | Medium | High | None |

**Rationale:** Multi-interface support (API + CLI) opens automation and makes tinkerdown usable from anywhere.

### Medium-Term (P2) - Polish & Scale

| Feature | Effort | Impact | Dependencies |
|---------|--------|--------|--------------|
| Source caching | Medium | Medium | None |
| Error handling | Medium | Medium | None |
| Computed fields | Medium | Medium | None |
| Date range filtering | Medium | Medium | Filtering |
| Build command | Medium | High | None |

### Long-Term (P3) - Ecosystem

| Feature | Effort | Impact | Dependencies |
|---------|--------|--------|--------------|
| Slack bot | Large | High | HTTP API |
| Charts | Large | Medium | None |
| Calendar view | Large | Medium | None |
| Kanban board | Large | Medium | None |
| Authentication | Large | Medium | None |
| Webhooks | Medium | Medium | HTTP API |

### Distribution (P1-P2) - Reach Non-Developers

| Feature | Effort | Impact | Dependencies |
|---------|--------|--------|--------------|
| Desktop App | Large | Very High | Build command |
| tinkerdown.dev (hosted) | Medium-Large | Very High | None |
| Share button | Small | High | tinkerdown.dev |

**Rationale:** These features expand the audience beyond developers to PMs, designers, and managers who won't use CLI.

---

## Summary

### The Gaps in One Sentence

> Tinkerdown can display and edit data, but can't track time, operators, or aggregates—and has no way to reach non-developers.

### Top Missing Features

**For Developers (Core Functionality):**
1. **Auto-timestamp** - Every log entry needs manual time
2. **Operator identity** - No "who did this"
3. **Snapshot capture** - Can't freeze point-in-time state
4. **CLI mode** - Only web UI, can't script or SSH
5. **Aggregations** - Can't sum expenses or count tasks

**For Non-Developers (Distribution):**
6. **Desktop App** - PMs/designers won't use CLI
7. **tinkerdown.dev** - Can't share without recipient installing

### Recommended Next Steps

**Phase 1: Core Features (2-3 weeks)**
1. Implement auto-timestamp + operator identity (1-2 days)
2. Implement snapshot capture (2-3 days)
3. Implement HTTP API (3-5 days)
4. Implement CLI mode (3-5 days)
5. Implement filtering + aggregations (3-5 days)

**Phase 2: Distribution (3-4 weeks)**
6. Implement build command (3-5 days)
7. Build Desktop App with Tauri/Wails (2-3 weeks)
8. Launch tinkerdown.dev hosted service (2-3 weeks)

### Target Audiences After Completion

| Audience | How They Use Tinkerdown |
|----------|------------------------|
| **Developers** | CLI, text editor, git |
| **DevOps/SRE** | Runbooks, CLI during incidents |
| **Product Managers** | Desktop app, shared links |
| **Designers** | Desktop app, embedded in docs |
| **Engineering Managers** | Desktop app, team dashboards |
| **Anyone** | tinkerdown.dev links, no install |

**Total estimated effort:**
- Core features: 2-3 weeks
- Distribution: 3-4 weeks
- **Total: 5-7 weeks to full platform**

After this, tinkerdown serves both use cases (runbooks + productivity) and both audiences (developers + non-developers).
