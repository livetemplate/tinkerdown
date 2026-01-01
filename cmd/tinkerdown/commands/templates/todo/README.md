# [[.Title]]

A todo list application built with tinkerdown.

## Quick Start

```bash
cd [[.ProjectName]]
tinkerdown serve
```

Then open http://localhost:8080

## Features

- SQLite-backed persistent storage
- Add, complete, and delete tasks
- Real-time updates via WebSocket

## Project Structure

```
[[.ProjectName]]/
├── index.md      # Main application page
├── tasks.db      # SQLite database (created automatically)
└── README.md     # This file
```

## Customization

Edit `index.md` to:
- Add more fields (priority, due date, category)
- Change the styling
- Add filtering or sorting

## Learn More

- [tinkerdown Documentation](https://github.com/livetemplate/tinkerdown)
- [SQLite Source Reference](https://github.com/livetemplate/tinkerdown/blob/main/docs/sources/sqlite.md)
