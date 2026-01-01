# [[.Title]]

A custom WASM data source for tinkerdown.

## Prerequisites

- [TinyGo](https://tinygo.org/getting-started/install/) - Required to compile Go to WASM

## Building

```bash
make build
```

This produces `[[.ProjectName]].wasm`.

## Usage

Copy the `.wasm` file to your tinkerdown project and configure it:

```yaml
sources:
  mydata:
    type: wasm
    path: "./sources/[[.ProjectName]].wasm"
    options:
      category: "example"  # Available as os.Getenv("category")
```

## Project Structure

```
[[.ProjectName]]/
├── source.go     # WASM source implementation
├── Makefile      # Build commands
└── README.md     # This file
```

## Required Exports

Your WASM module must export these functions:

| Export | Signature | Description |
|--------|-----------|-------------|
| `fetch` | `() -> i32` | Fetch data, return pointer to JSON array |
| `get_result_len` | `() -> i32` | Return length of last fetch result |

## Optional Exports

| Export | Signature | Description |
|--------|-----------|-------------|
| `free_result` | `() -> void` | Free memory from last result |
| `write` | `(action_ptr, action_len, data_ptr, data_len) -> i32` | Handle write operations |
| `get_error` | `() -> i32` | Pointer to error string |
| `get_error_len` | `() -> i32` | Length of error string |

## Data Format

The `fetch` function must return a JSON array of objects:

```json
[
  {"id": 1, "title": "Item One", "category": "example"},
  {"id": 2, "title": "Item Two", "category": "example"}
]
```

## Configuration

Options passed in the source config are available as environment variables:

```yaml
sources:
  mydata:
    type: wasm
    path: "./sources/[[.ProjectName]].wasm"
    options:
      api_key: "${MY_API_KEY}"  # Env var substitution works
      category: "products"
```

In your Go code:
```go
apiKey := os.Getenv("api_key")
category := os.Getenv("category")
```

## Learn More

- [tinkerdown Documentation](https://github.com/livetemplate/tinkerdown)
- [WASM Source Reference](https://github.com/livetemplate/tinkerdown/blob/main/docs/sources/wasm.md)
- [TinyGo WASI Documentation](https://tinygo.org/docs/guides/webassembly/)
