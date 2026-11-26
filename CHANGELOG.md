# Changelog

All notable changes to tinkerdown will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Earlier releases (v0.1.x) are documented in the
[GitHub releases page](https://github.com/livetemplate/tinkerdown/releases).


## [Unreleased]

### Added

- **Queries from a `.sql` file.** A stream with `sql: PATH`, or
  `--source sql:PATH`, lists the queries in a `.sql` file or in a folder's
  `.sql` files, one row per query, and reads them again when a file changes.
  A `-- name: NAME` header, as sqlc, aiosql and yesql write it, or HugSQL's
  `-- :name NAME`, starts each query, and a file with no header is one query
  named after the file. A click passes the query to your command as
  `TD_sql`; tinkerdown never runs the SQL itself. See
  [Queries from a .sql file](docs/sources/stream.md#queries-from-a-sql-file).

### Changed

- **A `.http` row's `entry` starts with its source's name**, as in `api/1`,
  where it was `1`. With two `.http` files on a page, Send ran the first
  file's request of that number, and a merge keyed on `fields.entry` showed
  one row for both. Markup that passes `{{.fields.entry}}` is unchanged; a
  test or link that names an entry such as `1` needs the new form.

### Fixed

- **`on:` refuses a built-in action's name.** A button named `run`, `refresh`
  or another built-in runs the built-in, so a merge with `on: run` never
  started. Config validation now says so.
- **Presentation mode no longer shows a page's CSS as text.** It set
  `display: block` on every element of the current slide. A `<style>`
  element then showed as text, and a grid lost its layout. Each element
  now keeps its own `display`.

## [v0.9.1] - 2026-11-25

### Fixed

- **A page from a URL runs its commands where you run `serve`.** Its
  `--source`, `--` and `--action` commands ran in the temporary directory
  that holds the fetched page, so `-- go test -json ./...` found no module.
  They now run in the directory `serve` runs in, where their `file:` paths
  already resolved, and a `--config` file's commands run in that file's
  directory (#386). A local page is unchanged.

## [v0.9.0] - 2026-11-25

### Added

- **Short names for pages on GitHub.** `tinkerdown serve
  github.com/OWNER/REPO[/PATH][@REF]` fetches the page at
  `https://raw.githubusercontent.com/OWNER/REPO/REF/PATH/index.md`, with the
  same checks as a URL. `REF` defaults to `HEAD`, and `#sha256=` pins it. The
  pin line offers the short name.
- **Plain commands.** Everything after `--` is one source command, so
  `tinkerdown serve --allow-exec page.md -- docker logs -f api` needs no
  quotes. `--source COMMAND`, `--source file:PATH` and `--source http:PATH`
  need no `NAME=`: such a source is labelled by its command's last plain word
  (`api`) or its file's name, with `-2`, `-3` when the label is taken, so it
  never replaces a page's source. `NAME=` still names a source, so
  `--source 'FOO=1 cmd'` is a source named `FOO`. To set a variable, write
  `--source 'env FOO=1 cmd'` or `-- FOO=1 cmd`.
- **`serve PAGE --help` lists what the page takes**: its `["*"]` merges with
  their kinds and hint, the sources its other merges read by name, and its
  buttons' actions. It needs no `--allow-exec` and starts nothing.

### Changed

- **`serve` refuses a flag it doesn't know.** It used to ignore one, so a
  mistyped flag ran the page without it. Flags that never existed, such as
  `--debug`, `--verbose` and `--production`, now fail; the guides that showed
  them are fixed. So does `--port=8080`, which was ignored too: serve says to
  write `--port 8080`. A `--port`, `--host`, `--config` or `--operator` with
  no value after it is refused instead of ignored.

- **`serve github.com/…` now fetches from GitHub.** Before, a directory named
  `github.com/…` in the current directory was served as a local path. It no
  longer is, so a checkout can't plant one to skip the remote page checks.
  Write `./github.com/…` to serve it locally.

## [v0.8.1] - 2026-11-24

### Fixed

- **A value that starts with `{{` and ends with `}}` shows as written.** The
  page showed such a value as empty, so a `.http` request's URL like
  `{{host}}/users/{{id}}` disappeared from the list. The embedded client is
  now `@livetemplate/client` 0.27.0, which renders every value as sent.

## [v0.8.0] - 2026-11-23

### Added

- **Runs a click starts.** A merge with `params` and `on: NAME` starts its
  inputs when a button named NAME is clicked, never when a page loads. So a
  reload, Back or a shared link can't repeat a command that changes
  something. The inputs run once per click, a click mid-run starts them
  again, and the block's `.Running` is true while they run. A click within
  250 ms of the page's last run waits for the 250 ms to pass.
- **`.http` files as a stream source.** `http: PATH` in config, or
  `--source NAME=http:PATH`, lists the requests in a `.http` file, one row
  per request, and follows the file as it's edited. Each request's filled-in
  method, URL, headers and body stay on the server and reach a command only
  through a click, as `TD_method`, `TD_url`, `TD_headers` and `TD_body`.
  Variables come from the file's `@name` lines, and environment variables
  only when named as `{{$processEnv NAME}}`. A variable's value can use
  other variables, so `@token = {{$processEnv TOKEN}}` works.
- **`line_max` on merges** raises the 16 KiB line cap, up to 16 MiB.
- **A `jsonindent` template function** indents a JSON object or array and
  returns other text unchanged.

### Fixed

- **Relative paths for a page from a URL.** A relative `file:` path in a
  `--source` flag pointed into the page's temporary directory. It's now taken
  from the directory `serve` runs in, and a relative path in a `--config`
  file from that file's directory.

## [v0.7.5] - 2026-11-22

### Fixed

- **Presentation mode no longer shows a blank screen on a page without `##`
  sections**, such as one whose only `##` is a tabs heading. Presentation
  mode shows one `##` section at a time, so a page with none had nothing to
  show. It's now one slide holding the whole page. Content before the first
  `##`, such as a tabs heading and its block, now shows on the first slide
  instead of on none. An `<h2>` inside a block no longer starts a slide,
  because its slide was blank too.

## [v0.7.4] - 2026-11-22

### Added

- **A `linkify` template function.** `{{linkify .msg}}` escapes a value and
  makes each `http://` or `https://` URL in it a link that opens in a new tab.
  Trailing punctuation stays outside the link. It's safe on text you don't
  control, such as log lines.

### Fixed

- **`go install github.com/livetemplate/tinkerdown/cmd/tinkerdown@latest`
  works again (#364).** Go leaves every directory named `vendor` out of a
  module's download. The embedded Prism, Mermaid, Chart.js and Pico files
  lived under `internal/assets/vendor/`, so the build from a download failed.
  They now live under `internal/assets/third_party/`, and a test checks the
  module's files against the rules the Go proxy uses.
- **Live updates for nested ranges, repeated values and maps** (livetemplate
  v0.25.13). A range whose items are each another range now renders. A word
  that appears twice in a list no longer makes every update resend the list.
  A range over a map keeps its keys in sorted order. `and` and `or` stop
  at the argument that decides them.
- **Ranging over a map nested in a row shows each key once.** tinkerdown adds
  a capitalised copy of each key so `{{.Name}}` reads `name`. It did that at
  every depth, so `{{range $k, $v := .fields}}` listed `ms` and `Ms`. Copies are
  now added only for state keys, a state map's keys and a row's own keys. A
  page that read a nested map under a capitalised name, such as `.user.Name`
  for JSON `name`, now gets nothing and must use `.user.name`.

### Changed

- The embedded client is now `@livetemplate/client` 0.25.12. It no longer
  ships prereview's attribute handlers, so `tinkerdown validate` now reports
  `lvt-spy`, `data-lvt-text-caret`, `data-lvt-text-select-button`,
  `data-lvt-url-hash` and `data-lvt-viewport-items` as unknown.

## [v0.7.3] - 2026-11-22

### Fixed

- **Reopening a log no longer shows its lines twice.** A merge with `params`
  kept its rows for 5 minutes after the last viewer left, but its inputs
  stopped after 30 seconds. Opening the same item again within those minutes
  restarted the input, which printed its output into rows the merge still
  held. A merge now drops its rows when its shortest input does, and starts again
  from its inputs' rows when it restarts. This affected
  the containers app's log pane, and the CI app's failed log.

## [v0.7.2] - 2026-11-22

### Added

- **`sort` on keyed merges.** `sort: -fields.createdAt` shows rows newest
  first, instead of in first-seen order. Numbers, and text that reads as a
  number, compare as numbers; other values compare as text. A new row
  reaches the page as an insert at its place, so screen readers and keyboard
  focus follow the order on screen. See
  [docs/sources/stream.md](docs/sources/stream.md#one-row-per-key).

## [v0.7.1] - 2026-11-22

### Added

- **A button shows `aria-busy` while its action runs.** The client tags each
  action with a ref, and the server sends it back on the update after the
  action. Until then the button (or `lvt-on:click` element, or form) is
  `aria-busy="true"`, which draws a spinner, and a second click sends
  nothing. See
  [docs/reference/lvt-attributes.md](docs/reference/lvt-attributes.md#aria-busy-while-an-action-runs).
- **`.Params` on a merge with `params`.** The block's state holds the values
  it took from the page URL, so `{{with .Params.name}}` can name what it
  shows. See
  [docs/sources/stream.md](docs/sources/stream.md#values-from-the-page-url).

### Changed

- **One theme button instead of three.** The toolbar's light, dark and auto
  buttons are now a single button that shows the current theme's icon. It opens
  a Light / Dark / System menu, upward when the toolbar sits in the sidebar
  footer. Ctrl+Shift+D still cycles the themes.

## [v0.7.0] - 2026-11-22

### Added

- **`serve --action NAME=COMMAND`.** A button named NAME on any page, including
  a page from a URL, runs COMMAND. The button's `data-*` values arrive as
  `$TD_<name>` environment variables, never pasted into the command, and a
  value that is empty, over 256 bytes, starts with `-` or holds a control
  character runs nothing. See
  [docs/reference/cli.md](docs/reference/cli.md#actions-from-the-command-line).
- **Merges with `params`.** `params: {name: {in: containers.fields.name}}`
  takes `?name=web` from the page URL and starts the merge's inputs with
  `TD_name=web`, so a row link opens that item's stream. A value is accepted
  only while it is in a row of the merge `in` names, and each set of values
  gets its own buffer, stopped 30 seconds after its last viewer. A `["*"]`
  merge doesn't take a `params` merge's inputs. On a page with a `params`
  merge, a link that only changes the query updates the values without a
  page load, and Back and Forward do the same. See
  [docs/sources/stream.md](docs/sources/stream.md#values-from-the-page-url).
- **`drop` on keyed merges.** `drop: fields.state=removed` removes a key's
  row when a matching row arrives, and the page gets a remove for it. Values
  may list alternatives as `a|b`. See
  [docs/sources/stream.md](docs/sources/stream.md#one-row-per-key).

### Fixed

- **A log pane that opens with many lines now follows new ones.**
  `lvt-fx:scroll="bottom-sticky"` stopped following when one update added
  many rows, such as a `params` merge opening a stream with its backlog.
  Fixed in `@livetemplate/client` 0.25.11, which tinkerdown now uses.
- **`serve` no longer hangs after a page edit while a tab is open.** An open
  WebSocket held the server's route lock for as long as the tab stayed open.
  A page edit then waited for that lock to rescan the pages, and every page
  request queued behind the rescan. The lock is now held only to find the
  connection's page.
- **A failed action's error shows on the page.** The block re-renders after a
  failed action, so the error it set is no longer lost.

## [v0.6.0] - 2026-11-21

### Added

- **Keyed merges: one row per key, updated in place.** A merge with
  `key: [fields.Package, fields.Test]` keeps the latest row for each key in
  first-seen order, and `collect: fields.Output` gathers a field's text across
  a key's rows; `collect_reset: fields.Action=run` starts it over per run. Keyed rows have a stable `key` for `data-key`, so updates
  reach the page as in-place range updates. See
  [docs/sources/stream.md](docs/sources/stream.md#one-row-per-key).
- **Every stream row has a `fields` map**, empty for plain-text and system
  rows, so a template can read `.fields.NAME` on any row. A template that did
  failed to render on the first plain-text row.
- **Field paths in filters.** `fields.Action = fail` reads a parsed JSON or
  logfmt field, in filter expressions, tabs headings and `f.` form fields.

## [v0.5.0] - 2026-11-21

### Added

- **`tinkerdown serve --source` adds stream sources from the command line.**
  `--source NAME=COMMAND` follows a command (with `--allow-exec`) and
  `--source NAME=file:PATH` follows a file. A merge with `inputs: ["*"]`
  takes every stream source, `accepts` declares what a page takes, and a
  tabs heading with `[*] source` shows one tab per source. When a page needs
  sources and has none, `serve` asks for them in a terminal. See
  [docs/sources/stream.md](docs/sources/stream.md#sources-from-the-command-line).
- **`tinkerdown serve` takes one page: a `.md` path, a `file://` URI or an
  `https://` URL.** A page from a URL may only declare merge sources, is
  checked for scripts and other unsafe markup, and is served with a strict
  Content-Security-Policy. `#sha256=` pins its content. See
  [docs/reference/cli.md](docs/reference/cli.md#pages-from-a-url).
- **`tinkerdown serve --help`** prints serve's own usage. It used to start
  serving the current directory.

- **`stream` and `merge` sources follow logs and other live output.** A
  `stream` source runs a long-lived command (`cmd:`, with `--allow-exec`) or
  follows a file (`file:`), and the page shows each new line as it arrives.
  A `merge` source combines streams into one timeline. Every page showing an
  input shares one process. Lines are parsed as JSON, logfmt or plain text.
  See [docs/sources/stream.md](docs/sources/stream.md).
- **Stream rows are coloured by level with no page CSS.** Rows with
  `class="lvl-{{.level}} stream-{{.stream}}"` get theme-aware colours for
  error, warn, debug and system rows.
- **`Pause`, `Resume` and `Clear` buttons** for stream and merge blocks. They
  change one page's view only.
- **Filters gain `~` and `and`, and `Filter` forms gain field filters.**
  `msg ~ timeout` keeps rows containing the text, ignoring case, and clauses
  join with `and`. A field named `f.<field>` adds a substring filter that
  combines with the selected tab.

### Fixed

- **A block whose source fails to build shows the error** instead of
  "Connecting..." forever. Only validation messages reach the page; the server
  log has the full error.
- **A server on a loopback address refuses requests for other host names**,
  so a website can't use DNS rebinding to read a local tinkerdown page.

- **`lvt-on:input`, `lvt-mod:debounce` and `lvt-mod:throttle` now work in
  blocks,** as `lvt-attributes.md` documents. `lvt-on:change` now also sends
  the value under the field's `name`, so several fields can feed one action.
- **Closing a page now closes its sources.** Each connection's sources, such
  as a sqlite database handle, were left open after the page closed.

## [v0.4.3] - 2026-11-18

### Fixed

- **An `lvt-on:*` click in an embedded app before its WebSocket opens is no
  longer lost.** The bundled browser client moves to `@livetemplate/client`
  0.25.9 (from 0.25.8), and the Go module to
  `github.com/livetemplate/livetemplate` v0.25.9 (from v0.25.7). The built-in
  `lvt-on:*` handlers now listen before the handshake and their sends wait for
  it, as custom `lvt-x:*` handlers have since v0.4.2.

## [v0.4.2] - 2026-11-17

### Fixed

- **An unknown page is a 404, not the first page.** serve redirected any path
  it had no route for to its first page. In a site with navigation, that hid a
  page missing from `tinkerdown.yaml`: its URL showed the home page with a 200.
  `/` still goes to the first page when there is no `index.md`. A path that
  names a page another way (a trailing slash, `.html`, `.md`, or `/index`)
  redirects to that page. The WebSocket endpoint also 404s an unknown page
  instead of serving the first page's handler.
- **A click in an embedded app before its WebSocket opens is no longer lost.**
  The bundled browser client moves to `@livetemplate/client` 0.25.8 (from
  0.25.6). The Go module moves to `github.com/livetemplate/livetemplate` v0.25.7
  (from v0.25.5). Custom `lvt-x:*` handlers in an `embed-lvt` app now arm on the
  server-rendered markup before the handshake, and their sends wait for it.
  Before, on a remote site the page was clickable before any handler listened.

### Added

- **`tinkerdown validate` fails on a page no navigation entry lists.** In a
  site with explicit navigation, serve routes only listed pages, so an unlisted
  `.md` file ships unreachable while every other check passes. Pages under a
  `_` directory are skipped, since they are only included.

## [v0.4.1] - 2026-11-16

### Changed — embedded client 0.25.6 and livetemplate v0.25.5

The bundled browser client now builds on `@livetemplate/client` 0.25.6 (up from
0.11.9), and the Go module requires `github.com/livetemplate/livetemplate`
v0.25.5 (up from v0.21.0). An embedded livetemplate app (`embed-lvt`) can now
use custom `lvt-x:*` attributes: the client reads handlers queued on
`window.lvtAttributes` and passes `ctx.signal` to each one. Put the handler
script inside the app's `<body>`, because `embed-lvt` inlines only the
`data-lvt-id` wrapper and drops scripts in `<head>`. v0.25.5 also keeps a
`<script>` at the start of `<body>` inside the wrapper. Before it, such an app
rendered a second copy of its page after the first update.

## [v0.4.0] - 2026-07-24

The ephemeral-UI reframe (M0–M5): Tinkerdown becomes the tool teams use to let an
LLM generate contextual, throwaway internal UIs against an **approved surface** —
governed at generation *and* runtime, on-brand by construction, saveable when a
workflow recurs. Everything below shipped since v0.3.9.

### Fixed — `tinkerdown cli` reaches WASM sources; wasm docs use the real config keys

`tinkerdown cli` now constructs `wasm` sources, so a WASM source has the same CRUD
surface on the CLI as under `serve`: `list` works for any module, and
`add`/`update`/`delete` work when the module exports `write` (a read-only module
refuses a write with a clear error rather than dropping it). `exec` remains
intentionally out of the CLI CRUD surface (it is a read-oriented, `--allow-exec`-gated
command runner), now documented in the code rather than a silent fallthrough. The
WASM source docs and the `wasm-source` template were also corrected: they used a
non-existent `module:` key (the real key is `path:`) and `config:` (real: `options:`),
so following them failed to load — plus a new *Read-only vs. writable* section
documents the `write`-export contract and the CLI parity.

### Added — a saved-skills gallery (`examples/gallery/`)

A discoverable home for captured workflows: the gallery is itself a plain
Tinkerdown app (an `lvt-source` over a read-only CSV, no custom JavaScript) that
lists each captured skill with what it stands up and where its stand-up steps
live. Because no source type reads a directory of `SKILL.md` frontmatter, the
gallery renders a committed static index (`skills.csv`) rather than scanning
`skills/` live; a test (`TestSavedSkillsGalleryInSync`) keeps that index in step
with the captured skills on disk — every `skills/` dir except the
framework-authoring skills — so a new capture cannot silently go unlisted.

### Changed — a saved skill now carries its house style

`/tinkerdown:save` captures the manifest **whole**, not trimmed to sources +
actions: the `styling` block (theme + design `tokens`) and any
`generation.style_guide` → `style-guide.md` travel with the capture, under the
same bundle-or-point rule as `seed.sql` / `app.md`. So a saved workflow re-runs
**on-brand** instead of falling back to bare defaults — closing the gap where a
capture kept a console's data surface but dropped its palette. The PII reference
skill demonstrates the point case: it points at the committed
`examples/pii-access-approval/`, whose manifest carries the house style, and a
test now guards that the style file travels and the manifest still round-trips
its `styling.tokens` + `generation.style_guide`.

### Changed — the generation skill now consumes the house style

The `/tinkerdown` generation skill reads a project's house style and authors
against it, closing a declared-but-unconsumed gap: `generation.style_guide` (a
markdown file of house tone/layout/do-don't) was a config field nothing ever
read. The skill now reads it — plus the `styling` block (`theme` +
`styling.tokens`) — and carries the on-brand rule into generation: **write
semantic HTML, never hardcode colours** and let the design tokens skin it, follow
the project's `style-guide.md`. The token vocabulary is documented in the skill
`reference.md` (guarded against `KnownStyleTokens` so a new token can't go
untaught). The whole style block stays optional — with none, generation falls
back to on-brand defaults. The PII reference app now ships a `style-guide.md` +
`styling.tokens` demonstrating a governed, on-brand console.

### Added — declarative design tokens (`styling.tokens`)

A project can now set its on-brand palette declaratively, without writing raw
CSS. `styling.tokens` in `tinkerdown.yaml` maps snake_case design-token names to
values — e.g. `accent: "#5a67d8"`, `card_bg: "#ffffff"` — and each drives the
matching CSS custom property (`--accent`, `--card-bg`, …) in every page's
`:root`, overriding the built-in default via the same mechanism `primary_color`
already uses for `--accent`. Because the override skins the *semantic* HTML the
generator emits, a generated UI is on-brand by construction rather than by
careful prompting. An unknown token key fails loudly at config load (naming the
key and listing the known tokens) so a typo can't silently skin nothing; token
values are sanitized against CSS-injection at render. Absent, built-in defaults
apply — existing projects are unaffected. A token override is a single value that
applies to **both** light and dark themes (same as `primary_color`); per-theme
palettes are not yet expressible.

### Added — runtime approved-surface enforcement

When a project declares a `generation:` block, Tinkerdown now enforces the
approved surface at **runtime**, not only at generation time: a running app — or
a webhook / crafted-message caller — may only invoke an **approved** action, and
a builtin write may only touch an **approved** source. The check runs on every
path a caller can reach (WebSocket custom actions, WebSocket builtin writes, and
the webhook endpoint, which share no chokepoint), so a caller bypassing the
browser cannot reach past the surface the operator approved — the server-side
gate the client-only `confirm:` never was. Projects with no `generation:` block
are unaffected (approval is opt-in).

### Changed — `validate` catches unresolved sources and incomplete action params

`tinkerdown validate` now reports two more "passes validate, breaks at serve"
classes:

- **Bound references** — a source bound via `lvt-source="…"` that resolves to no
  declared source (a typo) is reported with the list of declared sources, instead
  of erroring only at serve as "source not found". The declared universe is the
  file's frontmatter sources plus the tinkerdown.yaml governing it.
- **Action-param completeness** — for a `kind: sql` action a form invokes, every
  `:param` its statement references must be supplied by a form field or a `data-*`
  attribute (`:operator` is server-set and exempt); a missing one is named,
  instead of erroring at serve on the first substitution.

The state-ref diagnostic ("interactive block has no state reference") now teaches
the real fix — add `lvt-source` to the block's container — rather than an
undefined `state="block-id"`.

### Changed — `lvt-persist` reports a migration hint

Using `lvt-persist` now reports "unknown attribute — use lvt-source" instead of a
confusing "no state reference": Tinkerdown's server no longer supports the persist
model (use `lvt-source` with `type: sqlite` for a writable store).

### Changed — `validate` now checks that every lvt block compiles as a template

`tinkerdown validate` runs each lvt code block through the template parser
(livetemplate `v0.21.0`'s new `Validate`), so a template-syntax or composition
error inside a block — an unclosed `{{range}}`, an unknown function, an
unresolved component — is now reported with a line and message instead of
surfacing only at serve time as a block that silently renders nothing.
Validation resolves against the same component templates and helper functions
serve uses, so a block that renders under serve validates clean, and vice versa.

### Added — `split` block helper

Templates in an lvt block can call `{{split "a, b, c" ", "}}` to turn a
delimited string into a slice (e.g. comma-separated tags into individual values
for a `{{range}}`) — Tinkerdown's first base block helper, available to every
block at both parse and render.

### Added — Atomic multi-statement SQL actions (`statements:`)

A `kind: sql` action may now carry a `statements:` list instead of a single
`statement:`; the statements run in one transaction — all commit, or the first
error rolls all of them back:

```yaml
approve-export:
  kind: sql
  source: access_requests
  statements:
    - "INSERT INTO exports (...) SELECT ... FROM orders_pii LIMIT (SELECT row_cap FROM access_requests WHERE id = :id)"
    - "INSERT INTO audit_log (...) SELECT ... FROM access_requests WHERE id = :id"
    - "UPDATE access_requests SET status = 'approved' WHERE id = :id"
```

This exists so an action can change state **and** append its audit record
atomically — a partial success (state changed, audit missing) is impossible. An
action must set exactly one of `statement`/`statements`; both, or neither, is a
config error caught at load.

### Added — PII / data-export access-approval reference app

`examples/pii-access-approval/` — the M1 reference console: a request queue where
**Approve** runs a bounded, server-authoritative export (the row cap is read from
the request row, not sent by the client) plus a durable audit record, and **Deny**
records a decision without granting access. Built entirely on the manifest +
existing primitives.

### Fixed — policy lint no longer flags built-in source affordances

`tinkerdown validate` treated a writable source's own `Add`/`Delete`/`Toggle`/
`Refresh` controls as unapproved *actions*. These are intrinsic affordances,
governed by whether the source is approved and writable — not by the
approved-action set — so an approved writable source's Add form now lints clean.

### Added — `generation:` block: an approved surface for LLM-generated apps

A `tinkerdown.yaml` may now declare which of its sources and actions an LLM is allowed
to wire up when generating an app against the project:

```yaml
generation:
  sources: [requests]
  actions: [approve-request]
  style_guide: ./style-guide.md   # optional
```

Sources and actions also accept a `describes:` note — no runtime behavior, purely the
human-readable summary an operator reads when reviewing what a generated app does.

Approval is enforced as a **precedence tier**, not a prohibition. The documented
frontmatter-wins-over-config rule gains a top tier: an approved name is *pinned*, so a
page cannot redefine it. This is necessary because a page's frontmatter can declare its
own sources and actions — without pinning, a generated page could reference an approved
name while defining that name as something the operator never approved, defeating any
check that reasons about names alone. Attempts are logged rather than silently dropped.

Approval also makes site-level **actions reachable**. Previously a page could only
invoke actions declared in its own frontmatter, so an approved-actions list would have
named a surface no generated page could use. The fallback is limited to *approved*
actions: actions written for schedules or webhooks stay unreachable from pages.

An approved name that refers to nothing is rejected at config load. Such an entry would
otherwise be silently inert — and since approval is what pins a name, a typo in
`generation.sources` would *remove* a protection while appearing to add one.

**Projects without a `generation:` block are unaffected.**


### Changed — Upstream bump: livetemplate v0.10.0 → v0.19.1

Tinkerdown had been pinned to `livetemplate v0.10.0` while upstream shipped
nine minor releases. This adopts the current latest across the stack:

| Dependency | Was | Now |
|---|---|---|
| `github.com/livetemplate/livetemplate` | v0.10.0 | **v0.19.1** |
| `github.com/livetemplate/lvt/components` | pseudo-version `2026-02-28` | **v0.2.0** |
| `@livetemplate/client` | 0.14.3 | **0.18.2** |

The client version is not merely "latest" — livetemplate v0.18.0 added a
`ClientVersion` constant precisely because there is **no runtime server↔client
version handshake**, and v0.19.0 declares `0.18.2` as the wire-compatible pair
for this server release.

No source changes were required: tinkerdown's livetemplate API surface (10
symbols) is untouched by the range, and the one API removed upstream
(`WithStore`, v0.19.0) was never used here.

Behavior inherited from the bump, most consequentially two silent-update-loss
fixes in v0.18.1 (a range whose item statics changed had its update dropped,
leaving stale items on the page until a full reload; and a nil `lvt:"persist"`
field discarded the entire restored state on reconnect), plus v0.19.0's
per-item recursive range diffs, v0.18.0's client pinning, v0.17.0's
`WithParseFS` and scoped method precompute, and v0.16.0's `__ping__` liveness
heartbeat.

### Fixed — `make build` no longer rebuilds the client from stale `node_modules`

`build-client` ran `npm run build` with no install step, so it bundled whatever
happened to be in `client/node_modules` rather than what the lockfile pins. On a
checkout whose `node_modules` had drifted, this silently regenerated the
committed bundle from a stale dependency and reverted shipped fixes — including
the `data-lvt-force-update` handling that the checkbox e2e tests depend on. CI
never caught it because CI always installs fresh. Now runs `npm ci` first, which
installs exactly the lockfile and fails loudly on drift. ([#295])

[#295]: https://github.com/livetemplate/tinkerdown/issues/295


## [v0.2.1] - 2026-05-09

### Added — Site-wide include resolution

`tinkerdown serve` now resolves `include="..."` paths against the
**site root** (the content directory passed to `serve`) instead of
each markdown file's own parent directory. Cross-page references like
`include="../recipes/counter/_app/counter.go"` from a page in
`getting-started/` now succeed; paths that escape the site root are
still rejected.

Library API:

- New `tinkerdown.ParseFileInSite(path, siteRoot)` — uses `siteRoot`
  as the include-confinement boundary. Pass `siteRoot=""` to fall
  back to page-root confinement.
- Existing `tinkerdown.ParseFile(path)` is unchanged — single-page
  callers (CLI tools, library users rendering one .md file) keep the
  v0.2.0 page-root-confined default.

Motivation: the v1 page-root confinement (per the comment in
`page.go:103-109` of v0.2.0) made it impossible for a multi-page docs
site to maintain a single source of truth for code samples shared
across pages. The classic case: a "Getting Started" tutorial and a
"Recipes" deep-dive that both want to include from the same
deployable `_app/`. Site-wide root resolution unblocks that without
removing the security boundary — paths still must stay under the
site directory.

The `serve` and `validate` commands automatically use `ParseFileInSite`
with the configured content directory; no per-page frontmatter is
needed.


## [v0.2.0] - 2026-05-07

### Added — Literate authoring primitives

A toolkit for documentation that *shows the real code, running*. Three
composable primitives let an author pair source listings with live
widgets without any drift between docs and production.

#### `show-source` / `hide-source` flag on `lvt` blocks

Add `show-source` to a fenced ` ```lvt ` block to render the template
both as a syntax-highlighted listing and as the live, interactive
widget it produces. Page-level default via `lvt_show_source: true`
in frontmatter; individual blocks can opt out with `hide-source`.

#### `embed-lvt` block

Embeds a deployed LiveTemplate app inline. Tinkerdown fetches the
upstream HTML server-side at render time so the page paints fully
on first paint; the WebSocket attaches afterwards for live updates.

Attributes:
- `path=` — docs-side path to expose (default `/`)
- `upstream=` — HTTP origin of the deployed app; auto-registers a
  reverse-proxy from `path=` (no `tinkerdown.yaml` route entry needed)
- `server=` — alternative cross-origin direct mode (no proxy);
  remote app must include the docs origin in its `AllowedOrigins`
- `session=` — authoring intent for linked widgets (real cross-region
  state sharing comes from the upstream app's `BroadcastAction` calls
  + a constant-groupID authenticator, not from this attribute)
- `timeout=` — server-side fetch deadline (default `2s`)
- `height=` — CSS `min-height` for layout stability
- `show-source` — display upstream HTML alongside the live preview

#### `include="..."` fence attribute

Slices a real source file into a rendered code block:

- `lines="N-M"` — single 1-indexed inclusive range; auto-clamps if the
  file shrinks below the upper bound
- `lines="N-M,P-Q"` — multiple ranges, joined with a language-aware
  ellipsis comment (`// ...` for Go, `# ...` for Python/YAML,
  `<!-- ... -->` for HTML, etc.)
- `region="name"` — extracts content between
  `// >>> region:name` / `// <<< region:name` markers; survives
  line-number drift across edits
- `highlight="N-M"` — Prism Line Highlight overlay; numbers are
  file-absolute and match the rendered gutter

#### Auto source-link footer

Each `include=` block automatically renders a `counter.go:13-35`
footer link to the cited file at the cited range. Activated by
frontmatter:

```yaml
source_repo: https://github.com/yourorg/yourrepo
source_path: examples/literate-counter/index.md
```

Default git ref tracks the running tinkerdown binary's release
version (set via build-time ldflags); override per-page with
`source_ref:`. For `dev` builds, the ref falls back to `main`. URL
schemes are restricted to `http(s)://` to prevent XSS via author-set
`javascript:` or `data:` URLs.

#### Path confinement, dedent, and hot reload

- Include paths are resolved relative to the markdown file's
  directory, canonicalised (symlinks followed), and confined to
  that tree. Escape attempts (`../../etc/passwd`) are rejected with
  a warning; the page still renders.
- The longest common leading whitespace across non-blank lines is
  stripped, so a function-body slice renders flush-left without
  mangling its relative indentation.
- The file watcher tracks every included file via `SetExtraFiles`.
  Editing `_app/counter.go` while the page is open auto-refreshes
  the rendered snippet (and the source-link footer).
- File-size cap (`maxIncludeBytes = 10 MB`) on `include=` reads
  guards against accidentally pulling vendored binaries or large
  logs into the docs build.

#### Vendored Prism plugins

Prism Line Numbers + Line Highlight are vendored from prismjs@1.29.0
and loaded only on pages where `len(IncludedFiles) > 0`. Pages
without source-includes pay zero asset cost.

### Added — Reference examples

- `examples/literate-counter/`, `examples/literate-linked/` —
  `show-source` on inline `lvt` blocks bound to markdown data sources
- `examples/literate-counter-include/` — `include=` slicing a real
  `_app/counter.go` paired with `embed-lvt`; canonical A2 shape
- `examples/literate-linked-include/` — A3 pattern: two `embed-lvt`
  blocks sharing one upstream session via constant-groupID
  authenticator + `BroadcastAction` in handlers
- `examples/embed-counter/`, `examples/embed-shared-session/`,
  `examples/mixed-tracks/` — embed-only and combined-tracks examples
- In-tree authoring guide at `docs/guides/literate-docs.md` covering
  `show-source`, `include=`, and pairing with `embed-lvt`

### Fixed

- Embed event-delegator collision when two `embed-lvt` blocks point
  at the same upstream — wrapper IDs are now suffixed with the
  per-block id so livetemplate's event delegator routes correctly to
  each region.
- CommonMark-compliant fence opener recognition in the include
  preprocessor — `include=` fences inside lists and quotes are now
  resolved (previously skipped).
- Race in `isIncludedFile` watcher callback — acquire `s.mu.RLock`
  around page-route iteration; short-circuit on `.md` events so
  watcher doesn't redundantly scan the include set during Discover.

[v0.2.1]: https://github.com/livetemplate/tinkerdown/compare/v0.2.0...v0.2.1
[v0.2.0]: https://github.com/livetemplate/tinkerdown/compare/v0.1.20...v0.2.0
