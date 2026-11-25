# Stream and Merge Sources

A `stream` source follows a long-running command or a growing file, and the page shows each new line as it arrives. A `merge` source combines several streams into one timeline. Use them for logs: `docker logs -f`, `journalctl -f`, `kubectl logs -f`, a dev server's output, or `/var/log/app.log`. A stream can also list the requests in a `.http` file, and a merge can run a command when a button is clicked; see [Runs a click starts](#runs-a-click-starts).

## Configuration

```yaml
sources:
  web:
    type: stream
    cmd: docker logs -f --timestamps web
  app:
    type: stream
    file: ./logs/app.log
  all:
    type: merge
    inputs: [web, app]
```

A command needs `tinkerdown serve --allow-exec`, as an `exec` source does. A file does not.

## Options

| Option | Applies to | Description |
|--------|------------|-------------|
| `type` | both | `stream` or `merge` |
| `cmd` | stream | Command to follow. Runs with `/bin/sh -c` from the site directory, so pipes, quoting and `./script.sh` work. Set one of `cmd`, `file` and `http`. |
| `file` | stream | File to follow, absolute or relative to the site directory. |
| `http` | stream | A `.http` file, absolute or relative to the site directory: one row per request, read again whenever the file changes. See [Requests from a .http file](#requests-from-a-http-file). |
| `inputs` | merge | Names of the stream sources to combine, or `["*"]` for every stream source the page can see, including `--source` flags. |
| `key` | merge | Field paths (`NAME` or `fields.NAME`) that make the merge keyed: one row per distinct key, updated in place. See [One row per key](#one-row-per-key). |
| `collect` | merge | With `key`: a field path whose text is gathered across a key's rows into `collected`. |
| `collect_reset` | merge | With `collect`: `PATH=VALUE`, where VALUE may list alternatives as `a\|b`. A row that matches starts its key's `collected` over, such as `fields.Action=run\|start` for a test's or package's next run. |
| `drop` | merge | With `key`: `PATH=VALUE`, where VALUE may list alternatives as `a\|b` (a value can't contain `\|`, and a number matches as it prints). A row that matches removes its key's row and isn't stored, such as `fields.state=removed` for a removed container. |
| `sort` | merge | With `key`: a field, or `-` and a field for descending, such as `-fields.createdAt`. Rows show in that field's order instead of first-seen order. |
| `params` | merge | Values taken from the page URL's query, each with `in: MERGE.PATH`. See [Values from the page URL](#values-from-the-page-url). |
| `on` | merge | With `params`: the name of the button whose click starts the inputs, instead of the URL. See [Runs a click starts](#runs-a-click-starts). |
| `line_max` | merge | The longest line its inputs read, in bytes or with `KiB` or `MiB`, up to `16MiB` (default: `16KiB`). A longer line is cut and its row gets `truncated: true`. |
| `accepts` | merge | With `inputs: ["*"]`: the sources the page takes. `kinds` lists any of `cmd`, `file` and `http` (default: all), `min` is how many it needs, and `hint` is one line saying what to pass. |
| `format` | stream | How to parse each line: `auto` (default), `json`, `logfmt` or `lines`. |
| `env` | stream | Extra environment variables for `cmd`. `$VAR` references are expanded. |
| `restart` | stream | `on-exit` (default) restarts an exited command after 1s, doubling up to 30s. `never` leaves it stopped. |
| `backlog` | stream | Lines read from the end of `file` on start (default: `200`). |
| `buffer` | both | Rows kept (default: `2000`). Older rows are dropped. |
| `flush` | both | Minimum time between updates to one page (default: `100ms`). |

## Sources from the command line

A page can leave its sources to whoever runs it. Give it a merge with `inputs: ["*"]`, and pass the sources to `tinkerdown serve`:

```yaml
sources:
  all:
    type: merge
    inputs: ["*"]
    accepts:
      min: 1
      hint: "Logs to follow: a container, a dev server or a log file"
```

```sh
tinkerdown serve --allow-exec --source 'docker logs -f --timestamps api' --source file:./logs/app.log
# or, for one command, everything after --
tinkerdown serve --allow-exec -- docker logs -f --timestamps api
```

`--source COMMAND` follows a command and needs `--allow-exec`; `--source file:PATH` follows a file, and `--source http:PATH` lists a `.http` file's requests. Each may start with `NAME=` to name the source; without it, the source gets a label from its command or file name, as [the CLI reference](../reference/cli.md#serve) describes. A merge that reads a source by name, such as one with `params` or `on:`, needs `NAME=`. For a page from a URL, a relative `PATH` is taken from the directory `serve` runs in, and a relative path in a `--config` file from that file's directory. A flag with `NAME=` also replaces a page or `tinkerdown.yaml` source of that name, but never an approved one. A label never does: it becomes `logs-2` when the page uses `logs`. `serve` refuses a flag no page uses, and one of a kind the page's `accepts` leaves out.

`["*"]` takes every stream source, sorted by name. An approved merge takes only `tinkerdown.yaml` sources, so neither a page nor a flag can add one. With no sources yet, the merge is empty: the block can show an empty state with `{{if not .Inputs}}`, and `.Accepts.Hint` says what to pass.

To show one tab per source, write the tabs heading with `[*] source`:

```markdown
## [All] | [*] source
```

When a page needs sources and has none, `serve` asks for them in a terminal, then prints the flags to reuse next time. Without a terminal, as when an agent or CI runs it, `serve` never waits: it prints what to pass and serves the empty state. It prints a page's hint without control characters, quoted as the page's words.

## One row per key

Some streams report on things rather than log lines: `go test -json` writes several events per test, and `kubectl get pods -w` a line per change to a pod. A merge with `key` keeps one row per thing, the latest one:

```yaml
sources:
  tests:
    type: merge
    inputs: ["*"]
    key: [fields.Package, fields.Test]
    collect: fields.Output
```

A row whose key matches an earlier one replaces it in place, and keeps the earlier row's position, so the list doesn't jump. A keyed row has a `key` field, a short hex hash of its key values: use it as `data-key`, since `seq` changes with every update. Being hex, it is safe whatever the key fields hold. A field missing from a row counts as empty, so `go test`'s package-level events become one row per package. `buffer` counts keys, and past it the key updated least recently is dropped.

`collect` gathers a field's text from all of a key's rows into the row's `collected`, up to 64 KiB, keeping the end. A test's row then carries its output. When the same keys come back in a new run, `collect_reset: fields.Action=run` starts each key's text over at its `run` row, so the output is the latest run's:

```html
{{range .Data}}
<li data-key="{{.key}}" class="{{.fields.Action}}">
  {{.fields.Test}} {{.fields.Action}}
  <pre>{{.collected}}</pre>
</li>
{{end}}
```

Some keys end for good, such as a container that was removed. `drop: fields.state=removed` takes a matching row as the end of its key, so the key's row leaves the list and the page gets a remove for it. A later row for the same key starts a new row at the end of the list.

By default a keyed merge shows keys in the order it first saw them. `sort: -fields.createdAt` orders them by a field instead, newest first; without the `-` it's ascending. Numbers compare as numbers, and so does text that reads as a number (logfmt values are text), the same way a filter compares them. Numbers come before other text, which compares as text, so ISO times sort correctly. A row without the field goes last in either direction, and rows with equal values keep the order they arrived in. A new row reaches the page as an insert at its place, and a row whose value changes moves, so a list sorted by a creation time only ever grows at the top.

## Values from the page URL

A merge with `params` takes values from the page URL's query and starts its inputs with them, so a link opens one item's stream. Each value reaches the input's command as `TD_<name>`, never pasted into it:

```yaml
sources:
  containers:
    type: merge
    inputs: ["*"]
    key: [fields.name]
  logview:
    type: merge
    inputs: [logs]
    params:
      name: {in: containers.fields.name}
```

```sh
tinkerdown serve --allow-exec page.md --source docker='…' --source logs='docker logs -f --tail 200 -- "$TD_name"'
```

A row links to `?name={{.fields.name}}`, and `logview` then follows that container. Quote the variable, `"$TD_name"`, so a value stays one argument.

The block's `.Params` holds the values it took, by name, so it can say what it shows: `{{with .Params.name}}<h3>{{.}}</h3>{{end}}`. `.Params` is empty until every param is present and valid.

Because the page has a `params` merge, clicking a link that only changes the query doesn't load the page again. The address bar changes, the new values go to the server on the open connection, and only the `params` blocks update, so the rest of the page keeps its search text, tab and Pause. Back and Forward work the same way. A link opened in a new tab, or to another page, loads as usual.

Because anyone can write a link, every param needs `in`. A value is accepted only while it equals `PATH` in a row of `MERGE`, another merge on the same page, so a link opens only what the page already lists. The check runs again when `MERGE`'s rows change: a link opened before its row arrives starts once the row appears. Without every param in the query, the block shows no rows and nothing runs. A value that is empty, over 256 bytes, starts with `-`, or holds a control character sets the block's `.Error` instead, and nothing runs.

Each distinct set of values gets its own buffer and its own commands, which stop 30 seconds after the last page showing them closes. A `["*"]` merge on the page doesn't take the inputs a `params` merge lists, since they need its values. A `params` merge lists its inputs by name. Name it differently from its inputs: a `--source` flag replaces a page source of the same name.

## Requests from a .http file

A stream with `http` reads a `.http` file, the format the VS Code REST Client and JetBrains IDEs use, and shows one row per request. It reads the file again when its size or modification time changes, and each read replaces the rows, so a deleted request leaves the page.

```http
@host = https://api.example.com

### Create user
POST {{host}}/users
Authorization: Bearer {{$processEnv TOKEN}}
Content-Type: application/json

{"name": "Ada"}
```

`###` separates requests, and the text after it names the next one; `# @name x` names it too. `#` and `//` start comments, also between headers. `@name = value` defines a variable for the whole file, and can come anywhere outside a request: after a request line, a line like it is part of the body. A request is its request line (`METHOD URL`, an optional `HTTP/1.1`, or a bare URL for `GET`), then headers up to the first blank line, then the body up to the next `###`.

Each row's `fields` hold the request as written: `entry` (its position, `"1"` first), `name`, `method`, `url`, `headers` (one per line), `body`, and `text` (name, method and URL, for search). `{{name}}` stays as written in them.

The values with variables filled in stay on the server, and reach a command only through a click; see below. `{{name}}` takes the file's `@name` value. Because a plain `{{name}}` never reads the environment, a file gets only the environment variables it names as `{{$processEnv NAME}}`, the VS Code REST Client's syntax. That keeps a token out of the file: `TOKEN=… tinkerdown serve …`. A variable's value can use other variables, so `@token = {{$processEnv TOKEN}}` lets requests write `{{token}}`. An environment value is used as it is. A variable that refers to itself, directly or through others, is an error, and so is a chain more than 10 deep. One read of the file fills in at most 64 MiB, and the requests past that say so. A file that doesn't parse sets the block's `.Error`, naming the line, and the last good rows stay.

## Runs a click starts

A merge with `params` and `on: NAME` starts its inputs when a button named NAME is clicked, never because a page loaded. Use it for a command that changes something, such as sending a request, which a reload, Back or a shared link must not repeat.

```yaml
sources:
  api: {type: stream, http: ./api.http}
  requests: {type: merge, inputs: [api], key: [fields.entry]}
  send: {type: stream, cmd: 'printf "%s" "$TD_headers" | curl -sS -i -X "$TD_method" -H @- ${TD_body:+--data-binary "$TD_body"} -- "$TD_url"'}
  response:
    type: merge
    inputs: [send]
    on: send
    line_max: 1MiB
    params:
      entry: {in: requests.fields.entry}
```

```html
<button name="send" data-entry="{{.fields.entry}}">Send</button>
```

The button can be in any block on the page. Its `data-*` values are the params, checked like URL values: each must pass the value rules and be in its `in` merge. The command then gets each param as `TD_<name>`, plus the values the row stands for, which the browser never sees. For an `http` row these are `TD_method`, `TD_url`, `TD_headers` and `TD_body`, with variables filled in. These values may span lines. Each may be up to 128 KiB, counted as the whole `TD_name=value`, since Linux refuses a longer environment string. A request with an unset variable, a longer value, or a value holding a NUL byte is refused with the reason in the block's `.Error`, and nothing runs.

The inputs run once per click: an input isn't restarted when it exits, and has no timeout of its own, so give the command one (`curl --max-time 30`). A click while it runs stops the run, clears its rows and starts again. A click within 250 ms of the last run a page started waits until 250 ms have passed, and a later click in that time replaces it. Clicks sent in a loop then repeat a request at most four times a second, and the last click still runs. The block's `.Running` is true while it runs. Everyone who clicks the same values shares one run. A clean exit adds no row, and a failed one adds a `system` row with the exit status.

A merge with `on` ignores the page URL's query, and its inputs must run a command. `on` is 1-32 lowercase letters, digits or `_`, and a builtin action name never starts one.

## Row fields

Each line becomes a row with these fields:

| Field | Value |
|-------|-------|
| `seq` | Number unique across all streams, in arrival order. Use it as `data-key`. |
| `source` | Name of the stream source the line came from. |
| `stream` | `stdout`, `stderr`, `file`, or `system` for rows about the input itself, such as a restart. |
| `ts`, `time` | Line time as RFC 3339, and as `15:04:05.000` for display. |
| `level` | `trace`, `debug`, `info`, `warn`, `error`, `fatal`, or empty. |
| `msg` | The message. |
| `line` | The raw line. |
| `truncated` | `true` when the line was longer than 16 KiB and was cut. |
| `fields` | Other fields parsed from a JSON or logfmt line. |

With `format: auto`, a JSON line or a logfmt line (`level=warn msg="slow query"`) fills `level`, `msg` and `ts` from its fields, and anything else is plain text. A leading RFC 3339 timestamp, as `docker logs --timestamps` adds, sets `ts`.

A plain-text line gets a level only from how it starts, so a line like "no errors found" stays unlabelled:

| Line starts with | Level |
|------------------|-------|
| A level word, after any timestamps: `ERROR`, `[warn]`, `WARN:` | that level |
| A syslog priority: `<3>` | from the priority |
| A glog prefix: `E1120 19:02:11.123` | from the letter |
| `panic:` or `fatal error:` | `fatal` |

## Showing a stream

Bind an `lvt` block to the source and key each row by `seq`:

```html
<ol lvt-fx:scroll="bottom-sticky">
  {{range .Data}}
  <li data-key="{{.seq}}" class="lvl-{{.level}} stream-{{.stream}}">{{.time}} {{.msg}}</li>
  {{end}}
</ol>
```

Because every row has the same markup and a key, each update sends only the new rows. Set per-row differences with attribute values, like the `class` above, and not with `{{if}}` inside the row. A row that changes shape forces a full redraw of the list.

Tinkerdown colours rows by these classes, so a page needs no CSS for them. `lvl-error` and `lvl-fatal` rows are red, and `lvl-warn` rows are amber. `lvl-debug`, `lvl-trace` and `stream-system` rows are muted, and system rows are italic. The colours follow the light and dark themes. To change them, override the rules or set `--tinkerdown-warn-color`. Because the names are generic, any element in an lvt block with a `lvl-*` class gets these colours. Colour alone is a weak cue, so show the level as text too, as in `<i>{{.level}}</i>`.

`lvt-fx:scroll="bottom-sticky"` follows new rows until the reader scrolls up, and resumes when they scroll back to the bottom.

## Pause, Resume and Clear

These buttons change one page's view and never the stream itself:

```html
{{if .Paused}}<button name="Resume">Resume</button>{{else}}<button name="Pause">Pause</button>{{end}}
<button name="Clear">Clear</button>
```

`Pause` stops updates to the page, and `Resume` catches it up. `Clear` hides the rows shown so far.

## Filtering

Filter streams like any source, with a [tabs heading](../guides/data-sources.md#filtering) or a `Filter` form. A level select and a search box work together:

```html
<form name="Filter">
  <select name="f.level" lvt-on:change="Filter">
    <option value="">any level</option><option>error</option><option>warn</option>
  </select>
  <input type="search" name="f.msg" lvt-on:input="Filter" lvt-mod:debounce="250">
</form>
```

## How inputs run

Every page showing the same input shares one process and one buffer, so ten open tabs run `docker logs -f` once. An input starts when the first page opens. It keeps running for 5 minutes after the last page closes, so a page reopened within that time still has its history.

A command runs in its own process group. Stopping the server kills the whole group, including anything the command started.

A `file` path can point anywhere the server's user can read, as with the `json` and `csv` sources. The page author picks the path, so only serve pages you trust.
