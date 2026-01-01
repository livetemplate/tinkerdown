# GraphQL Source

Fetch data from GraphQL APIs.

## Configuration

```yaml
sources:
  issues:
    type: graphql
    url: https://api.github.com/graphql
    query_file: queries/issues.graphql
    result_path: repository.issues.nodes
```

## Options

| Option | Required | Description |
|--------|----------|-------------|
| `type` | Yes | Must be `graphql` |
| `url` | Yes | GraphQL endpoint URL |
| `query_file` | Yes | Path to `.graphql` file (relative to app directory) |
| `result_path` | Yes | Dot-notation path to extract array from response |
| `variables` | No | Map of query variables |
| `options` | No | Additional options (e.g., `auth_header`) |
| `timeout` | No | Request timeout (default: 10s) |
| `cache` | No | Cache configuration |

## Query File Format

Create a `.graphql` file with your query:

```graphql
# queries/issues.graphql
query GetIssues($owner: String!, $repo: String!) {
  repository(owner: $owner, name: $repo) {
    issues(first: 100) {
      nodes {
        number
        title
        state
        createdAt
        author {
          login
        }
      }
    }
  }
}
```

## Variables

Pass variables to your GraphQL query:

```yaml
sources:
  issues:
    type: graphql
    url: https://api.github.com/graphql
    query_file: queries/issues.graphql
    variables:
      owner: livetemplate
      repo: tinkerdown
    result_path: repository.issues.nodes
```

Variables support environment variable expansion:

```yaml
variables:
  owner: ${GITHUB_ORG}
  repo: ${GITHUB_REPO}
```

## Result Path Extraction

GraphQL responses often have nested data. Use `result_path` to extract the array:

### Response Structure

```json
{
  "data": {
    "repository": {
      "issues": {
        "nodes": [
          {"number": 1, "title": "Bug fix"},
          {"number": 2, "title": "Feature request"}
        ]
      }
    }
  }
}
```

### Configuration

```yaml
result_path: repository.issues.nodes
```

The source automatically extracts the `data` field from the GraphQL response, then navigates through the specified path (`repository.issues.nodes`) and returns the array.

## Authentication

Use `options.auth_header` for authenticated APIs:

```yaml
sources:
  github_issues:
    type: graphql
    url: https://api.github.com/graphql
    query_file: queries/issues.graphql
    variables:
      owner: livetemplate
      repo: tinkerdown
    result_path: repository.issues.nodes
    options:
      auth_header: "Bearer ${GITHUB_TOKEN}"
```

## Caching

Enable caching for rate-limited APIs:

```yaml
sources:
  github_data:
    type: graphql
    url: https://api.github.com/graphql
    query_file: queries/data.graphql
    result_path: repository.releases.nodes
    cache:
      ttl: 5m
      strategy: stale-while-revalidate
```

## Error Handling

GraphQL sources include built-in error handling:

- **HTTP errors**: Automatic retry with exponential backoff
- **GraphQL errors**: Detected and reported from the `errors` array in response
- **Path extraction errors**: Clear error if result_path doesn't resolve to an array
- **Circuit breaker**: Prevents repeated requests to failing endpoints

## Full Example

### Directory Structure

```
myapp/
├── tinkerdown.yaml
├── index.md
└── queries/
    └── issues.graphql
```

### Configuration

```yaml
# tinkerdown.yaml
sources:
  github_issues:
    type: graphql
    url: https://api.github.com/graphql
    query_file: queries/issues.graphql
    variables:
      owner: livetemplate
      repo: tinkerdown
    result_path: repository.issues.nodes
    options:
      auth_header: "Bearer ${GITHUB_TOKEN}"
    cache:
      ttl: 10m
      strategy: stale-while-revalidate
```

### Query File

```graphql
# queries/issues.graphql
query GetIssues($owner: String!, $repo: String!) {
  repository(owner: $owner, name: $repo) {
    issues(first: 50, states: OPEN, orderBy: {field: CREATED_AT, direction: DESC}) {
      nodes {
        number
        title
        state
        createdAt
        author {
          login
        }
        labels(first: 5) {
          nodes {
            name
          }
        }
      }
    }
  }
}
```

### Page

```html
<!-- index.md -->
---
title: GitHub Issues
---

# Open Issues

<table lvt-source="github_issues" lvt-columns="number,title,state,author.login:Author">
</table>
```

## Environment Variables

Use environment variables for sensitive data:

```yaml
sources:
  private_api:
    type: graphql
    url: ${GRAPHQL_ENDPOINT}
    query_file: queries/data.graphql
    result_path: data.items
    options:
      auth_header: "Bearer ${API_TOKEN}"
```

## Next Steps

- [REST Source](rest.md) - REST API integrations
- [Caching](../caching.md) - Caching strategies
- [Error Handling](../error-handling.md) - Error handling details
