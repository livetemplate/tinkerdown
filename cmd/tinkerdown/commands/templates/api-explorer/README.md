# {{.Title}}

A GitHub repository search tool built with [Tinkerdown](https://github.com/livetemplate/tinkerdown).

## Running

```bash
cd {{.ProjectName}}
tinkerdown serve
```

Open http://localhost:8080 in your browser.

## API

Uses the GitHub Search Repositories API. Rate limited to 10 requests/minute for unauthenticated requests.
