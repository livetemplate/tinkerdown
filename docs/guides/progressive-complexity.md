# Progressive Complexity

Tinkerdown lets you build increasingly complex apps by graduating through natural complexity tiers. Each tier builds on the previous one — nothing needs rewriting. Start with pure markdown and add capabilities as your needs grow.

## Tier 0: Pure Markdown

Write standard markdown. Task lists become interactive automatically — checkboxes toggle, new items can be added, changes persist to the file.

```markdown
# Shopping List
- [ ] Milk
- [x] Bread
- [ ] Eggs
```

No YAML, no HTML, no configuration. Just markdown.

**What you get:** Interactive checkboxes, add-item form, file persistence.

**Examples:** [auto-tasks](../../examples/auto-tasks/)

---

## Tier 1: Markdown + YAML Sources

Define data sources in YAML frontmatter and write standard markdown tables. Tinkerdown matches heading names to source names and auto-generates the appropriate UI.

```yaml
---
sources:
  expenses:
    type: sqlite
    db: ./expenses.db
    table: expenses
    readonly: false
---

# Expense Tracker

## Expenses
| Description | Category | Amount |
|-------------|----------|--------|
```

The heading "Expenses" matches the source named "expenses". Since the source is writable (`readonly: false`), Tinkerdown generates a full CRUD interface: data table with edit and delete buttons, plus an add form. Amount gets a number input because schema introspection detects it as a numeric column.

For read-only sources (REST APIs, JSON files), the table auto-populates with a refresh button and no CRUD controls.

### Computed Sources

Derive aggregated data from other sources without writing code:

```yaml
---
sources:
  expenses:
    type: sqlite
    db: ./expenses.db
    table: expenses
  by_category:
    type: computed
    from: expenses
    group_by: category
    aggregate:
      total: sum(amount)
      count: count()
---

## By Category
| Category | Total | Count |
|----------|-------|-------|
```

Supported aggregation functions: `sum()`, `count()`, `avg()`, `min()`, `max()`. Computed sources auto-refresh when the parent source changes.

### How Source Matching Works

Tinkerdown matches markdown table headings to source names using smart matching:

1. **Exact match:** Heading "Expenses" (slug: `expenses`) matches source `expenses`
2. **Underscore normalization:** Heading "By Category" (slug: `by-category`) also matches source `by_category`
3. **Word containment:** Heading "My Monthly Expenses" matches source `expenses` (the slug `my-monthly-expenses` contains `expenses` at a word boundary)
4. **No match:** The table renders as a normal static markdown table

Exact and underscore-normalized matches take precedence over word containment. If containment is ambiguous (heading matches multiple sources), Tinkerdown skips with a warning. Use `auto_bind: false` on a source to exclude it from matching — useful for generic names like `data` or `items`:

```yaml
sources:
  status:
    type: rest
    from: https://api.example.com/status
    auto_bind: false  # don't auto-match to heading "Status"
```

**What you get:** Auto-populated tables, CRUD for writable sources, schema-aware form inputs, computed aggregations.

**Examples:** [auto-table-sqlite](../../examples/auto-table-sqlite/), [auto-table-rest](../../examples/auto-table-rest/), [computed-source](../../examples/computed-source/)

---

## Tier 2: HTML + lvt-* Attributes

When auto-inference can't give you what you need — custom layouts, explicit action buttons, confirmation dialogs, datatable features, cross-source selects — use HTML elements with `lvt-*` attributes.

```html
<table lvt-source="tasks" lvt-columns="title,status" lvt-datatable lvt-actions="Complete,Delete">
</table>

<form lvt-submit="Add" lvt-reset-on:success>
  <input name="title" placeholder="New task" required>
  <select name="priority" lvt-source="priorities" lvt-value="id" lvt-label="name"></select>
  <button type="submit">Add</button>
</form>

<button lvt-click="ClearDone" lvt-confirm="Delete all completed tasks?">Clear Done</button>
```

The escape from Tier 1 to Tier 2 is clean: if the auto-generated UI isn't right, replace the markdown table with explicit HTML. The auto-generated template can serve as a starting point.

**What you get:** Explicit data binding, custom forms, action buttons with confirmation, datatable with sorting/pagination, cross-source selects.

**References:** [Auto-Rendering Guide](auto-rendering.md), [lvt-* Attributes Reference](../reference/lvt-attributes.md)

---

## Tier 3: Go Templates

For full control over rendering, use Go template syntax inside `lvt` code blocks:

````markdown
```lvt
<div lvt-source="expenses">
  {{if .Error}}
    <div class="error">{{.Error}}</div>
  {{else}}
    {{range .Data}}
    <div class="card">
      <h3>{{.Description}}</h3>
      <span class="badge">{{.Category}}</span>
      <span class="amount">${{.Amount}}</span>
      <button lvt-click="Delete" lvt-data-id="{{.Id}}">Remove</button>
    </div>
    {{end}}
  {{end}}
</div>
```
````

Go templates give you conditionals, loops, custom HTML structure, and access to the full state object (`.Data`, `.Error`, `.Errors`).

**What you get:** Conditional rendering, custom HTML layouts, multiple data iterations, error handling.

**References:** [Go Templates Guide](go-templates.md)

---

## Tier 4: Custom Data Sources

For data that doesn't fit built-in source types, write a custom source in TinyGo compiled to WASM. The module exports a `fetch` function that returns JSON data:

```yaml
sources:
  custom:
    type: wasm
    path: ./sources/custom.wasm
```

Use `tinkerdown new myapp --template wasm-source` to scaffold a working WASM source project with build scripts and a test app.

**What you get:** Arbitrary data fetching logic, compiled to WASM, runs server-side.

**References:** [WASM Source Docs](../sources/wasm.md)

---

## When to Graduate

| If you need... | Use |
|----------------|-----|
| Interactive task lists | Tier 0 — just write markdown checkboxes |
| Data tables from databases/APIs | Tier 1 — markdown tables + YAML sources |
| Aggregated dashboards | Tier 1 — computed sources |
| Custom form layouts | Tier 2 — HTML + lvt-* attributes |
| Conditional rendering | Tier 3 — Go templates |
| Custom data fetching | Tier 4 — WASM sources |

The key principle: **start at the lowest tier that works.** Each tier adds complexity but also capability. If you're writing HTML when a markdown table would suffice, you're working harder than you need to.
