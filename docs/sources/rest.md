# REST Source

Fetch data from REST APIs.

## Configuration

```yaml
sources:
  users:
    type: rest
    url: https://api.example.com/users
```

## Options

| Option | Required | Description |
|--------|----------|-------------|
| `type` | Yes | Must be `rest` |
| `url` | Yes | API endpoint URL |
| `method` | No | HTTP method (default: GET) |
| `headers` | No | HTTP headers |
| `body` | No | Request body (for POST/PUT) |
| `timeout` | No | Request timeout (default: 10s) |

## Examples

### Basic GET

```yaml
sources:
  users:
    type: rest
    url: https://api.example.com/users
```

### With Authentication

```yaml
sources:
  secure_data:
    type: rest
    url: https://api.example.com/data
    headers:
      Authorization: Bearer ${API_TOKEN}
      Accept: application/json
```

### POST Request

```yaml
sources:
  search_results:
    type: rest
    url: https://api.example.com/search
    method: POST
    headers:
      Content-Type: application/json
    body: |
      {"query": "test"}
```

### With Query Parameters

```yaml
sources:
  filtered_users:
    type: rest
    url: https://api.example.com/users?status=active&limit=100
```

## Response Handling

REST sources expect JSON responses. The response is automatically parsed.

### Array Response

```json
[
  {"id": 1, "name": "Alice"},
  {"id": 2, "name": "Bob"}
]
```

Direct use:

```html
<table lvt-source="users" lvt-columns="id,name">
</table>
```

### Nested Response

```json
{
  "data": [
    {"id": 1, "name": "Alice"},
    {"id": 2, "name": "Bob"}
  ],
  "meta": {"total": 100}
}
```

The source automatically extracts arrays from common patterns like `data`, `items`, `results`.

## Caching

Enable caching for API rate limiting:

```yaml
sources:
  api_data:
    type: rest
    url: https://api.example.com/data
    cache:
      ttl: 5m
      strategy: stale-while-revalidate
```

## Error Handling

REST sources include built-in error handling:

```yaml
sources:
  external_api:
    type: rest
    url: https://api.example.com/data
    timeout: 30s
```

Features:
- Automatic retry with exponential backoff
- Circuit breaker for repeated failures
- Configurable timeout

## Environment Variables

Use environment variables for sensitive data:

```yaml
sources:
  private_api:
    type: rest
    url: ${API_URL}
    headers:
      Authorization: Bearer ${API_TOKEN}
```

## Full Example

```yaml
# tinkerdown.yaml
sources:
  github_repos:
    type: rest
    url: https://api.github.com/users/livetemplate/repos
    headers:
      Accept: application/vnd.github.v3+json
    cache:
      ttl: 10m
      strategy: stale-while-revalidate
```

```html
<!-- index.md -->
<h2>Repositories</h2>
<table lvt-source="github_repos" lvt-columns="name,description,stargazers_count:Stars">
</table>
```

## Next Steps

- [Exec Source](exec.md) - Execute shell commands
- [Caching](../caching.md) - Caching strategies
