# Exec Source Enhancements Design (Issue #38)

**Date:** 2026-01-03
**Status:** Approved
**Issue:** #38 - 2.4 Exec Source

## Overview

Enhance the exec source with additional output formats (lines, CSV), environment variable support, and comprehensive tests including disabled-by-default verification.

## Decisions Made

| Topic | Decision | Rationale |
|-------|----------|-----------|
| Lines format | Simple line-by-line `{"line": "...", "index": N}` | Predictable, works with any output |
| CSV delimiter | Explicit with comma default | Covers 90% of cases, override when needed |
| Environment vars | Add `env` map to config | Matches REST source pattern, keeps secrets out of command |
| Command validation | None beyond exec.Command | Already safe from shell injection, user opted in |

## Config Structure

```yaml
sources:
  my_script:
    type: exec
    cmd: "./script.sh --flag value"
    format: json          # json (default), lines, csv
    delimiter: ","        # for csv format, default comma
    env:
      API_KEY: ${API_KEY}
      DEBUG: "true"
    timeout: 30s          # default 30s
```

## Implementation Details

### ExecSource Struct

```go
type ExecSource struct {
    name      string
    cmd       string
    siteDir   string
    format    string            // "json", "lines", "csv"
    delimiter string            // for csv, default ","
    env       map[string]string // environment variables (already expanded)
    timeout   time.Duration     // override default 30s
}
```

### Output Parsing

**Lines parser:**
- Split by newline, skip empty lines
- Each line becomes `{"line": "content", "index": N}`

**CSV parser:**
- First row is headers
- Use Go's `encoding/csv` package
- Configurable delimiter (default comma)
- Each row becomes `{"header1": "val1", "header2": "val2"}`

**JSON parser:**
- Existing logic unchanged
- Handles array, object, and NDJSON

### Environment Variables

- Inherit current environment via `os.Environ()`
- Custom env vars appended (override inherited)
- Expansion happens at construction time with `os.ExpandEnv()`

## Files to Modify

| File | Changes |
|------|---------|
| `parser.go` | Add `Format`, `Delimiter`, `Env` fields to `SourceConfig` |
| `internal/config/config.go` | Same fields in config's `SourceConfig` |
| `internal/source/exec.go` | New struct fields, `NewExecSourceWithConfig()`, lines/CSV parsers |
| `internal/source/exec_test.go` | Add 5 new tests |
| `internal/source/source.go` | Update factory to pass config |
| `internal/runtime/state.go` | Update exec source creation |

## Test Coverage

1. **TestExecSourceDisabledByDefault** - Verify error without `--allow-exec`
2. **TestExecSourceLinesFormat** - Plain text line parsing
3. **TestExecSourceCSVFormat** - CSV with headers
4. **TestExecSourceEnvVars** - Environment variable passing
5. **TestExecSourceCustomDelimiter** - TSV and other delimiters

## Out of Scope

- Command validation beyond exec.Command's safety
- Table/column auto-detection for lines format
- Shell execution mode (`sh -c`)

## Security Notes

- Exec sources remain disabled by default
- Requires explicit `--allow-exec` flag
- No shell interpretation (direct exec)
- Env vars expanded at config load, not at runtime
