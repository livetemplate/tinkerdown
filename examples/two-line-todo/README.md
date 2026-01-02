# Two-Line Todo

The simplest possible Tinkerdown app - a working todo list in just 5 lines of markdown.

## Quick Start

```bash
cd examples/two-line-todo
tinkerdown serve
```

Open http://localhost:8080 to see your interactive todo app.

## How It Works

Tinkerdown automatically detects task lists (`- [ ]` and `- [x]` syntax) and makes them interactive:
- Click a checkbox to toggle completion
- Changes persist to the markdown file

No configuration needed - pure markdown becomes a working app.

## What You Can Add

Want persistence or styling? Add frontmatter:

```markdown
---
title: "My Tasks"
persist: localstorage
---

## Tasks
- [ ] Task 1
```

This is Tier 1 (Pure Markdown) - the simplest form of a Tinkerdown app.
