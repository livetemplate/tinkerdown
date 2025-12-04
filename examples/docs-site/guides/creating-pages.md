---
title: "Creating Pages"
---

# Creating Pages

Learn how to create and structure your LivePage content.

## Basic Page Structure

Every LivePage document is a Markdown file with optional frontmatter:

```markdown
---
title: "My Page Title"
type: tutorial
---

# Page Heading

Your content goes here.
```

## Frontmatter Options

- `title` - The page title (shown in browser tab and navigation)
- `type` - Either "tutorial" or "site"
- `persist` - State persistence option ("localstorage", "sessionstorage", or "memory")
- `steps` - Number of steps in a tutorial

## Adding Content

Use standard Markdown syntax for content:

```markdown
## Headings

Use ## for major sections, ### for subsections.

## Lists

- Item 1
- Item 2
- Item 3

## Code Blocks

\`\`\`go
func main() {
    fmt.Println("Hello, LivePage!")
}
\`\`\`
```

## Interactive Blocks

Add server-side Go code with the `server` language tag:

```markdown
\`\`\`go server
type State struct {
    Counter int `json:"counter"`
}

// Increment handles the "increment" action
func (s *State) Increment(_ *livetemplate.ActionContext) error {
    s.Counter++
    return nil
}
\`\`\`
```

## Next Steps

Learn how to configure your site with [Configuration](/guides/configuration).
