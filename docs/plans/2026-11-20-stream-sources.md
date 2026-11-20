# Plan: streaming sources in tinkerdown, and a log viewer built on them

State — Done: Phases 0–4 (implementation PR open; plan PR awaiting signoff). Next: review, then `detectLevel` (user contribution).

## Context

The goal is a log viewer that streams logs from tailed files, docker containers and long-running commands (with stdout and stderr tagged). It is built as a tinkerdown app in `logviewer/`.

Tinkerdown cannot stream today, for three reasons:
- **Every source is one-shot.** `Source.Fetch` returns a whole slice (`internal/source/source.go:17`).
- **The exec source can't follow output.** `exec` waits for the process to exit and has a 30s timeout (`internal/source/exec.go:122`).
- **The server pushes only for markdown sources.** It pushes to the browser only when a `.md` file used by a `type: markdown` source changes (`internal/server/websocket.go:262`, `watcher.go:95`).

This milestone (M1) adds three things to tinkerdown: a streaming source primitive, a server push path for it, and richer filters. It then builds the log viewer on top. Scope was settled with the user:
- inputs are spawned commands (docker via `docker logs -f`) and tailed files
- docker containers come from a list in config
- M1 includes a merged "All" view

Livetemplate already has stream-mode ranges (`livetemplate/docs/proposals/streaming-range-rendering-proposal.md`, status Implemented). A keyed, homogeneous `{{range}}` sends only insert/append/remove ops, and the server keeps a hash per item instead of the item. So tinkerdown needs no new wire protocol, only a buffer and a push trigger.

## Product

The log viewer is one page. It has a tab per source plus an "All" tab, a filter toolbar, and a log list that follows the tail.

```
┌ Logs ─────────────────────────────────────────┐
│ [All] [web] [db] [app.log]                    │
├───────────────────────────────────────────────┤
│ Level [any ▾]   [🔍 text                   ]  │
│ [⏸ Pause]  [Clear]               ● live 2000  │
├───────────────────────────────────────────────┤
│ 17:02:11 web  INFO  GET /api/users 200 12ms   │
│ 17:02:11 db   WARN  slow query 812ms          │
│ 17:02:12 web  ERR!  panic: nil map  stderr    │
│ 17:02:12 app  INFO  job 41 done               │
│ 17:02:13 web  ···   restarted (exit 1) system │
│                                    ↓ following│
└───────────────────────────────────────────────┘
```

Each row shows its time, source, level, message, and the stream it came from when that isn't stdout. Level sets the row's colour, and stderr and system rows are marked. While paused, the `● live` indicator reads `⏸ paused`, and new rows wait until Resume.

On a phone, the toolbar wraps onto two lines, and each row drops the source column and puts the time on a line of its own:

```
┌ Logs ───────────────────┐
│ [All] [web] [db] [app]  │
│ Level[any▾] [🔍       ] │
│ [⏸] [Clear]   ● live    │
├─────────────────────────┤
│ 17:02:12 ERR! stderr    │
│ panic: nil map          │
│ 17:02:12 INFO           │
│ job 41 done             │
└─────────────────────────┘
```

| User journey | Screen element | Built on |
|---|---|---|
| Watch logs arrive live | log list, `↓ following` | `stream` source, hub push, `lvt-fx:scroll="bottom-sticky"` |
| Read back without being yanked to the bottom | scrolling up stops following | `bottom-sticky` |
| See every source in one timeline | All tab | `merge` source |
| Focus on one source | per-source tab | tinkerdown filter tabs (`source = web`) over the `all` block |
| Find errors | Level select | `f.level` filter |
| Search for text | 🔍 box (debounced) | `f.line` filter, `~` operator |
| Freeze the view to read it | ⏸ Pause / ▶ Resume | `Pause`/`Resume` actions |
| Start from a clean screen | Clear | `Clear` action (per tab only) |
| Notice a crashed process | `system` row | `cmd` restart with backoff |

### The tinkerdown files

The whole app is two files. Sources live in `tinkerdown.yaml`, and the page is one `lvt` block under a tabs heading. Tinkerdown already turns `## [All] | [web] source = web` into tabs, and clicking one sends `Filter {filter: "source = web"}` to the block below (`parser.go:761`, `client/src/core/tabs.ts`). So the per-source tabs need no new code.

