# Tinkerdown: Unified Design & Implementation Plan

**Date:** 2025-12-31
**Version:** 2.0
**Status:** Final Design

---

## Executive Summary

Tinkerdown turns markdown into apps. **If it's valid markdown, it's a working app.**

```markdown
# Todo
- [ ] Buy milk
```

That's a complete app. Two lines. No YAML. No HTML. No configuration.

### One Command, Three Modes

```bash
tinkerdown serve app.md              # Interactive UI
tinkerdown serve app.md --headless   # Background automation
tinkerdown serve app.md --cli        # Terminal interface
```

### The Three Pillars

| Pillar | What | Example |
|--------|------|---------|
| **Runbooks** | Incident procedures with tracking | `# DB Recovery` with steps |
| **Productivity** | Personal/team trackers | `## Tasks` with items |
| **Automation** | Scheduled bots, event handlers | `@daily:9am` triggers |

### Why Tinkerdown

```
┌─────────────────────────────────────────────────────────────────┐
│                    THE TINKERDOWN PROMISE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Need something?                                                │
│       │                                                         │
│       ▼                                                         │
│  Ask LLM → Get markdown → tinkerdown serve → Done               │
│                                                                 │
│  Properties:                                                    │
│  • Pure markdown (no HTML/YAML for 80% of apps)                 │
│  • Schema inferred from data                                    │
│  • Scheduling via @mentions                                     │
│  • Throwaway-OK (cheap to create, OK to discard)                │
│  • Data portable (git-backed, grep-able, yours forever)         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Table of Contents

1. [Markdown-Native Design](#markdown-native-design)
2. [Architecture](#architecture)
3. [Feature Dependency Graph](#feature-dependency-graph)
4. [Progressive Implementation Plan](#progressive-implementation-plan)
5. [Feature Specifications](#feature-specifications)
6. [Example Apps](#example-apps)
7. [Security Considerations](#security-considerations)
8. [Distribution Strategy](#distribution-strategy)

---

## Markdown-Native Design

### Core Principle

Markdown already has data structures. Tinkerdown recognizes them:

| Markdown | Schema | UI |
|----------|--------|-----|
| `- [ ] task` | `{text, done: bool}` | Checkbox list |
| `- item` | `{text}` | Simple list |
| `1. item` | `{text, order: int}` | Ordered list |
| `\| col \| col \|` | `{col: type, ...}` | Table with form |
| `term: def` | `{term, definition}` | Key-value list |

### Headings as Data Sources

The heading names the data. No anchors needed:

```markdown
## Tasks
- [ ] Design API
- [x] Write tests
- [ ] Deploy

## Expenses
| date | category | amount |
|------|----------|--------|
| 2024-01-15 | Food | $45.50 |
```

**System infers:**
- `## Tasks` → source named "tasks"
- `## Expenses` → source named "expenses"
- Task list → schema `{text, done}`
- Table → schema from columns and value patterns

### Automatic Schema Inference

Types inferred from data patterns:

| Pattern | Type | Input UI |
|---------|------|----------|
| `2024-01-15` | date | Date picker |
| `14:30` / `2:30pm` | time | Time picker |
| `123` / `45.67` | number | Number input |
| `$123.45` | currency | Currency input |
| `true` / `false` | boolean | Checkbox |
| `hello@example.com` | email | Email input |
| `https://...` | url | URL input |
| 3-10 unique strings | select | Dropdown |
| Everything else | text | Text input |

**Validation inference:**
- Value in every row → required
- Some rows empty → optional

### Scheduling with @mentions

Date mentions become triggers:

```markdown
## Tasks
- [ ] Submit report @friday
- [ ] Call dentist @tomorrow @9am
- [ ] Renew passport @2024-03-01
- [ ] Review PR @in:2hours

## Reminders
- Pay rent @monthly:1st
- Standup @daily:9am @weekdays
- Backup @weekly:sun @2am
```

**Date grammar:**

| Syntax | Meaning |
|--------|---------|
| `@today` | Today |
| `@tomorrow` | Tomorrow |
| `@friday` | Next Friday |
| `@2024-03-15` | Specific date |
| `@9am` | Time today |
| `@friday @3pm` | Date + time |
| `@in:2hours` | Relative |
| `@daily:9am` | Recurring daily |
| `@weekly:mon,wed` | Recurring weekly |
| `@monthly:1st` | Recurring monthly |
| `@yearly:mar-15` | Recurring yearly |

### Computed Values

Inline code with expressions:

```markdown
## Budget

**Total Income:** `sum(income.amount)`
**Total Expenses:** `sum(expenses.amount)`
**Balance:** `sum(income.amount) - sum(expenses.amount)`
**Tasks Done:** `count(tasks where done)` / `count(tasks)`
```

**Expression functions:**

| Function | Example |
|----------|---------|
| `count(source)` | `count(tasks)` |
| `count(source where expr)` | `count(tasks where done)` |
| `sum(source.field)` | `sum(expenses.amount)` |
| `avg(source.field)` | `avg(scores.value)` |
| `min/max(source.field)` | `min(tasks.due)` |

### Status Banners

Blockquotes with emoji:

