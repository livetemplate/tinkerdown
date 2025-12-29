# Tinkerdown

**Interactive documentation made easy**

Tinkerdown is a CLI tool for creating interactive tutorials, guides, and playgrounds using markdown files with embedded executable code blocks. Built on [LiveTemplate](https://github.com/livetemplate/livetemplate).

## Status

🚧 **Early Development** - Not yet ready for use

See [PROGRESS.md](PROGRESS.md) for implementation status and [docs/plans/2025-11-12-tinkerdown-design.md](docs/plans/2025-11-12-tinkerdown-design.md) for complete design.

## Vision

Writing interactive tutorials should be as easy as writing markdown:

```markdown
---
title: "Build a Counter"
---

# Learn LiveTemplate

## Server State

```go server readonly id="counter"
type CounterState struct {
    Counter int `json:"counter"`
}

// Increment handles the "increment" action
func (s *CounterState) Increment(_ *livetemplate.Context) error {
    s.Counter++
    return nil
}
```

## Try It Live

```lvt interactive state="counter"
<div>
    <h2>Count: {{.Counter}}</h2>
    <button lvt-click="increment">+1</button>
</div>
```

## Experiment

```go wasm editable
package main
import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
```
```

Then run:

```bash
tinkerdown serve
```

And get a fully interactive tutorial website!

## Key Features (Planned)

- 📝 **Markdown-first**: Write tutorials in markdown with special code blocks
- ⚡ **Real-time**: Interactive demos powered by LiveTemplate's reactivity
- 🎮 **Playgrounds**: Editable Go code that runs in browser via WebAssembly
- 🔒 **Secure**: Student code never touches server (WASM sandbox)
- 🚀 **Zero config**: `tinkerdown serve` just works
- 🎨 **Beautiful**: Built-in theme, looks professional out of the box
- 🔥 **Hot reload**: Changes reflect immediately during development

## Architecture

- **Dual execution model**: Author code runs on server (trusted), student code in browser (WASM sandbox)
- **Hybrid rendering**: Static markdown cached, code blocks dynamic
- **Multiplexed WebSocket**: Single connection for all interactive elements

## Configuration

Tinkerdown can be customized using a `tinkerdown.yaml` file in your project directory:

```yaml
# tinkerdown.yaml
title: "My Tutorial"
description: "Learn something awesome"

server:
  port: 8080
  host: localhost
  debug: false

styling:
  theme: clean              # Options: clean, dark, minimal
  primary_color: "#007bff"
  font: "system-ui"

blocks:
  auto_id: true
  id_format: "kebab-case"   # Options: kebab-case, camelCase, snake_case
  show_line_numbers: true

features:
  hot_reload: true

ignore:
  - "drafts/**"
  - "_*.md"
```

CLI flags override configuration file values:

```bash
tinkerdown serve --port 3000 --watch     # Override port and enable watch
tinkerdown serve --config custom.yaml    # Use custom config file
```

See `tinkerdown.yaml.example` for all available options.

### Frontmatter Configuration (Optional)

For single-file apps, you can define configuration directly in the markdown frontmatter, making `tinkerdown.yaml` optional:

```markdown
---
title: "My App"
sources:
  users:
    type: json
    file: users.json
  api_data:
    type: rest
    url: https://api.example.com/data
  db_users:
    type: pg
    query: "SELECT * FROM users"
styling:
  theme: dark
  primary_color: "#6366f1"
features:
  hot_reload: true
---

# My App

```lvt
<div lvt-source="users">
  {{range .Data}}<p>{{.Name}}</p>{{end}}
</div>
```
```

**Supported frontmatter config options:**

| Option | Description |
|--------|-------------|
| `sources` | Data sources (json, csv, rest, pg, exec) |
| `styling` | Theme configuration (theme, primary_color, font) |
| `blocks` | Code block settings (auto_id, id_format, show_line_numbers) |
| `features` | Feature flags (hot_reload) |

Frontmatter config takes precedence over `tinkerdown.yaml` when both exist.

## Development

```bash
# Clone
git clone https://github.com/livetemplate/tinkerdown.git
cd tinkerdown

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o tinkerdown ./cmd/tinkerdown
```

## Auto-Rendering

Tinkerdown can automatically generate HTML for common UI patterns:

```html
<!-- Select dropdown from data -->
<select lvt-source="countries" lvt-value="code" lvt-label="name">
</select>

<!-- Table with headers, rows, and actions -->
<table lvt-source="users" lvt-columns="name,email" lvt-actions="edit:Edit,delete:Delete">
</table>
```

See [Auto-Rendering Documentation](docs/auto-rendering.md) for full details.

## Documentation

- [Auto-Rendering](docs/auto-rendering.md) - Tables and select dropdowns from data sources
- [Design Document](docs/plans/2025-11-12-tinkerdown-design.md) - Complete architecture and design decisions
- [Progress Tracker](PROGRESS.md) - Implementation status and roadmap

## License

MIT

## Contributing

This project is in early development. Contributions welcome once core functionality is stable.
