# GraphQL Source Design

**Date:** 2025-12-31
**Status:** Approved
**Phase:** 4.1

## Overview

Add a GraphQL source type to tinkerdown, enabling data fetching from GraphQL APIs for display in tables, lists, and other lvt-source elements.

## Configuration

```yaml
sources:
  github_issues:
    type: graphql
    url: https://api.github.com/graphql
    query_file: queries/issues.graphql      # Required: path to .graphql file
    variables:                               # Optional: query variables
      owner: livetemplate
      repo: tinkerdown
    result_path: repository.issues.nodes    # Required: dot-path to extract array
    options:                                 # Optional: auth headers (same as REST)
      auth_header: "Bearer ${GITHUB_TOKEN}"
    timeout: 30s                            # Optional: request timeout
    cache:                                  # Optional: caching
      ttl: 5m
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `type` | Yes | Must be `graphql` |
| `url` | Yes | GraphQL endpoint URL |
| `query_file` | Yes | Relative path to `.graphql` file |
| `variables` | No | Map of variable name to value (supports `${ENV_VAR}` expansion) |
| `result_path` | Yes | Dot-notation path to extract array from response |
| `options` | No | Auth headers (same pattern as REST source) |
| `timeout` | No | Request timeout (default: 10s) |
| `cache` | No | Cache configuration (ttl, strategy) |
| `retry` | No | Retry configuration (max_retries, base_delay, max_delay) |

## Implementation

### New Files

```
internal/source/graphql.go      # GraphQL source implementation
internal/source/graphql_test.go # Unit tests
```

### Config Changes

Add to `internal/config/config.go` SourceConfig:

```go
QueryFile   string                 `yaml:"query_file,omitempty"`   // For graphql
Variables   map[string]interface{} `yaml:"variables,omitempty"`    // For graphql
ResultPath  string                 `yaml:"result_path,omitempty"`  // For graphql
```

### Source Registration

Add case in `internal/source/source.go` createSource():

```go
case "graphql":
    return NewGraphQLSource(name, cfg, siteDir)
```

### GraphQLSource Struct

```go
type GraphQLSource struct {
    name           string
    url            string
    queryFile      string
    variables      map[string]interface{}
    resultPath     string
    headers        map[string]string
    client         *http.Client
    retryConfig    RetryConfig
    circuitBreaker *CircuitBreaker
    siteDir        string
}
```

### Fetch Logic

1. Read query from file (relative to siteDir)
2. Build GraphQL request body: `{"query": "...", "variables": {...}}`
3. POST to endpoint with Content-Type: application/json
4. Add auth headers from options
5. Parse response, check for GraphQL errors
6. Extract array using result_path
7. Return data for template rendering

### Error Handling

- HTTP errors: Standard HTTP error handling with retry/circuit breaker
- GraphQL errors: If `errors` array present in response, fail with first error message
- Path extraction errors: Fail if result_path doesn't resolve to an array

### Path Extraction

```go
func extractPath(data map[string]interface{}, path string) ([]map[string]interface{}, error)
```

Example:
- Response: `{"repository": {"issues": {"nodes": [...]}}}`
- Path: `repository.issues.nodes`
- Result: The array at that path

## Testing

### Unit Tests

```go
func TestGraphQLSource_ReadQueryFile(t *testing.T)
func TestGraphQLSource_VariableExpansion(t *testing.T)
func TestExtractPath_Simple(t *testing.T)
func TestExtractPath_Nested(t *testing.T)
func TestExtractPath_NotFound(t *testing.T)
func TestExtractPath_NotArray(t *testing.T)
func TestGraphQLSource_GraphQLErrors(t *testing.T)
func TestGraphQLSource_HTTPErrors(t *testing.T)
```

### E2E Test

File: `lvtsource_graphql_e2e_test.go`
- Mock GraphQL server
- Full flow: config → fetch → render in browser

### Example

```
examples/lvt-source-graphql-test/
├── index.md
├── tinkerdown.yaml
└── queries/
    └── countries.graphql
```

## Documentation

Update:
- `docs/sources/graphql.md` - New file with full reference
- `docs/guides/data-sources.md` - Add GraphQL to overview
- `docs/reference/config.md` - Add new config fields

## Scope

### In Scope (v1)
- Read-only queries
- Query file support
- Variable substitution with env expansion
- Result path extraction
- Auth via headers (options)
- Retry/circuit breaker
- Caching support

### Out of Scope (future)
- Mutations (write operations)
- Subscriptions
- Inline queries in YAML
- Variables file