```markdown
> ✅ All systems operational

> ⚠️ `count(tasks where due < today and not done)` overdue

> 📊 `count(deals)` active deals | `sum(deals.value)` pipeline
```

### Actions

Button links and action links:

```markdown
[Add Task]                           → Button
[Clear Completed](action:clear-done) → Action link
[Export CSV](export:csv)             → Export
[← Back](back)                       → Navigation
```

### Tabs and Views

Bracketed heading names:

```markdown
## [All] Tasks | [Active] not done | [Done] done
```

Creates tabbed interface with automatic filtering.

### Outputs (Automation)

Header block for notifications:

```markdown
# Daily Standup Bot

> Slack: #team-updates
> Email: team@company.com

## Questions
- What did you do yesterday?
- What will you do today?

Notify @daily:9am @weekdays
```

---

## The Layered Approach

Three layers for different complexity:

### Layer 1: Pure Markdown (80% of apps)

```markdown
# Expense Tracker

## Expenses
| date | category | amount | note |
|------|----------|--------|------|
| 2024-01-15 | Food | $45 | Groceries |

## Summary
**Total:** `sum(expenses.amount)`
```

No YAML. No HTML. Just markdown.

### Layer 2: YAML for Advanced Features

```yaml
---
sources:
  tasks:
    schema:
      priority: select | Critical, High, Medium, Low
      title: required | min:3
    input: false  # read-only

outputs:
  slack:
    channel: "#alerts"
    token: ${SLACK_TOKEN}
---
```

YAML only for:
- External data sources (REST, DB)
- Custom validation rules
- Output configuration
- Complex actions

### Layer 3: HTML for Full Control

```html
<form lvt-submit="add" lvt-source="items">
  <input name="title" required>
  <button type="submit">Add</button>
</form>

{{range .items}}
<div class="card">
  <h3>{{.title}}</h3>
</div>
{{end}}
```

HTML + Go templates when you need:
- Custom layouts
- Complex interactions
- Conditional rendering

---

## Architecture

### Single Command, Multiple Modes

```
┌─────────────────────────────────────────────────────────────────┐
│                      tinkerdown serve                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  INPUT: app.md                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  # My App                                                │   │
│  │                                                          │   │
│  │  ## Tasks                                                │   │
│  │  - [ ] Example task @tomorrow                            │   │
│  │                                                          │   │
│  │  **Done:** `count(tasks where done)`                     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│              ┌───────────────┼───────────────┐                 │
│              ▼               ▼               ▼                 │
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐       │
│  │  Web Server   │  │   Triggers    │  │   HTTP API    │       │
│  │  (default)    │  │  (@mentions)  │  │  (always)     │       │
│  └───────────────┘  └───────────────┘  └───────────────┘       │
│                                                                 │
│  FLAGS:                                                         │
│  --headless     Skip web server, just run triggers              │
│  --port 8080    Web server port                                 │
│  --operator X   Set operator identity                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                        DATA FLOW                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  MARKDOWN SOURCES (auto-detected)                               │
│  ─────────────────────────────────                              │
│  ## Heading     ──┐                                             │
│  - [ ] tasks    ──┼──▶  Parse  ──▶  Infer Schema  ──▶  State   │
│  | table |      ──┤                                    │        │
│  - list         ──┘                                    │        │
│                                                        │        │
│  EXTERNAL SOURCES (YAML config)                        │        │
│  ──────────────────────────────                        │        │
│  rest: url      ──┐                                    │        │
│  sqlite: db     ──┼──▶  Fetch  ────────────────────────┘        │
│  postgres: dsn  ──┘                                             │
│                                                                 │
│  State  ──▶  Render  ──▶  UI / API / CLI                       │
│    │                                                            │
│    │  ACTIONS (modify state)                                    │
│    │  ────────────────────────                                  │
│    └──  add, delete, update, toggle  ◀──  User / Trigger       │
│                                                                 │
│  TRIGGERS (invoke actions)                                      │
│  ─────────────────────────                                      │
│  @daily:9am     ──┐                                             │
│  @friday        ──┼──▶  Execute Action  ──▶  Update State      │
│  webhook: /hook ──┤                              │              │
│  watch: *.pdf   ──┘                              ▼              │
│                                              OUTPUTS            │
│                                              ───────            │
│                                              > Slack: #ch       │
│                                              > Email: addr      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Feature Dependency Graph

```
┌─────────────────────────────────────────────────────────────────┐
│                    FEATURE DEPENDENCIES                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  LAYER 0: Foundation (unlocks markdown-native)                  │
│  ─────────────────────────────────────────────                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Heading   │  │  Table/List │  │   Schema    │             │
│  │   as Anchor │  │   Parsing   │  │  Inference  │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         └────────────────┼────────────────┘                     │
│                          ▼                                      │
│  LAYER 1: Core Features                                        │
│  ──────────────────────                                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │Auto-timestamp│  │  Computed   │  │  HTTP API   │             │
│  │+ Operator   │  │  Expressions│  │  + CLI mode │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                │                     │
│         ▼                ▼                ▼                     │
│  LAYER 2: Pillar Essentials                                    │
│  ──────────────────────────                                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  Snapshot   │  │   Tabs &    │  │  @schedule  │             │
│  │  + Steps    │  │  Filtering  │  │   Triggers  │             │
│  │  (Runbook)  │  │  (Product.) │  │  (Automat.) │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                │                     │
│         ▼                ▼                ▼                     │
│  LAYER 3: Pillar Completion                                    │
│  ──────────────────────────                                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │Status Banner│  │   Charts    │  │  Webhooks   │             │
│  │ + Actions   │  │ + Exports   │  │  + Outputs  │             │
│  └─────────────┘  └─────────────┘  └──────┬──────┘             │
│                                           │                     │
│                                           ▼                     │
│  LAYER 4: Distribution                                         │
│  ─────────────────────                                         │
│  ┌─────────────────────────────────────────────────┐           │
│  │  Build command / Desktop app / tinkerdown.dev   │           │
│  └─────────────────────────────────────────────────┘           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Feature → Pillar Matrix

