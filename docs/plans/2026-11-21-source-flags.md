# Plan: sources from the command line, so apps ship without demo sources

State — Done: nothing. Next: plan PR, then Phase 1.

## Context

The log viewer's `index.md` ships demo sources, because a tinkerdown page can't run without its sources written into it. Its review asked for a production page with no demo, and for a way to follow your own logs without editing the page ([#327](https://github.com/livetemplate/tinkerdown/issues/327)). This plan adds four generic pieces to tinkerdown:

- **`--source` flags:** `tinkerdown serve` takes stream sources on the command line.
- **`inputs: ["*"]`:** a merge source takes every stream source the page can see.
- **`[*] field` tabs:** a tabs heading gets one tab per input of the merge below it.
- **A serve prompt:** when a page needs sources and has none, `serve` asks for them in a terminal.
- **Help an agent can follow:** `tinkerdown serve --help` explains `--source` with examples and how to find sources, so a coding agent can set the app up from the help alone. Today `serve --help` ignores the flag and starts serving the current directory.

It also fixes one gap that a mistyped flag hits: a block whose source fails to build shows "Connecting..." forever.

The log viewer then ships with no sources. A user runs it like this:

```sh
tinkerdown serve --allow-exec . --source 'api=docker logs -f --timestamps api' --source 'app=file:./logs/app.log'
```

## Product

| User journey | What they see | Built on |
|---|---|---|
| Follow my logs without editing files | `serve ... --source name=...`, then a tab per source | `--source`, `inputs: ["*"]`, `[*] source` tabs |
| Start with no idea what to pass | `serve` asks, then prints the flags to reuse | serve prompt |
| Run from an agent or CI | no prompt; usage printed; page shows the empty state | terminal check |
| Let a coding agent set it up | `serve --help` says what to run and how to find sources | serve help |
| Mistype a source | the block shows the error, not "Connecting..." | block error |

A terminal run with no sources:

```
$ tinkerdown serve --allow-exec .
index.md follows logs, but no stream sources are set.
Add one as name=command or name=file:path. Leave empty to finish.
source> api=docker logs -f --timestamps api
source> app=file:./logs/app.log
source>
Next time, run:
  tinkerdown serve --allow-exec . --source 'api=docker logs -f --timestamps api' --source 'app=file:./logs/app.log'
🌐 Server running at http://localhost:8080
```

`tinkerdown serve --help` prints, and exits without serving:

```
Usage: tinkerdown serve [directory] [flags]

Serves the markdown pages in directory (default: current directory).

Flags:
  --source NAME=COMMAND    Follow a command's output as stream source NAME (needs --allow-exec)
  --source NAME=file:PATH  Follow a file as stream source NAME
  --allow-exec             Let sources run commands
  --port N, --host ADDR    Where to listen (default localhost:8080)
  ...

Finding sources to follow:
  docker ps --format '{{.Names}}'   -> --source 'NAME=docker logs -f --timestamps NAME'
  a project's start command          -> --source 'api=cd ../api && npm run dev'
  a log file                         -> --source 'app=file:/var/log/app.log'
  journalctl                         -> --source 'journal=journalctl -f -o short-iso'

Without a terminal, serve never prompts: pass every source as a flag.
```

Without a terminal (an agent, CI, a pipe), `serve` prints the same instructions once, without prompting, and starts. The page then shows its empty state:

```
┌ Logs ─────────────────────────────────────┐
│ [All]                                     │
├───────────────────────────────────────────┤
│ No log sources. Restart with              │
│   --source 'name=command'                 │
│   --source 'name=file:path'               │
└───────────────────────────────────────────┘
```

With the two sources above, the heading `## [All] | [*] source` renders as:

```
┌ Logs ─────────────────────────────────────┐
│ [All] [api] [app]                         │
│ 17:02:11 api  INFO  GET /users 200        │
│ 17:02:12 app  WARN  slow query 812ms      │
└───────────────────────────────────────────┘
```

A block whose source is invalid shows the error in place of its content:

```
┌───────────────────────────────────────────┐
│ ⚠ source "api": set exactly one of cmd or │
│   file                                    │
└───────────────────────────────────────────┘
```

## Decisions

- `--source name=value` adds a `stream` source. A value starting with `file:` follows that file; any other value is a command. The flag splits on the first `=` only, so commands may contain `=`.
  alternative: separate `--source-cmd` and `--source-file` flags
  because: one flag and one grammar serve both the command line and the prompt
- Flag sources take precedence over page frontmatter and `tinkerdown.yaml`, but not over approved sources. A flag naming an approved source fails at startup.
  because: a flag is the operator's explicit choice for this run, and approved sources are pinned by design (#324)
- A flag command still needs `--allow-exec`. Without it, `serve` refuses the flag at startup and names `--allow-exec`.
  alternative: treat a command typed by the operator as consent on its own
  because: one gate for every command is easier to reason about than per-source consent
- A merge marks "every stream source" with `inputs: ["*"]`, a list holding one `*`. It expands, sorted by name, to every stream source the merge can see: flag, page and site sources for an ordinary merge.
  alternative: a bare string `inputs: "*"` with a custom YAML unmarshaller
  because: `Inputs` stays `[]string`, and the config schema doesn't change shape
- An approved merge with `["*"]` expands through `mergeInputLookup`, so it sees only `tinkerdown.yaml` sources.
  because: otherwise page or flag sources would slip into an approved merge, which is the bypass #324 closed
- A merge whose `["*"]` expands to nothing is valid. Its block has no rows and `Inputs` is empty, and the page shows its own empty state. An explicit empty `inputs: []` stays an error.
  alternative: refuse to serve
  because: a page with no sources yet should still open and say what to do
- `GenericState` gains `Inputs []string`, the expanded input names of a merge block, for templates.
  alternative: none considered
- A tabs heading item `[*] field` expands to one tab per input of the lvt block's merge source, named after the input, with filter `field = input`. The parser records it as a dynamic tab group, and the server expands it when serving the page, after flags are known.
  alternative: render tabs in the block template from `.Inputs`, as `<button name="Filter" data-filter="source = {{.}}">`
  because: the heading keeps the tabs component's styling, ARIA roles and keyboard handling, and the page needs no tab markup
- `serve` prompts for sources only when stdin is a terminal and some page has a `["*"]` merge with no stream sources. Without a terminal it prints the instructions and serves the empty state.
  because: agents, CI and the e2e tests start `serve` without a terminal and must never hang
- `serve --help` (and `-h`) prints serve's own usage and exits 0 without serving. The usage leads with `--source`, gives copyable examples for docker, a dev server, a file and journald, and says that without a terminal every source must be a flag. The non-terminal message points to `serve --help`.
  because: an agent reads the help and the error output, not the docs site; the help has to be enough to finish the setup
- Prompt answers are not saved. After the prompt, `serve` prints the equivalent `--source` command line.
  alternative: write the answers to `tinkerdown.yaml`
  because: `serve` shouldn't change files the user didn't ask it to change
- A block whose source fails to build is sent to the page with its error as `.Error`, rendered by a default error view, instead of being dropped.
  alternative: none considered

## Files

tinkerdown (worktree `.worktrees/source-flags`):

- `cmd/tinkerdown/commands/serve.go`: parse `--source`; check them against `--allow-exec` and approved names; the prompt and the non-terminal message; `--help` and `-h`.
- `cmd/tinkerdown/main.go`: the top-level usage line for `serve` mentions `--source`.
- `cmd/tinkerdown/commands/source_flags.go` new?: `parseSourceFlag`, the prompt loop and `printReuse`, with tests.
- `internal/config/config.go`: hold flag sources apart from site sources (`FlagSources`).
- `internal/config/stream_sources.go`: `["*"]` in `ValidateStreamSource`; `ExpandMergeInputs(name, src, visible)`.
- `internal/server/websocket.go`: flag tier in `getEffectiveSource`; `["*"]` expansion through `mergeInputLookup`; send failed blocks with `.Error`.
- `internal/source/stream.go`: build a merge from the expanded inputs.
- `internal/runtime/state.go`: `Inputs` on `GenericState`.
- `parser.go`: record `[*] field` as a dynamic tab group.
- `internal/server/server.go`: expand dynamic tab groups when serving a page.
- `go.mod`: add `golang.org/x/term` for the terminal check.
- Tests: `source_flags_test.go` new?, `stream_source_e2e_test.go`, `internal/server/source_precedence_test.go`, `parser_test.go`.
- Docs: `docs/sources/stream.md` (`["*"]`, `Inputs`, `[*]` tabs), `docs/reference/cli.md` (`--source`), `docs/guides/data-sources.md` (tabs), `CHANGELOG.md`.

logviewer:

- `index.md`: no sources except `all: {type: merge, inputs: ["*"]}`; heading `## [All] | [*] source`; an empty state for `{{if not .Inputs}}`.
- `scripts/demo-logs.sh` and `logs/app.log` move to `e2e/testdata/`; the e2e test passes them as `--source` flags.
- `README.md` "Run it" and "Add your own logs": use `--source` and stop editing `index.md`. `AGENTS.md` shrinks to: install, ask the user which logs, run `tinkerdown serve --help`, then serve with `--source`.

## Tracker

Assumptions (A) are checked before building, risks (R) are what we'd act on, and gaps (G) are requirements no phase owns.

| # | Type | Item | Severity | Status |
|---|---|---|---|---|
| A1 | assumption | The server builds each page's HTML when serving it, or can post-process it, so dynamic tab groups expand per request with current sources. | high | open (Phase 2) |
| A2 | assumption | The client renders a block that arrives with only `.Error` set, so a failed block can reuse the normal path. | medium | open (Phase 1) |
| A3 | assumption | `serve` knows every page's sources before it listens (`Discover` runs first), so it can decide whether to prompt. | medium | open (Phase 3) |
| R1 | risk | Flag sources apply to every page of a multi-page site, which may surprise a site author. Mitigation: flags only feed pages that ask for them through `["*"]` or a name; document it. | medium | open |
| R2 | risk | The `[*]` tabs and the block's `.Inputs` could disagree if they resolve sources differently. Mitigation: both call `ExpandMergeInputs`. | medium | open |
| G1 | gap | Serving a page from a URL (#325) will want the same flag tier to override a remote page's sources. Not in this plan. | low | unowned |

## Acceptance criteria

- [ ] `/simplify` has run on each phase's diff.
- [ ] `go test ./cmd/tinkerdown/commands/...` covers `parseSourceFlag` (`a=b=c` keeps `b=c`; `file:` prefix; a missing name is refused), a command flag without `--allow-exec` refused, and a flag naming an approved source refused.
- [ ] `go test ./internal/config/...` covers `["*"]` expanding sorted, `["*"]` expanding to nothing being valid, and `inputs: []` still refused.
- [ ] `TestApprovedMergePinsItsInputs` gains a `["*"]` case: an approved merge sees no page or flag sources.
- [ ] A precedence test: a flag source beats a page source of the same name.
- [ ] `parser_test.go`: `## [All] | [*] source` records a dynamic tab group.
- [ ] `TestStreamSource_E2E` (chromedp, Docker Chrome) gains subtests:
  - a page with `inputs: ["*"]` served with two `--source` flags shows a tab per source, and each tab filters to it
  - with no sources, the page shows its empty state
  - an invalid source shows its error in the block
- [ ] `tinkerdown serve --help` prints the usage above and exits 0 without listening (a test in `cmd/tinkerdown/commands/cli_test.go`); every example in it is checked to parse with `parseSourceFlag`.
- [ ] The prompt: a test drives it through a pseudo-terminal and checks the printed reuse line; a run with stdin not a terminal doesn't block.
- [ ] logviewer: `cd e2e && go test ./...` passes with the demo passed as flags, and `index.md` has no `cmd` or `file`.
- [ ] Full tinkerdown suite green; pre-commit passes; `prereview check` clean on changed docs.

## Constraints

- A page with explicit `inputs` and fixed tabs behaves as today.
- Approved sources stay pinned: no flag, page or `["*"]` expansion reaches into an approved merge.
- `serve` never waits for input without a terminal.
- Commands, from any origin, run only with `--allow-exec`.
- e2e tests use real Docker Chrome and are never skipped.

## Contract

```
--source NAME=COMMAND        stream source running COMMAND (needs --allow-exec)
--source NAME=file:PATH      stream source following PATH
inputs: ["*"]                every stream source the merge can see, sorted by name
## [All] | [*] FIELD         one tab per merge input: label INPUT, filter "FIELD = INPUT"
.Inputs                      []string, a merge block's expanded inputs
```

Precedence, highest first: approved (`tinkerdown.yaml` with `generation.sources`), flags, page frontmatter, `tinkerdown.yaml`.

## Phases

### Phase 1: flags and `["*"]`
- Audit: A2. Trace `getEffectiveSource`, `mergeInputLookup` and `NewStreamSource` for where expansion belongs.
- Implementation: `--source` parsing and checks, the flag tier, `ExpandMergeInputs`, `Inputs`, block errors.
- Acceptance: `/simplify`, then the commands, config, precedence and approved-merge criteria.

### Phase 2: `[*]` tabs
- Audit: A1. Find where page HTML is produced per request, and how `--watch` reparses.
- Implementation: dynamic tab group in `parser.go`, expansion in the server.
- Acceptance: `/simplify`, the parser test and the e2e tab subtest.

### Phase 3: the prompt
- Audit: A3. Confirm `Discover` runs before listening.
- Implementation: terminal check, prompt loop, reuse line, non-terminal message, `serve --help`.
- Acceptance: `/simplify`, the prompt criterion, the e2e empty-state subtest, docs and CHANGELOG, the full suite.

### Phase 4: log viewer
- Audit: re-read the Contract against what shipped; write any drift into this plan in the same commit.
- Implementation: the logviewer files listed above.
- Acceptance: `/simplify`, the logviewer criterion, and a check in a real browser through chromedp screenshots.

## Delivery

1. Open the plan PR, run `/planprcommentsfix`, get signoff, merge. Never merge without signoff.
2. Create `.worktrees/source-flags` only after the plan is merged. One PR for Phases 1–3, then the logviewer change once it's released or built from `main`.
3. Release only through `release.sh`.
