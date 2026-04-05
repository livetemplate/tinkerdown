# Self-Contained Markdown Apps Design

**Date:** 2024-12-14
**Status:** Brainstorm / Future Reference

## Overview

A Tinkerdown markdown file can store its own data in native markdown formats within the document body. The file becomes both the application AND its database. Targeted at personal micro-tools (todo lists, bookmarks, habit trackers) with < 100 items.

## Design Decisions

| Decision | Choice |
|----------|--------|
| Data formats | Native markdown: task lists, bullet lists, tables |
| Binding syntax | `lvt-source="name"` with `type: markdown` in sources config |
| Format detection | Auto-detect with optional explicit declaration: `{#todos format=table}` |
| Persistence | Debounced write (~1 second) + immediate flush on page close via `sendBeacon()` |
| UI sync status | Visual indicator showing "Saving..." / "Saved" / "Conflict" states |
| Insert position | Developer controls via `lvt-insert="prepend\|append"`, default append |
| Item identification | Auto-generated hidden IDs via HTML comments: `<!-- id:abc123 -->` |
| External edits | File watcher detects changes, auto-reload or create conflict copy |
| Frontmatter | Sources config + app settings, not user data |
| Concurrency | Optimistic concurrency via mtime check, conflict copy strategy |
| Cross-file references | Source config with `file:` field, `readonly: false` for writes |

## Security Model

**CSRF Protection:**
- Server validates `Origin` and `Referer` headers on all mutating requests (POST, PUT, DELETE)
- Only accepts requests from `localhost` or the specific served domain
- Rejects requests with missing or mismatched origin headers
- Prevents malicious websites from sending requests to the local server

**Input Sanitization:**
- All user data is HTML-escaped on render (Go templates do this by default)
- Markdown-special characters (`|`, `-`, `[`, `]`, `#`) are escaped on write
- Anchor names validated: alphanumeric, hyphens, underscores only

**Content Security Policy:**
- Default: `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'`
- Applied automatically, no configuration needed
- Override only if external resources required (CDN scripts, fonts, APIs)

**Validation:**
- URL fields validated with `type="url"` and server-side check
- Maximum field lengths enforced
- No arbitrary script execution from stored data

## Data Model

**Frontmatter (YAML)** - Sources config + app settings:
```yaml
---
title: My App
pageSize: 20
sources:
  # Local anchor in same file (writable)
  todos:
    type: markdown
    anchor: "#todos"
    readonly: false

  # External file reference (writable)
  tasks:
    type: markdown
    file: "./tasks.md"
    anchor: "#active"
    readonly: false

  # Read-only lookup table
  categories:
    type: markdown
    file: "./data/categories.md"
    anchor: "#list"
    # readonly: true is the default
---
```

**Body sections with anchors** - User data:

| Markdown Format | Parsed As |
|-----------------|-----------|
| `- [ ] text <!-- id:xxx -->` | `[{id: "xxx", text: "...", done: bool}]` |
| `- item <!-- id:xxx -->` | `[{id: "xxx", text: "..."}]` |
| `\| col \| col \| <!-- id:xxx -->` | `[{id: "xxx", col1: "...", col2: "..."}]` |

**ID Generation:**
- IDs auto-generated on create: 8-character alphanumeric (e.g., `a1b2c3d4`)
- Optional prefix for readability: `bm_` for bookmarks, `todo_` for tasks
- Stored as trailing HTML comment (invisible in most markdown renderers)
- Used for all CRUD operations instead of content matching

## Markdown Source Type

The `markdown` source type extends the existing `lvt-source` architecture (alongside `exec`, `pg`, `rest`, `json`, `csv`).

**Configuration:**
```yaml
sources:
  todos:
    type: markdown
    anchor: "#todos"           # Required: section anchor in file
    readonly: false            # Optional: enable writes (default: true = read-only)

  external:
    type: markdown
    file: "./data.md"          # Optional: external file (default: current file)
    anchor: "#items"
```

**Supported Formats:**

| Markdown Format | Parsed Fields |
|-----------------|---------------|
| `- [ ] text` | `{id, text, done}` |
| `- item` | `{id, text}` |
| `\| col1 \| col2 \|` | `{id, col1, col2, ...}` |

**Comparison with Other Source Types:**

