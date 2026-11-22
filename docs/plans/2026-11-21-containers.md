# Plan: a containers onefile app, with actions and log streams from flags

State — Done: plan merged (#336), Phase 1 (`drop`). Next: Phase 2.

## Context

The next [onefile](https://github.com/livetemplate/onefile) app is `containers`: a live list of your Docker containers, with stop, restart and kill buttons, and a container's logs following live when you open it. Like the log viewer and the test runner, it runs from its URL, and every command it runs comes from your own flags.

A read-only prototype (2026-11-21, on v0.6.0) showed the list working, but a list without logs or buttons duplicates `docker ps`. It needs four generic additions to tinkerdown:

- **`drop` on keyed merges:** `drop: fields.state=removed` removes a key's row, so a removed container disappears instead of leaving a dead row.
- **`--action NAME=COMMAND`:** an exec action you declare on the command line, like `--source`. A page, even a remote one, can call it with a button but can't define it.
- **Values as environment variables:** a flag action or flag source gets each value as `$TD_<name>`, so `docker stop "$TD_name"` never parses the value as shell code.
- **Merges with `params`:** `params: {name: {in: containers.fields.name}}` takes `?name=web` from the page URL, accepts it only if `web` is a row in the page's `containers` merge, and starts the merge's inputs with `TD_name=web`. A row link `?name=web` then opens that container's logs.

The audit (2026-11-21) settled these facts:

- `docker ps -a --format` and `docker events --format` take Go templates, so one command prints both as JSON lines of one shape (checked on Docker 29.3.1 with a throwaway container: `create`, `start`, `die (2)`, `destroy`). Docker 29 events carry the container in `Actor.ID` and `Actor.Attributes.name`; only `die` has `exitCode`.
- The feed uses `{{…}}` for Docker's own templates, so tinkerdown can't use `{{…}}` placeholders in flag commands. Hence environment variables.
- Page-declared exec actions paste values into the command with `expandTemplate`. `sanitizeExecCommand` blocks `;|&$<>` but not quotes or spaces, so a value can add arguments.
- The WebSocket URL is built on the server as `/ws?page=PATH` (`server.go:1102`, `:3231`), so blocks never see the page's query today.
- Flag sources are parsed in `cmd/tinkerdown/commands/source_flags.go` and refused when they'd replace an approved source.

## Product

| User journey | What they see | Built on |
|---|---|---|
| See what's running | one row per container, updating live | `--source docker=…`, keyed merge on `fields.name`, `drop` |
| Notice a crash | the row turns red: `exited (137)` | `die` event with `exitCode` |
| Read a container's logs | click its name; logs follow live below the list | `<a href="?name=…">`, merge `params` with `in`, `--source logs=…` |
| Stop, restart or kill it | buttons on the row; kill asks first | `--action stop=…`, `data-confirm` |
| Filter | Running / Exited / Paused tabs, name search | `fields.state` tabs, `f.fields.name` |

```
tinkerdown serve --allow-exec https://raw.githubusercontent.com/livetemplate/onefile/main/containers/index.md \
  --source docker='docker ps -a --format "…"; exec docker events --format "…"' \
  --source logs='docker logs -f --tail 200 -- "$TD_name"' \
  --action stop='docker stop -- "$TD_name"' \
  --action restart='docker restart -- "$TD_name"' \
  --action kill='docker kill -- "$TD_name"'
```

```
┌ Containers ───────────────────────────────────────────────────┐
│ [All] [Running] [Exited] [Paused]   [🔍 name     ]  ● live 12 │
├───────────────────────────────────────────────────────────────┤
│ ● running  web     nginx:1.27    Up 2 hours    [⟳] [■] [✕]    │
│ ✗ exited   worker  app:dev       exited (137)  [⟳]            │
├ worker · logs ────────────────────────────────── ↓ following ─┤
│ 17:02:11 job 41 started                                       │
│ 17:02:12 panic: nil map            stderr                     │
└───────────────────────────────────────────────────────────────┘
```

On a phone, each row puts the image and status under the name and the buttons on their own line, and the logs panel sits below the list.

## Decisions

- Key rows by container name, not ID.
  alternative: key by ID
  because: `docker compose up` recreates a container under the same name with a new ID, and `ps` prints a short ID while events carry the full one
- Add `drop: PATH=VALUE` to keyed merges (`VALUE` may be `a|b`). A matching row removes its key's row and isn't stored; a later row for the key starts a new row at the end. `drop` and `collect_reset` share one `PATH=a|b` parser.
  alternative: keep removed containers as dimmed rows
  because: `--rm` containers get fresh names, so dead rows pile up to `buffer`
- Commands come from flags: `--action NAME=COMMAND` alongside `--source`. A flag action is reachable from every page, like an approved action, and refused when it would replace an approved action.
  alternative: a local config file; a local-only app
  because: chosen by the user; it keeps "one URL, one command" and the rule that a remote page never chooses what runs
- Flag commands get values as environment variables `TD_<name>`, never by text substitution. Names match `^[A-Za-z0-9_]{1,32}$`.
  alternative: shell-quote values into `{{.name}}` placeholders
  because: the Docker feed already uses `{{…}}`, and an environment variable can't be parsed as code; the prefix stops a value from replacing `PATH`
- A value is refused when it is empty, longer than 256 bytes, starts with `-`, or holds a control character.
  alternative: none considered
  because: a quoted value can still be read as an option (`--output=…`); README commands also pass `--`
- A value stays one argument only when the command quotes it: `"$TD_name"`. Unquoted, the shell splits `a b` into two. Tinkerdown can't enforce this; the docs and every README command quote it.
  alternative: none considered
- An action's values are the button's `data-*` attributes, as for page actions today.
  alternative: none considered
- Page-declared exec actions keep their current `{{…}}` expansion.
  alternative: move them to environment variables too
  because: that changes existing pages; recorded as G2
- `params` on a merge takes the values from the page URL query and starts the merge's inputs with them. Without every param present, the block renders with no rows and no input runs.
  alternative: a Filter-like action that sets the value from a button
  because: a link is shareable (`?name=web`), needs no cross-block messaging, and survives a reload
- Every param needs `in: MERGE.PATH`: a value is accepted only while it equals PATH in a row of MERGE on the same page. A link can then open only what the page already lists.
  alternative: free values; or open logs with a WebSocket action, which already refuses other origins
  because: anyone can write a link. A free value lets another site make your machine run your command with a value it picks, and an action loses the shareable link
- The `in` check runs again when MERGE's rows change. A link opened before the feed's first rows starts once the row appears; a value never listed starts nothing.
  alternative: none considered
- The page's query reaches the WebSocket: the server adds it to the `ws-url` it renders, and the handler passes it to block state.
  alternative: the client appends `location.search`
  because: the server already builds the URL, and the client then needs no change
- A merge with `params` gets a buffer per distinct set of values (keyed by name, config hash and values). Its inputs stop 30s after the last viewer leaves.
  alternative: run until shutdown, as streams do today
  because: every container opened would leave a `docker logs -f` running
- Kill asks first through the button's `data-confirm`. Stop and restart don't.
  alternative: none considered
- The feed is one copyable `--source` value. Events are filtered to `create`, `start`, `die`, `pause`, `unpause` and `destroy`, and Docker templates map them onto `ps` states (`start` → `running`, `die` → `exited`, `destroy` → `removed`). Names are printed with `{{json .Names}}`.
  alternative: all container events
  because: `attach`, `exec_*` and `health_status` would overwrite the row's state with noise
- `ps` rows show `ps`'s own status (`Up 2 hours`), which doesn't tick.
  alternative: compute uptime in the page
  because: `ps` JSON has no start timestamp

## Files

tinkerdown (worktree `.worktrees/containers-upstream`):
- `internal/stream/keyed.go`, `hub.go`: `drop` in `KeySpec` and `keyedRows.add`; per-values buffers for params merges, stopped 30s after the last unsubscribe.
- `internal/stream/proc.go`: start a cmd with extra `TD_*` environment variables.
- `internal/config/config.go`, `parser.go`, `stream_sources.go`: `Drop`, `Params` fields, mirrored; validation (`drop` and `params` need a merge, `drop` needs `key`, every param needs `in` naming a merge on the page and a field path); one `PATH=a|b` parser for `drop` and `collect_reset`; `FlagActions`.
- `internal/runtime/state.go` or `internal/server/websocket.go`: the `in` check against the named merge's current rows, rerun on its updates.
- `internal/config/params.go` new?: the `TD_` name and value rules, shared by actions and streams.
- `cmd/tinkerdown/commands/source_flags.go` (or `action_flags.go` new?): `--action NAME=COMMAND`, needs `--allow-exec`, refused over an approved action, listed in `serve --help`.
- `internal/runtime/actions.go`: run a flag action with `TD_*` values, no `expandTemplate`, no `sanitizeExecCommand`; the 30s timeout stays.
- `internal/server/server.go`: carry the page query into the `ws-url`. `websocket.go`: pass it to block state; reach flag actions from pages.
- Tests: `internal/stream` (drop; params buffers and stop), `internal/config` (validation, value rules), `internal/runtime` (flag action env, refused values), `cmd/tinkerdown/commands` (`--action` parsing), e2e `TestFlagActions_E2E`, `TestParamsMerge_E2E` new?, `TestKeyedMerge_E2E` (drop sends `["r", …]`).
- Docs: `docs/sources/stream.md` (`drop`, `params`), `docs/reference/cli.md` or the `serve` docs (`--action`, `TD_` values), `docs/reference/config.md`; `CHANGELOG.md` under `[Unreleased]`.

onefile:
- `containers/index.md` new?: merges `containers` (`inputs: ["*"]` accepting the feed, `key: [fields.name]`, `drop: fields.state=removed`) and `logs` (`inputs: [logs]`, `params: {name: {in: containers.fields.name}}`); row buttons and the logs panel.
- `containers/README.md` new? with the copyable command, `containers/docs/desktop.png`, `phone.png` new?
- `e2e/containers_test.go` new?; root `README.md` lists the app.

## Tracker

| # | Type | Item | Severity | Status |
|---|---|---|---|---|
| A1 | assumption | The feed survives `--source NAME='…'` quoting and `sh -c`, with the format strings' double quotes escaped. | high | open (Phase 4) |
| A2 | assumption | Removing a keyed row mid-range reaches the page as `["r", key]`. livetemplate's range diff emits `r` ops (`range_ops.go`). | medium | verified 2026-11-21 (`TestKeyedMerge_E2E`) |
| A3 | assumption | A merge input can't be used by a page both as a plain input and through a params merge; if it can, the plain use must refuse a source whose command reads `TD_*`. | medium | open (Phase 3) |
| A4 | assumption | `data-confirm` works on any button in an lvt block, not only in forms (the PII example uses it on a row button). | low | open (Phase 4) |
| R1 | risk | Anyone can write a link with `?name=`. `in` limits it to a container the page already lists, so the worst a hostile page can do is start logs for one of those; it can't read them (same-origin policy, Host allow-list). | low | mitigated (`in`) |
| R2 | risk | A rename leaves the old name's row behind. Mitigation if it bites: map `rename` to a drop of `oldName`. | low | accepted |
| G1 | gap | Remote pages can call flag actions but still can't declare their own. Intended. | low | owned (this plan) |
| G2 | gap | Page-declared exec actions still paste values with `{{…}}`, so a value can add arguments. | medium | unowned |

## Acceptance criteria

- [ ] `/simplify` has run on each phase's diff.
- [ ] `go test ./internal/stream/...`:
  - a `drop` row removes its key's row and isn't stored, and a later row re-adds at the end
  - a params merge gets one buffer per distinct values, and stops its inputs 30s after the last unsubscribe
  - a cmd sees its `TD_*` variables
- [ ] `go test ./internal/config/...`: refuses `drop` without `key`, `drop` or `params` on a non-merge, a bad param name; the value rules refuse empty, over 256 bytes, a leading `-`, a control character.
- [ ] `go test ./cmd/tinkerdown/commands/...`: `--action` parses `NAME=COMMAND`, needs `--allow-exec`, refuses a duplicate and an approved action's name.
- [ ] `go test ./internal/runtime/...`: a flag action runs with `TD_name` set from `data-name`, a value `$(id)` reaches the command as text, and a refused value never runs.
- [ ] Browser e2e (Docker Chrome):
  - `TestFlagActions_E2E`: clicking a row button runs the command with the row's value
  - `TestParamsMerge_E2E`: `?name=a` shows only `a`'s rows, and `?name=b` gets a separate buffer
  - `TestParamsMerge_E2E`: `?name=c`, not in the list, runs no command until a `c` row appears
  - `TestKeyedMerge_E2E`: a drop sends `["r", …]`
- [ ] onefile `cd e2e && go test -run TestContainers ./...`, with containers named `ofe2e-<pid>-*` and the README command plus `--filter name=ofe2e-<pid>`:
  - a running container shows `running`; one that exits with code 3 shows `exited (3)` in the Exited tab
  - Stop moves a row to exited; Kill asks first; `docker rm -f` removes the row
  - clicking a name opens its logs, and a line the container prints afterwards appears
  - search narrows by name; on a phone no cell leaves its row
  - `t.Cleanup` removes every `ofe2e-<pid>-*` container
- [ ] The onefile page passes every remote-page check when served from its raw GitHub URL.
- [ ] Full tinkerdown suite green; pre-commit passes; `prereview check` clean on changed docs.

## Constraints

- Merges without `drop` or `params`, page-declared actions and existing `--source` flags behave exactly as today.
- Nothing in tinkerdown knows about Docker.
- A remote page never defines a command: it only names sources and actions that flags supply.
- Flag actions and flag sources need `--allow-exec`.
- Every documented flag command quotes its values: `"$TD_name"`, never `$TD_name`.
- e2e tests use real Docker Chrome and a real Docker daemon, and are never skipped.

## Contract

```
drop: PATH=V           keyed merge: a row matching PATH=V (V may be a|b) removes its key's row, and is not stored
params:                merge: values from the page query; inputs start with TD_NAME=value; no rows until all are present
  NAME: {in: M.PATH}   a value is accepted only while it equals PATH in a row of merge M on the page
--action NAME=COMMAND  exec action from the command line; values arrive as $TD_<data-* name>
TD_ value rules        name ^[A-Za-z0-9_]{1,32}$; value 1-256 bytes, no leading "-", no control characters
```

The feed's JSON line: `{"name", "image", "state", "status"}`. `state` is one of `created`, `running`, `paused`, `exited`, `restarting`, `dead` (from `ps`), or `removed` (from `destroy`).

## Phases

### Phase 1: `drop` on keyed merges
- Audit: A2; every caller of `KeySpec` and the `collect_reset` parser.
- Implementation: `drop` per the Files list.
- Acceptance: `/simplify`, the stream and config `drop` criteria, `TestKeyedMerge_E2E`.

### Phase 2: `--action` flags and `TD_` values
- Audit: how page actions reach `executeCustomAction`, and how approved actions are found from a page.
- Implementation: the value rules, `--action`, the flag-action runner.
- Acceptance: `/simplify`, the commands, runtime and `TestFlagActions_E2E` criteria.

### Phase 3: merges with `params`
- Audit: A3; how a block's merge buffer is resolved per connection.
- Implementation: query into the WebSocket, per-values buffers, stop after the last viewer.
- Acceptance: `/simplify`, the params stream criteria, `TestParamsMerge_E2E`, docs, the full suite.

### Phase 4: the containers app
- Audit: A1, A4. Re-read the Contract against what shipped.
- Implementation: the onefile files.
- Acceptance: `/simplify`, the onefile criteria, screenshots through chromedp.

## Delivery

1. Open the plan PR, get signoff, merge. Never merge without signoff.
2. Create `.worktrees/containers-upstream` after the plan is merged. One PR per phase 1–3, or one for all three; release through `release.sh` and bump the Homebrew formula; then Phase 4 in onefile, against the released tinkerdown.
