# Project Structure

## Directory Layout

A typical Tinkerdown project has the following structure:

```
myapp/
├── index.md              # Main page (required)
├── about.md              # Additional pages
├── contact.md
├── tinkerdown.yaml       # Configuration file
├── _data/                # Data files (optional)
│   ├── tasks.json
│   └── users.csv
├── static/               # Static assets (optional)
│   ├── styles.css
│   └── images/
└── tasks.db              # SQLite database (if using sqlite source)
```

## File Types

### Markdown Pages (*.md)

Each `.md` file becomes a page in your app. Pages can contain:

- Standard Markdown content
- HTML with `lvt-*` attributes for interactivity
- Go template syntax for dynamic content

### Configuration (tinkerdown.yaml)

The main configuration file that defines:

- Data sources
- Styling options
- Server settings

See [Configuration Reference](../reference/config.md) for full details.

### Data Files (_data/)

The `_data/` directory contains static data files:

- JSON files (`.json`)
- CSV files (`.csv`)
- Markdown data files (`.md`)

### Static Assets (static/)

Static files like CSS, JavaScript, and images are served from the `static/` directory.

## Frontmatter

Each markdown page can have YAML frontmatter:

```yaml
---
title: My Page
description: A description of the page
sources:
  - tasks
  - users
---
```

See [Frontmatter Reference](../reference/frontmatter.md) for all options.

## Multi-Page Apps

Create additional pages by adding more `.md` files. Tinkerdown automatically:

- Generates navigation
- Creates routes for each page
- Maintains WebSocket connections per page

## Next Steps

- [Configuration Reference](../reference/config.md) - Full tinkerdown.yaml schema
- [Frontmatter Reference](../reference/frontmatter.md) - Page-level options
