# CLI Reference

Complete reference for the `tinkerdown` command-line interface.

## Commands

### serve

Start the development server.

```bash
tinkerdown serve [directory | page.md | file:// URI | https:// URL | short name] [flags]
```

**Arguments:**

| Argument | Description | Default |
|----------|-------------|---------|
| `directory` | Path to the app directory | Current directory |
| `page.md` or `file:///path/page.md` | Serve one local page at `/`, with its directory as the site root. Other pages in the directory aren't served. | |
| `https://…/page.md` | Fetch one page and serve it at `/`. Plain `http://` works only for localhost. | |
| `github.com/OWNER/REPO[/PATH][@REF]` | A short name for a page on GitHub, fetched like a URL. See [short names](#short-names). | |

### Pages from a URL

`tinkerdown serve https://…/page.md` fetches the page once at startup and prints its SHA-256. End the URL with `#sha256=HEX` to refuse a page whose content changed since you read it. The URL you type is fetched as given, even if it names a private address, since you chose it. A redirect, which the page's host controls, must go to a public `https://` address.

Because the page comes from someone else, it may only declare `merge` sources, so it takes its sources from your `--source` flags and its commands from your `--action` flags. `serve` refuses a page from a URL that holds:

- any other kind of source, any action, or a schedule
- frontmatter beyond `title`, `sources`, `styling`, `charts`, `blocks` and `features`
- an include, a `wasm` or `embed-lvt` block, or a server block that isn't an `lvt` block
- a `<script>`, `<meta>`, `<base>`, `<link>`, `<iframe>`, `<object>`, `<embed>`, `<svg>` or `<math>` element, markup inside a raw-text element such as `<style>`, an `on…` event handler, or a `javascript:` URL
- a template action in a tag name, attribute name, comment or raw-text element such as `<textarea>`, as in `<{{.tag}}>`, and the trim markers `{{-` and `-}}`. Actions may appear in text and attribute values.

The checks read the page's HTML and each template with an HTML tokenizer, the way a browser reads them.
- a merge `buffer` over 10,000 or a `flush` under 50ms

The page is also served with a strict Content-Security-Policy. It loads images and media only from the server, sends forms only to it, and runs no script but tinkerdown's own, so its template can't send your log rows to another site. A link a viewer clicks can still carry text away. It doesn't read a `tinkerdown.yaml` unless you pass `--config`.

To run a page that needs more, download it, review it, and serve the local copy with `tinkerdown serve page.md`.

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `-- COMMAND` | Add a stream source that follows COMMAND's output. Everything after the first `--` is one command, so it needs no quotes. A pipe would pipe `serve` itself, so give a pipeline as one `--source`: `--source 'docker logs -f api 2>&1 \| grep -v health'`. Needs `--allow-exec`. | none |
| `--source COMMAND` | Add a stream source that follows a command's output. Needs `--allow-exec`. Repeatable. | none |
| `--source file:PATH` | Add a stream source that follows a file. Repeatable. | none |
| `--source http:PATH` | Add a stream source that lists the requests in a `.http` file. Repeatable. | none |
| `--source sql:PATH` | Add a stream source that lists the queries in a `.sql` file, or in a folder's `.sql` files. Repeatable. | none |
| `--action NAME=COMMAND` | Run COMMAND when a button named NAME is clicked, on any page. Needs `--allow-exec`. Repeatable. | none |
| `--allow-exec` | Let sources and actions run commands | off |
| `--port`, `-p` | Server port | `8080` |
| `--host` | Server host | `localhost` |
| `--config`, `-c` | Config file | `tinkerdown.yaml` in the directory |
| `--watch`, `-w` | Reload pages when their files change | off |
| `--operator`, `-o` | Operator identity for audit fields | `$USER` |
| `--headless` | Serve only the API, webhooks and schedules | off |
| `--help`, `-h` | Print serve's usage and exit. After a page, also list what the page takes: see [what a page takes](#what-a-page-takes) | |

A `--source` value may start with `NAME=`, which names the source: `--source 'api=docker logs -f api'`. `NAME=` is the part before the first `=` when that part is 1-32 letters, digits, `_` or `-`, so `--source 'grep a=b log'` has no name. Write a command that starts with an environment assignment as `env FOO=1 cmd`. A source with no name gets a label: a command's last word that isn't a flag, with a path reduced to its base name (`docker logs -f api` is `api`, `tail -f /var/log/app.log` is `app`), or a file's name without its extension. When a label is taken, by another source or by a name the page uses, it becomes `api-2`, `api-3`. So a label never takes a name you give, and never replaces a page's source: only `NAME=` does that.

A `--source` flag reaches a page through a merge source with `inputs: ["*"]`, or, given a `NAME=`, by replacing a page source with that name. `serve` refuses a flag no page uses, and a flag of a kind the page's `accepts` leaves out. For a page from a URL, a relative `file:`, `http:` or `sql:` path is taken from the directory `serve` runs in, and commands from `--source`, `--` and `--action` run there. See [stream sources](../sources/stream.md#sources-from-the-command-line).

`serve` refuses a flag it doesn't know, such as a mistyped `--alow-exec`, instead of ignoring it. A flag's value goes after a space: `--port 8080`, not `--port=8080`.

When a page needs sources and has none, `serve` asks for them if it runs in a terminal, then prints the `--source` flags to reuse. Without a terminal it never waits: it prints what to pass and serves the page's empty state.

**Examples:**

```bash
# Start with defaults
tinkerdown serve

# Specify port
tinkerdown serve --port 3000

# Run a one-file app from GitHub, pinned to the content you reviewed
tinkerdown serve --allow-exec 'github.com/livetemplate/onefile/logs#sha256=…' -- docker logs -f api

# Follow a container and a file on a page that takes --source flags
tinkerdown serve --allow-exec --source 'docker logs -f --timestamps api' --source file:./logs/app.log

# See what a page takes before you run it
tinkerdown serve github.com/livetemplate/onefile/http --help
```

### What a page takes

`tinkerdown serve PAGE --help` loads the page, from its URL or short name too, and lists what it takes from the command line after the usage. It reads only what the page declares:

- a merge with `inputs: ["*"]`: the forms its `accepts` kinds take (`-- COMMAND`, `--source file:PATH`, `--source http:PATH`, `--source sql:PATH`), how many it needs, and its hint
- an input of another merge that no page or `tinkerdown.yaml` source defines: `--source NAME=…`, and whether a button or a row you open starts it
- a button whose name isn't a builtin action or a merge's `on:` button: `--action NAME=COMMAND`

```
github.com/livetemplate/onefile/http takes:
  --source http:PATH
      1 or more, shown in requests. The page says: "…"
  --source send=…
      run by the send button, shown in response
```

The names and hints are the page's text, so `serve` cleans them before printing. Help reads no `--source` or `--action` flag, so it needs no `--allow-exec` and starts nothing.

### Short names

A short name such as `github.com/livetemplate/onefile/logs` names a page on GitHub the way a Go import path names a package. `serve` expands it to `https://raw.githubusercontent.com/OWNER/REPO/REF/PATH/index.md` and fetches that URL, with the same checks:

- `@REF` picks a branch, tag or commit, as in `github.com/livetemplate/onefile/logs@main`. Without it, `serve` uses `HEAD`, the repo's default branch. A ref can't contain `/`, since the URL couldn't show where it ends.
- A `PATH` ending in `.md` names the page itself. Any other `PATH` names a directory and its `index.md`. With no `PATH`, the page is the repo's root `index.md`.
- `#sha256=HEX` after a short name pins the page, as after a URL.
- Each part may hold only letters, digits, `.`, `_` and `-`, and a path part can't be `.` or `..`. `serve` refuses any other name and says which rule it broke.
- A bare `github.com/…` is always a short name, even when a directory of that name exists. Otherwise a checkout could plant one and have its page served as a local page, without the pin or these checks. To serve a local checkout at that path, write `./github.com/…` or use `file://`.

Messages name both the short name and the URL it expanded to. The pin line offers the short name, so you can share back the line you typed.

### Actions from the command line

An `--action` command never has values pasted into it. Because a value reaches the command as an environment variable, the shell never reads it as code. Each `data-*` attribute of the clicked button becomes `TD_<name>`, so `<button name="stop" data-name="web">` runs the `stop` command with `TD_name=web`:

```bash
tinkerdown serve --allow-exec page.md --action 'stop=docker stop -- "$TD_name"'
```

Quote the variable: `"$TD_name"` stays one argument, while an unquoted `$TD_name` splits on spaces. The `--` keeps a value from being read as an option. A `--source` command read by a merge with `params` gets the page URL's values the same way; see [values from the page URL](../sources/stream.md#values-from-the-page-url).

A click runs nothing when a value is empty, longer than 256 bytes, starts with `-`, or holds a control character; the block shows the reason. An attribute whose name isn't 1-32 letters, digits or `_` is skipped. The command runs in the site directory with a 30-second timeout.

`serve` refuses an `--action` named like a builtin action (`pause`, `delete`, `filter`, …), since the builtin would run instead, and one that would replace an action approved in `tinkerdown.yaml`. An `--action` replaces a local page's action of the same name. Webhooks and the HTTP action API can't call it.

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
