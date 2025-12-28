---
title: "WASM Data Source"
sidebar: false
sources:
  quotes:
    type: wasm
    path: "./sources/quotes.wasm"
    options:
      category: "inspiration"
---

# WASM Data Source

This example demonstrates using a WebAssembly module as a data source.

WASM sources allow community-developed data sources to be distributed as `.wasm` modules, enabling:
- Custom data fetching logic
- Third-party API integrations
- Complex data transformations
- Sandboxed execution

## Quotes Display

```lvt
<main lvt-source="quotes">
    <h3>Inspirational Quotes</h3>
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <div style="display: flex; flex-direction: column; gap: 16px;">
        {{range .Data}}
        <blockquote style="border-left: 4px solid #007bff; padding-left: 16px; margin: 0;">
            <p style="font-style: italic; margin-bottom: 8px;">"{{.Text}}"</p>
            <footer style="color: #666;">— {{.Author}}</footer>
        </blockquote>
        {{end}}
    </div>
    {{end}}

    <button lvt-click="Refresh" style="margin-top: 16px;">Refresh</button>
</main>
```

## Configuration

```yaml
sources:
  quotes:
    type: wasm
    path: "./sources/quotes.wasm"  # Path to WASM module
    options:                        # Passed as env vars to module
      category: "inspiration"
```

## Building WASM Modules

WASM modules must export these functions:

### Required Exports

```
fetch() -> i32
  Returns pointer to JSON array result.
  Call get_result_len() after to get length.

get_result_len() -> i32
  Returns length of last fetch result.
```

### Optional Exports

```
free_result()
  Free memory from last result.

write(action_ptr i32, action_len i32, data_ptr i32, data_len i32) -> i32
  Handle write operations. Returns 0 on success.

get_error() -> i32
  Returns pointer to error string if write failed.

get_error_len() -> i32
  Returns length of error string.
```

### Example TinyGo Source

See `sources/quotes.go` for a complete example that can be compiled with:

```bash
tinygo build -o sources/quotes.wasm -target wasi sources/quotes.go
```

## Use Cases

- **API Integrations**: Fetch data from third-party APIs
- **Data Transformations**: Complex processing before display
- **Custom Protocols**: Support non-HTTP data sources
- **Community Sources**: Share data sources as distributable modules
