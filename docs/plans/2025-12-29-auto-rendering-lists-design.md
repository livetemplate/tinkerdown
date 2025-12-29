# Auto-Rendering Lists Design

**Date:** 2025-12-29
**Status:** Approved
**Task:** ROADMAP 1.2

## Overview

Add auto-rendering support for `<ul>` and `<ol>` elements with `lvt-source`, following the same pattern as tables and selects.

## Attributes

| Attribute | Required | Default | Description |
|-----------|----------|---------|-------------|
| `lvt-source` | Yes | - | Data source name |
| `lvt-field` | No | `{{.}}` | Field to display for object arrays |
| `lvt-actions` | No | - | Action buttons: `action:Label,action2:Label2` |
| `lvt-empty` | No | - | Message when data is empty |

## Examples

### String Array
```html
<ul lvt-source="tags">
</ul>
```
With `tags.json`: `["alpha", "beta", "gamma"]`

Renders:
```html
<ul>
  <li>alpha</li>
  <li>beta</li>
  <li>gamma</li>
</ul>
```

### Object Array with Field
```html
<ul lvt-source="users" lvt-field="name">
</ul>
```
Renders:
```html
<ul>
  <li>Alice</li>
  <li>Bob</li>
</ul>
```

### With Actions
```html
<ul lvt-source="tasks" lvt-field="title" lvt-actions="delete:×,edit:Edit">
</ul>
```
Renders:
```html
<ul>
  {{range .Data}}
  <li>
    {{.Title}}
    <button lvt-click="delete" lvt-data-id="{{.Id}}">×</button>
    <button lvt-click="edit" lvt-data-id="{{.Id}}">Edit</button>
  </li>
  {{end}}
</ul>
```

### Empty State
```html
<ol lvt-source="empty_source" lvt-empty="No items yet">
</ol>
```
Renders when empty:
```html
<ol>
  <li>No items yet</li>
</ol>
```

## Implementation

### Function: `autoGenerateListTemplate()`

Location: `page.go`

```go
func autoGenerateListTemplate(content string) string {
    // Regex matches <ul> or <ol> with lvt-source and empty content
    // Parses: lvt-field, lvt-actions, lvt-empty
    // Generates template with {{range .Data}}
}
```

### Processing Pipeline

Add to line ~183 in page.go:
```go
processedContent := autoGenerateTableTemplate(cb.Content)
processedContent = autoGenerateSelectTemplate(processedContent)
processedContent = autoGenerateListTemplate(processedContent)  // NEW
```

### Field Handling

- `lvt-field="name"` → `{{.Name}}` (titlecased)
- No field specified → `{{.}}` (works for string arrays)

## Files to Modify

1. `page.go` - Add `autoGenerateListTemplate()` function
2. `component_library_e2e_test.go` - Add list rendering tests
3. `examples/component-library-test/index.md` - Add list examples
4. `examples/component-library-test/tags.json` - New test data
5. `examples/component-library-test/tasks.json` - New test data
6. `docs/auto-rendering.md` - Document the feature

## Tests

1. `TestAutoListBasic` - string array renders as list items
2. `TestAutoListWithField` - object array with field extraction
3. `TestAutoListWithActions` - action buttons appended
4. `TestAutoListEmptyState` - empty message displayed
5. `TestAutoListOrderedList` - works with `<ol>` too
