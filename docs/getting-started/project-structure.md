# Project Structure

## Directory Layout

A typical Tinkerdown project has the following structure:

```
myapp/
├── index.md              # Main page (required)
├── about.md              # Additional pages
├── contact.md
├── _data/                # Data files (optional)
│   ├── tasks.json
│   └── users.csv
├── static/               # Static assets (optional)
│   ├── styles.css
│   └── images/
├── tasks.db              # SQLite database (if using sqlite source)
└── tinkerdown.yaml       # Optional: for complex shared configuration
```

## File Types

### Markdown Pages (*.md)

Each `.md` file becomes a page in your app. Pages can contain:

- Standard Markdown content
- HTML with `lvt-*` attributes for interactivity
- Go template syntax for dynamic content

## Frontmatter Configuration (Recommended)

Configure everything in your page's frontmatter - no separate config file needed:

```yaml
---
title: My Dashboard
description: Real-time metrics display

sources:
  tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks

  users:
    type: json
    path: ./_data/users.json

styling:
  theme: clean
---
```

This keeps configuration close to where it's used and makes single-file apps possible.

### tinkerdown.yaml (Optional)

Use `tinkerdown.yaml` only when you need:

- **Shared sources** across multiple pages
- **Complex caching** with stale-while-revalidate strategies
- **Server settings** like custom ports
- **Global styling** applied to all pages

See [Configuration Reference](../reference/config.md) for details.

### Data Files (_data/)

The `_data/` directory contains static data files:

- JSON files (`.json`)
- CSV files (`.csv`)
- Markdown data files (`.md`)

### Static Assets (static/)

Static files like CSS, JavaScript, and images are served from the `static/` directory.

## Multi-Page Apps

Create additional pages by adding more `.md` files. Tinkerdown automatically:

- Generates navigation
- Creates routes for each page
- Maintains WebSocket connections per page

### Navigation Configuration

Control navigation order and visibility in each page's frontmatter:

```yaml
---
title: Settings
nav:
  order: 3           # Order in navigation
  title: Config      # Override title in nav
  hidden: false      # Hide from navigation
---
```

## Next Steps

- [Frontmatter Reference](../reference/frontmatter.md) - All frontmatter options
- [Configuration Reference](../reference/config.md) - When to use tinkerdown.yaml
