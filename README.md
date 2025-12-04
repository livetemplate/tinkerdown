# Livepage

**Interactive documentation made easy**

Livepage is a CLI tool for creating interactive tutorials, guides, and playgrounds using markdown files with embedded executable code blocks. Built on [LiveTemplate](https://github.com/livetemplate/livetemplate).

## Status

🚧 **Early Development** - Not yet ready for use

See [PROGRESS.md](PROGRESS.md) for implementation status and [docs/plans/2025-11-12-livepage-design.md](docs/plans/2025-11-12-livepage-design.md) for complete design.

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
func (s *CounterState) Increment(_ *livetemplate.ActionContext) error {
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
livepage serve
```

And get a fully interactive tutorial website!

## Key Features (Planned)

- 📝 **Markdown-first**: Write tutorials in markdown with special code blocks
- ⚡ **Real-time**: Interactive demos powered by LiveTemplate's reactivity
- 🎮 **Playgrounds**: Editable Go code that runs in browser via WebAssembly
- 🔒 **Secure**: Student code never touches server (WASM sandbox)
- 🚀 **Zero config**: `livepage serve` just works
- 🎨 **Beautiful**: Built-in theme, looks professional out of the box
- 🔥 **Hot reload**: Changes reflect immediately during development

## Architecture

- **Dual execution model**: Author code runs on server (trusted), student code in browser (WASM sandbox)
- **Hybrid rendering**: Static markdown cached, code blocks dynamic
- **Multiplexed WebSocket**: Single connection for all interactive elements

## Configuration

Livepage can be customized using a `livepage.yaml` file in your project directory:

```yaml
# livepage.yaml
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
livepage serve --port 3000 --watch     # Override port and enable watch
livepage serve --config custom.yaml    # Use custom config file
```

See `livepage.yaml.example` for all available options.

## Development

```bash
# Clone
git clone https://github.com/livetemplate/livepage.git
cd livepage

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o livepage ./cmd/livepage
```

## Documentation

- [Design Document](docs/plans/2025-11-12-livepage-design.md) - Complete architecture and design decisions
- [Progress Tracker](PROGRESS.md) - Implementation status and roadmap

## License

MIT

## Contributing

This project is in early development. Contributions welcome once core functionality is stable.
