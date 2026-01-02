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

Tinkerdown is a tiered system. Start simple, add power as needed:

**Tier 1: Pure Markdown** (zero config)

| Block | What It Does |
|-------|--------------|
| `## Heading` | Creates a data source named after the heading |
| `- [ ] Task` | Interactive checkbox list |
| `\| Table \|` | Editable data grid with auto-generated forms |
| `[Button](action:x)` | Triggers actions |
| ``=count(x)`` | Computed values from your data |
| `@daily:9am` | Schedule triggers |

**Tier 2-3: YAML Frontmatter** (external data)

```yaml
---
sources:
  users: postgres://${DATABASE_URL}
  orders: ./local.db
  github: https://api.github.com/repos/...
types:
  expenses.amount: currency
---
```

Connect to Postgres, SQLite, REST APIs, shell commands. Override type inference.

**Tier 4: HTML + Go Templates** (full control)

```html
<table lvt-source="users" lvt-columns="name,email" lvt-actions="edit,delete">
</table>

{{range .items}}
  <div class="card">{{.title}}</div>
{{end}}
```

Custom layouts, conditional rendering, any CSS/JS you want.

**Tinkering means:** start with two lines of markdown, add a YAML source when you need real data, drop to HTML when you need custom UI. Each step is small, visible, and reversible.

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

