# Markdown-Native Micro Apps: Deep Dive Research

**Date:** 2025-12-31
**Status:** Research
**Problem:** How far can pure markdown take us in building micro apps without HTML/Go templates?

---

## The Elephant in the Room

Current tinkerdown requires:
1. **HTML** for forms and interactive elements
2. **Go html/template** for dynamic content rendering
3. **lvt-* attributes** on HTML elements for binding

This creates a learning curve that undermines the "micro apps in markdown" promise:
- Non-technical users can write markdown but not HTML
- Go templates have non-intuitive syntax (`{{range .items}}...{{end}}`)
- LLMs can generate this, but users can't maintain/modify it

**Core Question:** Can markdown markup itself handle forms and lists without escaping to HTML?

---

## 1. The User Spectrum

| Level | Knows | Can Build | Barrier |
|-------|-------|-----------|---------|
| Novice | Basic markdown, tables | Static docs | Any code |
| Intermediate | YAML, frontmatter | Config files | HTML syntax |
| Developer | HTML, templates | Full apps | None |

**Target:** Enable Novice/Intermediate users to build 80% of Runbooks and Productivity apps.

---

## 2. What Needs to Be Declarative

### Forms (Input Collection)
- Text inputs
- Selects/dropdowns
- Checkboxes
- Date pickers
- File uploads
- Submit buttons

### Lists (Data Display)
- Dynamic tables from data sources
- Filtered/sorted views
- Aggregations (counts, sums)
- Item actions (edit, delete)

### Actions
- Buttons that trigger mutations
- Links that navigate with context
- Confirmations

### Conditionals
- Show/hide based on data
- Different views for empty vs populated

---

## 3. Existing Markdown Form Syntaxes

### 3.1 GitHub-Flavored Markdown (Issue Templates)

```yaml
# .github/ISSUE_TEMPLATE/bug.yml
body:
  - type: input
    id: version
    attributes:
      label: Version
      placeholder: 1.0.0
  - type: dropdown
    id: severity
    attributes:
      label: Severity
      options:
        - Low
        - Medium
        - High
  - type: checkboxes
    id: terms
    attributes:
      label: Agreement
      options:
        - label: I agree to the terms
```

**Pros:** Clean YAML, widely understood
**Cons:** Separate file, not inline markdown

### 3.2 Markdown-It Checkbox Extension

```markdown
- [ ] Unchecked item
- [x] Checked item
```

**Pros:** Native markdown feel
**Cons:** Only checkboxes, no other input types

### 3.3 MDX (React Components in Markdown)

```mdx
# Contact Form

<Form action="/submit">
  <Input name="email" type="email" required />
  <Select name="topic" options={['Support', 'Sales']} />
  <Button type="submit">Send</Button>
</Form>
```

**Pros:** Full component flexibility
**Cons:** Requires JSX knowledge, React runtime

### 3.4 Obsidian Dataview Queries

```dataview
TABLE file.name, status, due
FROM "tasks"
WHERE status != "done"
SORT due ASC
```

**Pros:** SQL-like, powerful queries
**Cons:** Display only, no mutations

### 3.5 GUI-Style Databases

```
/database inline
```

Then GUI for columns/filters/sorts.

**Pros:** Zero syntax
**Cons:** Requires GUI, not markdown-representable

---

## 4. Proposed Approaches for Tinkerdown

### Approach A: Extended Markdown Tables

Use markdown tables as the primary UI primitive, with special column syntax.

#### Forms via Tables

```markdown
## Add Task

| Field | Input |
|-------|-------|
| Title | [text:title:required] |
| Priority | [select:priority:Low,Medium,High] |
| Due Date | [date:due] |
| Notes | [textarea:notes:3] |
| | [submit:Add Task:add_task] |
```

Renders as a form. The `[]` syntax defines input types.

#### Dynamic Lists via Tables

```markdown
## Tasks
<!-- source: tasks | filter: status != done | sort: due -->

| Title | Priority | Due | Actions |
|-------|----------|-----|---------|
| *auto-populated from source* |
```

The HTML comment directive tells tinkerdown to auto-populate.

**Pros:**
- Tables are familiar markdown
- Clear visual structure
- Easy to understand

**Cons:**
- Limited layout flexibility
- HTML comment directives feel hacky
- Complex forms get unwieldy

---

### Approach B: Fenced Code Blocks

Use fenced code blocks with custom languages.

#### Forms

~~~markdown
```form add_task
title: text | required | "Task title"
priority: select | Low, Medium, High | default: Medium
due: date
notes: textarea | rows: 3
---
[Add Task]
```
~~~

#### Lists

