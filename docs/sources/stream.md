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
| `inputs` | merge | Names of the stream sources to combine. |
| `format` | stream | How to parse each line: `auto` (default), `json`, `logfmt` or `lines`. |
| `env` | stream | Extra environment variables for `cmd`. `$VAR` references are expanded. |
| `restart` | stream | `on-exit` (default) restarts an exited command after 1s, doubling up to 30s. `never` leaves it stopped. |
| `backlog` | stream | Lines read from the end of `file` on start (default: `200`). |
| `buffer` | both | Rows kept (default: `2000`). Older rows are dropped. |
| `flush` | both | Minimum time between updates to one page (default: `100ms`). |

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
