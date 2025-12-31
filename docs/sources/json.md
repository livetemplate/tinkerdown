# JSON Source

Load data from JSON files.

## Configuration

```yaml
sources:
  config:
    type: json
    path: ./_data/config.json
```

## Options

| Option | Required | Description |
|--------|----------|-------------|
| `type` | Yes | Must be `json` |
| `path` | Yes | Path to JSON file |

## Examples

### Basic Usage

```yaml
sources:
  products:
    type: json
    path: ./_data/products.json
```

### Data File Structure

**Array of objects:**

```json
[
  {"id": 1, "name": "Widget", "price": 9.99},
  {"id": 2, "name": "Gadget", "price": 19.99}
]
```

**Object with array:**

```json
{
  "products": [
    {"id": 1, "name": "Widget", "price": 9.99},
    {"id": 2, "name": "Gadget", "price": 19.99}
  ]
}
```

## File Location

Place JSON files in the `_data/` directory:

```
myapp/
├── _data/
│   ├── products.json
│   ├── categories.json
│   └── config.json
├── index.md
└── tinkerdown.yaml
```

## Usage in Templates

### Auto-Rendering

```html
<table lvt-source="products" lvt-columns="name,price">
</table>
```

### Go Templates

```html
{{range .products}}
<div class="product">
  <h3>{{.name}}</h3>
  <span class="price">${{.price}}</span>
</div>
{{end}}
```

## Nested Data

JSON sources handle nested structures:

```json
{
  "company": {
    "name": "Acme Inc",
    "employees": [
      {"name": "Alice", "department": "Engineering"},
      {"name": "Bob", "department": "Sales"}
    ]
  }
}
```

Access nested data in templates:

```html
<h1>{{.company.name}}</h1>
{{range .company.employees}}
<p>{{.name}} - {{.department}}</p>
{{end}}
```

## Hot Reload

JSON files are re-read on each request in development mode. Changes are reflected immediately.

## Use Cases

- Configuration data
- Static content (navigation, footer links)
- Mock data during development
- Localization strings

## Full Example

```json
// _data/navigation.json
[
  {"title": "Home", "url": "/", "icon": "home"},
  {"title": "Products", "url": "/products", "icon": "box"},
  {"title": "About", "url": "/about", "icon": "info"}
]
```

```yaml
# tinkerdown.yaml
sources:
  nav:
    type: json
    path: ./_data/navigation.json
```

```html
<!-- index.md -->
<nav>
  {{range .nav}}
  <a href="{{.url}}">{{.title}}</a>
  {{end}}
</nav>
```

## Next Steps

- [CSV Source](csv.md) - Spreadsheet data
- [Data Sources Guide](../guides/data-sources.md) - Overview