~~~markdown
```list tasks
source: tasks
filter: status != done
sort: due asc
columns: title, priority, due
actions: edit, delete
empty: "No tasks yet"
```
~~~

**Pros:**
- Clean separation of declaration
- Familiar code fence syntax
- Easy to parse

**Cons:**
- Not visible in standard markdown preview
- Feels like config, not content

---

### Approach C: Markdown Links as Actions + Frontmatter Forms

Define forms in frontmatter, use markdown links for actions.

#### Frontmatter

```yaml
---
forms:
  add_task:
    fields:
      - name: title
        type: text
        required: true
      - name: priority
        type: select
        options: [Low, Medium, High]
    submit: Add Task
---
```

#### Content

```markdown
## Add Task

[Show Form](form:add_task)

## Current Tasks

[tasks: title, priority, due | filter: status=active | sort: due]

| Title | Priority | Due |
|-------|----------|-----|

Actions: [Edit](action:edit) | [Delete](action:delete?confirm=true)
```

**Pros:**
- Clean separation of structure (YAML) and content (markdown)
- Markdown links are natural for actions
- Frontmatter is already established pattern

**Cons:**
- Forms not visible where used
- Requires jumping between frontmatter and content

---

### Approach D: Inline Shortcodes (Hugo-Style)

Use `{{< >}}` shortcodes inline in markdown.

```markdown
## Add Task

{{< form action="add_task" >}}
  {{< input name="title" required >}}
  {{< select name="priority" options="Low,Medium,High" >}}
  {{< date name="due" >}}
  {{< submit >}}Add Task{{< /submit >}}
{{< /form >}}

## Current Tasks

{{< list source="tasks" filter="status!=done" sort="due" >}}
| Title | Priority | Due | Actions |
|-------|----------|-----|---------|
{{< /list >}}
```

**Pros:**
- Inline where needed
- Flexible nesting
- Established pattern (Hugo, Eleventy)

