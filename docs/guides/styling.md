# Styling

Customize the appearance of your Tinkerdown apps.

## Themes

Tinkerdown includes built-in themes:

```yaml
# tinkerdown.yaml
styling:
  theme: clean   # Options: clean, dark, minimal
```

### Available Themes

| Theme | Description |
|-------|-------------|
| `clean` | Light theme with clean typography (default) |
| `dark` | Dark mode theme |
| `minimal` | Minimal styling, good base for customization |

## Custom CSS

Add custom styles in the `static/` directory:

```
myapp/
├── static/
│   └── styles.css
└── ...
```

Reference in your markdown:

```html
<link rel="stylesheet" href="/static/styles.css">
```

## CSS Variables

Customize themes using CSS variables:

```css
:root {
  --primary-color: #007bff;
  --background-color: #ffffff;
  --text-color: #333333;
  --border-radius: 4px;
  --font-family: system-ui, sans-serif;
}
```

## Component Styling

### Tables

```css
/* Auto-rendered tables */
table[lvt-source] {
  width: 100%;
  border-collapse: collapse;
}

table[lvt-source] th {
  background: var(--header-bg);
  text-align: left;
}

table[lvt-source] td {
  padding: 8px;
  border-bottom: 1px solid var(--border-color);
}
```

### Forms

```css
/* Form inputs */
input, select, textarea {
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius);
}

button {
  background: var(--primary-color);
  color: white;
  border: none;
  padding: 8px 16px;
  cursor: pointer;
}
```

### Lists

```css
/* Auto-rendered lists */
ul[lvt-source], ol[lvt-source] {
  list-style: none;
  padding: 0;
}

ul[lvt-source] li {
  padding: 8px;
  border-bottom: 1px solid var(--border-color);
}
```

## Responsive Design

Tinkerdown apps are responsive by default. Add custom breakpoints:

```css
@media (max-width: 768px) {
  .sidebar {
    display: none;
  }

  table[lvt-source] {
    font-size: 14px;
  }
}
```

## Dark Mode

Enable dark mode with the `dark` theme or implement a toggle:

```css
@media (prefers-color-scheme: dark) {
  :root {
    --background-color: #1a1a2e;
    --text-color: #ffffff;
  }
}
```

## Next Steps

- [Auto-Rendering](auto-rendering.md) - Understand generated HTML structure
- [Configuration Reference](../reference/config.md) - Styling configuration options