1. [Markdown-Native Design](#markdown-native-design)
2. [Architecture](#architecture)
3. [v1.0 Roadmap](#v10-roadmap)
4. [Post-v1.0 Considerations](#post-v10-considerations)
5. [Summary](#summary)
6. [Appendix: Tinkering Stories](#appendix-tinkering-stories)

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

**Forgiving Inference:**
If a user enters data that violates the inferred type (e.g., "1234A" in a number column), the UI will not reject it. Instead, it will offer to **"Upgrade column to Text"**, preventing data loss and frustration. Visual indicators in table headers show inferred types.

### Auto-Generated Forms

Every data collection gets an input form automatically. No HTML required.

```markdown
## Expenses
| date | category | amount | note |
|------|----------|--------|------|
| 2024-01-15 | Food | $45.50 | Groceries |
```

**What you get:**

```ini
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

### Row Identification & Data Integrity

To safely edit rows without requiring users to manually add IDs, Tinkerdown uses a **Hybrid Identification Strategy**:

1.  **Explicit ID (Best):** If a row has `<!-- id:xyz -->`, it is used.
2.  **Content Hash (Default):** If no ID exists, a hash of the row content is used (e.g., `hash("Buy milk")`).
    *   *Limitation:* Duplicate rows (e.g., two "Buy milk" tasks) cannot be distinguished. The UI will warn about duplicates.
3.  **Index (Fallback):** Never used for writes, only for display if hashing fails.

**Explicit IDs are optional.** By default, the UI appends a hidden ID comment `<!-- id:xyz -->` when modifying a row to ensure future stability. However, users can disable this behavior to keep their markdown "clean", relying solely on content hashing (with the trade-off that duplicate rows cannot be distinguished).

### Write-back, Conflicts, and Formatting (Normative v1)

Tinkerdown treats the markdown file as the durable source of truth and writes changes back conservatively.

**Write-back scope**
- Lists: updates are line-oriented (toggle/update one list item line).
- Tables: updates replace only the affected row line(s) and may append `<!-- id:... -->` to stabilize row identity.
- Tinkerdown does not reflow unrelated prose or reformat entire sections.

**Formatting guarantees**
- Preserves surrounding whitespace and unrelated lines.
- Does not guarantee markdown table column alignment (padding). If the user wants aligned tables, that is an optional formatter step (not required for correctness).

**Conflict handling**
- If the file changes on disk since it was last read, writes fail with a conflict error and a `*.conflict-<timestamp>.md` copy is created.
- The UI should prompt the user to reload and reconcile.

### Scheduling with @mentions

Date mentions can represent *due dates* on items, or *triggers* for automation (see the normative rules below).

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

#### Scheduling & @mentions (Normative v1)

This section is **normative**: implementations should match these rules.

**Tokenization & scope**
- A schedule token starts with `@` and must be preceded by start-of-line or whitespace.
- Schedule tokens inside inline code spans, fenced code blocks, and HTML blocks are ignored.
- Unknown `@word` tokens (e.g., `@alice`) are treated as plain text in v1 (not schedules).
- Escape a literal schedule token with `\@` (e.g., `\@daily:9am`).

**Due dates vs triggers**
- `@...` tokens on list/table items are **metadata** (e.g., `due_at`) and do **not** execute automation.
- Automation triggers must be explicit using an imperative line:
  - `Notify @daily:9am @weekdays` (sends via `outputs:`)
  - `Run action:my_action @daily:9am` (invokes an action)

**Time zones & DST**
- Default timezone is the server/process local timezone.
- Optional override: `timezone: America/Los_Angeles` in frontmatter.
- Recurring schedules run on local wall-clock time. If a time is skipped due to DST, run at the next valid wall-clock instant. If a time occurs twice, run once (first occurrence) unless `schedule.duplicates: allow` is explicitly set.

**Parse failures**
- If a token cannot be parsed, it is ignored and a warning is surfaced inline (never silently breaks the page).

### Computed Values

Inline code with expressions:

```markdown
## Budget

**Total Income:** `=sum(income.amount)`
**Total Expenses:** `=sum(expenses.amount)`
**Balance:** `=sum(income.amount) - sum(expenses.amount)`
**Tasks Done:** `=count(tasks where done)` / `=count(tasks)`
```

**Expression functions:**

| Function | Example |
|----------|---------|
| `=count(source)` | `=count(tasks)` |
| `=count(source where expr)` | `=count(tasks where done)` |
| `=sum(source.field)` | `=sum(expenses.amount)` |
| `=avg(source.field)` | `=avg(scores.value)` |
| `=min/max(source.field)` | `=min(tasks.due)` |

**Error Visibility:**
If an expression fails (e.g., typo in source name), it renders an **inline error** (e.g., `(Error: source 'taskss' not found)`) instead of breaking the page or showing a blank space.

#### Expressions (Normative v1)

To avoid ambiguity with normal markdown code spans, **only** inline code spans that start with `=` are treated as expressions.

Examples:
- Expression: `=sum(expenses.amount)`
- Literal code: `sum(expenses.amount)`

**Escaping**
- If you need to show a literal that begins with `=`, write it as `` `\=literal` `` (renders as `\=literal`), or use fenced code blocks.

**Evaluation model**
- Expressions are read-only and cannot mutate state.
- Expressions evaluate against the current in-memory state at render time.
- Errors render inline (never crash the page).

### Status Banners

Blockquotes with emoji:

```markdown
> ✅ All systems operational

> ⚠️ `=count(tasks where due < today and not done)` overdue

> 📊 `=count(deals)` active deals | `=sum(deals.value)` pipeline
```

### Actions

Button links and action links:

```markdown
[Add Task]                           → Button
[Clear Completed](action:clear-done) → Action link
[Export CSV](export:csv)             → Export
[← Back](back)                       → Navigation
```

#### Action Model v1 (Normative)

Actions are named, parameterized operations that can be invoked from **UI links**, **triggers**, **CLI**, and the **HTTP API**.

**Declaration** (frontmatter)
- Actions are declared under `actions:`.
- If an action is invoked but not declared, it is an error (rendered inline / returned via API).

```yaml
---
actions:
  clear-done:
    kind: sql
    source: tasks
    statement: |
      DELETE FROM tasks WHERE done = true
    confirm: "Clear all completed tasks?"

  add-task:
    kind: sql
    source: tasks
    statement: |
      INSERT INTO tasks(text, done) VALUES (:text, false)
    params:
      text: { required: true }
---
```

**Invocation**
- Markdown links invoke actions by name: `[Clear Completed](action:clear-done)`.
- Triggers invoke actions via imperative lines: `Run action:clear-done @daily:9am`.
- Params come from (in priority order): explicit payload (API/CLI), UI form fields, or `params:` defaults.

**Security defaults**
- `kind: exec` is **disabled by default** and requires an explicit `--allow-exec` runtime flag.
- All SQL statements must be parameterized (named params like `:text`).
- Actions execute with the operator identity (`--operator`), available as `{{operator}}` for templating defaults/logging.

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

```ini
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
│  │  **Done:** `=count(tasks where done)`                    │   │
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

### Compatibility & Migration (v1)

This doc introduces markdown-native sources (headings/lists/tables). The existing `lvt-source` model remains valid.

**v1 compatibility goal:** support both styles in the same document:
- Markdown-native blocks are auto-parsed into sources by heading.
- `lvt-source` blocks reference sources defined via frontmatter or inferred from markdown-native blocks.

**Migration path:**
- Start with markdown-native Tier 1.
- Add frontmatter `sources:` only when you need external data or overrides.
- Use HTML templates only for custom rendering, not for basic CRUD.

### Schema & Configuration Tiers

Three levels of configuration based on user needs:

```ini
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

**Preservative Formatting:**
When writing back to the markdown file, the system uses "Preservative Formatting". It respects the user's existing indentation, spacing, and alignment style. It only modifies the specific lines that changed, minimizing visual jumps and preserving the "hand-crafted" feel of the document.

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

```ini
┌─────────────────────────────────────────────────────────────────┐
│                    UNIFIED SQL ENGINE                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Markdown tables → loaded into in-memory SQLite              │
│  2. External DBs → connected via drivers                        │
│  3. Cross-source queries → joined in local SQLite snapshots     │
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
│              │  (SQLite for markdown + cached  │                │
│              │   snapshots for external DBs)   │                │
│              └─────────────────────────────────┘                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### Cross-Source Queries (v1 constraints)

To keep v1 shippable, cross-source SQL is defined as **local joins over snapshots**:
- Each source (markdown, REST, Postgres, MySQL, SQLite) can be materialized into a local SQLite table/view.
- `query:` sources run against the local SQLite snapshot, not as a distributed query planner.
- External DB sources may still support **pass-through queries** (server-side filtering) when `from:` is set and the query references only that remote source.

**Write rules**
- Writes are allowed only to sources explicitly configured as writable (`sync: write|both` for markdown; explicit config for databases).
- `query:` result sets are always read-only (no implicit `UPDATE`/`DELETE` through views).

**Performance safety**
- Every external source must have caching controls (TTL) and an optional row/byte limit for snapshotting.
- The UI should surface when data is stale and when it will refresh.

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

```ini
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
│         (Conflict Check: Reload if file changed on disk)        │
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

### Milestone Scope Guardrails (Non-goals)

These are explicit **non-goals** to keep each milestone shippable.

- **Milestone 1:** No external sources; no automation triggers; no custom actions; no expressions (beyond static rendering).
- **Milestone 2:** Cross-source queries are limited to the v1 snapshot model (see Architecture); no distributed query planner; exec remains off by default.
- **Milestone 3:** Expressions require the `=` prefix and are read-only; action invocation requires declared `actions:`; no implicit mutations from expressions.
- **Milestone 4:** Schedules execute only from explicit imperative lines (`Notify ...`, `Run action:...`), never from due dates on items; webhook triggers must validate secrets.
- **Milestone 5:** Packaging/distribution only (no new language features); security focuses on secret hygiene and least-privilege runtime config.
- **Milestone 6:** Docs/examples/perf/security hardening; feature scope is frozen except for critical fixes.

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
| 1.8 | **Row Identification** | Hybrid ID strategy (Explicit > Hash) implemented |
| 1.9 | **Concurrency Control** | File watching detects external edits; UI prompts reload |
| 1.10 | **Example: Two-Line Todo** | `# Todo\n- [ ] task` runs as complete app |

**Security:** Input validation on all form submissions. Sanitize markdown content.

**Testing:** Golden file tests for parser output. Browser test for hot reload. Concurrency tests (write while file changes).

---

### Milestone 2: It Connects

**Goal:** Connect to external databases, APIs, and commands.

| Task | Description | Acceptance Criteria |
|------|-------------|---------------------|
| 2.1 | **SQLite Source** | `from: ./data.db` loads SQLite tables |
| 2.2 | **PostgreSQL Source** | `from: postgres://...` connects and queries |
| 2.3 | **REST Source** | `from: https://...` fetches JSON. Supports pass-through queries. |
| 2.4 | **Exec Source** | `type: exec` runs command. **Disabled by default.** |
| 2.5 | **Source Caching** | TTL and stale-while-revalidate work |
| 2.6 | **Cross-Source Queries** | SQL JOINs across markdown + external sources |
| 2.7 | **Auto-Timestamp** | `{{now}}` fills current date/time on submit |
| 2.8 | **Operator Identity** | `--operator alice` sets `{{operator}}` |
| 2.9 | **Example: Expense Tracker** | Markdown + SQLite source working together |

**Security:**
-   **Exec Sources:** Disabled by default. Require `--allow-exec` flag or explicit user confirmation.
-   **SQL:** Parameterized queries only. No string interpolation.
-   **REST:** Clarify that SQL-over-REST fetches all data; use pass-through for server-side filtering.

**Testing:** Golden tests for each source type. Integration test with test Postgres container. Verify exec sources fail without flag.

---

### Milestone 3: It Acts

**Goal:** Buttons trigger actions. API and CLI available.

| Task | Description | Acceptance Criteria |
|------|-------------|---------------------|
| 3.1 | **Action Buttons** | `[Button](action:name)` triggers named action |
| 3.2 | **Computed Expressions** | ``=count(tasks where done)`` evaluates live |
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
| 6.3 | **Charts** | ````chart` `` code blocks render visualizations |
| 6.4 | **Example Suite** | 10+ working examples covering all features |
| 6.5 | **Reduce Browser Tests** | Replace slow chromedp with fast protocol tests |
| 6.6 | **Performance Baseline** | Documented latency targets met |
| 6.7 | **Security Audit** | Exec sources sandboxed, input validation complete |
| 6.8 | **Release** | GitHub release with binaries, changelog, announcement |

**Testing:** Full test suite passes. Manual testing of all examples.

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

---

## Appendix: Tinkering Stories

Patterns of exploration that tinkerdown should enable. Not traditional user stories, but design principles.

| Category | Example | Requires |
|----------|---------|----------|
| **Discovery** | "I can start with any markdown file and see what happens" | Zero barrier |
| **Learning** | "I can read any app and understand what it does" | Readable syntax |
| **Composition** | "I can add a second source without breaking the first" | Independent pieces |
| **Iteration** | "I change the markdown and see the result instantly" | Hot reload |
| **Recovery** | "I can always hand-edit the markdown to fix a problem" | Human-editable |
| **Sharing** | "I share my app by sharing the markdown file" | Self-contained |

**Anti-patterns (if these happen, we've failed):**

- "I had to read the whole documentation before I could start"
- "I made a small change and everything broke"
- "I don't understand what this app does even though I'm looking at the source"
- "The LLM generated this and I have no idea how to modify it"