**Cons:**
- Looks like templates (what we're avoiding)
- Verbose
- Not standard markdown

---

### Approach E: Pure YAML with Markdown Sections

The most radical: everything declarative in YAML, markdown only for documentation.

```yaml
---
title: Task Tracker

sources:
  tasks:
    type: markdown
    anchor: "#tasks"

views:
  - type: form
    id: add_task
    fields:
      - {name: title, type: text, required: true}
      - {name: priority, type: select, options: [Low, Medium, High]}
    submit: {text: Add Task, action: add}

  - type: table
    source: tasks
    columns: [title, priority, due]
    filter: status != done
    sort: due
    actions:
      - {icon: edit, action: edit}
      - {icon: trash, action: delete, confirm: true}
---

# Task Tracker

Documentation and static content goes here in markdown.
The UI is entirely defined above.
```

**Pros:**
- Maximum declarativeness
- No escaping to any other syntax
- LLMs can generate perfect YAML
- Easy validation, tooling

**Cons:**
- Least markdown-like
- Content and UI separated
- Harder to see what renders where

---

## 5. Hybrid Recommendation: Layered Approach

Different complexity levels for different users:

### Layer 1: Pure Markdown (Novice)
For simple data display, use markdown tables with HTML comment directives:

```markdown
## Tasks
<!-- lvt-source="tasks" -->
| Title | Priority | Due |
|-------|----------|-----|

Click a row to see details.
```

The table auto-populates. No forms, no actions—just display.

### Layer 2: YAML + Markdown (Intermediate)
Forms and actions defined in frontmatter, referenced in content:

```yaml
---
forms:
  add:
    fields: [title: text, priority: select:Low|Medium|High]
    submit: Add
actions:
  delete: {confirm: "Delete this task?"}
---
```

```markdown
## Add Task
[form:add]

## Tasks
<!-- lvt-source="tasks" lvt-actions="edit,delete" -->
| Title | Priority | Due | |
|-------|----------|-----|-|
```

### Layer 3: Full Control (Developer)
HTML + Go templates for maximum flexibility:

```html
<form lvt-submit="add_task">
  <input name="title" required>
  <select name="priority">
    {{range .priorities}}<option>{{.}}</option>{{end}}
  </select>
  <button type="submit">Add</button>
</form>

{{range .tasks}}
<div class="task">
  <h3>{{.title}}</h3>
  <button lvt-click="delete" data-id="{{.id}}">Delete</button>
</div>
{{end}}
```

---

## 6. Concrete Syntax Proposal

After analysis, here's the recommended syntax:

### 6.1 Auto-Populating Tables

```markdown
<!-- lvt-source="tasks" lvt-filter="status=active" lvt-sort="due:asc" -->
| Title | Priority | Due Date |
|-------|----------|----------|
```

- HTML comment on line before table
- Table headers define columns to display
- No body rows needed—auto-populated
- Column names match source field names

### 6.2 Table Row Actions

```markdown
<!-- lvt-source="tasks" lvt-row-actions="edit,delete" -->
| Title | Priority | | |
|-------|----------|---|---|
```

- Empty header columns become action buttons
- `lvt-row-actions` defines which actions appear
- Actions reference frontmatter definitions

### 6.3 Forms in Frontmatter

```yaml
forms:
  add_task:
    title: "Add Task"
    fields:
      - name: title
        type: text
        label: "Task Title"
        required: true

      - name: priority
        type: select
        options: [Low, Medium, High]
        default: Medium

      - name: due
        type: date

      - name: tags
        type: multiselect
        source: tags  # Options from another source

    submit:
      text: "Add Task"
      action: add
      target: tasks
```

### 6.4 Form Placement

```markdown
## Add New Task

<!-- lvt-form="add_task" -->

---

## All Tasks
...
```

Or inline shorthand:

```markdown
## Add Task
[form:add_task]
```

### 6.5 Action Buttons (Non-Form)

```markdown
[Clear Completed](action:clear_completed)
[Export CSV](action:export?format=csv)
[Refresh](action:refresh)
```

### 6.6 Conditional Blocks

```markdown
<!-- lvt-if="tasks.length == 0" -->
No tasks yet. Add one above!
<!-- lvt-endif -->

<!-- lvt-if="tasks.length > 0" -->
| Title | Priority |
|-------|----------|
<!-- lvt-endif -->
```

---

## 7. Implementation Complexity

| Feature | Parser Work | Runtime Work | Risk |
|---------|-------------|--------------|------|
| Auto-tables | Medium | Medium | Low |
| Form frontmatter | Low | Medium | Low |
| Form placement | Low | Low | Low |
| Action links | Low | Medium | Low |
| Conditionals | High | Medium | Medium |
| Row actions | Medium | High | Medium |

### Recommended Implementation Order

1. **Auto-populating tables** (highest impact, medium effort)
2. **Form frontmatter + placement** (enables input without HTML)
3. **Action links** (natural markdown extension)
4. **Row actions** (completes CRUD)
5. **Conditionals** (polish, can wait)

---

## 8. Comparison: Before vs After

### Before (HTML + Templates)

```html
<form lvt-submit="add_task">
  <label>Title</label>
  <input type="text" name="title" required>

  <label>Priority</label>
  <select name="priority">
    <option>Low</option>
    <option selected>Medium</option>
    <option>High</option>
  </select>

  <button type="submit">Add Task</button>
</form>

<table>
  <thead>
    <tr><th>Title</th><th>Priority</th><th>Actions</th></tr>
  </thead>
  <tbody>
    {{range .tasks}}
    <tr>
      <td>{{.title}}</td>
      <td>{{.priority}}</td>
      <td>
        <button lvt-click="delete" data-id="{{.id}}">Delete</button>
      </td>
    </tr>
    {{end}}
  </tbody>
</table>
```

**Lines:** 25
**Knowledge required:** HTML, Go templates, lvt attributes

### After (Markdown-Native)

```yaml
---
forms:
  add_task:
    fields:
      - name: title, type: text, required: true
      - name: priority, type: select, options: [Low, Medium, High], default: Medium
    submit: Add Task
    target: tasks
---
```

```markdown
## Add Task
<!-- lvt-form="add_task" -->

## Tasks
<!-- lvt-source="tasks" lvt-row-actions="delete" -->
| Title | Priority | |
|-------|----------|-|
```

**Lines:** 14
**Knowledge required:** YAML (simple), markdown tables

---

## 9. What We Gain

1. **Lower barrier to entry** - No HTML knowledge needed
2. **LLM-friendly** - Structured YAML is easier to generate correctly
3. **Maintainable** - Users can understand and modify
4. **Portable** - Markdown renders reasonably in any viewer
5. **Validated** - YAML schema can catch errors early

## 10. What We Lose

1. **Flexibility** - Can't do arbitrary layouts
2. **Styling** - Limited customization
3. **Complex interactions** - Multi-step forms, wizards

**Mitigation:** Keep Layer 3 (full HTML/templates) available. Markdown-native is the 80% solution; escape hatch exists for 20%.

---

## 11. Trade-off: Table Columns Must Match Field Names

A design decision: should table columns auto-map to source fields by name?

### Option A: Strict Name Matching
```markdown
<!-- lvt-source="tasks" -->
| title | priority | due_date |
|-------|----------|----------|
```

Column headers must exactly match field names. Simple but inflexible.

### Option B: Display Names with Mapping
```yaml
sources:
  tasks:
    columns:
      title: Title
      priority: Priority Level
      due_date: Due Date
```

```markdown
<!-- lvt-source="tasks" -->
| Title | Priority Level | Due Date |
|-------|----------------|----------|
```

### Option C: Inline Mapping
```markdown
<!-- lvt-source="tasks" lvt-columns="title:Title,priority:Priority,due_date:Due" -->
| Title | Priority | Due |
|-------|----------|-----|
```

**Recommendation:** Option A for simplicity, Option B available in frontmatter for customization.

---

## 12. Open Questions

1. **Nested data** - How to display nested objects in tables?
2. **Joins** - Can tables reference multiple sources?
3. **Pagination** - How to handle large datasets?
4. **Validation** - How to specify field validation rules?
5. **Computed columns** - Can tables have calculated fields?

---

## 13. Next Steps

1. Prototype auto-populating tables with HTML comment directives
2. Implement form frontmatter schema
3. Test with real micro app examples
4. Gather user feedback on syntax preferences
5. Iterate on edge cases

---

## 14. Appendix: Example Micro Apps in Proposed Syntax

### A. Simple Task Tracker

```yaml
---
title: My Tasks
sources:
  tasks:
    type: markdown
    anchor: "#tasks-data"
    auto_fields:
      id: "{{uuid}}"
      created: "{{now:2006-01-02}}"
forms:
  add:
    fields:
      - name: title, type: text, required: true
      - name: priority, type: select, options: [Low, Medium, High]
      - name: due, type: date
    submit: Add Task
    target: tasks
actions:
  done:
    update: {status: done}
  delete:
    confirm: "Delete this task?"
    remove: true
---

# My Tasks

## Add Task
<!-- lvt-form="add" -->

## Active Tasks
<!-- lvt-source="tasks" lvt-filter="status!=done" lvt-sort="due:asc" lvt-row-actions="done,delete" -->
| title | priority | due | | |
|-------|----------|-----|-|-|

## Completed
<!-- lvt-source="tasks" lvt-filter="status=done" lvt-row-actions="delete" -->
| title | priority | |
|-------|----------|-|

---
#tasks-data
```

### B. Incident Runbook

```yaml
---
title: Database Incident Response
sources:
  steps:
    type: markdown
    anchor: "#steps-data"
    auto_fields:
      completed_at: ""
      completed_by: ""
forms:
  complete_step:
    fields:
      - name: notes, type: textarea, label: "Notes (optional)"
    submit: Mark Complete
    auto_fields:
      completed_at: "{{now:2006-01-02 15:04}}"
      completed_by: "{{operator}}"
    target: steps
---

# Database Incident Response

**Operator:** {{operator}}
**Started:** {{now:2006-01-02 15:04}}

## Checklist
<!-- lvt-source="steps" lvt-row-actions="complete_step" -->
| step | completed_by | completed_at | |
|------|--------------|--------------|--|

---
#steps-data
| step |
|------|
| Check database connectivity |
| Review error logs |
| Identify affected queries |
| Implement fix |
| Verify resolution |
| Update status page |
```

### C. Weekly Metrics Bot

```yaml
---
title: Weekly Metrics
triggers:
  - schedule: "0 9 * * 1"  # Monday 9am
    action: generate_report
sources:
  sales:
    type: rest
    url: ${API_URL}/sales/weekly
outputs:
  slack:
    channel: "#metrics"
    token: ${SLACK_TOKEN}
actions:
  generate_report:
    output: slack
    template: |
      *Weekly Sales Report*
      Total: ${{sales.total}}
      Orders: {{sales.count}}
      Top product: {{sales.top_product}}
---

# Weekly Metrics Report

## Sales Summary
<!-- lvt-source="sales" -->
| Metric | Value |
|--------|-------|
| total | |
| count | |
| top_product | |

[Regenerate Now](action:generate_report)
```

---

## 15. Conclusion

**The path forward:** A layered approach where:

1. **80% of apps** use markdown-native syntax (tables + frontmatter)
2. **20% escape** to HTML/templates when needed
3. **LLMs generate** the appropriate layer based on complexity

This keeps tinkerdown's core library focused (parsing markdown extensions, not new template languages) while dramatically lowering the barrier for simple apps.

The key innovations:
- **Auto-populating tables** via HTML comment directives
- **Form schemas** in frontmatter (not inline DSL)
- **Action links** using markdown link syntax
- **Row actions** via table column convention

This approach extends markdown naturally rather than inventing new syntax.
