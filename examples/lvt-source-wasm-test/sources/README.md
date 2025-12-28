# WASM Source Module

This directory contains example source code for a WASM data source module.

## Building

Requires [TinyGo](https://tinygo.org/getting-started/install/):

```bash
tinygo build -o quotes.wasm -target wasi quotes.go
```

## Interface

WASM modules must export these functions to work with tinkerdown:

### Required

| Export | Signature | Description |
|--------|-----------|-------------|
| `fetch` | `() -> i32` | Fetch data, return pointer to JSON array |
| `get_result_len` | `() -> i32` | Return length of last fetch result |

### Optional

| Export | Signature | Description |
|--------|-----------|-------------|
| `free_result` | `() -> void` | Free memory from last result |
| `write` | `(action_ptr, action_len, data_ptr, data_len) -> i32` | Handle write ops, return 0=success |
| `get_error` | `() -> i32` | Pointer to error string |
| `get_error_len` | `() -> i32` | Length of error string |

## Configuration

Options passed to the source in `tinkerdown.yaml` are available as environment variables:

```yaml
sources:
  quotes:
    type: wasm
    path: "./sources/quotes.wasm"
    options:
      category: "inspiration"  # Available as os.Getenv("category")
      api_key: "${API_KEY}"    # Env var substitution works
```

## Data Format

The `fetch` function must return a pointer to a JSON array of objects:

```json
[
  {"id": 1, "text": "Quote text", "author": "Author Name"},
  {"id": 2, "text": "Another quote", "author": "Another Author"}
]
```

Each object becomes a row in the template's `.Data` array.
