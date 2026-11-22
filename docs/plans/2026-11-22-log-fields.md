# Plan: log rows that show every field and link their URLs

State — Done: nothing. Next: plan sign-off, then Phase 1 (livetemplate).

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
  cost: removing the first of two equal items renames the survivor from `<hash>-2` to `<hash>`, which the client sees as one remove and one add instead of no change. That is still cheaper than resending the range.
- livetemplate: `and` and `or` evaluate their arguments one at a time and stop at the first one that decides the result, as Go's `text/template` does since Go 1.18. This applies only to the builtins, not to a function a page registers under the same name.
  alternative: none considered
- tinkerdown: a template function `linkify`, added to `blockHelperFuncs` (`internal/server/websocket.go:1343`) next to `split`. It takes any value, escapes all of its text, and turns each `http://` or `https://` URL into `<a href="…" target="_blank" rel="noopener noreferrer">`.
  - A URL starts at `http://` or `https://`, matched without regard to case, with at least one character after it. Only that match becomes an `href`.
  - A URL ends at whitespace, `"`, `'`, `<`, `>` or `\`.
  - Trailing `.`, `,`, `;`, `:`, `!`, `?` and an unmatched `)` stay outside the link. A `)` that closes a `(` inside the URL stays in it.
  - It matches on the raw text, then escapes each piece on its own: text with `html.EscapeString`, and the URL the same way in both the link text and the `href`. Matching after escaping would see `&#34;` instead of `"` and cut `&amp;` at its `;`.
  - It returns `template.HTML`.

  alternative: a script on the page, as onefile#5's workaround does
  because: the server renders every row, so one function works on every page with no `MutationObserver`. It also keeps the range in stream mode, because a page script that rewrites rows fights the diff.
  alternative: fix #554 and split words in the template
  because: a template can't trim punctuation or escape a URL for an `href`
- tinkerdown: rename `internal/assets/vendor/` to `internal/assets/third_party/`, and update the `//go:embed` lines and `ReadFile` paths in `assets.go`.
  alternative: none considered
- tinkerdown: a test builds the module's file list with `golang.org/x/mod/zip.CheckDir` and fails if it leaves out any file under `internal/`.
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
- `internal/assets/module_test.go` new?: the `CheckDir` test.
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
| A1 | assumption | The livetemplate client accepts a key with a `-2` suffix: it treats keys as opaque strings. | high | open (Phase 1 audit) |
| A2 | assumption | A row with a nested `{{range}}` over `.fields` keeps the outer range in stream mode, so new log lines still arrive as append ops. | high | open (Phase 1 audit, with a livetemplate test before the release; Phase 3 confirms it in WebSocket frames) |
| A3 | assumption | `golang.org/x/mod/zip.CheckDir` reports `vendor/` files as omitted, so the new test fails before the rename. | medium | open (Phase 2) |
| A4 | assumption | A map value that is a number or bool prints correctly through `linkify`'s `fmt.Sprint`. | low | open (Phase 2) |
| R1 | risk | `linkify` returns HTML built from log text, so a bug there is an XSS hole. Mitigation: every piece of output passes through `html.EscapeString`, only `http`/`https` become links, and the tests cover `<script>`, `javascript:`, a quote inside a URL, `&` in a query and a URL ending in `&`. | high | open |
| A5 | assumption | tinkerdown works on livetemplate v0.25.13 with a client bundle rebuilt from `@livetemplate/client` 0.25.12: its full suite and the onefile suite pass. | medium | open (Phase 2) |
| R2 | risk | Suffixing duplicate keys renames the second and later duplicates, so a connection open across the upgrade sees one remove and add for each. Mitigation: none needed; a server restart already starts new connections. | low | accepted |

## Acceptance criteria

- [ ] `/simplify` has run on each phase's diff.
- [ ] livetemplate `go test ./...` passes, with new tests:
  - `{{range}}{{range}}…{{end}}{{end}}` gives each outer item its inner range and a distinct key
  - `split "a a b" " "` gives keys `h(a)`, `h(a)-2`, `h(b)`, with `h(a)` unchanged from v0.25.12, and a later update to that range sends item ops, not a whole-range resend
  - removing the first `a` sends one remove and one add for the survivor, and leaves `h(b)` untouched
  - `and` and `or` stop at the deciding argument, so the `slice` example renders `n` without a panic
  - a page's own `and` function is still called with every argument
- [ ] livetemplate's own e2e or client tests pass, covering A1.
- [ ] tinkerdown `GOWORK=off go test ./...` passes, with:
  - the `CheckDir` test, which fails on main (A3) and passes after the rename
  - `linkify` table tests:
    - text: plain text, one URL, two URLs, a number value
    - URL edges: trailing punctuation, a URL in parentheses, balanced parentheses (`https://en.wikipedia.org/wiki/Foo_(bar)`), `https://` with nothing after it, `HTTP://` in capitals
    - escaping: `<script>`, `javascript:alert(1)`, a `"` inside a URL, `&` in a query, a URL ending in `&`, `&quot;` next to a URL, a `template.HTML` input (still escaped)
- [ ] After the v0.7.4 tag: `GOMODCACHE=$(mktemp -d) GOFLAGS=-modcacherw go install github.com/livetemplate/tinkerdown/cmd/tinkerdown@v0.7.4` succeeds.
- [ ] onefile `TestLogViewer` (chromedp) checks, with console logs, server logs, WebSocket frames and HTML captured:
  - a slog line shows `subject=…`, `to=…` and `body=…` after its message
  - the URL in `body=` is one `<a target="_blank">` with the exact `href` and the whole URL as its text, even where the line wraps
  - a `<script>` in a line renders as text
  - a line appended after the page loads arrives as an append op (A2)
  - searching `sign` shows the row with the matching text visible
  - at 390px the page has no sideways scroll
- [ ] The full onefile suite passes against released v0.7.4, and pre-commit passes in each repo without `--no-verify`.
- [ ] `prereview check` passes on the changed docs and READMEs.

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
  - Check A2 with a test: a stream-mode range whose rows hold a nested range over a map still sends append ops.
- Implementation: per the Files list.
- Acceptance: `/simplify`, then the livetemplate criteria. PR, sign-off, then v0.25.13 through `scripts/release.sh`.
- Learn: recorded here.

### Phase 2: tinkerdown#364 and `linkify`
- Audit: A4, A5, where the docs list template functions, and every other mention of `assets/vendor` or `vendor/` in the repo (Makefile, `.gitignore`, CI workflows, lint config, the client bundle's copy step, docs).
- Implementation: commit the `CheckDir` test first and show it failing on main (A3), then the rename, then the rest per the Files list, with livetemplate bumped to v0.25.13.
- Acceptance: `/simplify`, then the tinkerdown criteria. PR, sign-off, then v0.7.4 through `release.sh`, the `go install` check and the Homebrew PR.
- Learn: recorded here.

### Phase 3: onefile#4 and #5
- Audit: re-read the `logs` page and its e2e fixtures against this plan.
- Implementation: per the Files list.
- Acceptance: `/simplify`, then the onefile criteria, verified in Docker Chrome through chromedp and never with `curl`. PR and sign-off.
- Learn: recorded here; close the four issues from their PRs.
