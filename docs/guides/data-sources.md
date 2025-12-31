# Data Sources

Tinkerdown can connect to various data sources to power your interactive apps.

## Overview

Data sources are defined in your page's **frontmatter** (recommended) or in `tinkerdown.yaml` for shared sources.

## Quick Start

Add sources directly in your markdown file:

```markdown
---
title: My App
sources:
  tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks
---

<table lvt-source="tasks" lvt-columns="id,title,status">
</table>
```

## Available Source Types

| Type | Description | Use Case |
|------|-------------|----------|
| [sqlite](../sources/sqlite.md) | SQLite database | Local data storage, CRUD apps |
| [rest](../sources/rest.md) | REST API | External APIs, microservices |
| [graphql](../sources/graphql.md) | GraphQL API | GitHub API, complex queries |
| [exec](../sources/exec.md) | Shell commands | CLI tools, system info |
| [json](../sources/json.md) | JSON files | Static data, configuration |
| [csv](../sources/csv.md) | CSV files | Spreadsheet data, imports |
| [markdown](../sources/markdown.md) | Markdown files | Content management |
| [wasm](../sources/wasm.md) | WebAssembly modules | Custom sources |

## Frontmatter Configuration (Recommended)

Define sources in your page's frontmatter:

```yaml
---
title: Dashboard
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
---
```

## Using Sources in Pages

Reference sources using `lvt-source`:

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

## Shared Sources (tinkerdown.yaml)

For sources used across multiple pages, use `tinkerdown.yaml`:

```yaml
# tinkerdown.yaml
sources:
  # Available to all pages
  current_user:
    type: rest
    url: ${AUTH_API}/me
    headers:
      Authorization: Bearer ${AUTH_TOKEN}
```

See [Configuration Reference](../reference/config.md) for details.

## Caching

Enable caching for better performance:

```yaml
---
sources:
  users:
    type: rest
    url: https://api.example.com/users
    cache:
      ttl: 5m
      strategy: stale-while-revalidate
---
```

See the existing [Caching documentation](../caching.md) for details.

## Error Handling

Sources include built-in error handling:

- Automatic retry with exponential backoff
- Circuit breaker for failing sources
- Configurable timeout

```yaml
---
sources:
  external_api:
    type: rest
    url: https://api.example.com/data
    timeout: 30s
---
```

See the existing [Error Handling documentation](../error-handling.md) for details.

## Next Steps

- [SQLite Source](../sources/sqlite.md) - Database-backed apps
- [REST Source](../sources/rest.md) - REST API integrations
- [GraphQL Source](../sources/graphql.md) - GraphQL API integrations
- [Auto-Rendering](auto-rendering.md) - Automatic UI generation