`logviewer/tinkerdown.yaml`:

```yaml
sources:
  web: { type: stream, cmd: "docker logs -f --timestamps web" }
  db:  { type: stream, cmd: "docker logs -f --timestamps db", restart: never }
  app: { type: stream, file: ./logs/app.log, format: json, backlog: 500 }
  all: { type: merge, inputs: [web, db, app], buffer: 5000 }
```

`logviewer/index.md`:

~~~markdown
---
title: Logs
---

## [All] | [web] source = web | [db] source = db | [app] source = app

```lvt
<main lvt-source="all">
  <form name="Filter" class="toolbar">
    <select name="f.level" lvt-on:change="Filter">
      <option value="">any level</option>
      <option>error</option><option>warn</option><option>info</option><option>debug</option>
    </select>
    <input type="search" name="f.line" placeholder="search"
           lvt-on:input="Filter" lvt-mod:debounce="250">
  </form>
  {{if .Paused}}<button name="Resume">▶ Resume</button>{{else}}<button name="Pause">⏸ Pause</button>{{end}}
  <button name="Clear">Clear</button>
  <span class="status">{{if .Paused}}⏸ paused{{else}}● live{{end}} · {{len .Data}}</span>
  {{if .Error}}<p><mark>{{.Error}}</mark></p>{{end}}

  <ol class="log" lvt-fx:scroll="bottom-sticky">
    {{range .Data}}
    <li data-key="{{.seq}}" class="lvl-{{.level}} stream-{{.stream}}">
      <time>{{.time}}</time> <b class="src">{{.source}}</b>
      <i class="lvl">{{.level}}</i> <span class="msg">{{.msg}}</span>
    </li>
    {{end}}
  </ol>
</main>
```

<style>
.log .lvl-error, .log .lvl-fatal { color: var(--danger); }
.log .lvl-warn { color: var(--warning); }
.log .stream-system { opacity: .6; font-style: italic; }
@media (max-width: 600px) { .log .src { display: none; } .log time { display: block; } }
</style>
~~~

The `{{if}}`s sit outside the range, so every row still uses the same template and the range stays in stream mode. The phone layout is a CSS media query for the same reason. A per-row `{{if}}` would break stream mode.

## Decisions

- New source type `stream` with exactly one input: `cmd:` (long-running process) or `file:` (native tail). Setting both is a validation error.
  alternative: extend `exec` with a `follow: true` flag
  because: exec's semantics (exit, timeout, `manual`/Run, `FetchWithArgs`) conflict with a process that never exits
- New source type `merge` with `inputs: [a, b]`, which interleaves the named stream sources by arrival order.
  alternative: `stream` taking a list of inputs
  because: the page needs one block holding every source, and each input still needs its own buffer so another page can show it alone