| Feature | Runbooks | Productivity | Automation | Layer |
|---------|:--------:|:------------:|:----------:|:-----:|
| Heading as anchor | ✓ | ✓ | ✓ | 0 |
| Table/list parsing | ✓ | ✓ | ✓ | 0 |
| Schema inference | ✓ | ✓ | ✓ | 0 |
| Auto-timestamp | ✓ | ✓ | ✓ | 1 |
| Operator identity | ✓ | ○ | ○ | 1 |
| Computed expressions | ○ | ✓ | ○ | 1 |
| HTTP API | ✓ | ✓ | ✓ | 1 |
| CLI mode | ✓ | ○ | ✓ | 1 |
| Snapshot capture | ✓ | | | 2 |
| Step buttons | ✓ | | | 2 |
| Tabs & filtering | ○ | ✓ | ○ | 2 |
| @schedule triggers | | | ✓ | 2 |
| Status banners | ✓ | ✓ | ○ | 3 |
| Action buttons | ✓ | ✓ | ○ | 3 |
| Charts | | ✓ | | 3 |
| Exports (CSV/PDF) | ○ | ✓ | ○ | 3 |
| Webhook triggers | ○ | | ✓ | 3 |
| Outputs (Slack/Email) | ✓ | ○ | ✓ | 3 |
| Build command | ✓ | ✓ | ✓ | 4 |
| Desktop app | ○ | ✓ | | 4 |

✓ = Critical, ○ = Useful, blank = Not applicable

---

## Progressive Implementation Plan

This implementation plan is designed to be followed by an LLM across multiple sessions. Each task includes:

- **Prerequisites**: What must be done before starting
- **Files to modify/create**: Exact paths
- **Integration points**: How it connects to existing code
- **Acceptance criteria**: How to verify completion
- **Verification commands**: Tests to run

**Before starting any task:**
1. Read the Prerequisites section
2. Check if prerequisite tasks are complete
3. Read the relevant existing files listed
4. Follow the implementation steps
5. Run verification commands
6. Update the task status in this document

### Current Codebase Structure

```
tinkerdown/
├── cmd/
│   └── tinkerdown/          # CLI entry point
├── internal/
│   ├── source/              # Data sources (markdown, sqlite, rest, etc.)
│   │   ├── source.go        # Source interface
│   │   ├── markdown.go      # Markdown table/list source
│   │   └── ...
│   ├── server/              # HTTP server, WebSocket
│   ├── runtime/             # Actions, state management
│   ├── markdown/            # Markdown parser
│   ├── config/              # Configuration
│   └── compiler/            # Template compilation
├── web/                     # Frontend assets
└── examples/                # Working examples
```

### Overview: 12 Weeks, 6 Phases

```
┌─────────────────────────────────────────────────────────────────┐
│                    12-WEEK IMPLEMENTATION                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  PHASE 1 (Wk 1-2)    PHASE 2 (Wk 3-4)    PHASE 3 (Wk 5-6)     │
│  Markdown-Native     Core Features       Pillar Features       │
│                                                                 │
│  • Heading anchors   • Auto-timestamp    • Snapshots           │
│  • Table parsing     • Operator ID       • Step buttons        │
│  • List parsing      • Expressions       • Tabs                │
│  • Schema inference  • HTTP API          • Status banners      │
│  • Auto-CRUD UI      • CLI mode          • Action buttons      │
│                                                                 │
│  PHASE 4 (Wk 7-8)    PHASE 5 (Wk 9-10)   PHASE 6 (Wk 11-12)   │
│  Triggers/Outputs    Distribution        Polish                │
│                                                                 │
│  • @schedule parse   • Build command     • Charts              │
│  • Schedule runner   • Desktop app       • Templates           │
│  • Webhooks          • tinkerdown.dev    • Documentation       │
│  • Slack/Email                                                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Task Status Summary

| Phase | Task | Status |
|-------|------|--------|
| 1 | 1.1 Heading-as-Anchor | `[ ]` |
| 1 | 1.2 Table Parsing | `[ ]` |
| 1 | 1.3 List Parsing | `[ ]` |
| 1 | 1.4 Schema Inference | `[ ]` |
| 1 | 1.5 Auto-CRUD UI | `[ ]` |
| 2 | 2.1 Auto-Timestamp | `[ ]` |
| 2 | 2.2 Operator Identity | `[ ]` |
| 2 | 2.3 Computed Expressions | `[ ]` |
| 2 | 2.4 HTTP API | `[ ]` |
| 2 | 2.5 CLI Mode | `[ ]` |
| 3 | 3.1 Snapshot Capture | `[ ]` |
| 3 | 3.2 Step Status | `[ ]` |
| 3 | 3.3 Tabs | `[ ]` |
| 3 | 3.4 Status Banners | `[ ]` |
| 3 | 3.5 Action Buttons | `[ ]` |
| 4 | 4.1 @schedule Parsing | `[ ]` |
| 4 | 4.2 Schedule Runner | `[ ]` |
| 4 | 4.3 Webhook Triggers | `[ ]` |
| 4 | 4.4 Output Integrations | `[ ]` |
| 5 | 5.1 Build Command | `[ ]` |
| 5 | 5.2 Desktop App | `[ ]` |
| 5 | 5.3 Hosted Service | `[ ]` |
| 6 | 6.1 Charts | `[ ]` |
| 6 | 6.2 Template Gallery | `[ ]` |
| 6 | 6.3 Documentation | `[ ]` |

---

### Phase 1: Markdown-Native Foundation (Week 1-2)

#### Task 1.1: Heading-as-Anchor Detection

**Status:** `[ ] Not Started`

**Goal:** Recognize `## Heading` as a data source anchor without explicit `#anchor` syntax.

