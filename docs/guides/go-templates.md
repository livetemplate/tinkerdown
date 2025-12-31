# Go Templates

Tinkerdown uses Go's `html/template` package for dynamic content rendering.

## Basic Syntax

### Variables

```html
<!-- Display a variable -->
<p>Hello, {{.name}}!</p>

<!-- Access nested fields -->
<p>User: {{.user.email}}</p>
```

### Conditionals

```html
{{if .isLoggedIn}}
  <p>Welcome back!</p>
{{else}}
  <p>Please log in</p>
{{end}}

<!-- Comparison -->
{{if eq .status "active"}}
  <span class="badge-active">Active</span>
{{end}}
```

### Loops

```html
{{range .tasks}}
  <div class="task">
    <h3>{{.title}}</h3>
    <p>{{.description}}</p>
  </div>
{{end}}

<!-- With index -->
{{range $index, $task := .tasks}}
  <div>{{$index}}: {{$task.title}}</div>
{{end}}
```

### With Block

```html
{{with .user}}
  <p>Name: {{.name}}</p>
  <p>Email: {{.email}}</p>
{{end}}
```

## Template Functions

### Built-in Functions

| Function | Description | Example |
|----------|-------------|---------|
| `eq` | Equal | `{{if eq .a .b}}` |
| `ne` | Not equal | `{{if ne .status "done"}}` |
| `lt`, `le` | Less than | `{{if lt .count 10}}` |
| `gt`, `ge` | Greater than | `{{if gt .price 100}}` |
| `and` | Logical AND | `{{if and .a .b}}` |
| `or` | Logical OR | `{{if or .a .b}}` |
| `not` | Logical NOT | `{{if not .done}}` |
| `len` | Length | `{{len .items}}` |
| `index` | Array access | `{{index .items 0}}` |

### String Functions

```html
<!-- Print formatted -->
{{printf "Count: %d" .count}}

<!-- HTML output (use sparingly) -->
{{.rawHtml | safeHTML}}
```

## When to Use Go Templates vs Auto-Rendering

### Use Auto-Rendering For:

- Simple data tables: `<table lvt-source="tasks" lvt-columns="...">`
- Simple lists: `<ul lvt-source="items" lvt-field="name">`
- Select dropdowns: `<select lvt-source="options" ...>`

### Use Go Templates For:

- Custom layouts
- Conditional rendering
- Nested data structures
- Complex formatting

```html
<!-- Custom card layout - use Go templates -->
{{range .tasks}}
<div class="card {{if .done}}completed{{end}}">
  <h3>{{.title}}</h3>
  <span class="priority priority-{{.priority}}">{{.priority}}</span>
  <button lvt-click="Delete" lvt-data-id="{{.id}}">Delete</button>
</div>
{{end}}
```

## Combining with lvt-* Attributes

Go templates work seamlessly with `lvt-*` attributes:

```html
{{range .tasks}}
<div class="task">
  <span>{{.title}}</span>
  <button lvt-click="Toggle" lvt-data-id="{{.id}}">
    {{if .done}}Undo{{else}}Done{{end}}
  </button>
</div>
{{end}}
```

## Next Steps

- [Auto-Rendering](auto-rendering.md) - When templates aren't needed
- [lvt-* Attributes Reference](../reference/lvt-attributes.md) - Interactive elements
