# CLI Reference

Complete reference for the `tinkerdown` command-line interface.

## Commands

### serve

Start the development server.

```bash
tinkerdown serve [directory] [flags]
```

**Arguments:**

| Argument | Description | Default |
|----------|-------------|---------|
| `directory` | Path to the app directory | Current directory |

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--port`, `-p` | Server port | `8080` |
| `--host` | Server host | `localhost` |
| `--production` | Production mode | `false` |
| `--debug` | Enable debug logging | `false` |
| `--verbose` | Enable verbose logging | `false` |
| `--log-format` | Log format (text, json) | `text` |

**Examples:**

```bash
# Start with defaults
tinkerdown serve

# Specify port
tinkerdown serve --port 3000

# Production mode
tinkerdown serve --production

# Debug mode
tinkerdown serve --debug
```

### new

Create a new Tinkerdown app from a template.

```bash
tinkerdown new [options] <name>
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `name` | Name of the new app (creates directory) |

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--template` | Template to use | `basic` |
| `--list` | List available templates | - |

**Available Templates:**

| Template | Description |
|----------|-------------|
| `basic` | Minimal starter with interactive counter example |
| `todo` | Task list with SQLite CRUD operations |
| `dashboard` | Multi-source data display with tables and stats |
| `form` | Contact form with validation and persistence |
| `api-explorer` | REST API explorer with live data refresh |
| `cli-wrapper` | Wrap CLI tools with an interactive web form |
| `wasm-source` | Scaffold for building custom WASM data sources |

**Examples:**

```bash
# List available templates
tinkerdown new --list

# Create basic app (default template)
tinkerdown new myapp

# Create todo app
tinkerdown new --template=todo my-todos

# Create API explorer
tinkerdown new --template=api-explorer api-dashboard

# Create WASM source scaffold
tinkerdown new --template=wasm-source my-custom-source
```

**Template Details:**

- **basic**: A minimal starter with a simple interactive counter. Good for learning the basics.

- **todo**: A full todo list app with SQLite persistence. Demonstrates Add, Toggle, Delete operations.

- **dashboard**: Multi-source dashboard with tasks and team data from markdown files. Shows stat cards and tables.

- **form**: A contact form that saves submissions to SQLite. Shows form handling and validation.

- **api-explorer**: Connects to REST APIs (uses JSONPlaceholder demo). Shows caching and refresh.

- **cli-wrapper**: Wrap any CLI tool with a web form. Auto-generates inputs from command arguments.

- **wasm-source**: Scaffold for building custom WASM data sources with TinyGo. Includes source.go, Makefile, and documentation.

### validate

Validate a Tinkerdown app.

```bash
tinkerdown validate [directory] [flags]
```

**Arguments:**

| Argument | Description | Default |
|----------|-------------|---------|
| `directory` | Path to the app directory | Current directory |

**Checks performed:**

- Markdown syntax
- Source references exist
- Configuration validity
- WASM module paths

**Examples:**

```bash
# Validate current directory
tinkerdown validate

# Validate specific app
tinkerdown validate ./myapp
```

### version

Display version information.

```bash
tinkerdown version
```

## Global Flags

These flags work with all commands:

| Flag | Description |
|------|-------------|
| `--help`, `-h` | Display help |
| `--config` | Path to config file |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TINKERDOWN_PORT` | Default server port | `8080` |
| `TINKERDOWN_DEBUG` | Enable debug mode | `false` |

## Exit Codes

| Code | Description |
|------|-------------|
| `0` | Success |
| `1` | General error |
| `2` | Invalid configuration |
| `3` | Validation error |

## Next Steps

- [Configuration Reference](config.md) - tinkerdown.yaml options
- [Getting Started](../getting-started/quickstart.md) - First app tutorial
