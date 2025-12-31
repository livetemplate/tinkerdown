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
└── tinkerdown.yaml    # Configuration file
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

Edit `tinkerdown.yaml` to add a data source:

```yaml
sources:
  tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks
```

Then use it in your markdown:

```html
<table lvt-source="tasks" lvt-columns="id,title,status">
</table>
```

## Next Steps

- [Data Sources](../guides/data-sources.md) - Connect to databases, APIs, and more
- [Auto-Rendering](../guides/auto-rendering.md) - Automatic tables, lists, and selects
- [Project Structure](project-structure.md) - Understanding the file layout
