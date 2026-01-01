# {{.Title}}

A task manager built with [Tinkerdown](https://github.com/livetemplate/tinkerdown) and SQLite.

## Running

```bash
cd {{.ProjectName}}
tinkerdown serve
```

Open http://localhost:8080 in your browser.

## Data Storage

Tasks are stored in `tasks.db` (SQLite). The database is created automatically on first run.