- Each server owns one `stream.Hub`. The hub keeps one ring buffer per stream source, keyed by source name plus a hash of the resolved config, so the same name configured differently on two pages gets two buffers.
  alternative: one buffer per connection (what today's per-connection factory would produce)
  because: N tabs would mean N `tail -f` processes, each with its own history
- The hub is passed into the runtime through a new option on `runtime.NewGenericStateWithMetadata`, not a package global.
  alternative: a package-level default hub
  because: e2e tests run several servers in one process, and one server's shutdown would kill the others' buffers
- An input starts the first time any connection subscribes to it, and stops 5 minutes (`Hub.IdleStop`) after its last subscriber leaves, dropping its rows. A merge's subscribers hold its inputs.
  alternative: run until the server shuts down (the original decision)
  because: Phase 2's security review showed that an abandoned input, or a `cmd` edited under `--watch`, would run forever. A short idle period still keeps history across a quick tab reload.
- When a `cmd` exits: restart with backoff (1s doubling to 30s), and insert a `stream: system` row that says why. `restart: never` turns this off.
  alternative: none considered
- On shutdown, kill the whole process group (`Setpgid`, then signal `-pgid`).
  alternative: none considered
- `cmd` runs via `/bin/sh -c`, so pipes and quoting work (`docker logs -f x 2>&1 | grep -v health`). It requires `--allow-exec`, the same as `exec`.
  alternative: `strings.Fields`, as exec does today
  because: exec's splitting breaks the quoted `--format` strings that docker and journalctl commands need; the config author already controls the command
- A push sends a snapshot. When the buffer changes, each subscribed connection runs `Refresh`, which re-fetches the ring snapshot, then `sendUpdate`. The stream-mode diff turns this into append and remove ops.
  alternative: push only the new rows to the client
  because: a snapshot can't race the connection's first fetch, and filters, pause and clear stay ordinary per-connection state
- Pushes are coalesced per connection. Each connection gets a 1-slot channel and its own flusher goroutine, and flushes at most every `flush` (default `100ms`), so a slow tab never blocks the hub.
  alternative: none considered
- Stream subscriptions use their own registry in the WebSocket handler, not `RefreshSourcesForFile`.
  because: that path finds blocks through `sourceFiles`/`StateRef`, which only markdown sources register
- Each row is an immutable map, shared by every snapshot that holds it. The ring keeps the newest `buffer` rows (default 2000) and evicts from the head.
  alternative: `cache.max_rows`
  because: `max_rows` keeps the first N rows, which for logs are the oldest (`cached.go:118`)
- New per-connection builtin actions for `stream`/`merge` blocks:
  - `Pause` stops pushes to that connection.
  - `Resume` refreshes once, then resumes pushes.
  - `Clear` hides rows with `seq <= current`. It never empties the shared buffer.

  All three are added to `builtinActions` so the policy lint accepts them.
  alternative: none considered
- Filter grammar gets two generic additions:
  - an operator `field ~ text` (case-insensitive substring match)
  - an `and` that joins clauses

  The `Filter` action also composes form fields named `f.<field>` into `field ~ value` clauses, skipping empty values. A single form can then hold a level select and a search box.
  alternative: a separate log-specific search action
  because: substring match and AND help every tinkerdown source, not only logs
- `Filter` updates only the parts its message carries. A `filter` key replaces the tab expression, and `f.*` keys replace the field filters. The block shows rows that match both.
  alternative: none considered
  because: today `handleFilter` resets `activeFilter` to `""` when `filter` is missing (`state.go:563`), so a toolbar change would silently undo the selected tab
- `GenericState` exposes `Paused bool`, so templates can switch between Pause and Resume.
  alternative: none considered
- Lines are parsed according to `format`:

  | `format` | Behavior |
  |---|---|
  | `auto` (default) | tries JSON, then logfmt, then plain text with level detection |
  | `json` | JSON only |
  | `logfmt` | logfmt only |
  | `lines` | no parsing |

  Fields parsed from JSON or logfmt go under `fields`, never at the top level, so they can't overwrite the row fields.
  alternative: none considered
- Each line is read with a capped reader. A line longer than 16 KiB is cut at 16 KiB and the row gets `truncated: true`.
  because: `bufio.Scanner` gives up on a token over 64 KiB and stops reading the stream, so one stack trace would end the stream
- The file tailer polls every 250ms with `os.Stat` and detects rotation (inode change) and truncation (size below the read offset). On start it reads the last `backlog` lines (default 200).
  alternative: fsnotify, or the server watcher
  because: the server watcher only handles `.md` files and is off in headless mode, and fsnotify misses events on bind mounts and NFS
- The log template follows three rules so the range stays in stream mode:
  - every row keyed by `data-key="{{.seq}}"`
  - level shown as `class="lvl-{{.level}}"`, never with per-row `{{if}}`
  - rendered as a `<div>`/`<ul>`, never `<table lvt-source>`, because a table rebuilds a `datatable` on every refresh (`state.go` `buildDataTable`)

  alternative: none considered
- Follow mode uses the client's existing `lvt-fx:scroll="bottom-sticky"` (`client/dom/directives.ts:285`). The docs wrongly list this value as `sticky`, and this work fixes them.
  alternative: none considered
- `logviewer/` is a new git repo with its own Go module. Its e2e test builds `tinkerdown` from `TINKERDOWN_SRC` and runs `tinkerdown serve`, because Go forbids importing tinkerdown's `internal/` packages from another module.
  alternative: put the app in `tinkerdown/examples/`
  because: the user asked for a separate app; a smaller copy still goes in `tinkerdown/examples/stream-logs` as a test fixture

## Files to modify

tinkerdown (worktree `.worktrees/stream-sources`):
- `internal/stream/` new? `hub.go` (Hub, subscribe/unsubscribe, global `seq`), `ring.go`, `proc.go` (cmd runner), `tail.go`, `parse.go` (formats and `detectLevel`), and unit tests for each.
- `internal/source/stream.go` new?: `StreamSource` and `MergeSource`, thin views whose `Fetch` returns the hub snapshot.
- `internal/source/source.go`: add `stream` and `merge` to `createSource`.
- `internal/runtime/state.go`: add `stream` and `merge` to the second `createSource`, plus the hub option, `Pause`/`Resume`/`Clear`, `builtinActions` and the `f.*` filter composition.
- `internal/runtime/expr.go`: add the `~` operator and `and` to `parseWhereClause`.
- `internal/config/config.go`: add `Inputs`, `Buffer`, `Backlog`, `Restart` and `Flush` to `SourceConfig`, update the `Type` comment, and add validation.
- `parser.go`: mirror the new fields in `SourceConfig` (`:26`).
- `internal/server/websocket.go`: copy the new fields in `getEffectiveSource`; subscribe the stream blocks in `initializeInstances`; unsubscribe on close.
- `internal/server/server.go`: own the `Hub`, and shut it down in the server's close or shutdown path.
- `stream_source_e2e_test.go` new?, plus `examples/stream-logs/` new?
- `e2e_helpers_test.go` and `e2e_helpers_external_test.go`: add a server-log capture helper (`log.SetOutput` into a thread-safe buffer).
- Docs:
  - `docs/sources/stream.md` new?, which documents both `stream` and `merge`
  - `docs/guides/data-sources.md`: a table row
  - `docs/reference/config.md`: the type list
  - `docs/reference/lvt-attributes.md`: change `sticky` to `bottom-sticky`, and note the `~`/`and` filter additions
- `CHANGELOG.md`: entries under `[Unreleased] ### Added`.

logviewer:
- `logviewer/tinkerdown.yaml` new? and `logviewer/index.md` new?: as shown in § The tinkerdown files, with demo sources standing in for the docker ones in the shipped default.
- `logviewer/scripts/demo-logs.sh` new?: emits mixed JSON, logfmt and plain lines on stdout and stderr.
- `logviewer/go.mod` new?, `logviewer/e2e_test.go` new?, `logviewer/README.md` new?.

## Tracker: assumptions, risks and known gaps

This one table replaces the template's separate Assumptions, Known gaps and Risks sections. Assumptions (A) must be checked before building, risks (R) are what we'd act on, and gaps (G) are requirements no phase owns.

| # | Type | Item | Severity | Status |
|---|---|---|---|---|
| A1 | assumption | A keyed `{{range .Data}}` inside a GenericState lvt block runs in stream mode. When `Refresh` appends rows, the WebSocket sends `a`/`r` ops, not a full tree. | high | verified in Phase 0: append 2 + evict 1 sent `[["r","1"],["a",[…2 rows]]]`, 141 bytes against 962 for the full list |
| A2 | assumption | After `Filter` then clear-filter, later appends go back to granular ops. | high | verified in Phase 0: clearing the filter sent one full range (897 bytes), and the next append was granular again, so the fallback lasts one render, not the connection |
| A3 | assumption | The WebSocket handler has a close hook where subscriptions can be removed. | medium | verified in Phase 2: `ServeHTTP` returns on disconnect, but `Handler.Close` was never called; it now runs there and stops followers first |
| A4 | assumption | A tabs heading sends `Filter {filter}` to the lvt block directly below it (`parser.go:761`, `client/src/core/tabs.ts`, test at `tabs.test.ts:126`). | medium | verified 2026-11-20 by reading the code |
| A5 | assumption | A field event inside `<form name="Filter">` sends the form's named fields. | medium | false, found in Phase 3: tinkerdown's block client sent only `{value}` on change and ignored `lvt-on:input` and `lvt-mod:*`. Fixed in `client/src/blocks/interactive-block.ts`; each field sends its own name, which partial-update `Filter` needs |
| A6 | assumption | The docker daemon is reachable from tests. | low | verified 2026-11-20 |
| R1 | risk | Filtering makes each push a full replace of up to 2000 rows per tab. | high | retired: A1 and A2 hold; only the render that clears a filter sends the full list |
| R2 | risk | `sh -c` gives the config author a shell. Mitigation: gated by `--allow-exec`, exactly as `exec` is, and `docs/sources/stream.md` says so up front. | medium | accepted |
| G1 | gap | Because stdout and stderr are read on separate pipes, their relative order is lost; within one input, rows keep arrival order, which can differ from write order. Users who need exact order can pass `2>&1` and give up the tags. | medium | unowned |
| G2 | gap | Buffers live in one server process. History is not saved across restarts, and several tinkerdown instances don't share it. | low | unowned |
| G3 | gap | Tinkerdown's block client re-implements part of `@livetemplate/client`'s `EventDelegator` (this work added `lvt-on:input` and `lvt-mod:*`; `lvt-key` is still missing). The deeper fix is an upstream hook that lets a block embed the delegator without owning the socket, plus a trailing-edge throttle and cancel in its rate limiters. | medium | unowned: cross-repo follow-up |
| G4 | gap | Tinkerdown has no importable e2e helper package, so the logviewer copies the Docker Chrome and capture harness, and five tests in tinkerdown repeat the console and WebSocket capture. | low | unowned |

## Acceptance criteria

- [x] `/simplify` has run on each phase's diff ✅ (Phases 1, 2, and 3–4 together; findings applied in 48a4cd8, 5dea14a and logviewer 768aacb)
- [x] `go test ./internal/stream/...` passes ✅ (`-race`; plus `TestIdleInputStopsAndRestarts`), with tests for:
  - ring eviction
  - global `seq` order across inputs
  - a 1 MiB line truncated without killing the reader
  - tail rotation and truncation
  - `cmd` restart with backoff and a `system` row
  - process group killed on `Hub.Close` (no orphan `sleep` left in `pgrep`)
- [x] `go test ./internal/runtime/...` ✅ covers `~`, `and`, `f.*` composition, a `filter`-only message keeping the `f.*` filters (and the reverse), `Pause`/`Resume`/`Clear` with `Paused`, and `IsBuiltinAction("pause")`.
- [x] Config validation rejects `stream` with both `cmd` and `file`, and `merge` naming an unknown or non-stream input ✅ (`TestValidateStreamSources`). `--allow-exec` is a runtime flag, so a `cmd` stream without it is rejected when the source is built ✅ (`TestStreamSourceNeedsHubAndExecFlag`).
- [x] `TestStreamSource_E2E` (chromedp, Docker Chrome) ✅:
  - rows appear with no user action
  - captured WebSocket frames contain range append ops (A1 holds)
  - two pages share one process (`pgrep -c` == 1); the second page is a WebSocket connection, because a second Docker Chrome tab takes ~40s
  - `Pause` stops new rows and `Resume` catches up
  - an `f.line` filter narrows rows
  - an `alpine:3.21` container's logs stream through `docker logs -f` (busybox isn't pulled locally)
  - appending to a tailed file shows up within 1s

  The test captures console logs, server logs, WebSocket frames and rendered HTML.
