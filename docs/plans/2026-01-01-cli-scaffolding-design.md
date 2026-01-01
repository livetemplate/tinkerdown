# Enhanced CLI Scaffolding Design (Phase 3.1)

**Date:** 2026-01-01
**Status:** Approved

## Overview

Extend `tinkerdown new` with a `--template` flag supporting 7 templates, each demonstrating a distinct Tinkerdown pattern.

## Command Syntax

```bash
tinkerdown new myapp                    # defaults to "basic"
tinkerdown new myapp --template=todo    # use specific template
tinkerdown new myapp -t dashboard       # short flag
```

## Template Directory Structure

```
cmd/tinkerdown/commands/templates/
├── basic/           # kubectl pods + jq (exec source)
├── tutorial/        # renamed from current basic (Go server state)
├── todo/            # SQLite CRUD task list
├── dashboard/       # REST + exec multi-source
├── form/            # Contact form → SQLite
├── api-explorer/    # GitHub search with query params
└── wasm-source/     # WASM source scaffold + test app
```

## Template Details

### basic/ - Kubernetes Pods Dashboard

```
basic/
├── index.md
├── get-pods.sh
└── README.md
```

**get-pods.sh:**
```bash
#!/bin/bash
kubectl get pods -o json | jq '[.items[] | {name: .metadata.name, status: .status.phase, ready: (.status.containerStatuses[0].ready // false)}]'
```

**index.md frontmatter:**
```yaml
title: "{{.Title}}"
sources:
  pods:
    type: exec
    cmd: ./get-pods.sh
```

Content: Table displaying pod name, status, ready state. Refresh button. Instructions for customizing the kubectl command.

### tutorial/ - Go Server State Tutorial

Current `basic/` template renamed. No content changes - same Go server state tutorial with increment/message demo.

### todo/ - SQLite CRUD

```
todo/
├── index.md
└── README.md
```

Based on `examples/lvt-source-sqlite-test/`. SQLite source with table (done/text/priority columns), add form, toggle/delete actions. Database created on first run.

### dashboard/ - Multi-source Display

```
dashboard/
├── index.md
├── system-info.sh
└── README.md
```

**Frontmatter:**
```yaml
sources:
  users:
    type: rest
    url: https://jsonplaceholder.typicode.com/users
  system:
    type: exec
    cmd: ./system-info.sh
```

Content: Two sections - "API Users" table and "System Info" (disk usage, etc.). Demonstrates mixing source types.

### form/ - Contact Form → SQLite

```
form/
├── index.md
└── README.md
```

Simple contact form (name, email, message) with SQLite persistence. Shows submitted entries in a table below the form.

### api-explorer/ - GitHub Search with Query Params

```
api-explorer/
├── index.md
└── README.md
```

**Frontmatter:**
```yaml
sources:
  repos:
    type: rest
    url: https://api.github.com/search/repositories?q=${QUERY}&per_page=10
```

Input field for search query driving parameterized REST calls.

### wasm-source/ - Custom WASM Source Scaffold

```
wasm-source/
├── source.go
├── Makefile
├── README.md
└── test-app/
    ├── index.md
    └── tinkerdown.yaml
```

**source.go:** Working TinyGo example fetching from httpbin.org.

**Makefile:**
```makefile
build:
	tinygo build -o source.wasm -target=wasi source.go

test: build
	cd test-app && tinkerdown serve

clean:
	rm -f source.wasm
```

**README.md:** Documents WASM interface contract, build instructions, testing with test-app.

## Implementation Changes

### new.go Modifications

1. Add `--template` / `-t` flag with default `"basic"`
2. Validate template exists in embedded FS
3. Walk template directory recursively (handle nested dirs like `wasm-source/test-app/`)
4. Process `.md`, `.yaml`, `.sh` files through Go templates for `{{.Title}}` and `{{.ProjectName}}` substitution
5. Copy other files (`.go`, `Makefile`) as-is
6. Make `.sh` files executable (`chmod +x`)

### Template Validation

```go
var validTemplates = []string{"basic", "tutorial", "todo", "dashboard", "form", "api-explorer", "wasm-source"}
```

### Success Output

```
✨ Created new app: myapp (template: todo)

📁 Project structure:
   myapp/
   ├── index.md
   └── README.md

🚀 Next steps:
   cd myapp
   tinkerdown serve
```

## Testing Strategy

- **Unit tests:** Each template generates valid file structure
- **E2E tests:** Generated app runs with `tinkerdown serve`
- **Validation:** All templates pass `tinkerdown validate`

## Template Summary

| Template | Source Type | Files | Key Demo |
|----------|-------------|-------|----------|
| `basic` | exec | index.md, get-pods.sh, README.md | kubectl + jq |
| `tutorial` | (Go server) | index.md, README.md | Current basic, renamed |
| `todo` | sqlite | index.md, README.md | CRUD task list |
| `dashboard` | rest + exec | index.md, system-info.sh, README.md | Multi-source |
| `form` | sqlite | index.md, README.md | Form → database |
| `api-explorer` | rest | index.md, README.md | Parameterized API |
| `wasm-source` | wasm | source.go, Makefile, README.md, test-app/ | Custom source scaffold |
