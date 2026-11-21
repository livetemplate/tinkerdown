# Stream and Merge Sources

A `stream` source follows a long-running command or a growing file, and the page shows each new line as it arrives. A `merge` source combines several streams into one timeline. Use them for logs: `docker logs -f`, `journalctl -f`, `kubectl logs -f`, a dev server's output, or `/var/log/app.log`.

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
| `cmd` | stream | Command to follow. Runs with `/bin/sh -c` from the site directory, so pipes, quoting and `./script.sh` work. Set `cmd` or `file`, not both. |
| `file` | stream | File to follow, absolute or relative to the site directory. |
| `inputs` | merge | Names of the stream sources to combine, or `["*"]` for every stream source the page can see, including `--source` flags. |
| `key` | merge | Field paths (`NAME` or `fields.NAME`) that make the merge keyed: one row per distinct key, updated in place. See [One row per key](#one-row-per-key). |
| `collect` | merge | With `key`: a field path whose text is gathered across a key's rows into `collected`. |
| `collect_reset` | merge | With `collect`: `PATH=VALUE`, where VALUE may list alternatives as `a\|b`. A row that matches starts its key's `collected` over, such as `fields.Action=run\|start` for a test's or package's next run. |
| `drop` | merge | With `key`: `PATH=VALUE`, where VALUE may list alternatives as `a\|b` (a value can't contain `\|`, and a number matches as it prints). A row that matches removes its key's row and isn't stored, such as `fields.state=removed` for a removed container. |
| `params` | merge | Values taken from the page URL's query, each with `in: MERGE.PATH`. See [Values from the page URL](#values-from-the-page-url). |
| `accepts` | merge | With `inputs: ["*"]`: the sources the page takes. `kinds` lists `cmd`, `file` or both (default: both), `min` is how many it needs, and `hint` is one line saying what to pass. |
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
tinkerdown serve --allow-exec --source 'api=docker logs -f --timestamps api' --source 'app=file:./logs/app.log'
```

`--source NAME=COMMAND` follows a command and needs `--allow-exec`; `--source NAME=file:PATH` follows a file. A flag also replaces a page or `tinkerdown.yaml` source of the same name, but never an approved one. `serve` refuses a flag no page uses, and one of a kind the page's `accepts` leaves out.

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

Because anyone can write a link, every param needs `in`. A value is accepted only while it equals `PATH` in a row of `MERGE`, another merge on the same page, so a link opens only what the page already lists. The check runs again when `MERGE`'s rows change: a link opened before its row arrives starts once the row appears. Without every param in the query, the block shows no rows and nothing runs. A value that is empty, over 256 bytes, starts with `-`, or holds a control character shows an error instead.

Each distinct set of values gets its own buffer and its own commands, which stop 30 seconds after the last page showing them closes. A `["*"]` merge on the page doesn't take the inputs a `params` merge lists, since they need its values. A `params` merge lists its inputs by name. Name it differently from its inputs: a `--source` flag replaces a page source of the same name.

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
