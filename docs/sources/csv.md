# CSV Source

Load data from CSV files.

## Configuration

```yaml
sources:
  products:
    type: csv
    path: ./_data/products.csv
```

## Options

| Option | Required | Description |
|--------|----------|-------------|
| `type` | Yes | Must be `csv` |
| `path` | Yes | Path to CSV file |
| `delimiter` | No | Field delimiter (default: `,`) |
| `header` | No | First row is header (default: `true`) |

## Examples

### Basic Usage

```yaml
sources:
  inventory:
    type: csv
    path: ./_data/inventory.csv
```

### Custom Delimiter

```yaml
sources:
  data:
    type: csv
    path: ./_data/data.tsv
    delimiter: "\t"
```

### No Header Row

```yaml
sources:
  raw_data:
    type: csv
    path: ./_data/raw.csv
    header: false
```

## CSV File Format

**With headers (default):**

```csv
id,name,price,category
1,Widget,9.99,Electronics
2,Gadget,19.99,Electronics
3,Book,14.99,Books
```

**Without headers:**

```csv
1,Widget,9.99,Electronics
2,Gadget,19.99,Electronics
```

When `header: false`, columns are named `col0`, `col1`, `col2`, etc.

## Data Types

CSV values are parsed as strings by default. Numbers and booleans are auto-converted when possible.

## Usage in Templates

### Auto-Rendering

```html
<table lvt-source="products" lvt-columns="name,price,category">
</table>
```

### Go Templates

```html
{{range .products}}
<div class="product">
  <h3>{{.name}}</h3>
  <span>${{.price}}</span>
</div>
{{end}}
```

## File Location

Place CSV files in the `_data/` directory:

```
myapp/
├── _data/
│   ├── products.csv
│   ├── users.csv
│   └── inventory.csv
├── index.md
└── tinkerdown.yaml
```

## Use Cases

- Importing spreadsheet data
- Static datasets
- Report data
- Bulk data entry

## Full Example

```csv
# _data/employees.csv
id,name,department,salary
1,Alice,Engineering,95000
2,Bob,Sales,75000
3,Carol,Marketing,80000
```

```yaml
# tinkerdown.yaml
sources:
  employees:
    type: csv
    path: ./_data/employees.csv
```

```html
<!-- index.md -->
<h2>Employee Directory</h2>
<table lvt-source="employees" lvt-columns="name,department,salary">
</table>
```

## Next Steps

- [Markdown Source](markdown.md) - Content from markdown files
- [Data Sources Guide](../guides/data-sources.md) - Overview