- [x] `cd logviewer && go test ./...` e2e ✅ (`TestLogViewer`, 8 subtests; screenshots in `logviewer/docs/`). The All tab interleaves both demo inputs, and stderr rows carry `stream-stderr`. The level select still applies after switching to a source tab, and `bottom-sticky` stays at the bottom while rows arrive.
- [x] The full suite is green in tinkerdown ✅ (`go test -timeout 40m ./...`; three tests need the `./tinkerdown` binary that `make build` creates). `make test` now sets `-timeout 30m`, because the root package takes ~15 minutes. The repo has no pre-commit hook.
- [x] `prereview check` is clean on the changed docs ✅ (remaining findings in `CHANGELOG.md` and `lvt-attributes.md` predate this work)

## Constraints

- Existing sources keep their behavior: `exec` stays one-shot with `strings.Fields`, and the `RefreshSourcesForFile` markdown path is unchanged.
- No log-specific code outside `internal/stream/parse.go`, and level detection stays opt-in through `format`.
- Pushes never block the hub, so a stuck WebSocket write can't stall other tabs.
- Docs match the code (`cmd`, not `command`) and are checked against the code before commit. Today's `exec.md` drift isn't repeated, and isn't fixed here either beyond the `stream` page.
- e2e tests use real Docker Chrome and a real docker container. They are never skipped.

