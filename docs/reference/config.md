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
    url: https://api.example.com/users
    method: GET
    headers:
      Authorization: Bearer ${API_TOKEN}
```

### GraphQL Source

```yaml
sources:
  issues:
    type: graphql
    url: https://api.github.com/graphql
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
    url: https://api.example.com/data
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
    url: ${API_URL}
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
    url: ${AUTH_API}/me
    headers:
      Authorization: Bearer ${AUTH_TOKEN}
    cache:
      ttl: 10m
      strategy: stale-while-revalidate

  # Shared data with aggressive caching
  products:
    type: rest
    url: https://api.example.com/products
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
