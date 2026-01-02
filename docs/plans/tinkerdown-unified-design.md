# Tinkerdown: Unified Design & Implementation Plan

**Date:** 2026-01-02
**Version:** 3.1 - Consistency Pass
**Status:** Living Document

---

## Executive Summary

Tinkerdown turns markdown into apps. **If it's valid markdown, it's a working app.**

```markdown
# Todo
- [ ] Buy milk
```

That's a complete app. Two lines. No YAML. No HTML. No configuration.

### The Tinkering Philosophy

Tinkerdown enables **tinkering** - exploration, experimentation, and discovery. The goal isn't to prescribe what you build, but to give you building blocks simple enough to understand and powerful enough to matter.

```
┌─────────────────────────────────────────────────────────────────┐
│                     THE TINKERING LOOP                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│    Wonder → Try → See → Understand → Modify → Try again         │
│                                                                 │
│    "What if I..."                                               │
│         ↓                                                       │
│    Make a small change                                          │
│         ↓                                                       │
│    Immediate feedback                                           │
│         ↓                                                       │
│    "Oh, that's how it works"                                    │
│         ↓                                                       │
│    "Now what if I..."                                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Core properties:**
- **Learnable** - Master the grammar in an afternoon
- **Deterministic** - What's simple for LLMs is simple for humans
- **Composable** - Small pieces that combine without interference
- **Recoverable** - You can always hand-edit if needed
- **Portable** - Git-backed, grep-able, yours forever

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
3. [The Layered Approach](#the-layered-approach)
4. [Architecture](#architecture)
5. [Feature Dependency Graph](#feature-dependency-graph)
6. [Progressive Implementation Plan](#progressive-implementation-plan)
7. [Feature Specifications](#feature-specifications)
8. [Example Apps](#example-apps)
9. [Security Considerations](#security-considerations)
10. [Distribution Strategy](#distribution-strategy)
11. [Success Metrics](#success-metrics)
12. [Extended Roadmap (Post-v1.0)](#extended-roadmap-post-v10)
13. [Summary](#summary)

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

## The Layered Approach

Three layers for different complexity:

> **Terminology note:** "Layers" describe the UI/templating abstraction (Markdown → YAML → HTML). "Tiers" in the Architecture section describe configuration complexity (Zero Config → Type Hints → Full SQL). These map to the same progression - Layer 1 = Tier 1, etc.

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
    from: "#tasks"
    types:
      priority: select:Critical,High,Medium,Low
      title: text
    required: [title]

outputs:
  slack:
    channel: "#alerts"
    token: ${SLACK_TOKEN}
---
```

YAML only for:
- Type hints when inference is wrong
- External data sources (databases, REST)
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

## Feature Dependency Graph

Features build on each other. Each layer unlocks new tinkering capabilities.

```
┌─────────────────────────────────────────────────────────────────┐
│                    FEATURE DEPENDENCIES                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  LAYER 0: Foundation (It Works)                                 │
│  ──────────────────────────────                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Heading   │  │  Table/List │  │   Schema    │             │
│  │   as Anchor │  │   Parsing   │  │  Inference  │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         └────────────────┼────────────────┘                     │
│                          ▼                                      │
│  LAYER 1: Core Features (It Connects)                          │
│  ────────────────────────────────────                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │Auto-timestamp│  │  Computed   │  │  HTTP API   │             │
│  │+ Operator   │  │  Expressions│  │  + CLI mode │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                │                     │
│         ▼                ▼                ▼                     │
│  LAYER 2: Interactivity (It Acts)                              │
│  ────────────────────────────────                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  Snapshot   │  │   Tabs &    │  │   Action    │             │
│  │  + Steps    │  │  Filtering  │  │   Buttons   │             │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘             │
│         │                │                │                     │
│         ▼                ▼                ▼                     │
│  LAYER 3: Automation (It Reacts)                               │
│  ───────────────────────────────                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  @schedule  │  │   Charts    │  │  Webhooks   │             │
│  │  Triggers   │  │ + Exports   │  │  + Outputs  │             │
│  └─────────────┘  └─────────────┘  └──────┬──────┘             │
│                                           │                     │
│                                           ▼                     │
│  LAYER 4: Distribution (It Ships)                              │
│  ────────────────────────────────                              │
│  ┌─────────────────────────────────────────────────┐           │
│  │  Build command / Desktop app / tinkerdown.dev   │           │
│  └─────────────────────────────────────────────────┘           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Feature → Tinkering Stories

Each feature enables specific tinkering stories:

| Feature | Discovery | Learning | Composition | Iteration | Sharing | Layer |
|---------|:---------:|:--------:|:-----------:|:---------:|:-------:|:-----:|
| Heading as anchor | ✓ | ✓ | | | | 0 |
| Table/list parsing | ✓ | ✓ | | | | 0 |
| Schema inference | ✓ | | | ✓ | | 0 |
| Auto-timestamp | | ✓ | | | ✓ | 1 |
| Computed expressions | ✓ | ✓ | ✓ | | | 1 |
| HTTP API | | | ✓ | | ✓ | 1 |
| CLI mode | | | ✓ | | | 1 |
| Tabs & filtering | ✓ | | ✓ | | | 2 |
| Action buttons | ✓ | ✓ | ✓ | | | 2 |
| @schedule triggers | ✓ | | ✓ | | | 3 |
| Webhooks | | | ✓ | | | 3 |
| Outputs (Slack/Email) | | | ✓ | | ✓ | 3 |
| Charts/Exports | ✓ | | | | ✓ | 3 |
| Build command | | | | | ✓ | 4 |
| Desktop app | | | | | ✓ | 4 |

✓ = Enables this story type

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
│   ├── source/              # Data sources (see list below)
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

**Implemented source types (9 total):**

| Type | From | Description |
|------|------|-------------|
| `markdown` | `#heading` | Tables/lists in the .md file itself |
| `sqlite` | `./file.db` | SQLite database file |
| `postgres` | `postgres://...` | PostgreSQL database |
| `json` | `./file.json` or URL | JSON file or REST endpoint |
| `csv` | `./file.csv` | CSV file |
| `rest` | `https://...` | REST API with headers/auth |
| `exec` | shell command | Execute command, parse JSON output |
| `graphql` | `https://...` | GraphQL endpoint |
| `wasm` | `./module.wasm` | WebAssembly custom source |

