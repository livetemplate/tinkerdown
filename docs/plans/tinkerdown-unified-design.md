# Tinkerdown: Unified Design & Implementation Plan

**Date:** 2026-01-02
**Version:** 4.0 - Consolidated v1.0 Roadmap
**Status:** Living Document

---

## Executive Summary

Tinkerdown turns markdown into apps. **If it's valid markdown, it's a working app.**

```markdown
# Todo
- [ ] Buy milk
```

That's a complete app. Two lines. No YAML. No HTML. No configuration.

### The Building Blocks

Tinkerdown has a small grammar. These are the pieces you combine:

| Block | What It Does | Example |
|-------|--------------|---------|
| `# Heading` | Groups content | Section titles |
| `- [ ] Task` | Interactive checkbox | Tracks completion |
| `\| Table \|` | Editable data grid | Add/edit/delete rows |
| `[Button](action:x)` | Triggers actions | API calls, scripts |
| `` `expression` `` | Computed values | `count(x)`, `sum(x.field)` |
| `@schedule` | Runs on schedule | `@daily:9am`, `@friday` |

That's it. Six building blocks. Everything else is standard markdown rendered as content.

**Tinkering means:** you add a checkbox, see it work, add a table, connect it to a database, add a button, wire it to an API. Each step is small, visible, and reversible. When you read a tinkerdown file, you understand it. When an LLM generates one, you can verify it.

### One Command, Three Modes

```bash
tinkerdown serve app.md              # Interactive UI (default)
tinkerdown serve app.md --headless   # Background automation (triggers only)
tinkerdown cli app.md <command>      # Terminal interface (no server)
```

### What People Build

People have used tinkerdown for things we didn't anticipate. Common patterns include:

| Pattern | Example |
|---------|---------|
| **Runbooks** | Incident procedures with live system data and action buttons |
| **Trackers** | Personal/team task lists, expense logs, inventory |
| **Dashboards** | Database views, API monitors, system status |
| **Bots** | Scheduled notifications, automated reports |

But the point is: **you decide what to build**. We provide the blocks.

---

## Table of Contents

