# Plan: log rows that show every field and link their URLs

State — Done: plan signed off (2026-11-22, #366), Phase 1 (livetemplate#555, v0.25.13), Phase 2 (#367), the nested-alias fix (#369), v0.7.4, Phase 3 (onefile#6); all four issues closed. Next: the Homebrew formula (homebrew-tap#12).

## Context

This plan fixes four related issues, in three repos:

| Issue | Repo | What's wrong | Phase |
|---|---|---|---|
| [livetemplate#554](https://github.com/livetemplate/livetemplate/issues/554) | livetemplate | Some nested ranges render nothing, and `and` doesn't short-circuit | 1 |
| [tinkerdown#364](https://github.com/livetemplate/tinkerdown/issues/364) | tinkerdown | `go install …@latest` fails, because Go leaves `vendor/` directories out of a module's zip | 2 |
| [onefile#4](https://github.com/livetemplate/onefile/issues/4) | onefile | A log row shows only `msg`, so a URL in `body=` can't be seen | 3 |
| [onefile#5](https://github.com/livetemplate/onefile/issues/5) | onefile | URLs in a log line can't be clicked | 2, 3 |

The audit (2026-11-22, livetemplate main at v0.25.12, a throwaway test in `internal/parse`) found three separate bugs behind #554:

- **Lost content.** When a range item's whole body is another range, as in `{{range $w := …}}{{range $p := …}}…{{end}}{{end}}`, each outer item loses its content. Every item then gets the key `cbf29ce48422`, the hash of empty input. With a tag around the inner range, the same template renders correctly.
- **Duplicate keys.** Plain values are keyed by a hash of their content, so a word that appears twice (`split "a a" " "`) gives two items the same key. Since v0.25.10, a repeated key makes every update resend the whole range, so the list renders but is never patched in place.
- **No short-circuit.** `and` and `or` get all their arguments already evaluated (`evalFunctionCall`, `internal/parse/eval.go:209`). `{{if and (ge (len .s) 8) (eq (slice .s 0 7) "http://")}}` panics on a 2-character `.s`.

The audit also checked these:

- `takeString` deletes `time`, `level` and `msg` from a row's `fields` (`internal/stream/parse.go:195`). Ranging over `.fields` therefore shows only the other fields.
- livetemplate inserts a `template.HTML` value without escaping it (`valueToString`, `eval.go:629`).
- tinkerdown pins livetemplate v0.25.9 and `@livetemplate/client` `^0.25.11` (`client/package.json:18`). v0.25.10–v0.25.12 changed range keys (#547, #550), `bottom-sticky` and the embedded client.

## Product

A row shows its message, then each other field as `key=value`, with every `http://` or `https://` URL as a link that opens in a new tab. This is the slog line from onefile#4 on a 390px phone:

```
┌ Logs ───────────────────────────┐
│ 13:17:53.000 INFO               │
│ email                           │
│ subject=Your sign-in link       │
│ to=user@example.com             │
│ body=Sign in: http://localhost: │
│ 8080/signin/link?t=…  ← a link  │
└─────────────────────────────────┘
```

| User journey | What they see | Built on |
|---|---|---|
| Read a structured line | the message, then each other field, wrapped and muted | `{{range $k, $v := .fields}}` in `logs/index.md` |
| Open a link the app logged | the URL is a link, which opens in a new tab | `linkify` on `msg` and on each field value |
| Search for text in a field | the row that matches shows the matching text | fields are visible, and search already covers the whole line |

## Decisions

- livetemplate: a range item whose whole body is one range keeps that range as its content, so it's hashed and rendered like a wrapped one.
  alternative: none considered
- livetemplate: when two items in one range hash the same, the second and later get `<hash>-2`, `<hash>-3` and so on, in range order.
  alternative: key every plain value by its index
  because: index keys rename every item after an insert at the top, which turns a stream-mode prepend into a full update. Suffixing only the duplicates keeps every existing key byte-for-byte, which `appendJSONValue`'s wire-stability comment (`internal/keys/hash.go:85`) requires. It replaces v0.25.10's whole-range resend for plain values, while that fallback stays for keys a page sets with `data-key`.
  A suffixed key can't equal a real one, because real keys are 12 hex characters with no `-`. Suffixes go only on generated keys: a range keyed by `data-key` never gets them, so a page key like `abc-2` can't meet one.
  cost: removing the first of two equal items, or prepending an equal one, renames from `<hash>-2` to `<hash>`, so the client sees one remove and one add instead of no change or a pure insert. That is still cheaper than resending the range.
- livetemplate: `and` and `or` evaluate their arguments one at a time and stop at the first one that decides the result, as Go's `text/template` does since Go 1.18. This applies only to the builtins, not to a function a page registers under the same name.
  alternative: none considered
- tinkerdown: a template function `linkify`, added to `blockHelperFuncs` (`internal/server/websocket.go:1343`) next to `split`. It takes any value, escapes all of its text, and turns each `http://` or `https://` URL into `<a href="…" target="_blank" rel="noopener noreferrer">`.
  - A URL starts at `http://` or `https://`, matched without regard to case, with at least one character after it. Only that match becomes an `href`.
  - A URL ends at whitespace, `"`, `'`, `<`, `>` or `\`.
  - Trailing `.`, `,`, `;`, `:`, `!`, `?`, `”`, `’`, `。`, `，` and an unmatched `)`, `）`, `]` or `}` stay outside the link. A closing bracket that matches an opening one inside the URL stays in it.
  - Go's `regexp` (RE2), which runs in linear time, finds each candidate span, so a long line full of `http://` can't make it slow. A loop then trims the trailing punctuation and counts brackets, since RE2 can't balance them.
  - It matches on the raw text, then escapes each piece on its own: text with `html.EscapeString`, and the URL the same way in both the link text and the `href`. Matching after escaping would see `&#34;` instead of `"` and cut `&amp;` at its `;`.
  - It returns `template.HTML`.

  alternative: a script on the page, as onefile#5's workaround does
  because: the server renders every row, so one function works on every page with no `MutationObserver`. It also keeps the range in stream mode, because a page script that rewrites rows fights the diff.
  alternative: fix #554 and split words in the template
  because: a template can't trim punctuation or escape a URL for an `href`
- tinkerdown: rename `internal/assets/vendor/` to `internal/assets/third_party/`, and update the `//go:embed` lines and `ReadFile` paths in `assets.go`.
  alternative: none considered
- tinkerdown: a test builds the module's file list with `golang.org/x/mod/zip.CheckDir` and fails if it leaves out any file a `//go:embed` pattern uses (from `go list -json`).
  alternative: a CI step that runs `go install …@<tag>` from an empty module cache
  because: that runs only after a tag exists, too late to stop a broken release. `CheckDir` applies the same rules as the Go proxy, so the test also catches other omissions, such as a nested `go.mod`.
- onefile `logs`: each row shows `msg`, then a `<span class="fields">` holding one `<span>` per field, as `key=value`. Field names come from `range` over the map, sorted, as Go templates sort map keys. `msg` and every value go through `linkify`.
  alternative: show `.line` under the message (onefile#4's workaround)
  because: it shows the time and level twice
- onefile `logs`: fields wrap with `overflow-wrap: anywhere`, so a long token never scrolls the page sideways at 390px.
  alternative: none considered
- Releases go in order: livetemplate v0.25.13, then tinkerdown v0.7.4 (it needs v0.25.13 and adds `linkify` and the rename), then the onefile PR, which needs 0.7.4.
  alternative: none considered

## Files to modify

livetemplate (worktree `.worktrees/range-keys`):
- `internal/parse/range.go` (`buildRangeTreeWithStatics`) and, if the audit finds the item tree replaced there, `internal/parse/walk.go` new?: a body-only range kept, and duplicate keys suffixed.
- `internal/parse/eval.go` (`evalFunctionCall`): `and` and `or` evaluated lazily.
- `internal/parse/range_test.go`, `internal/parse/eval_test.go`: tests for the three bugs.
- `CHANGELOG.md`: `[Unreleased] ### Fixed`.

tinkerdown (worktree `.worktrees/log-links`):
- `internal/assets/vendor/` → `internal/assets/third_party/`, and `internal/assets/assets.go`.
- `module_zip_test.go` (module root): the `CheckDir` test. It moved from the planned `internal/assets/module_test.go` and checks the files `go list` reports as embedded, not every file under `internal/`, so the embeds in `cmd/tinkerdown/commands` are covered too.
- `internal/server/linkify.go` new? and `internal/server/linkify_test.go` new?; `internal/server/websocket.go`: `linkify` in `blockHelperFuncs`.
- `go.mod`, `go.sum`: livetemplate v0.25.13, and `golang.org/x/mod` (indirect today) as a direct requirement, used only by the test.
- `client/package.json`, `client/package-lock.json`, `internal/assets/client/*`: `@livetemplate/client` 0.25.12 and the rebuilt bundle (A5).
- The docs page that lists `split` (found in the Phase 2 audit): add `linkify`.
- `CHANGELOG.md`: `[Unreleased]`, with Added (`linkify`) and Fixed (`go install`, livetemplate's range keys and `and`).

onefile (worktree `.worktrees/log-fields`):
- `logs/index.md`: the fields span and `linkify`, plus CSS.
- `logs/README.md`, root `README.md`, `AGENTS.md`: needs tinkerdown 0.7.4.
- `e2e/logs_test.go`: the new checks, plus fixture lines with a URL and structured fields.
- `logs/docs/*.png`: retaken if the row's look changes.

## Tracker: assumptions and risks

| # | Type | Item | Severity | Status |
|---|---|---|---|---|
| A1 | assumption | The livetemplate client accepts a key with a `-2` suffix: it treats keys as opaque strings. | high | holds: the client reads `_k` as an opaque string (`getItemKey`, `findItemByKey` with `cssEscape`) |
| A2 | assumption | A row with a nested `{{range}}` over `.fields` keeps the outer range in stream mode, so new log lines still arrive as append ops. | high | held only after a fix: a map range visited keys in random order, so every update sent a reorder; livetemplate#555 sorts them |
| A3 | assumption | `golang.org/x/mod/zip.CheckDir` reports `vendor/` files as omitted, so the new test fails before the rename. | medium | holds: `CheckDir` omits all 18 files under `internal/assets/vendor/`, but reports absolute paths, so the test compares paths relative to the module root |
| A4 | assumption | A field value that isn't a string (a number, bool, nested map, slice or error) prints usefully through `linkify`'s `fmt.Sprint`. That's the intended behavior. | low | holds: `fmt.Sprint` prints maps (sorted) and slices, and the table tests cover both |
| R1 | risk | `linkify` returns HTML built from log text, so a bug there is an XSS hole. Mitigation: every piece of output passes through `html.EscapeString`, only `http`/`https` become links, and the tests cover `<script>`, `javascript:`, a quote inside a URL, `&` in a query and a URL ending in `&`. | high | open |
| A5 | assumption | tinkerdown works on livetemplate v0.25.13 with a client bundle rebuilt from `@livetemplate/client` 0.25.12: its full suite and the onefile suite pass. | medium | holds after removing five allowlist entries the new client no longer implements (Phase 2 Learn) |
| R2 | risk | Suffixing duplicate keys renames the second and later duplicates, so a connection open across the upgrade sees one remove and add for each. Mitigation: none needed; a server restart already starts new connections. | low | accepted |

## Acceptance criteria

- [x] `/simplify` has run on each phase's diff. ✅ (Phase 3's run also covered #369)
- [x] livetemplate `go test ./...` passes, with new tests: ✅ (livetemplate#555, `range_keys_test.go`, `internal/parse/issue554_test.go`)
  - `{{range}}{{range}}…{{end}}{{end}}` gives each outer item its inner range and a distinct key
  - `split "a a b" " "` gives keys `h(a)`, `h(a)-2`, `h(b)`, with `h(a)` unchanged from v0.25.12, and a later update to that range sends item ops, not a whole-range resend
  - removing the first `a`, or prepending another `a`, sends one remove and one add for the renamed item, and leaves `h(b)` untouched
  - `and` and `or` stop at the deciding argument and return its value, not a bool, so the `slice` example renders `n` without a panic. A nil or missing field as the first argument decides `and` as false, and `or` with no truthy argument returns its last one.
  - a page's own `and` function is still called with every argument
- [x] livetemplate's own e2e or client tests pass, covering A1. ✅ (livetemplate#555 CI)
- [x] tinkerdown `GOWORK=off go test ./...` passes, with: ✅ (#367, #369; `TestModuleZipKeepsEmbeddedFiles`, `TestLinkify`)
  - the `CheckDir` test, which fails on main (A3) and passes after the rename
  - `linkify` table tests:
    - text: plain text, one URL, two URLs, a number value, a map and a slice value (printed by `fmt.Sprint`, then linked and escaped)
    - URL edges: trailing punctuation, a URL in parentheses, balanced parentheses (`https://en.wikipedia.org/wiki/Foo_(bar)`), `https://` with nothing after it, `HTTP://` in capitals
    - URL edges: `((` nested in a URL, `(see https://a.com/x).`, a URL followed by `]` or `}`
    - URL edges, non-ASCII: a URL followed by `”`, `。` or `）`
    - escaping: `<script>`, `javascript:alert(1)`, `JavaScript:alert(1)`, `https://a.com/x"onmouseover=alert(1)` (the link stops at `"`), a `"` inside a URL, `&` in a query, a URL ending in `&`, `&quot;` next to a URL, a `template.HTML` input (still escaped)
- [x] After the v0.7.4 tag: `GOMODCACHE=$(mktemp -d) GOFLAGS=-modcacherw go install github.com/livetemplate/tinkerdown/cmd/tinkerdown@v0.7.4` succeeds. ✅
- [x] onefile `TestLogViewer` (chromedp) checks, with console logs, server logs, WebSocket frames and HTML captured: ✅ (onefile#6)
  - a slog line shows `subject=…`, `to=…` and `body=…` after its message
  - the URL in `body=` is one `<a target="_blank">` with the exact `href` and the whole URL as its text, even where the line wraps
  - a `<script>` in a line renders as text
  - a line appended after the page loads arrives as an append op (A2)
  - searching `sign` shows the row with the matching text visible
  - at 390px the page has no sideways scroll
- [x] The full onefile suite passes against released v0.7.4, and pre-commit passes in each repo without `--no-verify`. ✅ (run with the `go install`ed v0.7.4 binary)
- [x] `prereview check` passes on the changed docs and READMEs. ✅ (CHANGELOG's only findings are sentence length in older entries)

## Constraints

- Keys of non-duplicate range items stay byte-for-byte the same (`TestFormatHashPart_ByteEquivalentToJSON` and the existing range tests keep passing).
- `linkify` is generic: it knows nothing about logs and works on any value in any block.
- The onefile `logs` row template keeps one shape for every row: no `{{if}}` per row around the fields, so the range stays in stream mode.
- Each release and each PR merge waits for the user's sign-off.

## Phases

### Phase 1: livetemplate#554
- Audit:
  - Find where a body-only range's item tree loses its content (`walkAST` → `buildRangeTreeWithStatics` → `extractItemDynamics`).
  - Check whether stream mode's `RangeStreamState.Hashes` follows `AutoKey` or a separate content hash. If it's the content hash, duplicates need the suffix there too.
  - Check A1 in the client's keyed-range code.
  - Check A2 with a test: a stream-mode range whose rows hold a nested range over a map still sends append ops. Because map keys render sorted, an unchanged row produces no diff on a second render.
- Implementation: per the Files list.
- Acceptance: `/simplify`, then the livetemplate criteria. PR, sign-off, then v0.25.13 through `scripts/release.sh`.
- Learn:
  - A fourth bug: a range over a map visited its keys in Go's random order. livetemplate#555 sorts them like `text/template`.
  - `Execute` falls back to `html/template` when livetemplate's evaluator fails, so HTML-level tests passed on `main` for `and` and for map order. Tests of the evaluator check the built tree.
  - The multi-instance docker e2e couldn't start on `main` (Go 1.24 image, gitignored `go.sum`); livetemplate#555 fixes it.

### Phase 2: tinkerdown#364 and `linkify`
- Audit: A4, A5, where the docs list template functions, and every other mention of `assets/vendor` or `vendor/` in the repo (Makefile, `.gitignore`, CI workflows, lint config, the client bundle's copy step, docs).
- Implementation: run the `CheckDir` test on main and show it failing (A3). Because the pre-commit hook runs the suite, a failing test can't be committed alone, so it goes in with the rename. Then the rest per the Files list, with livetemplate bumped to v0.25.13.
- Acceptance: `/simplify`, then the tinkerdown criteria. PR, sign-off, then v0.7.4 through `release.sh`, the `go install` check and the Homebrew PR.
- Learn:
  - The docs listed a `safeHTML` function that doesn't exist; `docs/guides/go-templates.md` now lists `printf`, `split` and `linkify`.
  - A5 held after one fix: client 0.25.12 no longer ships prereview's attribute handlers, so `TestKnownAttributesAreReal` failed on five allowlist entries (`lvt-spy`, `data-lvt-text-caret`, `data-lvt-text-select-button`, `data-lvt-url-hash`, `data-lvt-viewport-items`). They're removed from `vocabulary.go`.

### Phase 3: onefile#4 and #5
- Audit: re-read the `logs` page and its e2e fixtures against this plan.
- Implementation: per the Files list.
- Acceptance: `/simplify`, then the onefile criteria, verified in Docker Chrome through chromedp and never with `curl`. PR and sign-off.
- Learn:
  - tinkerdown added a capitalised copy of every map key at every depth, so ranging over `.fields` showed each field twice (`ms` and `Ms`). The fix (signed off 2026-11-22) ships in v0.7.4, with copies only for state keys, a state map's keys and a row's own keys (`processMapValues`, `TestProcessStateMapAliases`).
  - Close the four issues from their PRs.