## Contract: config and row shape

Example config: § The tinkerdown files. Defaults: `format: auto`, `buffer: 2000`, `backlog: 200`, `restart: on-exit`, `flush: 100ms`, max line 16 KiB.

Row fields: `seq` (int, global) · `ts` (RFC3339; parsed when present, otherwise arrival time) · `time` (`ts` as `15:04:05.000`, for display) · `source` · `stream` (`stdout` | `stderr` | `file` | `system`) · `level` (`trace` | `debug` | `info` | `warn` | `error` | `fatal` | `""`) · `msg` · `line` (raw) · `truncated` (bool) · `fields` (map).

## Phases

### Phase 0: spike (A1, A2)
- Audit: read `livetemplate/internal/diff/range_ops.go` (`handleStreamModeRange`) and the het-range fallback in the proposal §5d.
- Implementation: a throwaway e2e test in the worktree renders a keyed range from an `exec` source, appends rows, refreshes, and dumps the WebSocket frames. It then applies a filter, clears it, and dumps again.
- Acceptance: A1/A2 recorded in the tracker with the frame dumps. The Filter design is confirmed or switched to R1's mitigation. The spike file is deleted.
- Learn: the het-range fallback is per render, not one-way as the proposal implied. A gorilla `ReadMessage` deadline error breaks the connection for good, so test readers use a goroutine and a channel.