1. [Tinkering Stories](#tinkering-stories)
2. [Markdown-Native Design](#markdown-native-design)
3. [Architecture](#architecture)
4. [v1.0 Roadmap](#v10-roadmap)
5. [Quick Reference](#quick-reference)
6. [Success Metrics](#success-metrics)
7. [Post-v1.0 Considerations](#post-v10-considerations)
8. [Summary](#summary)

---

## Tinkering Stories

These aren't traditional user stories ("As a [role], I want [feature]"). They're patterns of exploration that tinkerdown should enable.

### Discovery Stories

> **"I wonder if I can just..."**

| Story | What it requires |
|-------|------------------|
| I can start with any markdown file I already have and see what happens | Zero barrier to start |
| I can add one line and see it become interactive | Incremental enhancement |
| I can look at an example, copy it, and modify it | Learn by doing |
| I can break things without consequences - reload and I'm back | Safe experimentation |

### Learning Stories

> **"How does this actually work?"**

| Story | What it requires |
|-------|------------------|
| I can read any tinkerdown app and understand what it does | Readable syntax |
| When something doesn't work, the error tells me what right looks like | Helpful errors |
| I can view the source of any running app | Transparency |
| I can learn the grammar incrementally - don't need everything to start | Progressive disclosure |

### Composition Stories

> **"What if I combine these?"**

| Story | What it requires |
|-------|------------------|
| I can add a second data source without breaking the first | Independent pieces |
| I can copy a section from one app into another and it works | Self-contained blocks |
| I can connect any action to any source - they're interchangeable | Uniform interfaces |
| Multiple sources, actions, triggers compose without interference | No hidden coupling |

### Iteration Stories

> **"Let me try a different approach"**

| Story | What it requires |
|-------|------------------|
| I change the markdown and see the result instantly | Hot reload |
| I can try something, undo it, try something else | Fast experimentation |
| I can start simple and add complexity piece by piece | Progressive enhancement |
| I can rip out parts that don't work without breaking the rest | Graceful degradation |

### Recovery Stories

> **"Something's wrong, can I fix it?"**

| Story | What it requires |
|-------|------------------|
| When something breaks, I can understand why | Clear error messages |
| I can always hand-edit the markdown to fix a problem | Human-editable format |
| A broken section doesn't take down the whole app | Fault isolation |
| I can diff my changes and revert if needed | Version control friendly |

### Sharing Stories

> **"Look what I made"**

| Story | What it requires |
|-------|------------------|
| I share my app by sharing the markdown file - nothing else needed | Self-contained |
| Someone else can run it if they have tinkerdown - no setup | Portable |
| I can explain how it works by showing the markdown | Self-documenting |
| I can put it in git and collaborate | Text-based format |

### Anti-Stories

> **If these happen, we've failed:**

- "I had to read the whole documentation before I could start"
- "I made a small change and everything broke"
- "I don't understand what this app does even though I'm looking at the source"
- "I needed to set up three other things before tinkerdown would work"
- "I can't share this because it depends on my local setup"
- "The LLM generated this and I have no idea how to modify it"

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

Types inferred from data patterns, mapped to SQL types:

| Pattern | SQL Type | Input UI |
|---------|----------|----------|
| `2024-01-15` | `DATE` | Date picker |
| `14:30` / `2:30pm` | `TIME` | Time picker |
| `123` / `45.67` | `DECIMAL` | Number input |
| `$123.45` | `DECIMAL(10,2)` | Currency input |
| `true` / `false` | `BOOLEAN` | Checkbox |
| `hello@example.com` | `TEXT` | Email input |
| `https://...` | `TEXT` | URL input |
| 3-10 unique strings | `TEXT` + enum constraint | Dropdown |
| Everything else | `TEXT` | Text input |

**Validation inference (SQL constraints):**
- Value in every row → `NOT NULL`
- Some rows empty → nullable (no constraint)

### Auto-Generated Forms

Every data collection gets an input form automatically. No HTML required.

```markdown
## Expenses
| date | category | amount | note |
|------|----------|--------|------|
| 2024-01-15 | Food | $45.50 | Groceries |
```

**What you get:**

```
┌─────────────────────────────────────────────────────────────────┐
│  Add Expense                                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Date:     [📅 2024-01-16    ]  ← Date picker (from pattern)   │
│                                                                 │
│  Category: [▼ Food          ]  ← Dropdown (≤10 unique values)  │
│                                                                 │
│  Amount:   [$ 0.00          ]  ← Currency input (from $)       │
│                                                                 │
│  Note:     [                ]  ← Text input (default)          │
│                                                                 │
│            [ Add Expense ]                                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Form behavior:**
- Submit adds a new row to the markdown table
- Each row gets edit/delete actions
- Changes persist back to the `.md` file
- Real-time sync across browser tabs (WebSocket)

**Task lists get inline editing:**

```markdown
## Tasks
- [ ] Buy milk
- [x] Call mom
```

- Click checkbox → toggles `[ ]` ↔ `[x]` in the file
- Click text → inline edit
- "Add" button → appends new `- [ ]` item

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

YAML frontmatter for notifications (distinct from status banners):

```markdown
---
outputs:
  slack: "#team-updates"
  email: "team@company.com"
---
# Daily Standup Bot

## Questions
- What did you do yesterday?
- What will you do today?

Notify @daily:9am @weekdays
```

> **Note:** Outputs use YAML frontmatter, not blockquotes. Blockquotes with emoji (✅, ⚠️, ❌) are status banners for display.

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

### Schema & Configuration Tiers

Three levels of configuration based on user needs:

```
┌─────────────────────────────────────────────────────────────────┐
│                    CONFIGURATION TIERS                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  TIER 1: Zero Config (80% of users)                             │
│  ──────────────────────────────────                             │
│  Just write markdown. Types inferred from data patterns.        │
│                                                                 │
│    ## Expenses                                                  │
│    | date | category | amount |                                 │
│    |------|----------|--------|                                 │
│    | 2024-01-15 | Food | $45.50 |                               │
│                                                                 │
│    → date: DATE, category: TEXT, amount: DECIMAL (auto)         │
│                                                                 │
│  TIER 2: Type Hints (15% of users)                              │
│  ─────────────────────────────────                              │
│  Override inference with human-readable type names.             │
│                                                                 │
│    sources:                                                     │
│      expenses:                                                  │
│        types:                                                   │
│          amount: currency                                       │
│          priority: select                                       │
│                                                                 │
│  TIER 3: Full SQL (5% of users)                                 │
│  ──────────────────────────────                                 │
│  External databases, constraints, advanced queries.             │
│                                                                 │
│    sources:                                                     │
│      orders: postgres://${DATABASE_URL}                         │
│      legacy:                                                    │
│        from: ./data.db                                          │
│        query: SELECT * FROM orders WHERE status = 'pending'     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

#### Tier 1: Zero Configuration

No YAML needed. Tinkerdown infers everything from markdown:

```markdown
## Tasks
- [ ] Design the API
- [x] Write tests
- [ ] Deploy to production

## Expenses
| date | category | amount |
|------|----------|--------|
| 2024-01-15 | Food | $45.50 |
| 2024-01-16 | Transport | $12.00 |

## Team
| name | email | role |
|------|-------|------|
| Alice | alice@co.com | Engineer |
| Bob | bob@co.com | Designer |
```

**What gets inferred:**

| Data | Inferred Schema |
|------|-----------------|
| `## Tasks` | Table: `tasks` (text TEXT, done BOOLEAN) |
| `## Expenses` | Table: `expenses` (date DATE, category TEXT, amount DECIMAL) |
| `## Team` | Table: `team` (name TEXT, email TEXT, role TEXT) |

Types and constraints are inferred automatically from data patterns. See [Automatic Schema Inference](#automatic-schema-inference) for the full pattern detection rules.

---

#### Tier 2: Type Hints

When auto-inference gets it wrong, override with simple hints:

```yaml
---
sources:
  # Markdown source with type hints
  expenses:
    from: "#expenses"           # data comes from markdown section
    types:
      amount: currency
      date: date
      category: select          # auto-detect options from data

  # Another markdown source
  tasks:
    from: "#tasks"
    types:
      priority: select:Critical,High,Medium,Low    # explicit options
      due: date
---

## Expenses
| date | category | amount |
...
```

**Type hint vocabulary:**

| Hint | SQL Type | UI | Notes |
|------|----------|-----|-------|
| `text` | TEXT | Text input | Default |
| `number` | DECIMAL | Number input | |
| `integer` | INTEGER | Number input | Whole numbers |
| `currency` | DECIMAL(10,2) | Currency input | With symbol |
| `date` | DATE | Date picker | |
| `time` | TIME | Time picker | |
| `datetime` | TIMESTAMP | DateTime picker | |
| `boolean` | BOOLEAN | Checkbox | |
| `email` | TEXT | Email input | With validation |
| `url` | TEXT | URL input | With validation |
| `select` | TEXT | Dropdown | Options from data |
| `select:a,b,c` | TEXT + CHECK | Dropdown | Explicit options |
| `textarea` | TEXT | Multi-line | |
| `hidden` | TEXT | None | Not shown in forms |

**Shorthand vs Full form:**

```yaml
# Shorthand (source auto-detected from heading)
sources:
  expenses:
    types:
      amount: currency

# Full form (explicit anchor)
sources:
  expenses:
    from: "#expenses"
    types:
      amount: currency

# Shorthand for types only (dot notation)
types:
  expenses.amount: currency
  expenses.category: select
  tasks.priority: select:High,Medium,Low
```

**Required fields:**

```yaml
sources:
  expenses:
    from: "#expenses"
    types:
      amount: currency
    required:
      - date
      - amount
```

**Sync mode (for markdown sources):**

```yaml
sources:
  expenses:
    from: "#expenses"
    sync: both       # read | write | both (default: both)
```

| Mode | Behavior |
|------|----------|
| `read` | Markdown → App (markdown is source of truth, read-only in UI) |
| `write` | App → Markdown (changes written back to .md file) |
| `both` | Bidirectional (default) |

---

#### Tier 3: External Databases & Queries

For connecting to existing databases and running SQL queries:

```yaml
---
sources:
  # Simple: entire table (infer table from source name)
  users: postgres://${DATABASE_URL}

  # Explicit table name
  orders:
    from: postgres://${DATABASE_URL}
    table: customer_orders

  # Filtered query
  active_users:
    from: postgres://${DATABASE_URL}
    query: SELECT * FROM users WHERE active = true

  # Complex aggregation
  monthly_stats:
    from: postgres://${DATABASE_URL}
    query: |
      SELECT
        date_trunc('month', created_at) as month,
        count(*) as order_count,
        sum(total) as revenue
      FROM orders
      WHERE created_at >= '2024-01-01'
      GROUP BY 1
      ORDER BY 1

  # SQLite file
  archive: ./archive.db

  # SQLite with query
  recent_logs:
    from: ./data.db
    query: SELECT * FROM logs WHERE date > date('now', '-7 days')
---
```

**Query syntax reference:**

| Syntax | What It Does |
|--------|--------------|
| `source: postgres://url` | All rows from table (name = source name) |
| `source: { from: url, table: X }` | All rows from specific table |
| `source: { from: url, query: SELECT... }` | Custom SQL query |
| `source: { query: SELECT... }` | Cross-source query (no `from`) |

**Connection string formats:**

| Database | Format |
|----------|--------|
| PostgreSQL | `postgres://user:pass@host:5432/dbname` |
| MySQL | `mysql://user:pass@host:3306/dbname` |
| SQLite | `./path/to/file.db` or `sqlite:///path/to/file.db` |

**Environment variables:**

```yaml
sources:
  production: postgres://${DATABASE_URL}
  analytics: mysql://${MYSQL_URL}
```

**Parameterized queries:**

```yaml
sources:
  user_orders:
    from: postgres://${DATABASE_URL}
    query: SELECT * FROM orders WHERE user_id = :user_id
    params:
      user_id: ${CURRENT_USER}
```

---

#### Cross-Source Queries

Join data across markdown and external databases:

```yaml
---
sources:
  # Markdown source (shorthand)
  expenses: "#expenses"

  # Markdown source (full form)
  tasks:
    from: "#tasks"
    types:
      priority: select

  # External database
  employees:
    from: postgres://${DATABASE_URL}
    table: employees

  # Join across sources (markdown + database)
  report:
    query: |
      SELECT
        e.date,
        e.amount,
        emp.name as submitter
      FROM expenses e
      JOIN employees emp ON e.employee_id = emp.id
      WHERE e.date >= '2024-01-01'
---
```

**The `from:` keyword - unified syntax:**

| Source Type | Shorthand | Full Form |
|-------------|-----------|-----------|
| Markdown | `tasks: "#tasks"` | `tasks: { from: "#tasks" }` |
| SQLite | `data: ./app.db` | `data: { from: ./app.db }` |
| PostgreSQL | `users: postgres://...` | `users: { from: postgres://... }` |
| MySQL | `orders: mysql://...` | `orders: { from: mysql://... }` |

Use full form when you need additional options (types, query, table, required, etc.).

**How it works internally:**

```
┌─────────────────────────────────────────────────────────────────┐
│                    UNIFIED SQL ENGINE                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Markdown tables → loaded into in-memory SQLite              │
│  2. External DBs → connected via drivers                        │
│  3. Cross-source queries → federated via SQL                    │
│                                                                 │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐       │
│  │  ## Table   │     │  SQLite     │     │  PostgreSQL │       │
│  │  (markdown) │     │  (file.db)  │     │  (remote)   │       │
│  └──────┬──────┘     └──────┬──────┘     └──────┬──────┘       │
│         │                   │                   │               │
│         └─────────────┬─────┴─────────────┬─────┘               │
│                       │                   │                     │
│                       ▼                   ▼                     │
│              ┌─────────────────────────────────┐                │
│              │     Unified Query Layer         │                │
│              │  (SQLite for markdown + proxy   │                │
│              │   for external DBs)             │                │
│              └─────────────────────────────────┘                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

#### Schema Files (Optional)

For complex schemas, use external SQL files:

```yaml
---
schema: ./schema.sql    # Applied to markdown sources
sources:
  expenses: "#expenses"
  tasks: "#tasks"
---
```

**schema.sql:**

```sql
-- Only needed for constraints beyond type hints
CREATE TABLE expenses (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  date DATE NOT NULL,
  category TEXT DEFAULT 'misc',
  amount DECIMAL(10,2) NOT NULL CHECK(amount > 0),
  employee_id INTEGER REFERENCES employees(id)
);

CREATE TABLE tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL CHECK(length(title) >= 3),
  priority TEXT DEFAULT 'Medium'
    CHECK(priority IN ('Critical','High','Medium','Low')),
  done BOOLEAN DEFAULT false
);
```

**When to use schema files:**

| Need | Use |
|------|-----|
| Type hints only | Tier 2 YAML |
| CHECK constraints | Schema file |
| Foreign keys | Schema file |
| Indexes | Schema file |
| Complex defaults | Schema file |

---

#### HTML Templates (Full Control)

When you need complete control over layout and interactions, use HTML with `lvt-*` attributes and Go templates:

```html
<form lvt-submit="add" lvt-source="items">
  <input name="title" placeholder="New item" required>
  <button type="submit">Add</button>
</form>

<div class="grid">
  {{range .items}}
  <div class="card">
    <h3>{{.title}}</h3>
    <p>Added: {{.created_at | formatDate}}</p>
    <button lvt-click="delete" lvt-data-id="{{.id}}">Delete</button>
  </div>
  {{end}}
</div>

{{if eq (len .items) 0}}
<p class="empty">No items yet. Add one above!</p>
{{end}}
```

**Use HTML templates for:**
- Custom card/grid layouts
- Conditional rendering (`{{if}}`, `{{range}}`)
- Complex multi-step forms
- Custom styling and CSS classes
- Integrating with external JS libraries

**Progressive enhancement:** Start with pure markdown (Tier 1), add YAML when needed (Tier 2-3), drop to HTML only for specific sections that need custom rendering.

---

### Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                        DATA FLOW                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  MARKDOWN SOURCES (auto-detected)                               │
│  ─────────────────────────────────                              │
│  ## Heading     ──┐                                             │
│  - [ ] tasks    ──┼──▶  Parse  ──▶  SQL Schema  ──▶  SQL DB    │
│  | table |      ──┤                                    │        │
│  - list         ──┘                                    │        │
│                                                        │        │
│  EXTERNAL SOURCES (YAML config)                        │        │
│  ──────────────────────────────                        │        │
│  sqlite: db     ──┐                                    │        │
│  postgres: dsn  ──┼──▶  SQL Query  ────────────────────┘        │
│  mysql: dsn     ──┘                                             │
│                                                                 │
│  SQL DB  ──▶  Render  ──▶  UI / API / CLI                      │
│    │                                                            │
│    │  ACTIONS (SQL operations)                                  │
│    │  ────────────────────────                                  │
│    └──  INSERT, UPDATE, DELETE  ◀──  User / Trigger            │
│              │                                                  │
│              ▼ (if sync: write | both)                          │
│         Update markdown file                                    │
│                                                                 │
│  TRIGGERS (invoke actions)                                      │
│  ─────────────────────────                                      │
│  @daily:9am     ──┐                                             │
│  @friday        ──┼──▶  Execute SQL  ──▶  Update State         │
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

## v1.0 Roadmap

This roadmap is designed to be imported directly into a GitHub project. Each milestone becomes a GitHub milestone; each task becomes an issue.

### Milestone Overview

| # | Milestone | What Ships | Exit Criteria |
|---|-----------|------------|---------------|
| 1 | **It Works** | Markdown → interactive app | Pure markdown app runs, changes persist |
| 2 | **It Connects** | External data sources | Postgres, REST, exec sources work |
| 3 | **It Acts** | Buttons, forms, API | Action buttons trigger operations |
| 4 | **It Reacts** | Triggers & outputs | Schedules run, webhooks received |
| 5 | **It Ships** | Distribution | Build command produces standalone binary |
| 6 | **v1.0 Launch** | Polish & launch | Docs complete, examples work, release published |

---

### Milestone 1: It Works

**Goal:** A markdown file becomes an interactive app. Changes persist to the file.

| Task | Description | Acceptance Criteria |
|------|-------------|---------------------|
| 1.1 | **Testing Infrastructure** | Golden file tests work with `-update` flag |
| 1.2 | **Heading-as-Anchor** | `## Tasks` auto-detected as source named "tasks" |
| 1.3 | **Table Parsing** | Markdown tables parse to structured data with columns |
| 1.4 | **List Parsing** | Task lists, ordered lists, definition lists parse correctly |
| 1.5 | **Schema Inference** | Dates, numbers, booleans auto-detected from patterns |
| 1.6 | **Auto-CRUD UI** | Tables get add form, task lists get checkbox toggle |
| 1.7 | **Hot Reload** | File changes reflect in browser within 100ms |
| 1.8 | **Example: Two-Line Todo** | `# Todo\n- [ ] task` runs as complete app |

**Security:** Input validation on all form submissions. Sanitize markdown content.

**Testing:** Golden file tests for parser output. Browser test for hot reload.

---

### Milestone 2: It Connects

**Goal:** Connect to external databases, APIs, and commands.

| Task | Description | Acceptance Criteria |
|------|-------------|---------------------|
| 2.1 | **SQLite Source** | `from: ./data.db` loads SQLite tables |
| 2.2 | **PostgreSQL Source** | `from: postgres://...` connects and queries |
| 2.3 | **REST Source** | `from: https://...` fetches JSON with headers/auth |
| 2.4 | **Exec Source** | `type: exec` runs shell command, parses JSON output |
| 2.5 | **Source Caching** | TTL and stale-while-revalidate work |
| 2.6 | **Cross-Source Queries** | SQL JOINs across markdown + external sources |
| 2.7 | **Auto-Timestamp** | `{{now}}` fills current date/time on submit |
| 2.8 | **Operator Identity** | `--operator alice` sets `{{operator}}` |
| 2.9 | **Example: Expense Tracker** | Markdown + SQLite source working together |

**Security:** Parameterized queries only. No string interpolation in SQL. Exec sources log all commands.

**Testing:** Golden tests for each source type. Integration test with test Postgres container.

---

### Milestone 3: It Acts

**Goal:** Buttons trigger actions. API and CLI available.

| Task | Description | Acceptance Criteria |
|------|-------------|---------------------|
| 3.1 | **Action Buttons** | `[Button](action:name)` triggers named action |
| 3.2 | **Computed Expressions** | `` `count(tasks where done)` `` evaluates live |
| 3.3 | **Tabs & Filtering** | `## [All] \| [Active] not done` creates tabbed view |
| 3.4 | **Status Banners** | `> ✅ text` renders as styled banner |
| 3.5 | **HTTP API** | `GET/POST /api/sources/{name}` CRUD endpoints |
| 3.6 | **CLI Mode** | `tinkerdown cli app.md add tasks --text="..."` |
| 3.7 | **WebSocket Protocol Tests** | Direct WS tests without browser (fast) |
| 3.8 | **Example: Team Tasks** | Multi-user task board with filters |

**Security:** CORS headers configurable. Rate limiting on API endpoints.

**Testing:** API endpoint tests. CLI testscript tests. WebSocket protocol golden tests.

---

### Milestone 4: It Reacts

**Goal:** Schedules trigger actions. Webhooks received. Outputs notify.

| Task | Description | Acceptance Criteria |
|------|-------------|---------------------|
| 4.1 | **@schedule Parsing** | `@daily:9am` parsed from markdown |
| 4.2 | **Schedule Runner** | Cron-based trigger execution |
| 4.3 | **Webhook Triggers** | `POST /webhook/name` triggers action |
| 4.4 | **Slack Output** | `outputs: { slack: "#channel" }` sends messages |
| 4.5 | **Email Output** | `outputs: { email: "addr" }` sends email |
| 4.6 | **Headless Mode** | `--headless` runs triggers without web UI |
| 4.7 | **Example: Standup Bot** | Daily notification with team responses |

**Security:** Webhook secret validation. Output credentials from env vars only.

**Testing:** Schedule tests with mock clock. Webhook tests with test endpoints.

---

### Milestone 5: It Ships

**Goal:** Build standalone binaries. Desktop app. Distribution ready.

| Task | Description | Acceptance Criteria |
|------|-------------|---------------------|
| 5.1 | **Build Command** | `tinkerdown build app.md -o myapp` produces binary |
| 5.2 | **Embedded Assets** | Built binary includes all web assets |
| 5.3 | **Desktop App (Wails)** | Double-click .md opens in native window |
| 5.4 | **Homebrew Formula** | `brew install tinkerdown` works |
| 5.5 | **Docker Image** | `docker run tinkerdown serve app.md` works |
| 5.6 | **CLI Testscript Suite** | Black-box CLI tests with testscript |
| 5.7 | **Example: Distributable App** | Built binary shared and runs on another machine |

**Security:** Built binaries don't include source secrets. Env vars loaded at runtime.

**Testing:** Build output tests. Installation tests on clean systems.

---

### Milestone 6: v1.0 Launch

**Goal:** Documentation complete. Examples polished. Public release.

| Task | Description | Acceptance Criteria |
|------|-------------|---------------------|
| 6.1 | **User Documentation** | Getting started, syntax reference, examples |
| 6.2 | **Template Gallery** | 7+ starter templates via `tinkerdown new` |
| 6.3 | **Charts** | `` ```chart `` `` code blocks render visualizations |
| 6.4 | **Example Suite** | 10+ working examples covering all features |
| 6.5 | **Reduce Browser Tests** | Replace slow chromedp with fast protocol tests |
| 6.6 | **Performance Baseline** | Documented latency targets met |
| 6.7 | **Security Audit** | Exec sources sandboxed, input validation complete |
| 6.8 | **Release** | GitHub release with binaries, changelog, announcement |

**Testing:** Full test suite passes. Manual testing of all examples.

---

## Quick Reference

### Markdown Grammar

```
## Heading           → Data source (name from heading)
- [ ] item           → Task list: {text, done: bool}
- item               → Simple list: {text}
1. item              → Ordered list: {text, order: int}
| col | col |        → Table: {col: type, ...}

@today @tomorrow     → Date mentions
@daily:9am           → Schedule trigger
@weekly:mon,wed      → Recurring trigger

`count(x)`           → Computed value
`sum(x.field)`       → Aggregation

> ✅ Status          → Success banner
> ⚠️ Warning         → Warning banner

[Button]             → Action button
[Text](action:x)     → Action link
```

### YAML Configuration

```yaml
---
# Tier 2: Type hints
types:
  expenses.amount: currency
  tasks.priority: select:Critical,High,Medium,Low

# Tier 3: External sources
sources:
  users: postgres://${DATABASE_URL}
  data: ./local.db

# Outputs
outputs:
  slack: "#channel"
  email: "team@company.com"
---
```

### LVT Attributes (Tier 4: HTML)

| Attribute | Purpose | Example |
|-----------|---------|---------|
| `lvt-source` | Bind to data | `<table lvt-source="tasks">` |
| `lvt-submit` | Form action | `<form lvt-submit="add">` |
| `lvt-click` | Button action | `<button lvt-click="delete">` |
| `lvt-columns` | Column spec | `lvt-columns="name,email"` |
| `lvt-actions` | Row buttons | `lvt-actions="edit,delete"` |

---

## Success Metrics

**Primary metric:** "Users built things we didn't anticipate."

| Stage | Metric | Target |
|-------|--------|--------|
| **Try** | Install → first app running | < 2 min |
| **Understand** | Can explain any example | 100% |
| **Modify** | Successful first change | > 80% |
| **Compose** | Combine features without docs | > 60% |
| **Share** | App runs for others without help | 100% |

**Technical quality:**

| Metric | Target |
|--------|--------|
| Zero-config app works | 100% |
| Hot reload latency | < 100ms |
| Error messages suggest fix | 100% |

---

## Post-v1.0 Considerations

> Features below are **not committed**. They represent directions to explore based on user demand.

**High priority if demanded:**
- WASM Source SDK (`tinkerdown wasm init`)
- Authentication middleware (GitHub/OAuth)
- Pagination & sorting for large datasets

**Likely better solved elsewhere:**
- Rate limiting → reverse proxy (nginx)
- Complex UI components → Tier 4 HTML templates
- New database types → WASM modules

**Philosophy check before adding:**
1. Does it require config? Can 80% use case work without it?
2. Does WASM already solve this?
3. Is this a tinkerdown feature or deployment concern?
4. Does it increase learning curve?

---

## Summary

**Tinkerdown turns markdown into apps.**

```markdown
# Todo
- [ ] Try tinkerdown
```

Two lines. Working app. Zero configuration.

### The Tiers

| Tier | Who | How |
|------|-----|-----|
| 1 | 80% | Pure markdown |
| 2 | 15% | YAML type hints |
| 3 | 5% | External DBs, SQL |
| 4 | Power users | HTML + Go templates |

### The Milestones to v1.0

1. **It Works** - Markdown renders, changes persist
2. **It Connects** - External data sources
3. **It Acts** - Buttons, forms, API
4. **It Reacts** - Triggers, outputs
5. **It Ships** - Build, distribute
6. **v1.0 Launch** - Docs, examples, release

### Next Action

Start Milestone 1. Ship it. See what people build. Learn. Iterate.