**Prerequisites:** None (first task)

**Files to read first:**
- `internal/source/markdown.go` - Current markdown source implementation
- `internal/markdown/parser.go` - Markdown parsing utilities
- `internal/config/config.go` - How sources are configured

**Files to modify:**
- `internal/markdown/parser.go` - Add heading detection

**Files to create:**
- `internal/markdown/sources.go` - Auto-detected source extraction

**Implementation steps:**

1. Create `internal/markdown/sources.go`:
```go
package markdown

// AutoSource represents a data source auto-detected from markdown structure
type AutoSource struct {
    Name      string     // Derived from heading (e.g., "## Tasks" -> "tasks")
    Heading   string     // Original heading text
    Type      SourceType // table, task_list, unordered_list, ordered_list, definition_list
    StartLine int        // Line number where data starts
    EndLine   int        // Line number where data ends
    RawData   string     // Raw markdown content of the data
}

type SourceType int

const (
    SourceTypeTable SourceType = iota
    SourceTypeTaskList
    SourceTypeUnorderedList
    SourceTypeOrderedList
    SourceTypeDefinitionList
)

// DetectSources scans markdown content and returns all auto-detected sources
func DetectSources(content string) []AutoSource {
    // Implementation:
    // 1. Split content into lines
    // 2. Find all ## headings
    // 3. For each heading, check what follows:
    //    - Lines starting with "| " -> table
    //    - Lines starting with "- [ ]" or "- [x]" -> task_list
    //    - Lines starting with "- " -> unordered_list
    //    - Lines starting with "1. " -> ordered_list
    //    - Lines with term\n: definition pattern -> definition_list
    // 4. Extract the data block until next heading or EOF
}

// HeadingToSourceName converts "## My Tasks" to "my-tasks"
func HeadingToSourceName(heading string) string {
    // Remove ## prefix, lowercase, replace spaces with hyphens
}
```

2. Add tests in `internal/markdown/sources_test.go`

**Acceptance criteria:**
- [ ] `DetectSources()` correctly identifies tables after headings
- [ ] `DetectSources()` correctly identifies task lists after headings
- [ ] `DetectSources()` correctly identifies unordered lists after headings
- [ ] `HeadingToSourceName()` converts headings to valid source names
- [ ] All tests pass

**Verification commands:**
```bash
go test ./internal/markdown/... -v -run TestDetectSources
go test ./internal/markdown/... -v -run TestHeadingToSourceName
```

---

#### Task 1.2: Table Parsing

**Status:** `[ ] Not Started`

**Goal:** Parse markdown tables into structured data with column names and rows.

**Prerequisites:** Task 1.1 complete

**Files to read first:**
- `internal/source/markdown.go` - See how tables are currently parsed
- `internal/markdown/sources.go` - From Task 1.1

**Files to create:**
- `internal/markdown/table.go` - Table-specific parsing

**Implementation steps:**

1. Create `internal/markdown/table.go`:
```go
package markdown

// TableData represents parsed markdown table
type TableData struct {
    Columns []string             // Column names from header row
    Rows    []map[string]string  // Each row as column->value map
}

// ParseTable extracts structured data from markdown table
func ParseTable(raw string) (*TableData, error) {
    // 1. Split into lines
    // 2. First line is header: | col1 | col2 | col3 |
    // 3. Second line is separator: |------|------|------|
    // 4. Remaining lines are data rows
    // 5. Handle: empty cells, escaped |, whitespace trimming
}
```

**Acceptance criteria:**
- [ ] Correctly parses simple markdown tables
- [ ] Handles empty cells
- [ ] Handles cells with special characters
- [ ] Returns column names in order

**Verification commands:**
```bash
go test ./internal/markdown/... -v -run TestParseTable
```

---

#### Task 1.3: List Parsing

**Status:** `[ ] Not Started`

