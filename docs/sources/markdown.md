# Markdown Source

Load content from markdown files.

## Configuration

```yaml
sources:
  posts:
    type: markdown
    path: ./_data/posts/
```

## Options

| Option | Required | Description |
|--------|----------|-------------|
| `type` | Yes | Must be `markdown` |
| `path` | Yes | Path to directory containing markdown files |
| `glob` | No | File pattern (default: `*.md`) |

## Examples

### Basic Usage

```yaml
sources:
  articles:
    type: markdown
    path: ./_data/articles/
```

### Custom Pattern

```yaml
sources:
  docs:
    type: markdown
    path: ./_data/content/
    glob: "**/*.md"
```

## Markdown File Format

Each markdown file should have YAML frontmatter:

```markdown
---
title: My First Post
date: 2024-01-15
author: Alice
tags: [tech, tutorial]
---

# Introduction

This is the content of my post...
```

## Data Structure

Each file becomes an object with:

- All frontmatter fields
- `content`: The rendered HTML content
- `raw_content`: The raw markdown content
- `filename`: The filename without extension
- `path`: The file path

## Usage in Templates

### List All Posts

```html
{{range .posts}}
<article>
  <h2><a href="/post/{{.filename}}">{{.title}}</a></h2>
  <time>{{.date}}</time>
  <p>{{.description}}</p>
</article>
{{end}}
```

### Auto-Rendering Table

```html
<table lvt-source="posts" lvt-columns="title,date,author">
</table>
```

### Display Single Post

```html
{{with (index .posts 0)}}
<article>
  <h1>{{.title}}</h1>
  <div class="content">{{.content}}</div>
</article>
{{end}}
```

## File Organization

```
myapp/
├── _data/
│   └── posts/
│       ├── first-post.md
│       ├── second-post.md
│       └── drafts/
│           └── upcoming.md
├── index.md
└── tinkerdown.yaml
```

## Use Cases

- Blog posts
- Documentation pages
- FAQ content
- Changelog entries
- Team member profiles

## Full Example

```markdown
# _data/posts/hello-world.md
---
title: Hello World
date: 2024-01-15
author: Alice
description: My first post
tags: [intro, welcome]
---

Welcome to my blog! This is my first post.

## What to Expect

I'll be writing about:
- Technology
- Development
- Life
```

```yaml
# tinkerdown.yaml
sources:
  posts:
    type: markdown
    path: ./_data/posts/
```

```html
<!-- index.md -->
<h1>Blog</h1>

{{range .posts}}
<article class="post-preview">
  <h2>{{.title}}</h2>
  <p class="meta">{{.date}} by {{.author}}</p>
  <p>{{.description}}</p>
  <a href="/posts/{{.filename}}">Read more</a>
</article>
{{end}}
```

## Next Steps

- [WASM Source](wasm.md) - Custom sources with WebAssembly
- [Data Sources Guide](../guides/data-sources.md) - Overview