### Overview: Capability Milestones

Each milestone unlocks new tinkering possibilities. Ship each milestone, see what people build, learn.

```
┌─────────────────────────────────────────────────────────────────┐
│                    CAPABILITY MILESTONES                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  MILESTONE 1: It Works (Wk 1-2)                                 │
│  ───────────────────────────────                                │
│  Markdown → interactive app. Changes persist. Sources fetch.   │
│                                                                 │
│  Tinkering unlocked:                                            │
│  • Discovery: "I can run my existing markdown"                  │
│  • Learning: "I see how the syntax maps to UI"                  │
│  • Iteration: "Hot reload shows my changes"                     │
│                                                                 │
│  MILESTONE 2: It Connects (Wk 3-4)                              │
│  ─────────────────────────────────                              │
│  External databases, APIs, shell commands as data sources.     │
│                                                                 │
│  Tinkering unlocked:                                            │
│  • Composition: "I can pull in my actual data"                  │
│  • Discovery: "What if I connect to my database?"               │
│                                                                 │
│  MILESTONE 3: It Acts (Wk 5-6)                                  │
│  ──────────────────────────────                                 │
│  Action buttons, forms, HTTP API, CLI mode.                    │
│                                                                 │
│  Tinkering unlocked:                                            │
│  • Composition: "Buttons can trigger any action"                │
│  • Sharing: "Others can use my app via API"                     │
│                                                                 │
│  MILESTONE 4: It Reacts (Wk 7-8)                                │
│  ───────────────────────────────                                │
│  Schedules trigger, webhooks arrive, outputs notify.           │
│                                                                 │
│  Tinkering unlocked:                                            │
│  • Discovery: "What if it runs automatically?"                  │
│  • Composition: "Triggers + actions + outputs"                  │
│                                                                 │
│  MILESTONE 5: It Ships (Wk 9-10)                                │
│  ────────────────────────────────                               │
│  Build command, desktop app, distribution.                     │
│                                                                 │
│  Tinkering unlocked:                                            │
│  • Sharing: "I can give this to non-developers"                 │
│                                                                 │
│  MILESTONE 6: It Scales (Wk 11-12)                              │
│  ─────────────────────────────────                              │
│  Charts, templates, documentation, polish.                     │
│                                                                 │
│  Tinkering unlocked:                                            │
│  • Discovery: "What else can I visualize?"                      │
│  • Learning: "Examples show me patterns"                        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Milestone → Stories Mapping

| Milestone | Discovery | Learning | Composition | Iteration | Recovery | Sharing |
|-----------|:---------:|:--------:|:-----------:|:---------:|:--------:|:-------:|
| 1: Works | ✓ | ✓ | | ✓ | ✓ | |
| 2: Connects | ✓ | ✓ | ✓ | ✓ | ✓ | |
| 3: Acts | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 4: Reacts | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| 5: Ships | | | ✓ | | | ✓ |
| 6: Scales | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

### Task Status Summary

| Phase | Task | Status |
|-------|------|--------|
| 1 | 1.0 Testing Foundation | `[ ]` |
| 1 | 1.1 Heading-as-Anchor | `[ ]` |
| 1 | 1.2 Table Parsing | `[ ]` |
| 1 | 1.3 List Parsing | `[ ]` |
| 1 | 1.4 Schema Inference & Type Hints | `[ ]` |
| 1 | 1.5 Auto-CRUD UI | `[ ]` |
| 2 | 2.1 Auto-Timestamp | `[ ]` |
| 2 | 2.2 Operator Identity | `[ ]` |
| 2 | 2.3 Computed Expressions | `[ ]` |
| 2 | 2.4 HTTP API | `[ ]` |
| 2 | 2.5 CLI Mode | `[ ]` |
| 2 | 2.6 WebSocket Protocol Tests | `[ ]` |
| 3 | 3.1 Snapshot Capture | `[ ]` |
| 3 | 3.2 Step Status | `[ ]` |
| 3 | 3.3 Tabs | `[ ]` |
| 3 | 3.4 Status Banners | `[ ]` |
| 3 | 3.5 Action Buttons | `[ ]` |
| 3 | 3.6 Source Golden Tests | `[ ]` |
| 4 | 4.1 @schedule Parsing | `[ ]` |
| 4 | 4.2 Schedule Runner | `[ ]` |
| 4 | 4.3 Webhook Triggers | `[ ]` |
| 4 | 4.4 Output Integrations | `[ ]` |
| 5 | 5.1 Build Command | `[ ]` |
| 5 | 5.2 Desktop App | `[ ]` |
| 5 | 5.3 Hosted Service | `[ ]` |
| 5 | 5.4 CLI Testscript | `[ ]` |
| 6 | 6.1 Charts | `[ ]` |
| 6 | 6.2 Template Gallery | `[ ]` |
| 6 | 6.3 Documentation | `[ ]` |
| 6 | 6.4 Reduce Browser Tests | `[ ]` |

---

### Phase 1: Markdown-Native Foundation (Week 1-2)

#### Task 1.0: Testing Foundation

**Status:** `[ ] Not Started`

**Goal:** Set up golden file testing infrastructure for deterministic output verification.

**Prerequisites:** None (first task - enables testing for all subsequent tasks)

**Files to read first:**
- `docs/plans/2025-12-31-e2e-testing-strategy.md` - Testing strategy document
- `parser_test.go` - Existing test patterns
- `internal/source/markdown_test.go` - Existing source tests

**Files to create:**
- `internal/testutil/golden.go` - Golden file assertion helper
- `internal/testutil/scrub.go` - Scrubbers for non-deterministic data
- `testdata/` - Test fixture directory structure

**Implementation steps:**

1. Create `internal/testutil/golden.go`:
```go
package testutil

