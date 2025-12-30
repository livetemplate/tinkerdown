# Data Source Error Handling

Tinkerdown provides robust error handling for data sources with automatic retries, circuit breakers, and configurable timeouts.

## Features

- **Unified error types** - Consistent error handling across all source types
- **Automatic retries** - Exponential backoff with configurable delays
- **Circuit breaker** - Prevents cascade failures when services are down
- **Configurable timeouts** - Per-source timeout settings

## Configuration

Configure error handling in your `tinkerdown.yaml`:

```yaml
sources:
  api:
    type: rest
    url: https://api.example.com/data
    timeout: "30s"      # Request timeout (default: 10s)
    retry:
      max_retries: 3    # Number of retry attempts (default: 3, set to 0 to disable)
      base_delay: "100ms"  # Initial retry delay (default: 100ms)
      max_delay: "5s"      # Maximum retry delay (default: 5s)

  database:
    type: pg
    query: "SELECT * FROM users"
    timeout: "60s"
    retry:
      max_retries: 5
      base_delay: "200ms"
      max_delay: "10s"
```

## Retry Behavior

The retry system uses exponential backoff:

1. First retry: `base_delay` (e.g., 100ms)
2. Second retry: `base_delay * 2` (e.g., 200ms)
3. Third retry: `base_delay * 4` (e.g., 400ms)
4. ...up to `max_delay`

### Retryable Errors

The following errors trigger automatic retries:

- **HTTP 5xx errors** (500, 502, 503, 504)
- **HTTP 429** (Too Many Requests)
- **Connection errors** (refused, reset, no route to host)
- **Timeouts** (context deadline exceeded)
- **Temporary network failures**

### Non-Retryable Errors

These errors fail immediately without retry:

- **HTTP 4xx errors** (except 429)
- **Validation errors** (invalid configuration)
- **Parse errors** (invalid JSON/response format)

## Circuit Breaker

The circuit breaker prevents overwhelming a failing service:

### States

1. **Closed** (normal) - Requests flow through normally
2. **Open** - Service is unhealthy, requests are blocked
3. **Half-Open** - Testing if service has recovered

### Behavior

- Circuit opens after 5 failures within a 1-minute window (failures do not need to be consecutive)
- When open, requests fail immediately with a friendly message
- After 30 seconds, circuit transitions to half-open
- 2 successful requests close the circuit

### Configuration

Circuit breaker settings are currently not user-configurable and use sensible defaults:

| Setting | Default | Description |
|---------|---------|-------------|
| Failure Threshold | 5 | Failures to open circuit |
| Success Threshold | 2 | Successes to close circuit |
| Timeout | 30s | Time before half-open |
| Failure Window | 1 minute | Window for counting failures |

## Error Messages

Tinkerdown provides user-friendly error messages for templates:

| Error Type | User Message |
|------------|--------------|
| Connection failed | "Could not connect to data source. Please check your connection." |
| Timeout | "Request timed out. Please try again." |
| Circuit open | "Service temporarily unavailable. Please try again later." |
| HTTP 401 | "Authentication required." |
| HTTP 403 | "Access denied." |
| HTTP 404 | "Resource not found." |
| HTTP 429 | "Too many requests. Please slow down." |
| HTTP 5xx | "Server error. Please try again later." |

## Supported Sources

Error handling with retry and circuit breaker is available for:

- **rest** - REST API endpoints
- **pg** - PostgreSQL databases

Other source types (json, csv, markdown, sqlite, exec) have basic error handling without retry/circuit breaker since they typically don't benefit from retry logic.

## Example

```yaml
# tinkerdown.yaml
sources:
  products:
    type: rest
    url: https://api.shop.com/products
    timeout: "15s"
    retry:
      max_retries: 3
      base_delay: "100ms"
      max_delay: "2s"
```

```markdown
<!-- products.md -->
# Products

```lvt-source
source: products
```

{{range .items}}
- **{{.name}}**: ${{.price}}
{{end}}
```

If the API is temporarily unavailable:
1. First request fails → retry after 100ms
2. Second request fails → retry after 200ms
3. Third request fails → retry after 400ms
4. Fourth request fails → show error message

If failures continue, the circuit breaker opens and subsequent requests fail immediately until the service recovers.
