# Tinkerdown

**Build data-driven apps with markdown**

Tinkerdown is a CLI tool for creating interactive, data-driven applications using markdown files. Connect to databases, APIs, and files with zero boilerplate. Built on [LiveTemplate](https://github.com/livetemplate/livetemplate).

## Quick Start

```bash
# Install
go install github.com/livetemplate/tinkerdown/cmd/tinkerdown@latest

# Run an example
tinkerdown serve examples/lvt-source-file-test

# Open http://localhost:8080
```

## What You Can Build

Write markdown with embedded `lvt` blocks that connect to data sources:

```markdown
---
title: "User Dashboard"
sources:
  users:
    type: json
    file: users.json
---

# User Dashboard

<table lvt-source="users" lvt-columns="name,email,role" lvt-actions="edit:Edit,delete:Delete">
</table>
```

Run `tinkerdown serve` and get a fully interactive data table with action buttons.

## Key Features

- **Markdown-first**: Write apps in markdown with `lvt` code blocks
- **8 data sources**: JSON, CSV, REST APIs, PostgreSQL, SQLite, exec scripts, markdown, WASM
- **Auto-rendering**: Tables, selects, and lists generated from data
- **Real-time updates**: WebSocket-powered reactivity
- **Zero config**: `tinkerdown serve` just works
- **Hot reload**: Changes reflect immediately with `--watch`

## Data Sources

Connect to any data source in frontmatter or `tinkerdown.yaml`:

| Type | Description | Example |
|------|-------------|---------|
| `json` | JSON files | [lvt-source-file-test](examples/lvt-source-file-test) |
| `csv` | CSV files | [lvt-source-file-test](examples/lvt-source-file-test) |
| `rest` | REST APIs | [lvt-source-rest-test](examples/lvt-source-rest-test) |
| `pg` | PostgreSQL | [lvt-source-pg-test](examples/lvt-source-pg-test) |
| `sqlite` | SQLite databases | [lvt-source-sqlite-test](examples/lvt-source-sqlite-test) |
| `exec` | Shell commands (any language) | [lvt-source-exec-test](examples/lvt-source-exec-test) |
| `markdown` | Markdown files with anchors | [markdown-data-todo](examples/markdown-data-todo) |
| `wasm` | WASM modules | [lvt-source-wasm-test](examples/lvt-source-wasm-test) |

## Auto-Rendering

Generate HTML automatically from data sources:

```html
<!-- Select dropdown -->
<select lvt-source="countries" lvt-value="code" lvt-label="name">
</select>

<!-- Table with actions -->
<table lvt-source="users" lvt-columns="name,email" lvt-actions="edit:Edit,delete:Delete">
</table>

<!-- List with actions -->
<ul lvt-source="tasks" lvt-field="title" lvt-actions="delete:×">
</ul>
```

See [Auto-Rendering Documentation](docs/auto-rendering.md) for full details.

**Example:** [component-library-test](examples/component-library-test)

## Interactive Attributes

| Attribute | Description |
|-----------|-------------|
| `lvt-source` | Connect element to a data source |
| `lvt-click` | Handle click events |
| `lvt-submit` | Handle form submissions |
| `lvt-change` | Handle input changes |
| `lvt-confirm` | Show confirmation dialog before action |
| `lvt-data-*` | Pass data with actions |

## Configuration

Configure via `tinkerdown.yaml` or markdown frontmatter:

```yaml
# tinkerdown.yaml
title: "My App"
server:
  port: 8080
  debug: false
sources:
  users:
    type: json
    file: data/users.json
styling:
  theme: clean
features:
  hot_reload: true
```

Or inline in frontmatter:

```markdown
---
title: "My App"
sources:
  users:
    type: json
    file: users.json
---
```

CLI flags override config: `tinkerdown serve --port 3000 --watch`

## Development

```bash
git clone https://github.com/livetemplate/tinkerdown.git
cd tinkerdown
go mod download
go test ./...
go build -o tinkerdown ./cmd/tinkerdown
```

## Documentation

- [Auto-Rendering](docs/auto-rendering.md) - Tables, selects, and lists from data
- [Roadmap](ROADMAP.md) - Feature planning and implementation status
- [Design Document](docs/plans/2025-11-12-tinkerdown-design.md) - Architecture decisions

## License

MIT

## Contributing

Contributions welcome! See [ROADMAP.md](ROADMAP.md) for planned features and current priorities.
