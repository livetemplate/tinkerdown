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

Create a new Tinkerdown app.

```bash
tinkerdown new <name> [flags]
```

**Arguments:**

| Argument | Description |
|----------|-------------|
| `name` | Name of the new app (creates directory) |

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--template` | Template to use | `basic` |

**Available Templates:**

| Template | Description |
|----------|-------------|
| `basic` | Minimal app with one source |
| `todo` | SQLite CRUD with toggle/delete |
| `dashboard` | Multi-source data display |
| `form` | Contact form with SQLite persistence |

**Examples:**

```bash
# Create basic app
tinkerdown new myapp

# Create todo app
tinkerdown new myapp --template=todo
```

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
