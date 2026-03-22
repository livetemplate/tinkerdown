# Contributing Templates

How to add a new template to `tinkerdown new`.

## Directory Structure

Each template is a directory under `cmd/tinkerdown/commands/templates/`:

```
templates/
├── your-template/
│   ├── index.md          # Required: main page
│   ├── README.md         # Required: project README
│   ├── _data/            # Optional: data files
│   │   └── data.csv
│   └── queries/          # Optional: query files
│       └── query.graphql
└── CONTRIBUTING.md       # This file
```

## Template Delimiters

Templates use `<<` and `>>` as Go template delimiters (not `{{ }}`), because Tinkerdown's runtime templates use `{{ }}` for data binding.

**Correct:**
```
title: "<<.Title>>"
# <<.Title>>
cd <<.ProjectName>>
```

**Wrong (will not be substituted):**
```
title: "{{.Title}}"
title: "[[.Title]]"
```

## Available Template Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `<<.Title>>` | Project name converted to title case | `My App` |
| `<<.ProjectName>>` | Raw project name (directory basename) | `my-app` |

## File Processing Rules

Files are processed based on their extension:

| Extension | Processing |
|-----------|------------|
| `.md` | Go template (delimiters substituted) |
| `.yaml` | Go template (delimiters substituted) |
| `.sh` | Go template (delimiters substituted) + made executable |
| `.csv`, `.json`, `.graphql`, etc. | Copied as-is (no substitution) |

## Registering a Template

Add your template to `templateCatalog` in `cmd/tinkerdown/commands/new.go`:

```go
var templateCatalog = []templateInfo{
    // ...existing templates...
    {"your-template", "Short description of what it does", "Category"},
}
```

Categories: `Getting Started`, `Data Sources`, `Patterns`, `Advanced`.

## Naming Conventions

- Use kebab-case for template directory names: `csv-inventory`, `graphql-explorer`
- Keep names short and descriptive
- The name becomes the `--template=` flag value

## Testing

Add a test function in `cmd/tinkerdown/commands/new_test.go`:

```go
func TestNewCommandYourTemplate(t *testing.T) {
    tmpDir := t.TempDir()
    projectDir := filepath.Join(tmpDir, "test-project")
    defer chdir(t, tmpDir)()

    err := NewCommand([]string{"test-project"}, "your-template")
    if err != nil {
        t.Fatalf("NewCommand failed: %v", err)
    }

    assertFileExists(t, projectDir, "index.md")
    assertFileExists(t, projectDir, "README.md")

    // Verify template-specific content
    content := readFile(t, filepath.Join(projectDir, "index.md"))
    if !strings.Contains(content, "expected content") {
        t.Error("Expected content not found")
    }
}
```

Then run:

```bash
GOWORK=off go test ./cmd/tinkerdown/commands/ -run TestNewCommand -v
```

## Checklist

- [ ] Directory created under `templates/`
- [ ] `index.md` with frontmatter sources and content
- [ ] `README.md` with quick start, features, and structure
- [ ] Uses `<<` `>>` delimiters (not `[[` `]]` or `{{` `}}`)
- [ ] Registered in `templateCatalog` in `new.go`
- [ ] Test added in `new_test.go`
- [ ] Data files (CSV, JSON, etc.) included if needed
