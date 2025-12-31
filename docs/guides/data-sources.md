# Data Sources

Tinkerdown can connect to various data sources to power your interactive apps.

## Overview

Data sources are defined in `tinkerdown.yaml` and can be referenced in your markdown pages using `lvt-source` attributes.

## Available Source Types

| Type | Description | Use Case |
|------|-------------|----------|
| [sqlite](../sources/sqlite.md) | SQLite database | Local data storage, CRUD apps |
| [rest](../sources/rest.md) | REST API | External APIs, microservices |
| [exec](../sources/exec.md) | Shell commands | CLI tools, system info |
| [json](../sources/json.md) | JSON files | Static data, configuration |
| [csv](../sources/csv.md) | CSV files | Spreadsheet data, imports |
| [markdown](../sources/markdown.md) | Markdown files | Content management |
| [wasm](../sources/wasm.md) | WebAssembly modules | Custom sources |

## Basic Configuration

```yaml
# tinkerdown.yaml
sources:
  tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks

  users:
    type: rest
    url: https://api.example.com/users

  system_info:
    type: exec
    command: uname -a
```

## Using Sources in Pages

Reference sources in your markdown using `lvt-source`:

```html
<!-- Auto-render a table -->
<table lvt-source="tasks" lvt-columns="id,title,status">
</table>

<!-- Auto-render a list -->
<ul lvt-source="users" lvt-field="name">
</ul>

<!-- Auto-render a select -->
<select lvt-source="categories" lvt-value="id" lvt-label="name">
</select>
```

## Caching

Enable caching to improve performance:

```yaml
sources:
  users:
    type: rest
    url: https://api.example.com/users
    cache:
      ttl: 5m
      strategy: stale-while-revalidate
```

See [Caching Guide](../caching.md) for details.

## Error Handling

Sources include built-in error handling:

- Automatic retry with exponential backoff
- Circuit breaker for failing sources
- Configurable timeouts

```yaml
sources:
  external_api:
    type: rest
    url: https://api.example.com/data
    timeout: 30s
    retry:
      max_attempts: 3
      backoff: exponential
```

See [Error Handling Guide](../error-handling.md) for details.

## Next Steps

- [SQLite Source](../sources/sqlite.md) - Database-backed apps
- [REST Source](../sources/rest.md) - API integrations
- [Auto-Rendering](auto-rendering.md) - Automatic UI generation
