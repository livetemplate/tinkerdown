# E2E Black-Box Testing Strategy for Deterministic Output

> **Status**: Proposal
> **Date**: 2025-12-31
> **Goal**: Verify tinkerdown produces deterministic, expected output for given inputs

## Problem Statement

Current e2e tests:
- Use `strings.Contains()` - doesn't catch unintended output changes
- Rely on `Sleep()` for timing - slow and flaky
- No snapshot/golden file testing - can't detect output drift
- Inline assertions only - no reusable expected outputs

We need black-box testing that verifies:
1. **Markdown → HTML** transformation is deterministic
2. **Source data rendering** produces expected output
3. **CLI commands** produce expected output
4. **HTTP API responses** match expectations

## Recommended Approach

### Layer 1: Golden File Testing (HTML Output)

Use golden files to verify rendered output matches expectations.

**Library**: Custom minimal implementation or `github.com/sebdah/goldie`

```go
// internal/testing/golden.go
func Assert(t *testing.T, name string, actual []byte) {
    golden := filepath.Join("testdata", "golden", name)
    if *update {
        os.WriteFile(golden, actual, 0644)
        return
    }
    expected, _ := os.ReadFile(golden)
    if !bytes.Equal(actual, expected) {
        t.Errorf("output mismatch for %s\ndiff:\n%s", name, diff(expected, actual))
    }
}
```

**Test structure:**
```
testdata/
├── fixtures/                    # Input files
│   ├── simple.md
│   ├── with-table.md
│   └── with-sources.md
└── golden/                      # Expected outputs
    ├── simple.html
    ├── with-table.html
    └── with-sources.html
```

**Usage:**
```go
func TestRenderSimple(t *testing.T) {
    input := readFixture(t, "simple.md")
    output := render(input)
    golden.Assert(t, "simple.html", output)
}
```

**Update golden files:**
```bash
go test -update ./...
```

### Layer 2: Scrubbers for Non-Deterministic Data

Some output contains dynamic data (timestamps, IDs). Use scrubbers to normalize.

```go
type Scrubber func([]byte) []byte

var DefaultScrubbers = []Scrubber{
    // Replace timestamps with placeholder
    func(b []byte) []byte {
        re := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)
        return re.ReplaceAll(b, []byte("TIMESTAMP"))
    },
    // Replace UUIDs with placeholder
    func(b []byte) []byte {
        re := regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
        return re.ReplaceAll(b, []byte("UUID"))
    },
    // Replace WebSocket connection IDs
    func(b []byte) []byte {
        re := regexp.MustCompile(`ws-conn-[a-zA-Z0-9]+`)
        return re.ReplaceAll(b, []byte("WS-CONN-ID"))
    },
}

func Scrub(data []byte, scrubbers ...Scrubber) []byte {
    for _, s := range scrubbers {
        data = s(data)
    }
    return data
}
```

### Layer 3: CLI Black-Box Testing with testscript

Use `github.com/rogpeppe/go-internal/testscript` for CLI testing.

**Test file format** (`.txtar`):
```txtar
# testdata/scripts/serve_basic.txtar

# Setup: create test markdown file
exec cat input.md
stdout 'Hello'

# Run tinkerdown render (future command)
exec tinkerdown render input.md
stdout '<!DOCTYPE html>'
stdout '<h1>Hello</h1>'

# Compare full output
cmp stdout expected.html

-- input.md --
# Hello

World
-- expected.html --
<!DOCTYPE html>
<html>
<head>...</head>
<body>
<h1>Hello</h1>
<p>World</p>
</body>
</html>
```

**Test runner:**
```go
func TestScripts(t *testing.T) {
    testscript.Run(t, testscript.Params{
        Dir: "testdata/scripts",
        Setup: func(env *testscript.Env) error {
            // Add tinkerdown binary to PATH
            return nil
        },
    })
}
```

### Layer 4: HTTP Response Testing

For source endpoints and API responses:

```go
func TestSourceEndpoint(t *testing.T) {
    srv := setupTestServer(t, "testdata/fixtures/with-sources.md")

    resp, _ := http.Get(srv.URL + "/_sources/users")
    body, _ := io.ReadAll(resp.Body)

    // Scrub and compare
    scrubbed := Scrub(body, DefaultScrubbers...)
    golden.Assert(t, "source-users-response.json", scrubbed)
}
```

### Layer 5: Visual Regression (Future)

For pixel-perfect rendering verification:
- Screenshot comparison with tolerance
- CSS regression detection
- Could use Playwright or similar

## Test Categories

| Category | Tool | What It Tests |
|----------|------|---------------|
| Render output | Golden files | Markdown → HTML transformation |
| CLI commands | testscript | Command-line interface behavior |
| Source data | Golden + scrubbers | Data source fetching and rendering |
| API responses | HTTP golden | Endpoint response format |
| Interactive | chromedp (existing) | WebSocket and user interactions |

## Migration Plan

### Phase 1: Foundation (Week 1)
- [ ] Create `testdata/` directory structure
- [ ] Implement minimal golden file helper
- [ ] Implement basic scrubbers

### Phase 2: HTML Testing (Week 2)
- [ ] Add golden file tests for core rendering
- [ ] Cover: headings, paragraphs, code blocks, tables
- [ ] Cover: lvt-* attribute rendering

### Phase 3: Source Testing (Week 3)
- [ ] Add golden tests for source data rendering
- [ ] Add mock sources for deterministic testing
- [ ] Test: JSON, CSV, exec, markdown sources

### Phase 4: CLI Testing (Week 4)
- [ ] Add testscript infrastructure
- [ ] Test: serve command output
- [ ] Test: error messages

### Phase 5: Existing Test Improvement
- [ ] Replace `Sleep()` with polling/waiting
- [ ] Add golden file assertions to existing tests
- [ ] Remove flaky timing-dependent tests

## Example Test Suite

```go
// render_test.go
package tinkerdown_test

import (
    "testing"
    "github.com/livetemplate/tinkerdown/internal/testutil"
)

func TestRender(t *testing.T) {
    tests := []struct {
        name    string
        fixture string
        golden  string
    }{
        {"simple heading", "simple.md", "simple.html"},
        {"with table", "with-table.md", "with-table.html"},
        {"lvt source", "lvt-source.md", "lvt-source.html"},
        {"code blocks", "code-blocks.md", "code-blocks.html"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            input := testutil.ReadFixture(t, tt.fixture)
            output := render(input)
            testutil.Golden(t, tt.golden, output)
        })
    }
}
```

## Benefits

| Benefit | Description |
|---------|-------------|
| **Regression detection** | Any output change is caught |
| **Documentation** | Golden files document expected behavior |
| **Easy updates** | Single `-update` flag refreshes all |
| **Fast feedback** | String comparison is instant |
| **Deterministic** | Scrubbers remove non-determinism |
| **Portable** | Text files work everywhere |

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Brittle tests | Careful golden file structure, good scrubbers |
| Large diffs | Focus on semantic content, not formatting |
| Update fatigue | Group related tests, update in batches |
| Missing coverage | Combine with existing behavior tests |

## Decision

**Recommended**: Start with Layer 1 (Golden files) + Layer 2 (Scrubbers) as they provide the highest value with lowest implementation cost. Layer 3 (testscript) can be added for comprehensive CLI testing.

## References

- [goldie](https://github.com/sebdah/goldie) - Golden file testing
- [testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript) - CLI testing
- [go-cmp](https://github.com/google/go-cmp) - Semantic comparison
- [chromedp](https://github.com/chromedp/chromedp) - Browser automation (existing)