| Source Type | Data Location | Writeable | Use Case |
|-------------|---------------|-----------|----------|
| `markdown` | Same/external .md file | Yes | Self-contained apps |
| `json` | Local .json file | No | Static data |
| `csv` | Local .csv file | No | Imported data |
| `exec` | Command output | No | Dynamic/computed |
| `rest` | HTTP API | No | Remote data |
| `pg` | PostgreSQL | No | Database queries |

## Binding Syntax

```html
<div lvt-source="todos">
  {{range .Data}}
    <div data-id="{{.id}}">{{.text}}</div>
  {{end}}
</div>
```

**Filtering and Sorting:**
```html
<input lvt-filter="todos" placeholder="Search...">
<table lvt-source="todos" lvt-sort="name" lvt-sort-order="asc">
  ...
</table>
```

## Cross-File Data References

Reference data sections from other markdown files in the same project via source configuration.

**Source Configuration:**
```yaml
sources:
  # Local anchor (same file)
  todos:
    type: markdown
    anchor: "#todos"

  # External file
  contacts:
    type: markdown
    file: "./contacts.md"
    anchor: "#people"

  # Subdirectory
  categories:
    type: markdown
    file: "./data/categories.md"
    anchor: "#list"

  # Writable external source
  tasks:
    type: markdown
    file: "./tasks.md"
    anchor: "#active"
    readonly: false
```

**Access Modes:**

| Mode | Config | Behavior |
|------|--------|----------|
| Read-only | `readonly: true` (default) | Data displayed but not editable. |
| Read-write | `readonly: false` | Enables CRUD operations on source. |

**Use Cases:**
- Dashboard aggregating data from multiple files
- Shared lookup tables (categories, tags, contacts)
- Separating data files from presentation files

**Example - Dashboard with external data:**
```markdown
---
title: Project Dashboard
sources:
  tasks:
    type: markdown
    file: "./tasks.md"
    anchor: "#active"
    readonly: false
  team:
    type: markdown
    file: "./team.md"
    anchor: "#members"
---

# Projects Overview

## Active Tasks (from tasks.md)

<!-- Form to add tasks to external file -->
<form lvt-action="add" lvt-source="tasks">
  <input name="text" placeholder="New task" required>
  <button type="submit">Add Task</button>
</form>

<!-- Table displaying and allowing edits to external file -->
<table lvt-source="tasks" lvt-sort="status">
  {{range .Data}}
  <tr data-id="{{.id}}">
    <td>{{.text}}</td>
    <td>{{.status}}</td>
    <td><button name="delete" lvt-source="tasks" data-id="{{.id}}">×</button></td>
  </tr>
  {{end}}
</table>

## Team (read-only from shared file)
<ul lvt-source="team">
  {{range .Data}}<li>{{.name}} - {{.role}}</li>{{end}}
</ul>
```

**Security Constraints:**
- Paths must resolve within the served directory (no `../` escaping out)
- Only `.md` files can be referenced for markdown sources
- Path traversal attempts → validation error on page load
- Paths canonicalized before validation to prevent `..` bypasses

**Error Handling:**
- External file missing → error logged, empty data rendered
- External file exists but anchor missing → error logged, empty data rendered
- External file unreadable → error shown, empty data rendered

**File Watching:**
- All referenced files watched for changes (both dev and production modes)
- External file changes trigger targeted WebSocket push to refresh affected `lvt-source` bindings
- Only the changed data section is re-rendered, not full page reload
- Circular references detected and prevented at parse time
- In `--dev` mode: additional hot reload for template/code changes

**Write Operations to External Files:**
- Require `readonly: false` in source config
- Same optimistic concurrency rules apply to external files (mtime check before write)
- Same ID-based matching for CRUD operations
- Undo/redo scoped per-file (each file has its own undo stack)
- `lvt-filter` and `lvt-sort` work normally with external data

## CRUD Operations

| Action | UI Trigger | Markdown Result |
|--------|------------|-----------------|
| Create | Form submit / button | New item with generated ID appended/prepended |
| Read | Page load | Section parsed, data rendered |
| Update | Checkbox toggle, inline edit | Item matched by ID, modified in place |
| Delete | Delete button with `data-id` | Item matched by ID, removed |

**Built-in action handlers for markdown sources:**
- `add` - Creates new item from form data, generates ID
- `toggle` - Toggles `done` field on item matched by ID
- `delete` - Removes item matched by ID
- `update` - Updates fields on item matched by ID

**Usage with lvt-source:**
```html
<form lvt-action="add" lvt-source="todos">
  <input name="text" required>
  <button type="submit">Add</button>
</form>

<button name="delete" lvt-source="todos" data-id="{{.id}}">×</button>
```

