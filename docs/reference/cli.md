# CLI Reference

Complete reference for the `tinkerdown` command-line interface.

## Commands

### serve

Start the development server.

```bash
tinkerdown serve [directory | page.md | file:// URI | https:// URL] [flags]
```

**Arguments:**

| Argument | Description | Default |
|----------|-------------|---------|
| `directory` | Path to the app directory | Current directory |
| `page.md` or `file:///path/page.md` | Serve one local page at `/`, with its directory as the site root. Other pages in the directory aren't served. | |
| `https://…/page.md` | Fetch one page and serve it at `/`. Plain `http://` works only for localhost. | |

### Pages from a URL

`tinkerdown serve https://…/page.md` fetches the page once at startup and prints its SHA-256. End the URL with `#sha256=HEX` to refuse a page whose content changed since you read it.

Because the page comes from someone else, it may only declare `merge` sources, so it takes its sources from your `--source` flags. `serve` refuses a page from a URL that holds:

- any other kind of source, any action, or a schedule
- an include, a `wasm` or `embed-lvt` block, or a server block that isn't an `lvt` block
- a `<script>`, `<meta>`, `<base>`, `<link>`, `<iframe>`, `<object>` or `<embed>` element, an `on…` event handler, or a `javascript:` URL
- a template action in a tag name or attribute name, such as `<{{.tag}}>`. Actions may appear in text and in quoted attribute values.
- a merge `buffer` over 10,000 or a `flush` under 50ms

The page is also served with a strict Content-Security-Policy. It loads images and media only from the server, sends forms only to it, and runs no script but tinkerdown's own, so its template can't send your log rows to another site. A link a viewer clicks can still carry text away. It doesn't read a `tinkerdown.yaml` unless you pass `--config`.

To run a page that needs more, download it, review it, and serve the local copy with `tinkerdown serve page.md`.

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--source NAME=COMMAND` | Add a stream source that follows a command's output. Needs `--allow-exec`. Repeatable. | none |
| `--source NAME=file:PATH` | Add a stream source that follows a file. Repeatable. | none |
| `--allow-exec` | Let sources run commands | off |
| `--port`, `-p` | Server port | `8080` |
| `--host` | Server host | `localhost` |
| `--config`, `-c` | Config file | `tinkerdown.yaml` in the directory |
| `--watch`, `-w` | Reload pages when their files change | off |
| `--operator`, `-o` | Operator identity for audit fields | `$USER` |
| `--headless` | Serve only the API, webhooks and schedules | off |
| `--help`, `-h` | Print serve's usage and exit | |

A `--source` flag reaches a page through a merge source with `inputs: ["*"]`, or by replacing a page source with the same name. `serve` refuses a flag no page uses, and a flag of a kind the page's `accepts` leaves out. See [stream sources](../sources/stream.md#sources-from-the-command-line).

When a page needs sources and has none, `serve` asks for them if it runs in a terminal, then prints the `--source` flags to reuse. Without a terminal it never waits: it prints what to pass and serves the page's empty state.

**Examples:**

```bash
# Start with defaults
tinkerdown serve

# Specify port
tinkerdown serve --port 3000

# Run a one-file app from a URL, pinned to the content you reviewed
tinkerdown serve --allow-exec 'https://raw.githubusercontent.com/livetemplate/logviewer/main/index.md#sha256=…' --source 'api=docker logs -f api'

# Follow a container and a file on a page that takes --source flags
tinkerdown serve --allow-exec --source 'api=docker logs -f --timestamps api' --source 'app=file:./logs/app.log'
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