import (
    "bytes"
    "flag"
    "os"
    "path/filepath"
    "testing"
)

var update = flag.Bool("update", false, "update golden files")

// AssertGolden compares actual output against golden file.
// Run with -update flag to update golden files.
func AssertGolden(t *testing.T, name string, actual []byte) {
    t.Helper()
    golden := filepath.Join("testdata", "golden", name)

    if *update {
        os.MkdirAll(filepath.Dir(golden), 0755)
        if err := os.WriteFile(golden, actual, 0644); err != nil {
            t.Fatalf("failed to update golden file: %v", err)
        }
        return
    }

    expected, err := os.ReadFile(golden)
    if err != nil {
        t.Fatalf("failed to read golden file %s: %v", golden, err)
    }

    if !bytes.Equal(actual, expected) {
        t.Errorf("output mismatch for %s\n\nExpected:\n%s\n\nActual:\n%s",
            name, string(expected), string(actual))
    }
}

// ReadFixture reads a test fixture file.
func ReadFixture(t *testing.T, name string) []byte {
    t.Helper()
    path := filepath.Join("testdata", "fixtures", name)
    data, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("failed to read fixture %s: %v", name, err)
    }
    return data
}
```

2. Create `internal/testutil/scrub.go`:
```go
package testutil

import "regexp"

// Scrubber replaces non-deterministic values with placeholders
type Scrubber struct {
    Pattern     *regexp.Regexp
    Replacement string
}

// DefaultScrubbers for tinkerdown-specific patterns
var DefaultScrubbers = []Scrubber{
    {regexp.MustCompile(`(lvt|auto-persist-lvt)-\d+`), "BLOCK_ID"},
    {regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`), "TIMESTAMP"},
    {regexp.MustCompile(`"duration":\d+`), `"duration":0`},
    {regexp.MustCompile(`\?v=\d+`), "?v=VERSION"},
    {regexp.MustCompile(`nonce="[^"]+"`), `nonce="NONCE"`},
}

// Scrub applies all scrubbers to data
func Scrub(data []byte, scrubbers ...Scrubber) []byte {
    if len(scrubbers) == 0 {
        scrubbers = DefaultScrubbers
    }
    for _, s := range scrubbers {
        data = s.Pattern.ReplaceAll(data, []byte(s.Replacement))
    }
    return data
}
```

3. Create testdata directory structure:
```
testdata/
├── fixtures/           # Input files for tests
│   └── .gitkeep
└── golden/             # Expected outputs
    └── .gitkeep
```

**Acceptance criteria:**
- [ ] `AssertGolden()` correctly compares actual vs expected output
- [ ] `AssertGolden()` with `-update` flag creates/updates golden files
- [ ] `ReadFixture()` loads test input files
- [ ] `Scrub()` removes non-deterministic values (timestamps, IDs)
- [ ] All testutil tests pass
- [ ] Directory structure exists with `.gitkeep` files

**Verification commands:**
```bash
go test ./internal/testutil/... -v
# Test update mode
go test ./internal/testutil/... -v -update
```

**Testing requirements for subsequent tasks:**
After this task, all new features MUST include:
1. Golden file test for output (parser, sources, HTTP)
2. Scrubbed comparison for dynamic content
3. Fixture files in `testdata/fixtures/`
4. Expected output in `testdata/golden/`

---

#### Task 1.1: Heading-as-Anchor Detection

**Status:** `[ ] Not Started`

**Goal:** Recognize `## Heading` as a data source anchor without explicit `#anchor` syntax.

**Prerequisites:** Task 1.0 complete (testing infrastructure)

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

#### Task 1.4: Schema Inference & Type Hints

**Status:** `[ ] Not Started`

**Goal:** Tiered schema system - auto-inference (Tier 1) with type hint overrides (Tier 2).

**Prerequisites:** Tasks 1.2 and 1.3 complete

**Files to create:**
- `internal/schema/infer.go` - Pattern-based type inference
- `internal/schema/hints.go` - Type hint parsing and application
- `internal/schema/types.go` - Type definitions and SQL mapping