## Edge Cases & Error Handling

**Missing/Invalid Anchors:**
- Source references anchor but no section exists → Error logged, empty data
- Section without anchor syntax → Ignored (must have explicit `{#anchor}`)
- Invalid anchor name (special chars) → Validation error on page load

**Data Integrity:**
- Delete item but ID not found → No-op, warn in console
- Duplicate IDs (corruption) → Auto-repair: regenerate ID for second occurrence, warn in `--dev`
- ID collision on create → Regenerate with different ID
- Missing IDs on data items → Auto-assign IDs on file load (items matching data pattern)
- ID syntax broken (partial comment) → Attempt repair or strip and regenerate
- Integrity check command: `livemdtools check <file>` to validate and report issues

**File Operations:**
- File unwritable → Error shown, changes stay in memory, retry offered
- File mtime changed during write → Create conflict copy, notify user
- Page close with unsaved changes → `sendBeacon()` flush + `beforeunload` warning
- External file modification detected → If no local changes: auto-reload. If local changes pending: save to conflict copy, then reload

**Concurrent Access (Optimistic Concurrency Control):**
- Read file mtime before each write operation
- If mtime changed since last read → abort write, create conflict copy (`file.conflict-{timestamp}.md`)
- Same file opened in multiple tabs → Warn on second tab, coordinate via localStorage
- Same external file referenced by multiple pages → mtime check prevents silent overwrites
- Multi-device via file sync → Conflict copies preserve both versions (like Dropbox/Syncthing)
- No reliance on OS-level file locks (unreliable on network shares and sync services)

**Format Edge Cases:**
- Mixed formats in one section → First format wins, warn in `--dev`
- Size limit (>100 items) → Soft warning, suggest external database source (pg, json)
- Special characters in data → Auto-escaped on write, unescaped on read

**Undo/Redo:**
- Last 50 operations stored in memory
- `Ctrl+Z` / `Ctrl+Y` keybindings
- Operations logged: `{type: "delete", id: "xxx", data: {...}, timestamp: ...}`

## Complete Example

```markdown
---
title: My Bookmarks
pageSize: 20
sources:
  bookmarks:
    type: markdown
    anchor: "#bookmarks"
    readonly: false
---

## Bookmarks {#bookmarks}
| name       | url                      | tags       |
|------------|--------------------------|------------|
| GitHub     | https://github.com       | dev, code  | <!-- id:bm_a1b2c3 -->
| Hacker News| https://news.ycombinator.com | news    | <!-- id:bm_d4e5f6 -->

# {{.Config.title}}

<form lvt-action="add" lvt-source="bookmarks">
  <input name="name" placeholder="Name" required maxlength="100">
  <input name="url" placeholder="URL" required type="url">
  <input name="tags" placeholder="Tags (comma-separated)">
  <button type="submit">Add Bookmark</button>
</form>

<input lvt-filter="bookmarks" placeholder="Search bookmarks...">

<table lvt-source="bookmarks" lvt-sort="name">
  <thead>
    <tr><th>Name</th><th>URL</th><th>Tags</th><th></th></tr>
  </thead>
  <tbody>
  {{range .Data}}
    <tr data-id="{{.id}}">
      <td>{{.name}}</td>
      <td><a href="{{.url}}">{{.url}}</a></td>
      <td>{{.tags}}</td>
      <td>
        <button name="delete" lvt-source="bookmarks" data-id="{{.id}}"
                data-confirm="Delete {{.name}}?">×</button>
      </td>
    </tr>
  {{end}}
  </tbody>
</table>
```

**Run it:**
```bash
tinkerdown serve bookmarks.md
```

## Key Properties

