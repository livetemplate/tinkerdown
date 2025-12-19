# Self-Contained Markdown Apps Design

**Date:** 2024-12-14
**Status:** Brainstorm / Future Reference

## Overview

A LivePage markdown file can store its own data in native markdown formats within the document body. The file becomes both the application AND its database. Targeted at personal micro-tools (todo lists, bookmarks, habit trackers) with < 100 items.

## Design Decisions

| Decision | Choice |
|----------|--------|
| Data formats | Native markdown: task lists, bullet lists, tables |
| Binding syntax | Markdown anchors: `lvt-data="#todos"` |
| Format detection | Auto-detect based on content |
| Persistence | Debounced write (~1 second) |
| Insert position | Developer controls via `lvt-insert="prepend\|append"`, default append |
| Item identification | Content-based matching, optional explicit IDs |
| External edits | Hot reload in `--dev` mode only |
| Frontmatter | Config only, not user data |

## Data Model

**Frontmatter (YAML)** - App configuration only:
```yaml
---
theme: light
pageSize: 20
---
```

**Body sections with anchors** - User data:

| Markdown Format | Parsed As |
|-----------------|-----------|
| `- [ ] text` / `- [x] text` | `[{text: "...", done: bool}]` |
| `- item` | `[{text: "..."}]` |
| `\| col \| col \|` tables | `[{col1: "...", col2: "..."}]` |

## Binding Syntax

```html
<div lvt-data="#todos">
  {{range .}}...{{end}}
</div>
```

## CRUD Operations

| Action | UI Trigger | Markdown Result |
|--------|------------|-----------------|
| Create | Form submit / button | New item appended/prepended |
| Read | Page load | Section parsed, data rendered |
| Update | Checkbox toggle, inline edit | Item modified in place |
| Delete | Delete button | Item removed from list/table |

**Built-in action handlers for `lvt-data`:**
- `add` - Creates new item from form data
- `toggle` - Toggles `done` field on matched item
- `delete` - Removes matched item
- `update` - Updates fields on matched item

## Edge Cases & Error Handling

**Missing/Invalid Anchors:**
- `lvt-data="#foo"` but no section exists → Error logged, empty data
- Section without anchor syntax → Ignored (must have explicit `{#anchor}`)

**Data Integrity:**
- Delete item but content not found → No-op, warn in console
- Duplicate content without explicit IDs → First match wins, warn in `--dev`
- Update changes text (identifier) → Treat as delete old + create new

**File Operations:**
- File unwritable → Error shown, changes stay in memory
- Concurrent write conflict → Last write wins, log warning
- Page close with unsaved changes → beforeunload warning

**Format Edge Cases:**
- Mixed formats in one section → First format wins
- Size limit (>100 items) → Soft warning, suggest `lvt-persist`

## Complete Example

```markdown
---
title: My Bookmarks
pageSize: 20
---

## Bookmarks {#bookmarks}
| name       | url                      | tags       |
|------------|--------------------------|------------|
| GitHub     | https://github.com       | dev, code  |
| Hacker News| https://news.ycombinator.com | news    |

# {{.Config.title}}

<form lvt-action="add" lvt-data="#bookmarks">
  <input name="name" placeholder="Name" required>
  <input name="url" placeholder="URL" required>
  <input name="tags" placeholder="Tags (comma-separated)">
  <button type="submit">Add Bookmark</button>
</form>

<table lvt-data="#bookmarks">
  <thead>
    <tr><th>Name</th><th>URL</th><th>Tags</th><th></th></tr>
  </thead>
  <tbody>
  {{range .}}
    <tr>
      <td>{{.name}}</td>
      <td><a href="{{.url}}">{{.url}}</a></td>
      <td>{{.tags}}</td>
      <td><button lvt-click="delete" lvt-data-name="{{.name}}">×</button></td>
    </tr>
  {{end}}
  </tbody>
</table>
```

**Run it:**
```bash
livepage serve bookmarks.md
```

## Key Properties

- Single file = app + data
- Editable in any markdown editor (Obsidian, VS Code, GitHub)
- Git-friendly (text diffs)
- No external database
- Works offline