**Implementation steps:**

```go
package schema

import "regexp"

// TypeHint represents user-friendly type names (Tier 2)
type TypeHint string

const (
    HintText     TypeHint = "text"
    HintNumber   TypeHint = "number"
    HintInteger  TypeHint = "integer"
    HintCurrency TypeHint = "currency"
    HintDate     TypeHint = "date"
    HintTime     TypeHint = "time"
    HintDatetime TypeHint = "datetime"
    HintBoolean  TypeHint = "boolean"
    HintEmail    TypeHint = "email"
    HintURL      TypeHint = "url"
    HintSelect   TypeHint = "select"
    HintTextarea TypeHint = "textarea"
    HintHidden   TypeHint = "hidden"
)

// Column represents a schema column with inferred or hinted type
type Column struct {
    Name       string
    Hint       TypeHint    // User-provided hint (Tier 2)
    SQLType    string      // Mapped SQL type
    NotNull    bool
    Default    string
    Options    []string    // For select types: "select:a,b,c"
    InputType  string      // HTML input type for UI
}

// TypeMapping maps hints to SQL and UI types
var TypeMapping = map[TypeHint]struct {
    SQLType   string
    InputType string
}{
    HintText:     {"TEXT", "text"},
    HintNumber:   {"DECIMAL", "number"},
    HintInteger:  {"INTEGER", "number"},
    HintCurrency: {"DECIMAL(10,2)", "text"},  // with currency formatting
    HintDate:     {"DATE", "date"},
    HintTime:     {"TIME", "time"},
    HintDatetime: {"TIMESTAMP", "datetime-local"},
    HintBoolean:  {"BOOLEAN", "checkbox"},
    HintEmail:    {"TEXT", "email"},
    HintURL:      {"TEXT", "url"},
    HintSelect:   {"TEXT", "select"},
    HintTextarea: {"TEXT", "textarea"},
    HintHidden:   {"TEXT", "hidden"},
}

// InferType determines type from data values (Tier 1)
func InferType(values []string) TypeHint

// ParseHint parses user hint like "select:High,Medium,Low"
func ParseHint(hint string) (TypeHint, []string)

// ApplyHints merges inferred schema with user hints (Tier 2 overrides Tier 1)
func ApplyHints(inferred []Column, hints map[string]string) []Column
```

**Pattern detection for Tier 1:**

```go
var patterns = map[*regexp.Regexp]TypeHint{
    regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`):           HintDate,
    regexp.MustCompile(`^\d{1,2}:\d{2}(:\d{2})?(am|pm)?$`): HintTime,
    regexp.MustCompile(`^-?\d+$`):                       HintInteger,
    regexp.MustCompile(`^-?\d+\.\d+$`):                  HintNumber,
    regexp.MustCompile(`^[$€£¥]\d[\d,]*(\.\d{2})?$`):    HintCurrency,
    regexp.MustCompile(`^(true|false)$`):                HintBoolean,
    regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`):          HintEmail,
    regexp.MustCompile(`^https?://`):                    HintURL,
}

func InferType(values []string) TypeHint {
    // 1. Check patterns against all non-empty values
    // 2. If ≤10 unique values, suggest HintSelect
    // 3. Default to HintText
}
```

**Example - Tier 1 (auto-inferred):**

```markdown
## Expenses
| date | category | amount |
|------|----------|--------|
| 2024-01-15 | Food | $45.50 |
```

→ Inferred: `date: DATE, category: TEXT (select), amount: DECIMAL(10,2)`

**Example - Tier 2 (with hints):**

```yaml
types:
  expenses.amount: currency
  expenses.priority: select:Critical,High,Medium,Low
```

**Acceptance criteria:**
- [ ] `InferType()` correctly identifies date, time, number, currency, boolean, email, URL patterns
- [ ] `InferType()` suggests select for ≤10 unique string values
- [ ] `ParseHint()` correctly parses "select:a,b,c" syntax
- [ ] `ApplyHints()` correctly overrides inferred types with user hints
- [ ] Type mapping produces correct SQL types and HTML input types
- [ ] NOT NULL inferred when all rows have values

**Verification commands:**
```bash
go test ./internal/schema/... -v -run TestInferType
go test ./internal/schema/... -v -run TestParseHint
go test ./internal/schema/... -v -run TestApplyHints
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

#### Task 2.6: WebSocket Protocol Tests

**Status:** `[ ] Not Started`

**Goal:** Add direct WebSocket protocol tests without browser automation.

**Prerequisites:** Task 2.4 (HTTP API) complete, Task 1.0 (Testing Foundation) complete

**Files to read first:**
- `internal/server/websocket.go` - WebSocket handler and MessageEnvelope format
- `internal/testutil/golden.go` - Golden file helpers from Task 1.0

**Files to create:**
- `websocket_protocol_test.go` - Direct WebSocket tests
- `testdata/fixtures/ws-counter.md` - Test app for WebSocket
- `testdata/golden/ws-initial-state.json` - Expected initial message
- `testdata/golden/ws-after-action.json` - Expected post-action message

**Implementation steps:**

