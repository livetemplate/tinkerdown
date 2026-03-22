---
title: "<<.Title>>"
sources:
  countries:
    type: graphql
    from: https://countries.trevorblades.com/graphql
    query_file: queries/countries.graphql
    result_path: countries
---

# <<.Title>>

Browse countries from a public GraphQL API.

## Countries

```lvt
<table lvt-source="countries" lvt-columns="emoji:Flag,name:Country,code:Code,capital:Capital,currency:Currency,continent.name:Continent" lvt-empty="Loading countries...">
</table>
```

---

## How It Works

This app fetches data from a **GraphQL API** using a `.graphql` query file.

### GraphQL Source Configuration

```yaml
sources:
  countries:
    type: graphql
    from: https://countries.trevorblades.com/graphql
    query_file: queries/countries.graphql
    result_path: countries
```

- `from:` — the GraphQL endpoint URL
- `query_file:` — path to the `.graphql` query (relative to app directory)
- `result_path:` — dot-notation path to extract the array from the response

### Query File

The query lives in `queries/countries.graphql` — edit it to change which fields are fetched.

### Nested Fields

Use dot notation in `lvt-columns` for nested data: `continent.name:Continent` maps to `{ continent: { name: "..." } }`.
