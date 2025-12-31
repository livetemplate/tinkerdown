# Configuration Reference

Complete reference for `tinkerdown.yaml`.

## File Location

Place `tinkerdown.yaml` in your app's root directory:

```
myapp/
├── tinkerdown.yaml
├── index.md
└── ...
```

## Full Schema

```yaml
# Server settings
server:
  port: 8080
  host: localhost

# Styling options
styling:
  theme: clean  # clean, dark, minimal

# Data sources
sources:
  source_name:
    type: sqlite|rest|exec|json|csv|markdown|wasm
    # Type-specific options...
    cache:
      ttl: 5m
      strategy: simple|stale-while-revalidate
    timeout: 10s
```

## Server Configuration

```yaml
server:
  port: 8080           # Server port (default: 8080)
  host: localhost      # Server host (default: localhost)
```

## Styling Configuration

```yaml
styling:
  theme: clean         # Theme name (default: clean)
  # Options: clean, dark, minimal
```

## Source Configuration

### Common Options

All sources support these options:

```yaml
sources:
  example:
    type: <source_type>    # Required: sqlite, rest, exec, json, csv, markdown, wasm
    cache:                 # Optional: caching configuration
      ttl: 5m              # Time-to-live
      strategy: simple     # simple or stale-while-revalidate
    timeout: 10s           # Optional: request timeout
```

### SQLite Source

```yaml
sources:
  tasks:
    type: sqlite
    path: ./data.db        # Path to SQLite database
    query: SELECT * FROM tasks  # Query to execute
```

### REST Source

```yaml
sources:
  users:
    type: rest
    url: https://api.example.com/users  # API endpoint
    method: GET                         # HTTP method (default: GET)
    headers:                            # Optional headers
      Authorization: Bearer ${API_TOKEN}
    body: |                             # Optional request body (for POST/PUT)
      {"filter": "active"}
```

### Exec Source

```yaml
sources:
  system_info:
    type: exec
    command: uname -a       # Shell command to execute
    shell: /bin/sh          # Shell to use (default: /bin/sh)
```

### JSON Source

```yaml
sources:
  config:
    type: json
    path: ./_data/config.json  # Path to JSON file
```

### CSV Source

```yaml
sources:
  products:
    type: csv
    path: ./_data/products.csv  # Path to CSV file
    delimiter: ","              # Field delimiter (default: ,)
    header: true                # First row is header (default: true)
```

### Markdown Source

```yaml
sources:
  posts:
    type: markdown
    path: ./_data/posts/       # Directory containing markdown files
    glob: "*.md"               # File pattern (default: *.md)
```

### WASM Source

```yaml
sources:
  custom:
    type: wasm
    module: ./custom.wasm      # Path to WASM module
    config:                    # Optional config passed to module
      api_key: ${API_KEY}
```

## Caching Configuration

```yaml
sources:
  api_data:
    type: rest
    url: https://api.example.com/data
    cache:
      ttl: 5m                  # Cache duration (e.g., 5m, 1h, 30s)
      strategy: simple         # Caching strategy
```

### Cache Strategies

| Strategy | Description |
|----------|-------------|
| `simple` | Return cached data until TTL expires |
| `stale-while-revalidate` | Return stale data immediately, refresh in background |

## Environment Variables

Use `${VAR_NAME}` syntax for environment variable substitution:

```yaml
sources:
  api:
    type: rest
    url: ${API_URL}
    headers:
      Authorization: Bearer ${API_TOKEN}
```

## Validation

Validate your configuration:

```bash
tinkerdown validate
```

## Examples

### Minimal Configuration

```yaml
sources:
  tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks
```

### Multi-Source App

```yaml
styling:
  theme: clean

sources:
  tasks:
    type: sqlite
    path: ./data.db
    query: SELECT * FROM tasks

  users:
    type: rest
    url: https://api.example.com/users
    cache:
      ttl: 10m
      strategy: stale-while-revalidate

  system:
    type: exec
    command: uptime
```

## Next Steps

- [Data Sources Guide](../guides/data-sources.md) - Using sources
- [Frontmatter Reference](frontmatter.md) - Per-page configuration
