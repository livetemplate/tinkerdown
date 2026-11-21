# Plan: a test runner onefile app, and keyed stream rows

State — Done: Phases 0–1 (branch `feat/keyed-merges`). Next: PR review, release, then Phase 2 in onefile.

## Context

The next [onefile](https://github.com/livetemplate/onefile) app is a test runner: run your tests in watch mode, see one row per test with its status, filter to failures, and read a failing test's output. Like the log viewer, it runs from its URL and takes the test command as a `--source` flag.

It needs three generic additions to tinkerdown, because `go test -json` writes several events per test (`run`, `output` lines, then `pass`, `fail` or `skip`):

- **Keyed merges:** a merge with `key: [fields.Package, fields.Test]` keeps one row per key, the latest event, updated in place.
- **Collected text:** `collect: fields.Output` appends a field's text across a key's events, so a test's row carries its output.
- **Field paths:** filters and tabs can read nested fields, as in `[Failed] fields.Action = fail`. Today they only see top-level fields, and parsed JSON lands under `fields`.

The same additions serve the container board suggested next (one row per container from `docker ps` or `kubectl get pods -w`), so nothing here is test-specific.

## Product

| User journey | What they see | Built on |
|---|---|---|
| Run tests on every save | one row per test, updating live | `--source 'go=watchexec -e go -- go test -json ./...'`, keyed merge |
| Find what broke | the Failed tab | `fields.Action = fail` tab |
| Read why it broke | the test's output under its row | `collect: fields.Output` |
| Search for a test | the search box | `f.fields.Test` filter |
| Several suites | a tab per suite | `[*] source` tabs |

```
tinkerdown serve --allow-exec https://raw.githubusercontent.com/livetemplate/onefile/main/tests/index.md \
  --source 'go=watchexec -e go -- go test -json ./...'
```

```
┌ Tests ──────────────────────────────────────────────┐
│ [All] [Failed] [Passed] [Skipped]   [All] [go]      │
│ [🔍 test name          ]        ● watching · 214    │
├─────────────────────────────────────────────────────┤
│ ✓ pass   api      TestCreateUser            0.01s   │
│ ✗ fail   api      TestDeleteUser            0.03s   │
│   ▸ users_test.go:42: want 204, got 500             │
│ … run    store    TestMigrate                       │
│ – skip   store    TestPostgres (needs DB)           │
└─────────────────────────────────────────────────────┘
```

On a phone, each row puts the package above the test name, and output wraps under it.

## Decisions

- A merge may declare `key`, a list of field paths. Rows whose key values match replace the earlier row in place; the buffer holds `buffer` keys, dropping the key least recently updated.
  alternative: a separate `type: table` source that folds a stream
  because: a merge is what a page from a URL may declare (#330), and keying is a view over rows the merge already receives
- A keyed row keeps the position of the key's first row, so a test list doesn't jump as tests finish.
  alternative: move a key to the end on every update
  because: livetemplate's keyed ranges send an in-place `["u", key, diff]` for a changed item, so updating in place is also the smallest update on the wire
- `collect: <field path>` appends that field's text across a key's rows into the row's `collected`, capped at 64 KiB per key, keeping the end.
  alternative: none considered
- Rows missing a key field use the empty string for it, so `go test`'s package-level events become one row per package.
  alternative: drop rows without a full key
  because: the package row carries the package's own pass or fail
- Field paths: a filter field written `fields.Action` reads `row["fields"]["Action"]`. Plain names behave as today.
  alternative: lift parsed fields to the top level
  because: `fields` was kept separate so a JSON line can't overwrite `seq`, `source` or `ts`; paths keep that guarantee
- `f.<path>` form fields take paths too (`f.fields.Test`).
  alternative: none considered
- The app ships for `go test -json` first. Its key and collect fields come from frontmatter, so pytest's `--report-log` JSON lines or other runners can follow with their own page.
  alternative: one page that detects the runner
  because: each runner names its fields differently, and a page per runner stays readable

## Files

tinkerdown (worktree `.worktrees/keyed-merges`):

- `internal/stream/hub.go`: keyed `Buffer` (key → slot index, first-seen order, least-recently-updated eviction, `collected`); `Merge` takes key and collect.
- `internal/config/config.go`, `parser.go`: `Key []string`, `Collect string` on `SourceConfig` and its page mirror; `pageSourceConfig` copies them.
- `internal/config/stream_sources.go`: validate `key` and `collect` (merge only, field paths).
- `internal/source/stream.go`: pass key and collect to `hub.Merge`.
- `internal/runtime/expr.go`: field paths in `getFieldValue`; `internal/runtime/state.go`: `f.<path>` fields.
- Tests: `internal/stream/stream_test.go`, `internal/runtime/stream_state_test.go`, `stream_source_e2e_test.go`.
- Docs: `docs/sources/stream.md` (`key`, `collect`), `docs/guides/data-sources.md` (field paths), `CHANGELOG.md`.

onefile:

- `tests/index.md` new?: a `["*"]` merge with `key: [fields.Package, fields.Test]`, `collect: fields.Output`, `accepts: {min: 1, hint: ...}`; tabs on `fields.Action`; search; rows with output under failures.
- `tests/README.md` new?: run lines for `go test` with `watchexec`, `gotestsum --watch` and a plain re-run.
- `e2e/tests_test.go` new?, `e2e/testdata/tests/` new?: a small Go module with passing, failing and skipped tests.
- `README.md`: a row in the apps table.

## Tracker

| # | Type | Item | Severity | Status |
|---|---|---|---|---|
| A1 | assumption | A keyed row updated in place reaches the page as an `["u", key, diff]` op, not a full range redraw, through tinkerdown's GenericState refresh. | high | holds (`TestKeyedMerge_E2E` asserts the op) |
| A2 | assumption | Keying belongs in the hub's merge buffer, not per page: every page showing the merge shares one keyed view, as they share rows today. | medium | holds (`keyedRows` in the merge `Buffer`) |
| A3 | assumption | `watchexec` and `gotestsum --watch` both keep running and emit a new `go test -json` run per change. | medium | open (Phase 2) |
| R1 | risk | A new run reuses keys, so a test that was deleted keeps its last row until it is evicted or the server restarts. Documented; the planned mitigation (a package `start` marks rows stale) was dropped, since it is test-specific and the Constraints keep tinkerdown generic. | medium | accepted |
| R2 | risk | Collected output can be large for chatty tests. Mitigation: the 64 KiB cap per key, keeping the end, where failures print. | low | open |
| G1 | gap | Runners that write one JSON document at the end (jest, vitest) don't stream per test; they need a reporter that writes JSON lines. | low | unowned |

## Acceptance criteria

- [ ] `/simplify` has run on each phase's diff.
- [x] `go test ./internal/stream/...`: a keyed merge keeps one row per key in first-seen order, updates in place, evicts the least recently updated key past `buffer`, and caps `collected` at 64 KiB keeping the end.
- [x] `go test ./internal/runtime/...`: `fields.Action = fail`, `fields.Test ~ user` and `f.fields.Test` match nested fields; plain names behave as before.
- [x] Config validation refuses `key` or `collect` on a non-merge source, and a path that isn't a field name or `fields.<name>`.
- [x] A browser e2e (Docker Chrome, `TestKeyedMerge_E2E`): a test's row goes from `run` to `pass` in place, and the WebSocket frames carry `["u", …]` for it, not a redraw.
- [ ] onefile `cd e2e && go test -run TestTests ./...`: rows per test with the right status; the Failed tab shows only failures; a failing test shows its output; search narrows by name; phone layout has no horizontal scroll.
- [ ] The onefile page passes every remote-page check when served from its raw GitHub URL.
- [ ] Full tinkerdown suite green; pre-commit passes; `prereview check` clean on changed docs.

## Constraints

- Merges without `key` behave exactly as today.
- Nothing in tinkerdown knows about tests: keying, collecting and paths are generic.
- The onefile page stays a valid page from a URL: merge sources only, no actions, no scripts.
- e2e tests use real Docker Chrome and are never skipped.

## Contract

```
key: [PATH, ...]       on a merge: one row per distinct key, updated in place, first-seen order
collect: PATH          on a keyed merge: the field's text across a key's rows, in .collected (64 KiB, keep end)
collect_reset: PATH=V  a row matching PATH=V (V may be a|b) starts its key's .collected over
PATH                   NAME (a row field) or fields.NAME (a parsed JSON or logfmt field)
[Tab] fields.Action = fail   field paths work in tab and Filter expressions
f.fields.Test          Filter form fields take paths
```

## Phases

### Phase 0: spike (A1, A2)
- Audit: how `GenericState` builds `.Data` from a merge buffer, and livetemplate's kept-item update path (`generateKeptItemUpdateOps`).
- Implementation: a throwaway test that replaces one row in a buffer snapshot and dumps the WebSocket frames.
- Acceptance: A1 and A2 recorded here; the spike file is deleted.
- Learn: the audit found that pages key rows by `seq`, which changes on every update, so an update would be a remove and an insert. Keyed rows carry a stable `key` field for `data-key`. With that, livetemplate sends `["u", key, diff]`; the spike was folded into `TestKeyedMerge_E2E`, which asserts it.

### Phase 1: keyed merges, collect, field paths
- Audit: every caller of `hub.Merge` and `getFieldValue`.
- Implementation: per the tinkerdown Files list.
- Acceptance: `/simplify`, the stream, runtime, validation and e2e criteria, docs, the full suite.
- Learn: the onefile e2e found two gaps. A row without parsed fields (plain text, or a `system` row such as a restart) had `fields: nil`, and a template reading `.fields.Action` failed to render every update after it; rows now always carry a `fields` map. And `go test` reruns (a restart, or `watchexec`) appended each run's output to the last, so `collect_reset: PATH=VALUE` starts a key's text over; the page sets `fields.Action=run|start`, since a package row has a `start` but no `run`.

### Phase 2: the tests app
- Audit: A3. Re-read the Contract against what shipped.
- Implementation: the onefile files.
- Acceptance: `/simplify`, the onefile criteria, screenshots through chromedp.

## Delivery

1. Open the plan PR, get signoff, merge. Never merge without signoff.
2. Create `.worktrees/keyed-merges` after the plan is merged. One PR for Phases 0–1; release through `release.sh`; then Phase 2 in onefile, against the released tinkerdown.
