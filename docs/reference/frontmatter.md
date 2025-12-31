# Frontmatter Reference

Page-level configuration using YAML frontmatter.

## Overview

Add YAML frontmatter at the beginning of any markdown page:

```markdown
---
title: My Page
description: A page description
---

# Page Content
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

Sources used by this page (for documentation/validation).

```yaml
---
sources:
  - tasks
  - users
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

## Full Example

```yaml
---
title: Task Dashboard
description: Manage your daily tasks
sources:
  - tasks
  - categories
layout: wide
nav:
  order: 1
  title: Tasks
---

# Task Dashboard

<table lvt-source="tasks" lvt-columns="title,status,category">
</table>
```

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

- [Configuration Reference](config.md) - Global configuration
- [Project Structure](../getting-started/project-structure.md) - File layout
