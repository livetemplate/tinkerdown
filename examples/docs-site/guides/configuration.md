---
title: "Configuration"
---

# Configuration

Configure your LivePage site using `livepage.yaml`.

## Configuration File

Create a `livepage.yaml` file in your project root:

```yaml
title: "My Documentation"
description: "A comprehensive guide"
type: "site"

site:
  home: "index.md"
  logo: "/assets/logo.svg"
  repository: "https://github.com/username/repo"

navigation:
  - title: "Getting Started"
    path: "getting-started"
    collapsed: false
    pages:
      - title: "Introduction"
        path: "getting-started/intro.md"

server:
  port: 8080
  host: "localhost"
  debug: true

styling:
  theme: "clean"
  primary_color: "#007bff"
  font: "system-ui"

features:
  hot_reload: true
```

## Site Types

### Tutorial Mode

For single-page tutorials:

```yaml
type: "tutorial"
```

### Site Mode

For multi-page documentation:

```yaml
type: "site"
site:
  home: "index.md"
  logo: "/assets/logo.svg"
```

## Navigation

Define your site structure:

```yaml
navigation:
  - title: "Section Name"
    path: "section-path"
    collapsed: false
    pages:
      - title: "Page Name"
        path: "path/to/page.md"
```

## Server Options

Configure the development server:

```yaml
server:
  port: 8080        # Server port
  host: "localhost" # Server host
  debug: true       # Enable debug logging
```

## Styling

Customize the appearance:

```yaml
styling:
  theme: "clean"              # Theme name
  primary_color: "#007bff"    # Primary brand color
  font: "system-ui"           # Font family
```

## Features

Enable/disable features:

```yaml
features:
  hot_reload: true  # Auto-reload on file changes
```
