# <<.Title>>

A countries browser built with [Tinkerdown](https://github.com/livetemplate/tinkerdown).

## Quick Start

```bash
cd <<.ProjectName>>
tinkerdown serve
```

Open http://localhost:8080 in your browser.

## Features

- GraphQL API as data source
- Query defined in a separate `.graphql` file
- Nested field access (`continent.name`)
- Auto-rendered table via `lvt-columns`

## Project Structure

```
<<.ProjectName>>/
├── index.md                    # Countries browser page
├── queries/
│   └── countries.graphql       # GraphQL query
└── README.md                   # This file
```

## Customization

Edit `queries/countries.graphql` to change the query. Update `result_path` and `lvt-columns` in `index.md` to match.

## API

This template uses the free [Countries GraphQL API](https://countries.trevorblades.com/graphql) — no authentication required.

## Learn More

- [Tinkerdown Documentation](https://github.com/livetemplate/tinkerdown)
- [GraphQL Source Reference](https://github.com/livetemplate/tinkerdown/blob/main/docs/sources/graphql.md)
