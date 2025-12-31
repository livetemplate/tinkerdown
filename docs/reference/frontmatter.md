# Frontmatter Reference

Page-level configuration using YAML frontmatter. **This is the recommended way to configure Tinkerdown apps.**

## Overview

Add YAML frontmatter at the beginning of any markdown page:

```markdown
---
title: My Page
description: A page description
sources:
  tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks
---

# Page Content

<table lvt-source="tasks" lvt-columns="id,title,status">
</table>
```

## Available Options

### title

Page title (used in `<title>` and navigation).

```yaml
---
title: Dashboard
---
```

### description

Page description (used in meta tags).

```yaml
---
description: View and manage your tasks
---
```

### sources

Define data sources for this page. This is the **recommended** way to configure sources.

```yaml
---
sources:
  tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks

  users:
    type: rest
    url: https://api.example.com/users

  config:
    type: json
    path: ./_data/config.json
---
```

#### Source Types

All source types can be defined in frontmatter:

| Type | Example |
|------|---------|
| `sqlite` | `type: sqlite`<br>`path: ./data.db`<br>`query: SELECT * FROM tasks` |
| `rest` | `type: rest`<br>`url: https://api.example.com/data` |
| `json` | `type: json`<br>`path: ./_data/data.json` |
| `csv` | `type: csv`<br>`path: ./_data/data.csv` |
| `exec` | `type: exec`<br>`command: uname -a` |
| `markdown` | `type: markdown`<br>`path: ./_data/posts/` |
| `wasm` | `type: wasm`<br>`module: ./custom.wasm` |

See [Data Sources Guide](../guides/data-sources.md) for full details on each type.

### styling

Page styling options.

```yaml
---
styling:
  theme: clean    # Options: clean, dark, minimal
---
```

### layout

Page layout template.

```yaml
---
layout: wide    # Options: default, wide, minimal
---
```

### nav

Navigation settings.

```yaml
---
nav:
  order: 1           # Order in navigation
  title: Home        # Override title in nav
  hidden: false      # Hide from navigation
---
```

### auth (Future)

Authentication requirements.

```yaml
---
auth: required
# or
auth:
  provider: github
  allowed_orgs: [mycompany]
---
```

## Complete Example

A fully-configured single-file app:

```markdown
---
title: Task Dashboard
description: Manage your daily tasks

sources:
  tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks ORDER BY created_at DESC

  categories:
    type: json
    path: ./_data/categories.json

styling:
  theme: clean

nav:
  order: 1
  title: Tasks
---

# Task Dashboard

## All Tasks

<table lvt-source="tasks" lvt-columns="title:Task,status:Status,category:Category" lvt-actions="Edit,Delete">
</table>

## Add Task

<form lvt-submit="AddTask">
  <input name="title" placeholder="Task title" required>
  <select name="category" lvt-source="categories" lvt-value="id" lvt-label="name">
  </select>
  <button type="submit">Add Task</button>
</form>
```

## When to Use tinkerdown.yaml Instead

Use `tinkerdown.yaml` for:

- **Shared sources** used across multiple pages
- **Complex caching** configurations
- **Server settings** (port, host)
- **Environment variables** with secrets

```yaml
# tinkerdown.yaml - for complex multi-page apps
server:
  port: 3000

sources:
  # Shared across all pages
  auth_user:
    type: rest
    url: ${AUTH_API_URL}/user
    headers:
      Authorization: Bearer ${AUTH_TOKEN}
    cache:
      ttl: 10m
      strategy: stale-while-revalidate
```

See [Configuration Reference](config.md) for `tinkerdown.yaml` details.

## Multi-Page Navigation

For multi-page apps, Tinkerdown auto-generates navigation from frontmatter:

```
myapp/
├── index.md          # nav.order: 1 (Home)
├── tasks.md          # nav.order: 2 (Tasks)
├── settings.md       # nav.order: 3 (Settings)
└── about.md          # nav.hidden: true (not in nav)
```

## Accessing Frontmatter in Templates

Frontmatter values are available in templates:

```html
<h1>{{.Title}}</h1>
<p>{{.Description}}</p>
```

## Next Steps

- [Data Sources Guide](../guides/data-sources.md) - Detailed source configuration
- [Configuration Reference](config.md) - When to use tinkerdown.yaml