**Goal:** Parse markdown lists (task lists, ordered, unordered) into structured data.

**Prerequisites:** Task 1.1 complete

**Files to create:**
- `internal/markdown/list.go` - List-specific parsing

**Implementation steps:**

```go
package markdown

type TaskItem struct {
    Text string
    Done bool
}

type ListItem struct {
    Text  string
    Order int // For ordered lists, 0 for unordered
}

func ParseTaskList(raw string) []TaskItem
func ParseUnorderedList(raw string) []ListItem
func ParseOrderedList(raw string) []ListItem
func ParseDefinitionList(raw string) []map[string]string
```

**Acceptance criteria:**
- [ ] Correctly parses task lists with checked/unchecked items
- [ ] Correctly parses unordered lists
- [ ] Correctly parses ordered lists (preserving order)
- [ ] Correctly parses definition lists

**Verification commands:**
```bash
go test ./internal/markdown/... -v -run TestParse
```

---

#### Task 1.4: Schema Inference

**Status:** `[ ] Not Started`

**Goal:** Automatically infer field types from data values.

**Prerequisites:** Tasks 1.2 and 1.3 complete

**Files to create:**
- `internal/markdown/schema.go` - Schema inference logic

**Implementation steps:**

```go
package markdown

import "regexp"

type FieldType int

const (
    FieldTypeText FieldType = iota
    FieldTypeNumber
    FieldTypeCurrency
    FieldTypeDate
    FieldTypeTime
    FieldTypeBoolean
    FieldTypeEmail
    FieldTypeURL
    FieldTypeSelect
)

type Field struct {
    Name     string
    Type     FieldType
    Required bool
    Options  []string // For FieldTypeSelect
}

var (
    datePattern     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
    currencyPattern = regexp.MustCompile(`^\$[\d,]+(\.\d{2})?$`)
    // ... other patterns
)

func InferSchema(columns []string, rows []map[string]string) []Field
func InferFieldType(values []string) FieldType
```

**Acceptance criteria:**
- [ ] Correctly identifies date fields (YYYY-MM-DD format)
- [ ] Correctly identifies currency fields ($XX.XX format)
- [ ] Correctly identifies number, boolean, email, URL fields
- [ ] Identifies select fields when <= 10 unique values
- [ ] Correctly determines required vs optional fields

**Verification commands:**
```bash
go test ./internal/markdown/... -v -run TestInferSchema
```

---

#### Task 1.5: Auto-CRUD UI Generation

**Status:** `[ ] Not Started`

**Goal:** Automatically generate form + display UI for auto-detected sources.

**Prerequisites:** Tasks 1.1-1.4 complete

**Files to modify:**
- `internal/server/server.go` - Add auto-source detection to render pipeline

**Files to create:**
- `internal/render/autosource.go` - Auto-generate HTML for sources
- `web/components/auto-form.html` - Form template
- `web/components/auto-table.html` - Table template
- `web/components/auto-tasklist.html` - Task list template

**Implementation steps:**

1. Create render pipeline that detects auto-sources and generates HTML
2. Form generation based on schema type → input type mapping
3. Display generation for tables, task lists, other lists

**Acceptance criteria:**
- [ ] Auto-detected tables render with form above
- [ ] Auto-detected task lists render with add input
- [ ] Form fields match inferred schema types
- [ ] Add/toggle/delete actions work for all source types

**Verification commands:**
```bash
# Create test file and run
cat > /tmp/test-auto.md << 'EOF'
# Test App

## Tasks
- [ ] Test task 1
- [x] Test task 2
EOF

tinkerdown serve /tmp/test-auto.md
go test ./... -v -run TestAutoSource
```

---

### Phase 2: Core Features (Week 3-4)

#### Task 2.1: Auto-Timestamp

**Status:** `[ ] Not Started`

**Goal:** Auto-fill `{{now}}` template with current timestamp on form submit.

**Prerequisites:** Phase 1 complete

**Files to modify:**
- `internal/runtime/actions.go` - Add auto-field processing

**Acceptance criteria:**
- [ ] `{{now:2006-01-02}}` fills with current date
- [ ] `{{now:15:04}}` fills with current time
- [ ] Auto-detected sources with date/time columns auto-fill

---

#### Task 2.2: Operator Identity

**Status:** `[ ] Not Started`

**Goal:** Support `--operator` flag and `{{operator}}` template.

**Prerequisites:** Task 2.1 complete

**Files to modify:**
- `cmd/tinkerdown/main.go` - Add --operator flag
- `internal/config/config.go` - Store operator
- `internal/runtime/actions.go` - Process {{operator}} template

**Acceptance criteria:**
- [ ] `tinkerdown serve app.md --operator alice` sets operator
- [ ] `{{operator}}` resolves to operator value
- [ ] Operator available in templates as `.Operator`

---

#### Task 2.3: Computed Expressions

**Status:** `[ ] Not Started`

**Goal:** Evaluate inline expressions like `` `sum(expenses.amount)` ``.

**Prerequisites:** Phase 1 complete

**Files to create:**
- `internal/expr/parser.go` - Expression parser
- `internal/expr/eval.go` - Expression evaluator
- `internal/expr/functions.go` - Built-in functions (count, sum, avg, min, max)

