# Configuration Reference (tinkerdown.yaml)

Reference for `tinkerdown.yaml` - the **optional** configuration file for complex apps.

> **Recommendation:** For most apps, configure sources directly in [frontmatter](frontmatter.md). Use `tinkerdown.yaml` only for shared configuration across multiple pages or complex setups.

## When to Use tinkerdown.yaml

| Use Case | Recommendation |
|----------|----------------|
| Single-page app | Use frontmatter |
| Simple multi-page app | Use frontmatter per page |
| Shared sources across pages | Use `tinkerdown.yaml` |
| Complex caching strategies | Use `tinkerdown.yaml` |
| Server settings (port, host) | Use `tinkerdown.yaml` |
| Secrets via environment variables | Use `tinkerdown.yaml` |

## File Location

Place `tinkerdown.yaml` in your app's root directory:

```
myapp/
├── tinkerdown.yaml    # Optional
├── index.md
└── ...
```

## Full Schema

```yaml
# Server settings (can't be in frontmatter)
server:
  port: 8080
  host: localhost

# REST API (optional)
api:
  enabled: true
  auth:
    api_key: ${API_KEY}
    header_name: X-API-Key
    keys:
      - name: reader
        key: ${READ_KEY}
        permissions: [read]
  cors:
    origins: ["http://localhost:3000"]
  rate_limit:
    requests_per_second: 10
    burst: 20
    max_tracked_ips: 10000

# Global styling (can also be per-page in frontmatter)
styling:
  theme: clean  # clean, dark, minimal

# Shared data sources
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

Server settings **must** be in `tinkerdown.yaml` (not available in frontmatter):

```yaml
server:
  port: 8080           # Server port (default: 8080)
  host: localhost      # Server host (default: localhost)
```

## API Configuration

The optional `api:` block enables a REST API for programmatic access to your app's data sources.

```yaml
api:
  enabled: true        # Enable REST API endpoints (default: false)
```

### Authentication

Configure `api.auth` to require API key authentication on all API endpoints.

#### Legacy single key

The simplest setup — one key with full permissions (read, write, delete):

```yaml
api:
  enabled: true
  auth:
    api_key: ${API_KEY}
```

#### Multi-key with permissions

For finer-grained access, define named keys with specific permissions:

```yaml
api:
  enabled: true
  auth:
    keys:
      - name: dashboard
        key: ${DASHBOARD_KEY}
        permissions: [read]
      - name: admin
        key: ${ADMIN_KEY}
        permissions: [read, write, delete]
```

Available permissions:

| Permission | Allowed HTTP methods |
|------------|---------------------|
| `read`     | GET, HEAD           |
| `write`    | POST, PUT, PATCH    |
| `delete`   | DELETE              |

> **Note:** `OPTIONS` requests bypass permission checks entirely to support CORS preflight.

Both formats can coexist — the legacy `api_key` is treated as a key named "default" with full permissions.

#### Custom header

By default, keys are sent via the `X-API-Key` header. To use Bearer token authentication instead:

```yaml
api:
  enabled: true
  auth:
    api_key: ${API_KEY}
    header_name: Authorization  # Expects "Authorization: Bearer <token>"
```

> **Secure default:** If `api_key` references an environment variable (e.g., `${API_KEY}`) and that variable is **not set**, authentication is still treated as **enabled**. No key will match, so all API requests are rejected. Auth is never silently disabled by a missing env var.

### CORS

Configure allowed origins for cross-origin API requests:

```yaml
api:
  enabled: true
  cors:
    origins:
      - "http://localhost:3000"
      - "https://myapp.example.com"
```

### Rate Limiting

Protect API endpoints with per-IP rate limiting:

```yaml
api:
  enabled: true
  rate_limit:
    requests_per_second: 10   # Requests per second per IP (default: 10)
    burst: 20                 # Burst allowance (default: 20)
    max_tracked_ips: 10000    # Max unique IPs tracked (default: 10000)
```

## Styling Configuration

Can be in frontmatter or `tinkerdown.yaml`. Config file applies globally:

```yaml
styling:
  theme: clean         # Theme name (default: clean)
  # Options: clean, dark, minimal
