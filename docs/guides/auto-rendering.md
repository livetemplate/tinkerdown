# Auto-Rendering Components

Tinkerdown can automatically generate HTML for common UI patterns from data sources. This eliminates boilerplate template code for tables and select dropdowns.

## Select Dropdowns

Transform an empty `<select>` element into a fully populated dropdown by connecting it to a data source.

### Basic Usage

```html
<select lvt-source="countries" lvt-value="code" lvt-label="name">
</select>
```

With a `countries.json` data file:
```json
[
  {"code": "US", "name": "United States"},
  {"code": "GB", "name": "United Kingdom"},
  {"code": "DE", "name": "Germany"}
]
```

This renders as:
```html
<select>
  <option value="US">United States</option>
  <option value="GB">United Kingdom</option>
  <option value="DE">Germany</option>
</select>
```

### Attributes

| Attribute | Required | Default | Description |
|-----------|----------|---------|-------------|
| `lvt-source` | Yes | - | Name of the data source (JSON file, REST endpoint, etc.) |
| `lvt-value` | No | `id` | Field name to use for the option's `value` attribute |
| `lvt-label` | No | `name` | Field name to use for the option's display text |

### Default Fields

If you don't specify `lvt-value` and `lvt-label`, Tinkerdown uses sensible defaults:

```html
<!-- Uses "id" for value and "name" for label -->
<select lvt-source="items">
</select>
```

Your data should have `id` and `name` fields:
```json
[
  {"id": 1, "name": "Option One"},
  {"id": 2, "name": "Option Two"}
]
```

### Case Insensitivity

Field names are case-insensitive. All of these work:
- `lvt-value="code"` matches `code`, `Code`, or `CODE` in your data
- `lvt-label="name"` matches `name`, `Name`, or `NAME` in your data

### Preserving Existing Content

If your `<select>` already has content (options or template code), Tinkerdown won't override it:

```html
<!-- This is preserved as-is -->
<select lvt-source="countries">
  <option value="">Select a country...</option>
  {{range .Data}}
  <option value="{{.Code}}">{{.Name}}</option>
  {{end}}
</select>
```

### Other Attributes

Standard HTML attributes are preserved:

```html
<select lvt-source="countries" lvt-value="code" lvt-label="name"
        class="form-select" id="country-picker" name="country" required>
</select>
```

Renders as:
```html
<select class="form-select" id="country-picker" name="country" required>
  <option value="US">United States</option>
  ...
</select>
```

---

## Tables

Transform an empty `<table>` element into a fully populated table with headers and rows.

### Basic Usage

```html
<table lvt-source="users" lvt-columns="name,email,role">
</table>
```

With a `users.json` data file:
```json
[
  {"name": "Alice", "email": "alice@example.com", "role": "Admin"},
  {"name": "Bob", "email": "bob@example.com", "role": "User"}
]
```

This renders as:
```html
<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Email</th>
      <th>Role</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Alice</td>
      <td>alice@example.com</td>
      <td>Admin</td>
    </tr>
    <tr>
      <td>Bob</td>
      <td>bob@example.com</td>
      <td>User</td>
    </tr>
  </tbody>
</table>
```

### Attributes

| Attribute | Required | Default | Description |
|-----------|----------|---------|-------------|
| `lvt-source` | Yes | - | Name of the data source |
| `lvt-columns` | No | Auto-discover | Comma-separated list of column names |
| `lvt-actions` | No | - | Action buttons: `action:Label,action2:Label2` |
| `lvt-empty` | No | `"No data"` | Message to show when data is empty |
| `lvt-datatable` | No | - | Enable rich datatable mode with sorting |

### Auto-Discover Columns

Without `lvt-columns`, Tinkerdown discovers columns from the first data row:

```html
<!-- Columns auto-discovered from data -->
<table lvt-source="users">
</table>
```

### Action Buttons

Add action buttons to each row:

```html
<table lvt-source="users" lvt-columns="name,email" lvt-actions="edit:Edit,delete:Delete">
</table>
```

Renders an "Actions" column with buttons:
```html
<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Email</th>
      <th>Actions</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Alice</td>
      <td>alice@example.com</td>
      <td>
        <button lvt-click="edit" lvt-data-id="...">Edit</button>
        <button lvt-click="delete" lvt-data-id="...">Delete</button>
      </td>
    </tr>
    ...
  </tbody>
</table>
```

### Empty State

Customize the message shown when data is empty:

```html
<table lvt-source="users" lvt-columns="name,email" lvt-empty="No users found">
</table>
```

When `users` is empty, renders:
```html
<table>
  <thead>...</thead>
  <tbody>
    <tr>
      <td colspan="2">No users found</td>
    </tr>
  </tbody>
</table>
```

### Rich Datatable Mode

For advanced features like sorting, use the `lvt-datatable` attribute:

```html
<table lvt-source="users" lvt-columns="name,email,role" lvt-datatable>
</table>
```

This uses the [datatable component](https://github.com/livetemplate/components/tree/main/datatable) which provides:
- Column sorting (click headers)
- Striped rows
- Hover effects

**Note:** Rich mode requires the external datatable component and may need additional CSS for proper styling.

---

## Lists

Transform an empty `<ul>` or `<ol>` element into a fully populated list from a data source.

### Basic Usage

```html
<ul lvt-source="tags" lvt-field="name">
</ul>
```

With a `tags.json` data file:
```json
[
  {"id": 1, "name": "alpha"},
  {"id": 2, "name": "beta"},
  {"id": 3, "name": "gamma"}
]
```

This renders as:
```html
<ul>
  <li>alpha</li>
  <li>beta</li>
  <li>gamma</li>
</ul>
```

**Note:** Data sources must be arrays of objects. Raw string arrays are not currently supported by the runtime.

### Attributes

| Attribute | Required | Default | Description |
|-----------|----------|---------|-------------|
| `lvt-source` | Yes | - | Name of the data source |
| `lvt-field` | No | `{{.}}` | Field name to display for object arrays |
| `lvt-actions` | No | - | Action buttons: `action:Label,action2:Label2` |
| `lvt-empty` | No | - | Message to show when data is empty |

### Object Arrays with Field

For arrays of objects, use `lvt-field` to specify which field to display:

```html
<ul lvt-source="tasks" lvt-field="title">
</ul>
```

With `tasks.json`:
```json
[
  {"id": 1, "title": "Complete project setup"},
  {"id": 2, "title": "Write documentation"},
  {"id": 3, "title": "Add unit tests"}
]
```

Renders as:
```html
<ul>
  <li>Complete project setup</li>
  <li>Write documentation</li>
  <li>Add unit tests</li>
</ul>
```

### Action Buttons

Add action buttons to each list item:

```html
<ul lvt-source="tasks" lvt-field="title" lvt-actions="delete:×,edit:Edit">
</ul>
```

Renders action buttons alongside each item:
```html
<ul>
  <li>
    Complete project setup
    <button lvt-click="delete" lvt-data-id="1">×</button>
    <button lvt-click="edit" lvt-data-id="1">Edit</button>
  </li>
  ...
</ul>
```

**Note:** `lvt-actions` requires `lvt-field` to be specified (object arrays only). Each object must have an `id` field, which is used for the `lvt-data-id` attribute on action buttons. Actions are ignored for simple string arrays.

### Empty State

Show a message when the data source is empty:

```html
<ol lvt-source="steps" lvt-empty="No steps defined yet">
</ol>
```

When `steps` is empty, renders:
```html
<ol>
  <li>No steps defined yet</li>
</ol>
```

### Ordered Lists

Works with both `<ul>` (unordered) and `<ol>` (ordered) lists:

```html
<ol lvt-source="steps" lvt-field="description">
</ol>
```

---

## Data Sources

Select, table, and list auto-rendering work with any Tinkerdown data source:

### JSON Files

```yaml
# tinkerdown.yaml (or frontmatter)
sources:
  users:
    type: json
    file: data/users.json
```

### REST APIs

```yaml
sources:
  products:
    type: rest
    url: https://api.example.com/products
```

### PostgreSQL

```yaml
sources:
  orders:
    type: pg
    query: "SELECT id, customer, total FROM orders"
```

### Executable Scripts

```yaml
sources:
  report:
    type: exec
    cmd: python scripts/generate_report.py
```

---

## XSS Prevention

All user-provided data is automatically escaped to prevent XSS attacks:

```json
[{"name": "<script>alert('xss')</script>", "email": "test@example.com"}]
```

Renders safely as:
```html
<td>&lt;script&gt;alert('xss')&lt;/script&gt;</td>
```

---

## Full Example

```markdown
---
title: "User Management"
sources:
  users:
    type: json
    file: users.json
  roles:
    type: json
    file: roles.json
---

# User Management

## Add User

<form lvt-submit="add_user">
  <input name="name" placeholder="Name" required>
  <input name="email" type="email" placeholder="Email" required>
  <select lvt-source="roles" lvt-value="id" lvt-label="name" name="role">
  </select>
  <button type="submit">Add User</button>
</form>

## Users

<table lvt-source="users" lvt-columns="name,email,role" lvt-actions="edit:Edit,delete:Delete" lvt-empty="No users yet">
</table>
```

With `users.json`:
```json
[
  {"id": 1, "name": "Alice", "email": "alice@example.com", "role": "Admin"},
  {"id": 2, "name": "Bob", "email": "bob@example.com", "role": "User"}
]
```

And `roles.json`:
```json
[
  {"id": "admin", "name": "Administrator"},
  {"id": "user", "name": "Regular User"},
  {"id": "guest", "name": "Guest"}
]
```