**Acceptance criteria:**
- [ ] `count(tasks)` returns number of items
- [ ] `count(tasks where done)` returns filtered count
- [ ] `sum(expenses.amount)` returns sum
- [ ] Expressions update on WebSocket state change

---

#### Task 2.4: HTTP API

**Status:** `[ ] Not Started`

**Goal:** Expose REST API for state and actions.

**Prerequisites:** Phase 1 complete

**Files to create:**
- `internal/server/api.go` - API handlers

**Routes:**
```
GET  /api/state              → Full state JSON
GET  /api/sources/{name}     → Single source data
POST /api/sources/{name}     → Add item
DELETE /api/sources/{name}/{id} → Delete item
POST /api/action/{name}      → Execute action
```

**Acceptance criteria:**
- [ ] All routes work as documented
- [ ] API respects CORS headers

---

#### Task 2.5: CLI Mode

**Status:** `[ ] Not Started`

**Goal:** Command-line interface for actions without browser.

**Prerequisites:** Task 2.4 complete

**Files to create:**
- `cmd/tinkerdown/cli.go` - CLI subcommand

**Commands:**
```bash
tinkerdown cli app.md sources
tinkerdown cli app.md data tasks
tinkerdown cli app.md add tasks --text="Buy milk"
tinkerdown cli app.md delete tasks --id=3
```

**Acceptance criteria:**
- [ ] All commands work as documented
- [ ] Changes persist to markdown file

---

### Phase 3: Pillar Features (Week 5-6)

#### Task 3.1: Snapshot Capture

**Status:** `[ ] Not Started`

**Goal:** Capture exec source output at a point in time for runbooks.

**Prerequisites:** Phase 2 complete

---

#### Task 3.2: Step Status Buttons

**Status:** `[ ] Not Started`

**Goal:** Track runbook step completion status.

**Prerequisites:** Task 3.1 complete

---

#### Task 3.3: Tabs

**Status:** `[ ] Not Started`

**Goal:** Parse `## [Tab] Label` syntax for tabbed views.

**Prerequisites:** Phase 2 complete

**Files to create:**
- `internal/markdown/tabs.go` - Tab parsing

---

#### Task 3.4: Status Banners

**Status:** `[ ] Not Started`

**Goal:** Render `> emoji text` as styled status banners.

**Prerequisites:** Phase 2 complete

---

#### Task 3.5: Action Buttons

**Status:** `[ ] Not Started`

**Goal:** Parse `[Button Text]` and `[Text](action:name)` as action buttons.

**Prerequisites:** Phase 2 complete

---

### Phase 4: Triggers & Outputs (Week 7-8)

#### Task 4.1: @schedule Parsing

**Status:** `[ ] Not Started`

**Goal:** Parse `@daily:9am` mentions from markdown content.

**Prerequisites:** Phase 3 complete

**Files to create:**
- `internal/triggers/parser.go` - @mention parser
- `internal/triggers/schedule.go` - Cron conversion

---

#### Task 4.2: Schedule Runner

**Status:** `[ ] Not Started`

**Goal:** Execute triggers at scheduled times.

**Prerequisites:** Task 4.1 complete

**Files to create:**
- `internal/triggers/runner.go` - Cron runner using robfig/cron

---

#### Task 4.3: Webhook Triggers

**Status:** `[ ] Not Started`

**Goal:** Accept incoming webhooks to trigger actions.

**Prerequisites:** Task 4.2 complete

---

#### Task 4.4: Output Integrations

**Status:** `[ ] Not Started`

**Goal:** Send notifications to Slack, email, webhooks.

**Prerequisites:** Task 4.2 complete

**Files to create:**
- `internal/outputs/slack.go`
- `internal/outputs/email.go`
- `internal/outputs/webhook.go`

---

### Phase 5: Distribution (Week 9-10)

#### Task 5.1: Build Command

**Status:** `[ ] Not Started`

**Goal:** Bundle app into standalone executable.

**Prerequisites:** Phase 4 complete

---

#### Task 5.2: Desktop App

**Status:** `[ ] Not Started`

**Goal:** Wails-based desktop application.

**Prerequisites:** Task 5.1 complete

---

#### Task 5.3: Hosted Service

**Status:** `[ ] Not Started`

**Goal:** tinkerdown.dev for running apps from GitHub.

**Prerequisites:** Task 5.1 complete

---

### Phase 6: Polish (Week 11-12)

#### Task 6.1: Charts

**Status:** `[ ] Not Started`

**Goal:** Render chart code blocks as visualizations.

---

#### Task 6.2: Template Gallery

**Status:** `[ ] Not Started`

**Goal:** Collection of starter templates.

---

#### Task 6.3: Documentation

**Status:** `[ ] Not Started`

**Goal:** Complete user documentation.

---

### Session Handoff

When ending a session, update this section:

**Last Updated:** [DATE]
**Last Task Completed:** [TASK ID]
**Next Task:** [TASK ID]
**Blockers:** [Any issues encountered]
**Notes for Next Session:**
- [Important context]
- [Decisions made]
- [Files modified]

---

## Feature Specifications

### Complete Markdown Grammar