- Single file = app + data (or multi-file with cross-references)
- Editable in any markdown editor (Obsidian, VS Code, GitHub)
- Git-friendly (text diffs, hidden IDs don't clutter display)
- No external database
- Works offline
- Undo/redo support
- Safe concurrent access with optimistic concurrency control
- Reliable saves even on abrupt page close
- Cross-file data references for dashboards and shared data

## Sync Recommendations

For multi-device usage:
- **Git:** Commit after each session, pull before opening
- **Syncthing/Dropbox:** Conflict copies created automatically if simultaneous edits detected
- **Obsidian Sync:** Works natively with markdown files

## Limitations

- Maximum ~100 items per section (soft limit, performance degrades beyond)
- No multi-line content in table cells
- No nested data structures in markdown tables (use JSON code blocks for complex data)
- Single-user optimized (no real-time collaboration)
- Cross-file writes require `readonly: false` in source config (read-only by default)
- Maximum 10 markdown source references per page (performance)
- Circular file references not allowed
- Markdown source type supports: task lists, bullet lists, tables

---

## Implementation Plan

### Phase 1: Core Markdown Source (Read-Only)

**Goal:** Parse markdown sections and return data via `lvt-source`

**Files to Create:**
| File | Purpose |
|------|---------|
| `internal/source/markdown.go` | `MarkdownSource` implementing `Source` interface |
| `internal/source/markdown_test.go` | Unit tests for parsing logic |

**Files to Modify:**
| File | Change |
|------|--------|
| `internal/source/source.go` | Add `"markdown"` case to `createSource()` factory |
| `internal/compiler/lvtsource.go` | Add `generateMarkdownSourceCode()` function |
| `internal/config/config.go` | Add `Anchor` field to `SourceConfig` |

**Source Implementation:**
```go
type MarkdownSource struct {
    name     string
    filePath string  // empty = current file
    anchor   string  // e.g., "#todos"
    readonly bool
    format   string  // auto-detected: "tasklist", "bulletlist", "table"
}

func (s *MarkdownSource) Fetch(ctx context.Context) ([]map[string]interface{}, error) {
    // 1. Read markdown file
    // 2. Find section by anchor
    // 3. Detect format (task list, bullet list, table)
    // 4. Parse items with IDs
    // 5. Return []map[string]interface{}
}
```

**Parsing Logic:**
```
## Todos {#todos}
- [ ] Buy milk <!-- id:a1b2c3 -->
- [x] Call mom <!-- id:d4e5f6 -->

→ [{id: "a1b2c3", text: "Buy milk", done: false},
   {id: "d4e5f6", text: "Call mom", done: true}]
```

### Phase 2: Write Operations (CRUD)

**Goal:** Modify markdown file on add/update/delete actions

**New Handlers:**
| Action | Behavior |
|--------|----------|
| `add` | Append/prepend item with generated ID |
| `delete` | Remove item by ID |
| `update` | Modify item fields by ID |
| `toggle` | Flip `done` boolean for task lists |

**Write Flow:**
1. Receive action via WebSocket
2. Read current file, check mtime
3. Parse section, apply change
4. Re-serialize section to markdown
5. Write file (or create conflict copy if mtime changed)
6. Broadcast update via WebSocket

**Files to Create:**
| File | Purpose |
|------|---------|
| `internal/source/markdown_writer.go` | Write operations for markdown sources |

### Phase 3: File Watching & Live Updates

**Goal:** Auto-refresh UI when external edits detected

**Implementation:**
- Use existing `fsnotify` watcher
- On file change → re-parse section → push via WebSocket
- Target specific `lvt-source` bindings (not full page reload)

**Files to Modify:**
| File | Change |
|------|--------|
| `internal/server/watcher.go` | Track markdown source files |
| `internal/server/websocket.go` | Add source-specific refresh message |

### Phase 4: Concurrency & Conflict Handling

**Goal:** Safe concurrent access with conflict detection

**Implementation:**
- Store mtime on read
- Check mtime before write
- If changed → create `file.conflict-{timestamp}.md`
- Notify user via UI sync status indicator

---

## E2E Test Plan

**Test File:** `lvtsource_markdown_e2e_test.go`

### Read Tests
| Test | Scenario |
|------|----------|
| `TestMarkdownSource_TaskList` | Parse `- [ ] item` format, verify data renders |
| `TestMarkdownSource_BulletList` | Parse `- item` format |
| `TestMarkdownSource_Table` | Parse `| col | col |` format |
| `TestMarkdownSource_ExternalFile` | Read from `file:` path |
| `TestMarkdownSource_MissingAnchor` | Verify empty data + error logged |

### Write Tests
| Test | Scenario |
|------|----------|
| `TestMarkdownSource_AddItem` | Submit form → verify item in markdown |
| `TestMarkdownSource_DeleteItem` | Click delete → verify item removed |
| `TestMarkdownSource_ToggleCheckbox` | Toggle → verify `done` field changed |
| `TestMarkdownSource_UpdateItem` | Inline edit → verify markdown updated |
| `TestMarkdownSource_IDGeneration` | Add item → verify ID comment appended |

### Concurrency Tests
| Test | Scenario |
|------|----------|
| `TestMarkdownSource_ExternalEdit` | Modify file externally → verify WebSocket refresh |
| `TestMarkdownSource_ConflictCopy` | Concurrent edit → verify conflict file created |
| `TestMarkdownSource_MtimeCheck` | Race condition → verify no data loss |

### Edge Case Tests
| Test | Scenario |
|------|----------|
| `TestMarkdownSource_DuplicateID` | Two items same ID → verify auto-repair |
| `TestMarkdownSource_MissingID` | Item without ID → verify ID assigned |
| `TestMarkdownSource_SpecialChars` | Pipe, brackets in data → verify escaped |

---

## Example Apps

### Example 1: Markdown Data Todo List

**Directory:** `examples/markdown-data-todo/`

```
markdown-data-todo/
├── tinkerdown.yaml
└── index.md
```

**index.md:**
```markdown
---
title: My Todos
sources:
  todos:
    type: markdown
    anchor: "#tasks"
    readonly: false
---

## Tasks {#tasks}
- [ ] Learn livemdtools <!-- id:todo_001 -->
- [x] Read the docs <!-- id:todo_002 -->

# {{.Config.title}}

<form lvt-action="add" lvt-source="todos">
  <input name="text" placeholder="New task..." required>
  <button type="submit">Add</button>
</form>

<ul lvt-source="todos">
  {{range .Data}}
  <li data-id="{{.id}}">
    <input type="checkbox" lvt-on:click="toggle" lvt-source="todos"
           data-id="{{.id}}" {{if .done}}checked{{end}}>
    {{.text}}
    <button name="delete" lvt-source="todos" data-id="{{.id}}">×</button>
  </li>
  {{end}}
</ul>
```

### Example 2: Markdown Data Bookmarks

**Directory:** `examples/markdown-data-bookmarks/`

```markdown
---
title: My Bookmarks
sources:
  bookmarks:
    type: markdown
    anchor: "#links"
    readonly: false
---

## Links {#links}
| name | url | tags |
|------|-----|------|
| GitHub | https://github.com | dev | <!-- id:bm_001 -->
| HN | https://news.ycombinator.com | news | <!-- id:bm_002 -->

# {{.Config.title}}

<form lvt-action="add" lvt-source="bookmarks">
  <input name="name" placeholder="Name" required>
  <input name="url" type="url" placeholder="URL" required>
  <input name="tags" placeholder="Tags">
  <button type="submit">Add</button>
</form>

<input lvt-filter="bookmarks" placeholder="Search...">

<table lvt-source="bookmarks" lvt-sort="name">
  <thead><tr><th>Name</th><th>URL</th><th>Tags</th><th></th></tr></thead>
  <tbody>
  {{range .Data}}
  <tr data-id="{{.id}}">
    <td>{{.name}}</td>
    <td><a href="{{.url}}">{{.url}}</a></td>
    <td>{{.tags}}</td>
    <td><button name="delete" lvt-source="bookmarks" data-id="{{.id}}">×</button></td>
  </tr>
  {{end}}
  </tbody>
</table>
```

### Example 3: Markdown Data Dashboard

**Directory:** `examples/markdown-data-dashboard/`

```
markdown-data-dashboard/
├── tinkerdown.yaml
├── index.md          # Dashboard
├── tasks.md          # Task data
└── contacts.md       # Contact data
```

**index.md:**
```markdown
---
title: Project Dashboard
sources:
  tasks:
    type: markdown
    file: "./tasks.md"
    anchor: "#active"
    readonly: false
  contacts:
    type: markdown
    file: "./contacts.md"
    anchor: "#team"
---

# {{.Config.title}}

## Active Tasks
<table lvt-source="tasks">
  {{range .Data}}
  <tr><td>{{.text}}</td><td>{{.status}}</td></tr>
  {{end}}
</table>

## Team Contacts (read-only)
<ul lvt-source="contacts">
  {{range .Data}}<li>{{.name}} - {{.email}}</li>{{end}}
</ul>
```

---

## Progress Tracker

| Phase | Status | Milestone |
|-------|--------|-----------|
| Phase 1 | ✅ | Read-only markdown source |
| Phase 2 | ✅ | Write operations (CRUD) |
| Phase 3 | ✅ | File watching & live updates |
| Phase 4 | ✅ | Concurrency & conflict handling |
| E2E Tests | ✅ | All test scenarios passing |
| Examples | ✅ | Todo, Bookmarks, Dashboard working |