1. Create WebSocket test helper:
```go
// websocket_protocol_test.go
package tinkerdown_test

import (
    "strings"
    "testing"
    "github.com/gorilla/websocket"
)

func connectWebSocket(t *testing.T, serverURL string) *websocket.Conn {
    wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/ws"
    conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
    if err != nil {
        t.Fatalf("WebSocket connect failed: %v", err)
    }
    return conn
}

func TestWebSocketInitialState(t *testing.T) {
    srv := setupTestServer(t, "testdata/fixtures/ws-counter.md")
    defer srv.Close()

    conn := connectWebSocket(t, srv.URL)
    defer conn.Close()

    // Read initial state message
    _, msg, err := conn.ReadMessage()
    if err != nil {
        t.Fatalf("Failed to read message: %v", err)
    }

    scrubbed := testutil.Scrub(msg)
    testutil.AssertGolden(t, "ws-initial-state.json", scrubbed)
}

func TestWebSocketAction(t *testing.T) {
    srv := setupTestServer(t, "testdata/fixtures/ws-counter.md")
    defer srv.Close()

    conn := connectWebSocket(t, srv.URL)
    defer conn.Close()

    // Skip initial state
    conn.ReadMessage()

    // Send increment action
    action := `{"blockID":"lvt-0","action":"increment","data":{}}`
    conn.WriteMessage(websocket.TextMessage, []byte(action))

    // Read response
    _, msg, _ := conn.ReadMessage()

    scrubbed := testutil.Scrub(msg)
    testutil.AssertGolden(t, "ws-after-action.json", scrubbed)
}
```

**Acceptance criteria:**
- [ ] Direct WebSocket connection works without chromedp
- [ ] Initial state message matches golden file
- [ ] Action response matches golden file
- [ ] Tests run in < 100ms (vs 5s for browser tests)
- [ ] Scrubbers handle block IDs and timestamps

**Verification commands:**
```bash
go test -v -run TestWebSocket
# Compare timing vs browser test
go test -v -run TestWebSocketInitialState -count=10
```

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

#### Task 3.6: Source Golden Tests

**Status:** `[ ] Not Started`

**Goal:** Add golden file tests for all 9 source types (see source types list above).

**Prerequisites:** Task 1.0 (Testing Foundation) complete, existing source implementations

**Files to read first:**
- `internal/source/source.go` - Source interface
- `internal/source/*.go` - All source implementations
- `internal/testutil/golden.go` - Golden file helpers

**Files to create:**
- `internal/source/golden_test.go` - Golden tests for all sources
- `testdata/sources/fixtures/` - Input files for each source type
- `testdata/sources/golden/` - Expected JSON output for each source

**Implementation steps:**

1. Create fixture files for deterministic testing:
```
testdata/sources/
├── fixtures/
│   ├── users.json          # JSON source input
│   ├── products.csv        # CSV source input
│   ├── tasks.md            # Markdown source input
│   ├── test.db             # SQLite database
│   └── echo-json.sh        # Deterministic exec script
└── golden/
    ├── json-users.json     # Expected parsed output
    ├── csv-products.json
    ├── markdown-tasks.json
    ├── sqlite-users.json
    └── exec-data.json
```

2. Create golden tests for each source type:
```go
// internal/source/golden_test.go
package source_test

func TestJSONSourceGolden(t *testing.T) {
    src, _ := source.NewJSONFileSource("users",
        "testdata/sources/fixtures/users.json", ".")
    data, _ := src.Fetch(context.Background())

    output, _ := json.MarshalIndent(data, "", "  ")
    testutil.AssertGolden(t, "sources/golden/json-users.json", output)
}

func TestCSVSourceGolden(t *testing.T) {
    src, _ := source.NewCSVFileSource("products",
        "testdata/sources/fixtures/products.csv", ".", nil)
    data, _ := src.Fetch(context.Background())

    output, _ := json.MarshalIndent(data, "", "  ")
    testutil.AssertGolden(t, "sources/golden/csv-products.json", output)
}

func TestMarkdownSourceGolden(t *testing.T) {
    src, _ := source.NewMarkdownSource("tasks",
        "testdata/sources/fixtures/tasks.md", "tasks", ".", "", true)
    data, _ := src.Fetch(context.Background())

    output, _ := json.MarshalIndent(data, "", "  ")
    testutil.AssertGolden(t, "sources/golden/markdown-tasks.json", output)
}

func TestExecSourceGolden(t *testing.T) {
    src, _ := source.NewExecSource("data",
        "testdata/sources/fixtures/echo-json.sh", ".")
    data, _ := src.Fetch(context.Background())

    output, _ := json.MarshalIndent(data, "", "  ")
    testutil.AssertGolden(t, "sources/golden/exec-data.json", output)
}

func TestSQLiteSourceGolden(t *testing.T) {
    src, _ := source.NewSQLiteSource("users",
        "testdata/sources/fixtures/test.db", "users", ".", true)
    data, _ := src.Fetch(context.Background())

    output, _ := json.MarshalIndent(data, "", "  ")
    testutil.AssertGolden(t, "sources/golden/sqlite-users.json", output)
}
```

**Acceptance criteria:**
- [ ] Golden tests exist for: json, csv, markdown, sqlite, exec sources
- [ ] All source outputs are deterministic (same input → same output)
- [ ] Exec source uses deterministic script (no timestamps, random data)
- [ ] Tests can be updated with `-update` flag
- [ ] All golden tests pass

**Verification commands:**
```bash
go test ./internal/source/... -v -run Golden
# Update golden files after intentional changes
go test ./internal/source/... -v -run Golden -update
```

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