```
# DOCUMENT STRUCTURE
─────────────────────────────────────────────────────────

# Heading 1              → App title
## Heading 2             → Section + data source (if followed by data)
## [Tab] Label           → Tab navigation
### Heading 3            → Subsection


# DATA SOURCES
─────────────────────────────────────────────────────────

## Tasks                 → Source "tasks" (name from heading)
- [ ] Task item          → Task list: {text, done: bool}

## Shopping              → Source "shopping"
- Item                   → Simple list: {text}

## Steps                 → Source "steps"
1. First step            → Ordered list: {text, order: int}

## Contacts              → Source "contacts"
| name | email |         → Table: {name, email, ...}
|------|-------|
| Alice | alice@co |

## Config                → Source "config"
API Key                  → Definition list: {term, definition}
: sk-12345


# SCHEDULING (@mentions)
─────────────────────────────────────────────────────────

@today                   → Today
@tomorrow                → Tomorrow
@friday                  → Next Friday
@2024-03-15              → Specific date
@9am                     → Time (today)
@friday @3pm             → Date + time
@in:2hours               → Relative

@daily:9am               → Every day at 9am
@daily:9am @weekdays     → Weekdays only
@weekly:mon              → Every Monday
@weekly:mon,wed,fri      → Multiple days
@monthly:1st             → First of month
@monthly:last-friday     → Last Friday
@yearly:mar-15           → Annually


# EXPRESSIONS
─────────────────────────────────────────────────────────

`count(source)`          → Count items
`count(source where x)`  → Filtered count
`sum(source.field)`      → Sum values
`avg(source.field)`      → Average
`min(source.field)`      → Minimum
`max(source.field)`      → Maximum


# UI ELEMENTS
─────────────────────────────────────────────────────────

> ✅ Status text          → Success banner
> ⚠️ Warning text         → Warning banner
> ❌ Error text           → Error banner
> 📊 Stats                → Info banner

[Button Text]            → Action button
[Text](action:name)      → Action link
[Export](export:csv)     → Export link
[← Back](back)           → Navigation

---                      → Section divider


# OUTPUTS (in blockquotes)
─────────────────────────────────────────────────────────

> Slack: #channel        → Slack output
> Email: addr@co.com     → Email output
> Webhook: https://...   → Webhook output


# USER/TAG MENTIONS
─────────────────────────────────────────────────────────

@username                → User reference
#tag                     → Tag/category
```

### YAML Schema (Optional Overrides)

```yaml
---
# Only needed for advanced features

# External data sources
sources:
  api_data:
    type: rest
    url: https://api.example.com/data
    headers:
      Authorization: Bearer ${API_TOKEN}
    cache: 5m

  database:
    type: postgres
    connection: ${DATABASE_URL}
    query: SELECT * FROM items

# Custom validation (override inferred schema)
  tasks:
    schema:
      title: required | min:3 | max:100
      priority: select | Critical, High, Medium, Low

# Webhook triggers (can't be expressed in markdown)
triggers:
  - webhook:
      path: /github
      secret: ${WEBHOOK_SECRET}
    action: handle_github

# Output configuration
outputs:
  slack:
    token: ${SLACK_TOKEN}
  email:
    smtp: smtp.gmail.com
    user: ${EMAIL_USER}
    pass: ${EMAIL_PASS}

# App metadata
title: My App
icon: 📋
theme: dark
---
```

### LVT Attributes (HTML Layer)

For Layer 3 full control:

| Attribute | Element | Purpose | Status |
|-----------|---------|---------|--------|
| `lvt-source` | table, ul, select | Bind to data source | ✅ Implemented |
| `lvt-submit` | form | Action on submit | ✅ Implemented |
| `lvt-click` | button | Action on click | ✅ Implemented |
| `lvt-columns` | table | Column specification | ✅ Implemented |
| `lvt-actions` | table | Row actions | ✅ Implemented |
| `lvt-empty` | table, ul | Empty state text | ✅ Implemented |
| `lvt-field` | ul | Field to display | ✅ Implemented |
| `lvt-data-*` | button | Data attributes for actions | ✅ Implemented |
| `lvt-filter` | table, ul | Filter expression | 🔲 Planned |
| `lvt-aggregate` | span | Aggregation | 🔲 Planned |
| `lvt-chart` | div | Chart type | 🔲 Planned |

> **Note:** Before implementing new lvt-* attributes, verify they don't conflict
> with existing implementations in the livetemplate/client repository.

---

## Example Apps

See [tinkerdown-example-apps-plan.md](tinkerdown-example-apps-plan.md) for planned examples using the markdown-native syntax.

**Implementation milestones:**

| Example | Functional After |
|---------|------------------|
| Two-Line Todo | Week 2 |
| Expense Tracker | Week 4 |
| Team Tasks, Runbook, Meeting Notes, Inventory | Week 6 |
| Standup Bot, Health Monitor, CRM, Habit Tracker | Week 8 |

After each milestone, move working examples to `examples/`.

---

## Security Considerations

### Trust Model

Tinkerdown apps run locally with user permissions. Trust boundaries:

```
┌─────────────────────────────────────────────────────────────────┐
│                       TRUST LEVELS                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  TRUSTED (User Controls)                                        │
│  ─────────────────────────                                      │
│  • Markdown content (user writes)                               │
│  • YAML configuration (user defines sources)                    │
│  • Local file paths (user specifies)                            │
│                                                                 │
│  UNTRUSTED (External)                                           │
│  ─────────────────────                                          │
│  • REST API responses                                           │
│  • Database query results                                       │
│  • User input in forms                                          │
│                                                                 │
│  DANGEROUS (Requires Review)                                    │
│  ───────────────────────────                                    │
│  • exec sources (run shell commands)                            │
│  • Custom sources (arbitrary code)                              │
│  • Downloaded/shared .md apps                                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Security Guidelines

#### For Exec Sources

```yaml
# ⚠️ DANGEROUS: Never pass untrusted input to shell
sources:
  bad:
    type: exec
    command: "grep {{user_input}} file.txt"  # INJECTION RISK

  safe:
    type: exec
    command: "./validated-script.sh"
    # Script validates input internally
```

**Requirements for exec sources:**
1. Validate all inputs before shell execution
2. Use allowlists, not blocklists
3. Prefer structured APIs over shell commands
4. Log all command executions in production

#### For Custom Sources

```python
# custom-source.py
import sys, json

def main():
    data = json.loads(sys.stdin.read())

    # ✅ Validate input
    if "query" not in data:
        sys.exit(1)

    # ✅ Sanitize before use
    user_id = str(data["query"].get("user_id", ""))
    if not user_id.isalnum():
        sys.exit(1)

    # ✅ Use parameterized queries
    # cursor.execute("SELECT * FROM users WHERE id = ?", (user_id,))
```

#### For Shared Apps

When using apps from others:
1. **Review the markdown** - Check for exec sources and suspicious commands
2. **Check YAML sources** - Understand what data the app accesses
3. **Run in isolation** - Consider containers for untrusted apps
4. **Verify origin** - Prefer apps from trusted sources

### Security Roadmap

| Phase | Feature | Priority |
|-------|---------|----------|
| 1 | Input validation helpers | High |
| 1 | Exec source sandboxing option | High |
| 2 | Permission prompts for dangerous ops | Medium |
| 2 | App signing for trusted sources | Medium |
| 3 | Container isolation mode | Low |
| 3 | Capability-based permissions | Low |

### Error Handling for Custom Sources

Custom sources should handle errors gracefully:

```python
#!/usr/bin/env python3
import sys, json

def main():
    try:
        data = json.loads(sys.stdin.read())
        # Process...
        result = {"columns": [...], "rows": [...]}
        print(json.dumps(result))
        sys.exit(0)
    except json.JSONDecodeError:
        print(json.dumps({"error": "Invalid JSON input"}), file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(json.dumps({"error": str(e)}), file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
```

**Error handling requirements:**
- Exit 0 for success, non-zero for errors
- Write errors to stderr as JSON
- Never expose sensitive data in error messages
- Log errors for debugging but sanitize user data

---

## Distribution Strategy

```
┌─────────────────────────────────────────────────────────────────┐
│                    DISTRIBUTION TIERS                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  TIER 1: Developers (CLI)                                       │
│  ─────────────────────────                                      │
│  brew install tinkerdown                                        │
│  tinkerdown serve app.md                                        │
│                                                                 │
│  TIER 2: Power Users (Build)                                    │
│  ───────────────────────────                                    │
│  tinkerdown build app.md -o myapp                               │
│  ./myapp                                                        │
│                                                                 │
│  TIER 3: Non-Developers (Desktop)                               │
│  ─────────────────────────────────                              │
│  Download Tinkerdown.app                                        │
│  Double-click .md file to open                                  │
│                                                                 │
│  TIER 4: Anyone (Web)                                           │
│  ─────────────────────                                          │
│  tinkerdown.dev/gh/user/repo/app.md                             │
│  No install required                                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Success Metrics

### By Pillar

| Pillar | Metric | Target |
|--------|--------|--------|
| **All** | Pure markdown app works | 100% |
| **All** | Zero-config for basic apps | Yes |
| **Runbooks** | Time to log action | < 5 sec |
| **Productivity** | LLM first-try success | > 90% |
| **Automation** | Schedule reliability | 99.9% |

### Adoption

| Metric | 3 months | 6 months |
|--------|----------|----------|
| GitHub stars | 500 | 1000 |
| CLI installs | 200 | 500 |
| Apps created | 1000 | 5000 |

---

## Summary

### The Vision

```markdown
# Todo
- [ ] Try tinkerdown
```

**Two lines. Working app. Zero configuration.**

### The Layers

| Layer | For | Uses |
|-------|-----|------|
| 1 | 80% of apps | Pure markdown |
| 2 | Advanced config | YAML frontmatter |
| 3 | Full control | HTML + templates |

### The Pillars

| Pillar | Key Feature | Complete at |
|--------|-------------|-------------|
| Runbooks | Snapshots, steps | Week 6 |
| Productivity | Tabs, computed | Week 6 |
| Automation | @triggers, outputs | Week 8 |

### Next Action

**Week 1, Day 1:** Implement heading-as-anchor detection. Parse `## Tasks` and recognize the following list/table as the "tasks" source.

This single feature unlocks the markdown-native promise: valid markdown becomes working app.
