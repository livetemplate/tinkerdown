---
name: tinkerdown
description: Build single-file web apps in markdown. No React. No build step. Just run.
triggers:
  - tinkerdown
  - single-file app
  - markdown app
  - no-build app
  - internal tool
  - admin dashboard
  - one-file app
---

# Tinkerdown: One-File AI App Builder

Build working web apps in a single markdown file. No React. No build step. Just run.

## When to Use Tinkerdown

Use Tinkerdown for:
- **Internal tools** - Admin dashboards, data viewers, CRUD apps
- **Prototypes** - Quick interactive demos
- **Personal utilities** - Task managers, trackers, simple apps
- **Data displays** - Tables, forms, dashboards connected to databases/APIs

**Don't use Tinkerdown for:**
- Public-facing marketing sites (use static site generators)
- Apps requiring complex client-side state (use React/Vue)
- Real-time multiplayer games

## Quick Start

### 1. Create a markdown file

Create `myapp.md`:

```markdown
---
title: "My App"
---

# My App

\`\`\`lvt
<div>
    <h2>Add Item</h2>
    <form name="save" lvt-persist="items">
        <input type="text" name="title" required>
        <button type="submit">Add</button>
    </form>

    <h2>Items</h2>
    {{if .Items}}
    <ul>
        {{range .Items}}
        <li>
            {{.Title}}
            <button name="Delete" data-id="{{.Id}}">Delete</button>
        </li>
        {{end}}
    </ul>
    {{else}}
    <p>No items yet.</p>
    {{end}}
</div>
\`\`\`
```

### 2. Run it

```bash
tinkerdown serve myapp.md
```

### 3. Open in browser

Navigate to `http://localhost:3000` - your app is running!

## Key Concepts

| Concept | What It Does |
|---------|--------------|
| `lvt-persist` | Auto-saves form data to SQLite. Creates table, generates CRUD. |
| `lvt-source` | Connects to external data (PostgreSQL, REST API, CSV, JSON, scripts) |
| `name` (on button/form) | Triggers server action on click (button) or form submission (form) |
| `data-*` | Passes data with actions (e.g., `data-id="123"`) |
| frontmatter sources | Define data sources in frontmatter - no `tinkerdown.yaml` needed! |

## Reference

See [reference.md](./reference.md) for complete API documentation:
- File structure and frontmatter
- All `lvt-*` attributes
- Source configuration (pg, rest, csv, json, exec)
- Template syntax (Go templates)
- Components (datatable, dropdown)

## Examples

See [examples/](./examples/) for complete working apps:
1. [Todo App](./examples/01-todo-app.md) - Basic CRUD with `lvt-persist`
2. [Dashboard](./examples/02-dashboard.md) - Data display with `lvt-source`
3. [Contact Form](./examples/03-contact-form.md) - Form handling
4. [Blog](./examples/04-blog.md) - Multi-page with partials
5. [Inventory](./examples/05-inventory.md) - PostgreSQL integration
6. [Survey](./examples/06-survey.md) - Multi-step forms
7. [Booking](./examples/07-booking.md) - Date/time handling
8. [Expense Tracker](./examples/08-expense-tracker.md) - CSV import
9. [FAQ](./examples/09-faq.md) - Accordion component
10. [Status Page](./examples/10-status-page.md) - Real-time updates