#### Task 5.4: CLI Testscript

**Status:** `[ ] Not Started`

**Goal:** Add testscript-based black-box tests for CLI commands.

**Prerequisites:** Task 2.5 (CLI Mode) complete, Task 1.0 (Testing Foundation) complete

**Files to read first:**
- `cmd/tinkerdown/commands/*.go` - CLI command implementations
- [testscript docs](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript)

**Files to create:**
- `cli_test.go` - Testscript runner
- `testdata/cli/validate.txtar` - Validate command tests
- `testdata/cli/serve.txtar` - Serve command tests
- `testdata/cli/new.txtar` - New command tests

**Implementation steps:**

1. Add testscript dependency:
```bash
go get github.com/rogpeppe/go-internal/testscript
```

2. Create testscript runner:
```go
// cli_test.go
package main_test

import (
    "os"
    "os/exec"
    "path/filepath"
    "testing"

    "github.com/rogpeppe/go-internal/testscript"
)

func TestCLI(t *testing.T) {
    testscript.Run(t, testscript.Params{
        Dir: "testdata/cli",
        Setup: func(env *testscript.Env) error {
            // Build tinkerdown binary for testing
            binPath := filepath.Join(env.WorkDir, "tinkerdown")
            cmd := exec.Command("go", "build", "-o", binPath, "./cmd/tinkerdown")
            cmd.Dir = ".."
            return cmd.Run()
        },
    })
}
```

3. Create test files in txtar format:
```txtar
# testdata/cli/validate.txtar

# Test validate with valid markdown
exec tinkerdown validate valid.md
stdout 'valid'
! stderr .

# Test validate with invalid source config
! exec tinkerdown validate invalid.md
stderr 'unsupported source type'

-- valid.md --
# My App

## Tasks
- [ ] First task
- [ ] Second task

-- invalid.md --
---
sources:
  bad: { type: unknown }
---
# Invalid App
```

```txtar
# testdata/cli/new.txtar

# Test creating new app from template
exec tinkerdown new todo myapp.md
exists myapp.md
grep 'Tasks' myapp.md

# Clean up
rm myapp.md
```

**Acceptance criteria:**
- [ ] testscript infrastructure works
- [ ] `validate` command tested with valid/invalid inputs
- [ ] `new` command tested with template creation
- [ ] `serve` command tested (startup, port binding)
- [ ] Error messages verified in tests
- [ ] All testscript tests pass

**Verification commands:**
```bash
go test -v -run TestCLI
```

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

#### Task 6.4: Reduce Browser Tests

**Status:** `[ ] Not Started`

**Goal:** Replace slow chromedp tests with faster alternatives, keep only essential browser tests.

**Prerequisites:** Tasks 2.6, 3.6, 5.4 complete (alternative test coverage in place)

**Files to read first:**
- `*_e2e_test.go` - All existing chromedp tests (16 files)
- `docs/plans/2025-12-31-e2e-testing-strategy.md` - Testing strategy

**Current state:**
- 16 e2e test files using chromedp
- Each test takes ~5s (browser startup, navigation, sleep waits)
- Total e2e test time: ~80 seconds

**Target state:**
- 5 essential browser tests (critical user flows only)
- WebSocket protocol tests replace most browser tests
- Golden file tests for output verification
- Total test time: ~10 seconds

**Tests to keep (require real browser):**

| Test | Why Browser Needed |
|------|-------------------|
| `TestLvtClickAction` | JavaScript event handling |
| `TestLvtSourceRendersData` | DOM rendering after WS message |
| `TestFormSubmitAction` | Form submission + validation |
| `TestRealTimeUpdate` | Live state sync display |
| `TestNavigationWorks` | Client-side routing |

**Tests to convert/remove:**

| Current Test | Replacement |
|--------------|-------------|
| `TestLvtSourceExec` | WebSocket protocol test + Source golden test |
| `TestLvtSourceJSON` | Source golden test |
| `TestLvtSourceCSV` | Source golden test |
| `TestLvtSourceMarkdown` | Source golden test |
| `TestLvtSourcePg` | Source golden test (with test DB) |
| `TestLvtSourceSQLite` | Source golden test |
| `TestFrontmatterConfig` | Parser unit test |
| `TestMermaidDiagrams` | Parser golden test |
| `TestSearch` | HTTP endpoint test |
| `TestPlayground` | HTTP endpoint test |

**Implementation steps:**

1. Verify coverage exists for each test being removed:
```bash
# For each test to remove, verify replacement exists
go test -v -run TestJSONSourceGolden  # Replaces TestLvtSourceJSON
go test -v -run TestWebSocketAction   # Replaces browser action tests
```

2. Create consolidated browser test file:
```go
// browser_e2e_test.go - Essential browser tests only
package tinkerdown_test

func TestBrowserEssentials(t *testing.T) {
    t.Run("lvt-click triggers action", testLvtClickAction)
    t.Run("form submit works", testFormSubmitAction)
    t.Run("real-time update displays", testRealTimeUpdate)
    t.Run("navigation works", testNavigationWorks)
    t.Run("source data renders", testSourceRendersData)
}
```

3. Remove redundant test files:
```bash
# After verifying coverage, remove redundant tests
git rm lvtsource_file_e2e_test.go
git rm lvtsource_rest_e2e_test.go
# ... etc
```

