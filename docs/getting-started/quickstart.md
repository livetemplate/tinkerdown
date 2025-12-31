# Quickstart Guide

Create your first Tinkerdown app in 5 minutes.

## Create a New App

```bash
tinkerdown new myapp
cd myapp
```

## Project Structure

```
myapp/
├── index.md           # Your main page (markdown + interactive elements)
└── _data/             # Optional data files
```

## Run the Development Server

```bash
tinkerdown serve
```

Open http://localhost:8080 in your browser.

## Add Interactivity

Edit `index.md` to add interactive elements:

```markdown
---
title: My First App
---

# Welcome to My App

<button lvt-click="SayHello">Click Me</button>

<div id="output">{{.message}}</div>
```

## Add a Data Source

Define sources directly in your page's frontmatter:

```markdown
---
title: Task List
sources:
  tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks
---

# My Tasks

<table lvt-source="tasks" lvt-columns="id,title,status">
</table>
```

### For Complex Configurations

If you have many sources or complex configurations shared across pages, you can use `tinkerdown.yaml`:

```yaml
# tinkerdown.yaml (optional - for complex multi-page apps)
sources:
  tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks
    cache:
      ttl: 5m
      strategy: stale-while-revalidate
```

See [Configuration Reference](../reference/config.md) for when to use `tinkerdown.yaml`.

## Next Steps

- [Data Sources](../guides/data-sources.md) - Connect to databases, APIs, and more
- [Auto-Rendering](../guides/auto-rendering.md) - Automatic tables, lists, and selects
- [Project Structure](project-structure.md) - Understanding the file layout