### Phase 1: `internal/stream`
- Audit: confirm the `SourceConfig` field names don't collide, and check `exec.go`'s env handling to reuse it.
- Implementation: ring, hub, proc, tail, parse. **User contribution:** `detectLevel(line string) string` in `parse.go`, 5 to 10 lines. You choose how plain-text lines map to levels: keyword prefixes like `ERROR`, bracketed `[warn]`, syslog `<3>`, and how strictly to match so that "no errors found" doesn't count as an error.
- Acceptance: `/simplify`, then the `internal/stream` criteria above.

### Phase 2: wiring (source types, push, actions, filter)
- Audit: A3, A5; trace `initializeInstances` and connection close; confirm both `createSource` switches and `getEffectiveSource`.
- Implementation: per the Files list.
- Acceptance: `/simplify`, then the runtime and validation criteria above.
- Learn: every render JSON-round-trips the whole state (`GetStateAsInterface`), so each push costs O(buffer) per page; fine at 2000 rows and 10 pushes a second, the first thing to optimize at larger buffers. An approved merge must resolve its inputs from the manifest, or a page can redefine them.

### Phase 3: tinkerdown e2e, example, docs
- Audit: A6. Check what `e2e_helpers` already captures.
- Implementation: the server-log capture helper, `TestStreamSource_E2E`, `examples/stream-logs`, docs, CHANGELOG.
- Acceptance: `/simplify`, the e2e criterion, the full suite, `prereview check` on the docs.
- Learn: Docker Chrome's network path stalls frame delivery for ~7s once early on (a raw host client sees no gap), so the e2e waits for steady delivery before timing checks. Docker Chrome runs with `--cpus 0.5`, so a second browser tab takes ~40s; the two-pages check uses a second WebSocket instead.

### Phase 4: log viewer app
- Audit: re-read the contract above against what shipped. Any drift is written into this plan in the same commit.
- Implementation: the `logviewer/` files.
- Learn: chromedp ties a tab to the context of its first `Run`; a first `Run` on a timeout context closes the tab when it returns.
- Acceptance: `/simplify`, then the logviewer e2e criterion. Verified by hand in a real browser through chromedp screenshots, never with `curl`.

## Delivery

1. Copy this plan to `tinkerdown/docs/plans/2026-11-20-stream-sources.md`.
2. Run `/simplifyplan`, open the plan PR, run `/planprcommentsfix`, get signoff, merge. Never merge without signoff.
3. Create the `.worktrees/stream-sources` branch only after the plan is merged. Open one PR per phase, or Phases 1–3 together.
4. Release only through `release.sh`.
