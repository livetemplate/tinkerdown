# Plan: a containers onefile app, and dropping keyed rows

State — Done: audit (below). Next: plan PR, then Phase 1.

## Context

The next [onefile](https://github.com/livetemplate/onefile) app is `containers`: a live list of your Docker containers, one row per container, showing its state as it changes. Like the log viewer and the test runner, it runs from its URL and takes its feed as a `--source` command.

It reuses the test runner's keyed merges: the key is the container name, and each `docker events` line updates that container's row in place. It needs one generic addition to tinkerdown, because keyed merges can update a row but never remove one:

- **Dropping rows:** `drop: fields.state=removed` on a keyed merge removes a key's row when a matching row arrives. Without it, every `docker run --rm` leaves a dead row until it is evicted.

The audit (2026-11-21, Docker 29.3.1) settled the feed:

- `docker ps -a --format` and `docker events --format` both take Go templates, so one command can print both as JSON lines of the same shape. A combined feed was run against a throwaway container and printed `create`, `start`, `die (2)` and `destroy` rows for it.
- Docker 29 events have no top-level `id` or `status`. The container is in `Actor.ID` and `Actor.Attributes.name`, and only `die` carries `Actor.Attributes.exitCode`.
- Docker's templates support `if eq`, so the command maps event names onto `ps` states (`start` → `running`, `die` → `exited`, `destroy` → `removed`). Tabs then filter one field.

## Product

| User journey | What they see | Built on |
|---|---|---|
| See what's running | one row per container, updating live | `--source docker=…`, keyed merge on `fields.name` |
| Notice a crash | the row turns red and reads `exited (137)` | `die` event with `exitCode` |
| Find stopped containers | the Exited tab | `fields.state = exited` tab |
| Find a container | the search box | `f.fields.name` filter |
| Removed containers disappear | the row goes away | `drop: fields.state=removed` |

```
┌ Containers ───────────────────────────────────────────┐
│ [All] [Running] [Exited] [Paused]                     │
│ [🔍 name              ]                ● live · 12    │
├───────────────────────────────────────────────────────┤
│ ● running  web        nginx:1.27        Up 2 hours    │
│ ● running  db         postgres:17       Up 2 hours    │
│ ✗ exited   worker     app:dev           exited (137)  │
│ ‖ paused   cache      redis:7           paused        │
└───────────────────────────────────────────────────────┘
```

On a phone, each row puts the image under the name, and the status wraps under both:

```
┌ Containers ─────────────┐
│ [All] [Running] [Exited]│
│ [🔍 name     ] ● live 12│
├─────────────────────────┤
│ ● web          running  │
│   nginx:1.27            │
│   Up 2 hours            │
│ ✗ worker       exited   │
│   app:dev               │
│   exited (137)          │
└─────────────────────────┘
```

## Decisions

- Key rows by container name, not ID.
  alternative: key by ID
  because: `docker compose up` recreates a container under the same name with a new ID, and `ps` prints a short ID while events carry the full one
- M1 is read-only: no stop, restart or logs buttons.
  alternative: exec actions per row
  because: remote pages refuse actions, so buttons would break the run-from-URL story. Exec actions also paste row values into the command unescaped: `sanitizeExecCommand` blocks `;|&$<>` but not quotes or spaces, so a row value could add docker arguments
- Add `drop: PATH=VALUE` to keyed merges, parsed like `collect_reset` (`VALUE` may be `a|b`). A matching row removes its key's row and is not stored. A later row for the key starts a new row at the end.
  alternative: keep removed containers as dimmed `removed` rows
  because: `--rm` containers and e2e runs get fresh names, so dead rows pile up to `buffer`
- `drop` and `collect_reset` share one parser for `PATH=a|b`.
  alternative: none considered
- The feed is one command, given as a copyable `--source` value in the README: `docker ps -a --format …; exec docker events --filter … --format …`.
  alternative: ship a `feed.sh` in onefile and have the user download it
  because: an extra download step breaks "one URL, one command"
- Events are filtered to `create`, `start`, `die`, `pause`, `unpause` and `destroy`.
  alternative: all container events
  because: `attach`, `exec_*` and `health_status` would overwrite the row's state with noise
- Names are printed with `{{json .Names}}`, so the feed is valid JSON whatever the name holds.
  alternative: none considered
- Status text for `ps` rows is `ps`'s own (`Up 2 hours`), and doesn't tick. Event rows show the event and exit code.
  alternative: compute uptime in the page
  because: `ps` gives no start timestamp in its JSON, and templates have no clock
- The page doesn't offer a suggested command when started without `--source`.
  alternative: a serve prompt that offers the page's suggested command for one keypress
  because: a remote page would then choose what runs, behind one Enter; `accepts.hint` already shows the command as text

## Files

tinkerdown (worktree `.worktrees/drop-keyed-rows`):
- `internal/stream/keyed.go`: `drops(r)`; `add` removes the key's row and index entry, and reindexes the rows after it.
- `internal/stream/hub.go`: `KeySpec` gains `DropPath`, `DropValue`.
- `internal/config/config.go`, `parser.go`: `Drop string yaml:"drop"`, mirrored; `pageSourceConfig` copies it.
- `internal/config/stream_sources.go`: `validateKey` checks `drop` (needs `key`, `PATH=VALUE`); one shared `PATH=a|b` parser for `drop` and `collect_reset`; `KeySpec()` fills the drop fields.
- Tests: `internal/stream` (drop removes, re-adds at the end, alternatives), `internal/config` (validation), `source_flags_e2e_test.go` `TestKeyedMerge_E2E` (a drop sends `["r", key]`).
- Docs: `docs/sources/stream.md` (`drop` row, one-row-per-key section); `CHANGELOG.md` under `[Unreleased]`.

onefile:
- `containers/index.md` new?: the page, frontmatter merge `inputs: ["*"]`, `key: [fields.name]`, `drop: fields.state=removed`, `accepts` with the feed command as the hint.
- `containers/README.md` new?, `containers/docs/desktop.png` and `phone.png` new?
- `e2e/containers_test.go` new?: see Acceptance criteria.
- Root `README.md`: list the app.

## Tracker

| # | Type | Item | Severity | Status |
|---|---|---|---|---|
| A1 | assumption | The feed command survives `--source NAME='…'` quoting and `sh -c` unchanged, with the format strings' double quotes escaped. | high | open (Phase 2) |
| A2 | assumption | Removing a keyed row mid-range reaches the page as `["r", key]`, not a redraw. livetemplate's range diff emits `r` ops (`range_ops.go`). | medium | open (Phase 1) |
| A3 | assumption | `docker events` keeps running when no container changes, and exits only when the daemon goes away; tinkerdown's restart backoff then reconnects. | medium | open (Phase 2) |
| R1 | risk | A rename leaves the old name's row behind, since the feed doesn't map `rename`. Mitigation if it bites: map `rename` to a drop of `oldName`. | low | accepted |
| G1 | gap | A crashed container's last log lines. Tinkerdown can't start one `docker logs -f` per row, so this needs a per-key input, not a page change. | medium | unowned |
| G2 | gap | Stop and restart buttons. They need a `pattern` on action params (so a value can't add arguments), and still won't run from a URL. | medium | unowned (M2) |

## Acceptance criteria

- [ ] `/simplify` has run on each phase's diff.
- [ ] `go test ./internal/stream/...`: a `drop` row removes its key's row and isn't stored; a later row for the key is added at the end; `a|b` alternatives match; merges without `drop` are unchanged.
- [ ] Config validation refuses `drop` without `key`, a `drop` that isn't `PATH=VALUE`, and `drop` on a non-merge source.
- [ ] `TestKeyedMerge_E2E`: a drop row sends `["r", …]` for the key, and the row leaves the page.
- [ ] onefile `cd e2e && go test -run TestContainers ./...`, with containers named `ofe2e-<pid>-*` and the README feed plus `--filter name=ofe2e-<pid>`:
  - a running `sleep` container shows `running` in the Running tab
  - a container that exits with code 3 shows `exited (3)` in the Exited tab
  - `docker pause` moves it to Paused; `docker rm -f` removes its row
  - search narrows by name
  - phone layout: no cell leaves its row
  - `t.Cleanup` removes every `ofe2e-<pid>-*` container
- [ ] The onefile page passes every remote-page check when served from its raw GitHub URL.
- [ ] Full tinkerdown suite green; pre-commit passes; `prereview check` clean on changed docs.

## Constraints

- Keyed merges without `drop`, and merges without `key`, behave exactly as today.
- Nothing in tinkerdown knows about Docker; `drop` is generic.
- The onefile page stays a valid page from a URL: merge sources only, no actions, no scripts.
- e2e tests use real Docker Chrome and a real Docker daemon, and are never skipped.

## Contract

```
drop: PATH=V           on a keyed merge: a row matching PATH=V (V may be a|b) removes its key's row, and is not stored
```

The feed's JSON line, for both `ps` and event rows: `{"name", "image", "state", "status"}`. `state` is one of `created`, `running`, `paused`, `exited`, `restarting`, `dead` (from `ps`), or `removed` (from `destroy`).

## Phases

### Phase 1: `drop` on keyed merges
- Audit: A2; every caller of `KeySpec` and the `collect_reset` parser.
- Implementation: per the tinkerdown Files list.
- Acceptance: `/simplify`, the stream, validation and e2e criteria, docs, the full suite.

### Phase 2: the containers app
- Audit: A1, A3. Re-read the Contract against what shipped.
- Implementation: the onefile files.
- Acceptance: `/simplify`, the onefile criteria, screenshots through chromedp.

## Delivery

1. Open the plan PR, get signoff, merge. Never merge without signoff.
2. Create `.worktrees/drop-keyed-rows` after the plan is merged. One PR for Phase 1; release through `release.sh` and bump the Homebrew formula; then Phase 2 in onefile, against the released tinkerdown.