4. Update CI to run fast tests first:
```yaml
# .github/workflows/test.yml
jobs:
  fast-tests:
    - go test -v -short ./...  # Skip browser tests
  browser-tests:
    needs: fast-tests
    - go test -v -run Browser ./...
```

**Acceptance criteria:**
- [ ] Only 5 essential browser tests remain
- [ ] All removed tests have equivalent coverage via golden/protocol tests
- [ ] Total test time reduced from ~80s to ~10s
- [ ] CI runs fast tests before slow browser tests
- [ ] No regression in feature coverage

**Verification commands:**
```bash
# Verify test time improvement
time go test ./... -v

# Verify no coverage regression
go test ./... -cover
```

---

## Feature Specifications

> **Note:** This section provides a quick reference for all tinkerdown syntax. For detailed explanations and examples, see [Markdown-Native Design](#markdown-native-design) and [Architecture](#architecture).

### Complete Markdown Grammar (Quick Reference)

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


# OUTPUTS (in YAML frontmatter)
─────────────────────────────────────────────────────────

outputs:
  slack: "#channel"      → Slack output
  email: "addr@co.com"   → Email output
  webhook: "https://..." → Webhook output


# USER/TAG MENTIONS
─────────────────────────────────────────────────────────

@username                → User reference
#tag                     → Tag/category
```

### YAML Configuration (Quick Reference)

Only needed when you want to go beyond zero-config defaults. For detailed tier explanations, see [Architecture](#architecture).

**Tier 2 - Type Hints:**

```yaml
---
# Override inferred types with simple hints
sources:
  expenses:
    from: "#expenses"         # markdown section
    types:
      amount: currency
      category: select

  tasks:
    from: "#tasks"
    types:
      priority: select:Critical,High,Medium,Low
      due: date
    required: [title, due]

# Or use shorthand (auto-detects from heading)
types:
  expenses.amount: currency
  tasks.priority: select:Critical,High,Medium,Low
---
```

**Tier 3 - External Databases:**

```yaml
---
sources:
  # External databases (schema lives in DB)
  users: postgres://${DATABASE_URL}
  orders: mysql://${MYSQL_URL}
  archive: ./archive.db

  # With custom queries
  pending:
    from: postgres://${DATABASE_URL}
    query: SELECT * FROM orders WHERE status = 'pending'

  # REST APIs
  github:
    from: https://api.github.com/repos/user/repo
    headers:
      Authorization: Bearer ${GITHUB_TOKEN}
    cache: 5m
---
```

**Full Configuration (all options):**

```yaml
---
# Type hints (Tier 2)
types:
  expenses.amount: currency
  tasks.priority: select:Critical,High,Medium,Low

# Sources (Tier 3)
sources:
  users: postgres://${DATABASE_URL}
  report:
    query: |
      SELECT e.*, u.name as submitter
      FROM expenses e JOIN users u ON e.user_id = u.id

# Schema file for constraints (Tier 3)
schema: ./schema.sql

# Triggers
triggers:
  - webhook:
      path: /github
      secret: ${WEBHOOK_SECRET}
    action: handle_github

# Outputs
outputs:
  slack:
    token: ${SLACK_TOKEN}
  email:
    smtp: smtp.gmail.com
    user: ${EMAIL_USER}
    pass: ${EMAIL_PASS}

# Metadata
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

See [tinkerdown-example-apps-plan.md](tinkerdown-example-apps-plan.md) for planned examples.

**When examples become functional:**

| Example | Requires Milestone |
|---------|-------------------|
| Two-Line Todo | 1: Works |
| Expense Tracker | 2: Connects |
| Team Tasks, Meeting Notes, Inventory | 3: Acts |
| Runbook with live data | 3: Acts |
| Standup Bot, Health Monitor | 4: Reacts |
| Distributable apps | 5: Ships |

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

### Tinkering Health

The real success metric: **"Users built things we didn't anticipate."**

| Signal | What it means |
|--------|---------------|
| Unexpected use cases in issues/discussions | People are exploring |
| Users sharing apps that surprise us | Composition is working |
| "How do I do X?" questions (not complaints) | Curiosity, not frustration |
| Users modifying LLM-generated apps | They understand and can tinker |

### Anti-Metrics

| Signal | What it means |
|--------|---------------|
| "I can't figure out how to start" | Barrier too high |
| "It broke and I don't know why" | Errors not helpful |
| "I need help understanding this app" | Syntax not readable |
| Users only using exact examples | Not feeling safe to experiment |

### Tinkering Funnel

| Stage | Metric | Target |
|-------|--------|--------|
| **Try** | Time from install to first app running | < 2 minutes |
| **Understand** | Can read and explain any example | 100% |
| **Modify** | Successfully make a change on first try | > 80% |
| **Compose** | Combine two features without docs | > 60% |
| **Share** | App runs for someone else without help | 100% |

### Technical Quality

| Metric | Target |
|--------|--------|
| Pure markdown app works (zero-config) | 100% |
| LLM generation success rate | > 90% |
| Error messages suggest fix | 100% |
| Hot reload latency | < 100ms |
| Graceful degradation (partial failure) | Yes |

---

## Extended Roadmap (Post-v1.0)

> **Note:** These are future directions to consider after the 12-week core plan. Features here are **not committed** - they represent ideas that need validation against user demand and tinkerdown's core philosophy of markdown-first simplicity.

### Already Implemented

| Feature | Status | PR |
|---------|--------|-----|
| Data Source Error Handling (retry, circuit breaker, timeout) | ✅ Complete | - |
| Source Caching Layer (TTL, stale-while-revalidate) | ✅ Complete | #18 |
| Multi-Page WebSocket Support | ✅ Complete | #19 |
| Enhanced CLI Scaffolding (7 templates) | ✅ Complete | #24 |
| Documentation Structure | ✅ Complete | #20 |
| GraphQL Source | ✅ Complete | #22 |

### Future Directions

#### Developer Experience

**WASM Source SDK** - High priority for ecosystem growth:
```bash
tinkerdown wasm init github-issues   # Scaffold new WASM source
tinkerdown wasm build                 # Compile TinyGo to WASM
tinkerdown wasm test                  # Test module locally
```

**Debug Mode** - Useful for troubleshooting:
```bash
tinkerdown serve app.md --debug      # Source fetch timing, WS messages
```

**Validation improvements** - Extend existing `tinkerdown validate`:
- Warn on unused source definitions
- Type-check template patterns

> **Not adopting:** Hot Reload for YAML config. Markdown hot reload already works. Separate YAML reload adds complexity for rare use case.

#### Production Hardening

**Authentication Middleware** - Essential for shared deployments:
```yaml
auth:
  provider: github
  allowed_orgs: [mycompany]
```

**Health Endpoints** - Standard for production:
- `GET /health` - Liveness check
- `GET /ready` - Source connectivity check

**Graceful Shutdown** - Drain connections properly on SIGTERM.

> **Consider carefully:** Rate limiting. May be better handled by a reverse proxy (nginx, Cloudflare) than built into tinkerdown.

#### UI Enhancements

**Pagination & Filtering** - Needed for real data volumes:
```html
<table lvt-source="users" lvt-paginate="20" lvt-sortable>
</table>
```

**Charts** - Already in Phase 6 core roadmap.

> **Not adopting:** Component library (modals, toasts, accordions). These add significant complexity and are better served by the HTML layer (Layer 3) where users can use any JS library they prefer. Tinkerdown should not become a UI framework.

#### Data Sources

> **Not adopting:** MongoDB, S3, Redis as native sources. The WASM source system already enables these integrations without bloating the core binary. Community can build these as WASM modules.

> **Not adopting:** Source Composition with CEL. Template-level filtering (`{{range where .Status "active"}}`) is sufficient for most cases. CEL adds a complex dependency for marginal benefit.

**Webhook Source** - Already covered in Phase 4 (Task 4.3).

#### Advanced Features

> **Not adopting:** Offline Support. Tinkerdown is a local-first tool - your markdown file is already on disk. Adding service workers, offline queues, and sync logic adds significant complexity for a scenario that conflicts with the core use case (edit markdown, see changes).

> **Not adopting:** WASM Marketplace. Premature optimization. Focus on making WASM sources easy to build and share via git. A marketplace can come later if there's demand.

> **Defer:** API Endpoint Mode with OpenAPI generation. Task 2.4 HTTP API provides basic CRUD. Full OpenAPI/Swagger adds complexity. Revisit based on user demand.

### Philosophy Checklist

Before adding post-v1.0 features, validate against these principles:

1. **Does it require config?** If yes, can the 80% use case work without it?
2. **Does WASM already solve this?** New source types should be WASM modules, not core code.
3. **Is this a tinkerdown feature or a deployment concern?** Auth, rate limiting, SSL often belong in the deployment layer.
4. **Does it increase the learning curve?** Every new concept makes the two-line todo harder to explain.

### What v1.0 Must Prove

Before any extended roadmap work:

1. **Zero-config works** - `## Tasks` → working CRUD app
2. **LLM generation works** - AI can create tinkerdown apps reliably
3. **Users adopt it** - Real usage validates the concept
4. **WASM extensibility works** - Community can build custom sources

Features beyond the core plan should be driven by observed user needs, not speculative requirements

---

## Summary

### The Philosophy

**Tinkerdown enables tinkering.** We don't prescribe what you build - we give you blocks simple enough to understand and powerful enough to matter.

```
Wonder → Try → See → Understand → Modify → Try again
```

### The Promise

```markdown
# Todo
- [ ] Try tinkerdown
```

**Two lines. Working app. Zero configuration.**

But more importantly: **you can read it, understand it, modify it, share it.**

### The Grammar

Learnable in an afternoon. Fits on an index card:

```
## Heading        → Data source
- [ ] item        → Task list
| col | col |     → Table
`count(x)`        → Computed value
@daily:9am        → Schedule trigger
[Button](action:x)→ Action button
```

### The Layers

| Layer | For | Uses |
|-------|-----|------|
| 1 | 80% of apps | Pure markdown |
| 2 | Advanced config | YAML frontmatter |
| 3 | Full control | HTML + templates |

### The Milestones

| Milestone | What it unlocks |
|-----------|----------------|
| 1: Works | Discovery, Learning, Iteration |
| 2: Connects | Composition with real data |
| 3: Acts | Buttons, forms, API |
| 4: Reacts | Automation, triggers |
| 5: Ships | Distribution, sharing |
| 6: Scales | Polish, examples |

### The Success Metric

**"Users built things we didn't anticipate."**

### Next Action

Ship Milestone 1: A markdown file becomes an interactive app that persists changes.

Enable the tinkering loop. See what people build. Learn. Iterate.