```

## Source Configuration

Sources in `tinkerdown.yaml` are available to **all pages**. Page-specific sources should go in frontmatter.

### Common Options

```yaml
sources:
  example:
    type: <source_type>    # Required: sqlite, rest, graphql, exec, json, csv, markdown, wasm
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
    path: ./data.db
    query: SELECT * FROM tasks
```

### REST Source

```yaml
sources:
  users:
    type: rest
    from: https://api.example.com/users
    method: GET
    headers:
      Authorization: Bearer ${API_TOKEN}
```

### GraphQL Source

```yaml
sources:
  issues:
    type: graphql
    from: https://api.github.com/graphql
    query_file: queries/issues.graphql  # Path to .graphql file
    variables:                           # Optional query variables
      owner: livetemplate
      repo: tinkerdown
    result_path: repository.issues.nodes # Dot-path to extract array
    options:
      auth_header: "Bearer ${GITHUB_TOKEN}"
```

GraphQL-specific options:

| Option | Required | Description |
|--------|----------|-------------|
| `query_file` | Yes | Path to `.graphql` file (relative to app directory) |
| `variables` | No | Map of query variables (supports `${ENV_VAR}` expansion) |
| `result_path` | Yes | Dot-notation path to extract array from response |
| `options.auth_header` | No | Authorization header value |

### Exec Source

```yaml
sources:
  system_info:
    type: exec
    command: uname -a
```

### JSON Source

```yaml
sources:
  config:
    type: json
    path: ./_data/config.json
```

### CSV Source

```yaml
sources:
  products:
    type: csv
    path: ./_data/products.csv
    delimiter: ","
    header: true
```

### Markdown Source

```yaml
sources:
  posts:
    type: markdown
    path: ./_data/posts/
    glob: "*.md"
```

### WASM Source

```yaml
sources:
  custom:
    type: wasm
    module: ./custom.wasm
    config:
      api_key: ${API_KEY}
```

## Caching Configuration

Caching is where `tinkerdown.yaml` shines - complex cache strategies:

```yaml
sources:
  api_data:
    type: rest
    from: https://api.example.com/data
    cache:
      ttl: 5m                  # Cache duration
      strategy: stale-while-revalidate  # Background refresh
```

### Cache Strategies

| Strategy | Description |
|----------|-------------|
| `simple` | Return cached data until TTL expires |
| `stale-while-revalidate` | Return stale data immediately, refresh in background |

## Environment Variables

Use `${VAR_NAME}` syntax for secrets - a key reason to use `tinkerdown.yaml`:

```yaml
sources:
  api:
    type: rest
    from: ${API_URL}
    headers:
      Authorization: Bearer ${API_TOKEN}
```

## Validation

Validate your configuration:

```bash
tinkerdown validate
```

## Example: Complex Multi-Page App

When you have shared authentication, caching, and multiple pages:

```yaml
# tinkerdown.yaml
server:
  port: 3000

styling:
  theme: dark

sources:
  # Shared auth - used by all pages
  current_user:
    type: rest
    from: ${AUTH_API}/me
    headers:
      Authorization: Bearer ${AUTH_TOKEN}
    cache:
      ttl: 10m
      strategy: stale-while-revalidate

  # Shared data with aggressive caching
  products:
    type: rest
    from: https://api.example.com/products
    cache:
      ttl: 1h
      strategy: stale-while-revalidate

  # Shared database
  orders:
    type: sqlite
    path: ./data/orders.db
    query: SELECT * FROM orders
```

Then each page uses these sources:

```markdown
---
title: Dashboard
# No need to redefine sources - they come from tinkerdown.yaml
---

# Dashboard

Welcome, {{.current_user.name}}!

<table lvt-source="orders" lvt-columns="id,product,status">
</table>
```

## Priority: Frontmatter vs Config File

When the same source is defined in both places:

1. **Frontmatter wins** for that page
2. Config file provides defaults

## Next Steps

- [Frontmatter Reference](frontmatter.md) - Recommended configuration approach
- [Data Sources Guide](../guides/data-sources.md) - Using sources
