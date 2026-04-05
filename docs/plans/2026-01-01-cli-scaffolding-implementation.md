# CLI Scaffolding Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `--template` flag to `tinkerdown new` command with 7 templates demonstrating different Tinkerdown patterns.

**Architecture:** Rename existing `templates/basic/` to `templates/tutorial/`, create 6 new template directories, update `new.go` to parse `--template` flag and recursively process template directories with variable substitution.

**Tech Stack:** Go, embed.FS, text/template, cobra-style CLI flags

---

## Task 1: Rename basic → tutorial

**Files:**
- Rename: `cmd/tinkerdown/commands/templates/basic/` → `cmd/tinkerdown/commands/templates/tutorial/`

**Step 1: Rename the directory**

```bash
cd /Users/adnaan/code/livetemplate/tinkerdown/.worktrees/cli-scaffolding
mv cmd/tinkerdown/commands/templates/basic cmd/tinkerdown/commands/templates/tutorial
```

**Step 2: Verify the rename**

```bash
ls cmd/tinkerdown/commands/templates/
```
Expected: `tutorial/` directory exists

**Step 3: Commit**

```bash
git add -A
git commit -m "refactor: rename templates/basic to templates/tutorial"
```

---

## Task 2: Create basic template (kubectl pods)

**Files:**
- Create: `cmd/tinkerdown/commands/templates/basic/index.md`
- Create: `cmd/tinkerdown/commands/templates/basic/get-pods.sh`
- Create: `cmd/tinkerdown/commands/templates/basic/README.md`

**Step 1: Create basic directory**

```bash
mkdir -p cmd/tinkerdown/commands/templates/basic
```

**Step 2: Create get-pods.sh**

```bash
cat > cmd/tinkerdown/commands/templates/basic/get-pods.sh << 'EOF'
#!/bin/bash
# Fetch Kubernetes pods and format as JSON array
# Customize this command for your cluster

kubectl get pods -o json 2>/dev/null | jq '[.items[] | {
  name: .metadata.name,
  namespace: .metadata.namespace,
  status: .status.phase,
  ready: (.status.containerStatuses[0].ready // false)
}]' 2>/dev/null || echo '[]'
EOF
chmod +x cmd/tinkerdown/commands/templates/basic/get-pods.sh
```

**Step 3: Create index.md**

Create file `cmd/tinkerdown/commands/templates/basic/index.md`:

```markdown
---
title: "{{.Title}}"
sources:
  pods:
    type: exec
    cmd: ./get-pods.sh
---

# {{.Title}}

A simple Kubernetes pods dashboard built with Tinkerdown.

## Pods

```lvt
<main lvt-source="pods">
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    <p>Make sure kubectl is configured and accessible.</p>
    {{else if not .Data}}
    <p>No pods found. Run <code>kubectl get pods</code> to verify your cluster connection.</p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>Name</th>
                <th>Namespace</th>
                <th>Status</th>
                <th>Ready</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>{{.name}}</td>
                <td>{{.namespace}}</td>
                <td>{{.status}}</td>
                <td>{{if .ready}}✓{{else}}✗{{end}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}

    <button name="Refresh" style="margin-top: 16px;">Refresh</button>
</main>
```

## Customizing

Edit `get-pods.sh` to change what data is displayed. For example:

- Show deployments: `kubectl get deployments -o json`
- Filter by namespace: `kubectl get pods -n my-namespace -o json`
- Show services: `kubectl get services -o json`
```

**Step 4: Create README.md**

Create file `cmd/tinkerdown/commands/templates/basic/README.md`:

```markdown
# {{.Title}}

A Kubernetes pods dashboard built with [Tinkerdown](https://github.com/livetemplate/tinkerdown).

## Prerequisites

- `kubectl` configured with cluster access
- `jq` installed for JSON processing

## Running

```bash
cd {{.ProjectName}}
tinkerdown serve
```

Open http://localhost:8080 in your browser.

## Customizing

Edit `get-pods.sh` to change the kubectl command or jq filter.
```

**Step 5: Commit**

```bash
git add cmd/tinkerdown/commands/templates/basic/
git commit -m "feat: add basic template (kubectl pods)"
```

---

## Task 3: Create todo template (SQLite CRUD)

**Files:**
- Create: `cmd/tinkerdown/commands/templates/todo/index.md`
- Create: `cmd/tinkerdown/commands/templates/todo/README.md`

**Step 1: Create todo directory**

```bash
mkdir -p cmd/tinkerdown/commands/templates/todo
```

**Step 2: Create index.md**

Create file `cmd/tinkerdown/commands/templates/todo/index.md`:

```markdown
---
title: "{{.Title}}"
sources:
  tasks:
    type: sqlite
    db: "./tasks.db"
    table: tasks
    readonly: false
---

# {{.Title}}

A simple task manager with SQLite persistence.

## Tasks

```lvt
<main lvt-source="tasks">
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>Done</th>
                <th>Task</th>
                <th>Priority</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>
                    <input type="checkbox" {{if .Done}}checked{{end}}
                           lvt-on:click="Toggle" data-id="{{.Id}}">
                </td>
                <td {{if .Done}}style="text-decoration: line-through; opacity: 0.6"{{end}}>
                    {{.Text}}
                </td>
                <td>{{.Priority}}</td>
                <td>
                    <button name="Delete" data-id="{{.Id}}"
                            style="color: red; border: 1px solid red; background: transparent; border-radius: 4px; cursor: pointer; padding: 2px 8px;">
                        Delete
                    </button>
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
    <p><small>Total: {{len .Data}} tasks</small></p>
    {{end}}

    <hr style="margin: 16px 0;">

    <h3>Add New Task</h3>
    <form name="Add" style="display: flex; gap: 8px; flex-wrap: wrap; align-items: center;">
        <input type="text" name="text" placeholder="Task description..." required
               style="flex: 1; min-width: 200px; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
        <select name="priority" style="padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
            <option value="low">Low</option>
            <option value="medium" selected>Medium</option>
            <option value="high">High</option>
        </select>
        <button type="submit"
                style="padding: 8px 16px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;">
            Add Task
        </button>
    </form>
</main>
```

## How It Works

- **Add**: Submit the form to create a new task
- **Toggle**: Click checkbox to mark done/undone
- **Delete**: Remove a task permanently
- Data persists in `tasks.db` SQLite database
```

**Step 3: Create README.md**

Create file `cmd/tinkerdown/commands/templates/todo/README.md`:

```markdown
# {{.Title}}

A task manager built with [Tinkerdown](https://github.com/livetemplate/tinkerdown) and SQLite.

## Running

```bash
cd {{.ProjectName}}
tinkerdown serve
```

Open http://localhost:8080 in your browser.

## Data Storage

Tasks are stored in `tasks.db` (SQLite). The database is created automatically on first run.
```

**Step 4: Commit**

```bash
git add cmd/tinkerdown/commands/templates/todo/
git commit -m "feat: add todo template (SQLite CRUD)"
```

---

## Task 4: Create dashboard template (REST + exec)

**Files:**
- Create: `cmd/tinkerdown/commands/templates/dashboard/index.md`
- Create: `cmd/tinkerdown/commands/templates/dashboard/system-info.sh`
- Create: `cmd/tinkerdown/commands/templates/dashboard/README.md`

**Step 1: Create dashboard directory**

```bash
mkdir -p cmd/tinkerdown/commands/templates/dashboard
```

**Step 2: Create system-info.sh**

```bash
cat > cmd/tinkerdown/commands/templates/dashboard/system-info.sh << 'EOF'
#!/bin/bash
# System information as JSON
# Works on macOS and Linux

if command -v df &> /dev/null; then
    df -h 2>/dev/null | awk 'NR>1 {print "{\"filesystem\":\""$1"\",\"size\":\""$2"\",\"used\":\""$3"\",\"available\":\""$4"\",\"use_percent\":\""$5"\",\"mount\":\""$6"\"}"}' | jq -s '.'
else
    echo '[]'
fi
EOF
chmod +x cmd/tinkerdown/commands/templates/dashboard/system-info.sh
```

**Step 3: Create index.md**

Create file `cmd/tinkerdown/commands/templates/dashboard/index.md`:

```markdown
---
title: "{{.Title}}"
sources:
  users:
    type: rest
    url: https://jsonplaceholder.typicode.com/users
  system:
    type: exec
    cmd: ./system-info.sh
---

# {{.Title}}

A multi-source dashboard combining REST API data and local system information.

## API Users

```lvt
<main lvt-source="users">
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Email</th>
                <th>Company</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>{{.id}}</td>
                <td>{{.name}}</td>
                <td>{{.email}}</td>
                <td>{{if .company}}{{.company.name}}{{end}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}
    <button name="Refresh">Refresh Users</button>
</main>
```

## System Disk Usage

```lvt
<main lvt-source="system">
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>Filesystem</th>
                <th>Size</th>
                <th>Used</th>
                <th>Available</th>
                <th>Use %</th>
                <th>Mount</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>{{.filesystem}}</td>
                <td>{{.size}}</td>
                <td>{{.used}}</td>
                <td>{{.available}}</td>
                <td>{{.use_percent}}</td>
                <td>{{.mount}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}
    <button name="Refresh">Refresh System Info</button>
</main>
```

## About This Dashboard

This demonstrates combining multiple data sources:
- **REST API**: Users from JSONPlaceholder
- **Exec**: Local disk usage via `df -h`
```

**Step 4: Create README.md**

Create file `cmd/tinkerdown/commands/templates/dashboard/README.md`:

```markdown
# {{.Title}}

A multi-source dashboard built with [Tinkerdown](https://github.com/livetemplate/tinkerdown).

## Running

```bash
cd {{.ProjectName}}
tinkerdown serve
```

Open http://localhost:8080 in your browser.

## Data Sources

- **users**: REST API (JSONPlaceholder)
- **system**: Local shell command (df -h)
```

**Step 5: Commit**

```bash
git add cmd/tinkerdown/commands/templates/dashboard/
git commit -m "feat: add dashboard template (REST + exec)"
```

---

## Task 5: Create form template (Contact → SQLite)

**Files:**
- Create: `cmd/tinkerdown/commands/templates/form/index.md`
- Create: `cmd/tinkerdown/commands/templates/form/README.md`

**Step 1: Create form directory**

```bash
mkdir -p cmd/tinkerdown/commands/templates/form
```

**Step 2: Create index.md**

Create file `cmd/tinkerdown/commands/templates/form/index.md`:

```markdown
---
title: "{{.Title}}"
sources:
  submissions:
    type: sqlite
    db: "./submissions.db"
    table: submissions
    readonly: false
---

# {{.Title}}

A contact form with SQLite persistence.

## Submit a Message

```lvt
<main lvt-source="submissions">
    <form name="Add" style="max-width: 500px; display: flex; flex-direction: column; gap: 12px;">
        <div>
            <label for="name" style="display: block; margin-bottom: 4px; font-weight: bold;">Name</label>
            <input type="text" id="name" name="name" required
                   style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
        </div>
        <div>
            <label for="email" style="display: block; margin-bottom: 4px; font-weight: bold;">Email</label>
            <input type="email" id="email" name="email" required
                   style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
        </div>
        <div>
            <label for="message" style="display: block; margin-bottom: 4px; font-weight: bold;">Message</label>
            <textarea id="message" name="message" rows="4" required
                      style="width: 100%; padding: 8px; border: 1px solid #ccc; border-radius: 4px;"></textarea>
        </div>
        <button type="submit"
                style="padding: 10px 20px; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer; align-self: flex-start;">
            Submit
        </button>
    </form>

    <hr style="margin: 24px 0;">

    <h2>Submissions</h2>
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else if not .Data}}
    <p>No submissions yet.</p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>Name</th>
                <th>Email</th>
                <th>Message</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>{{.Name}}</td>
                <td>{{.Email}}</td>
                <td>{{.Message}}</td>
                <td>
                    <button name="Delete" data-id="{{.Id}}"
                            style="color: red; border: 1px solid red; background: transparent; border-radius: 4px; cursor: pointer; padding: 2px 8px;">
                        Delete
                    </button>
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}
</main>
```
```

**Step 3: Create README.md**

Create file `cmd/tinkerdown/commands/templates/form/README.md`:

```markdown
# {{.Title}}

A contact form built with [Tinkerdown](https://github.com/livetemplate/tinkerdown) and SQLite.

## Running

```bash
cd {{.ProjectName}}
tinkerdown serve
```

Open http://localhost:8080 in your browser.

## Data Storage

Submissions are stored in `submissions.db` (SQLite). The database is created automatically.
```

**Step 4: Commit**

```bash
git add cmd/tinkerdown/commands/templates/form/
git commit -m "feat: add form template (contact form)"
```

---

## Task 6: Create api-explorer template (GitHub search)

**Files:**
- Create: `cmd/tinkerdown/commands/templates/api-explorer/index.md`
- Create: `cmd/tinkerdown/commands/templates/api-explorer/README.md`

**Step 1: Create api-explorer directory**

```bash
mkdir -p cmd/tinkerdown/commands/templates/api-explorer
```

**Step 2: Create index.md**

Create file `cmd/tinkerdown/commands/templates/api-explorer/index.md`:

```markdown
---
title: "{{.Title}}"
persist: localstorage
sources:
  repos:
    type: rest
    url: https://api.github.com/search/repositories?q=${query}&per_page=10
---

# {{.Title}}

Search GitHub repositories using the GitHub API.

## Search

```lvt
<main lvt-source="repos">
    <form name="SetQuery" style="margin-bottom: 16px; display: flex; gap: 8px;">
        <input type="text" name="query" placeholder="Search repositories..." value="{{.Query}}"
               style="flex: 1; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
        <button type="submit"
                style="padding: 8px 16px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;">
            Search
        </button>
    </form>

    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else if .Data}}
    <p><small>Found {{len .Data.items}} repositories</small></p>
    <table>
        <thead>
            <tr>
                <th>Repository</th>
                <th>Stars</th>
                <th>Language</th>
                <th>Description</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data.items}}
            <tr>
                <td><a href="{{.html_url}}" target="_blank">{{.full_name}}</a></td>
                <td>{{.stargazers_count}}</td>
                <td>{{.language}}</td>
                <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{.description}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{else}}
    <p>Enter a search term to find repositories.</p>
    {{end}}
</main>
```

## How It Works

1. Enter a search term (e.g., "golang cli")
2. Click Search to query GitHub's API
3. Results show repository name, stars, language, and description

**Note:** GitHub API has rate limits. For production use, add authentication.
```

**Step 3: Create README.md**

Create file `cmd/tinkerdown/commands/templates/api-explorer/README.md`:

```markdown
# {{.Title}}

A GitHub repository search tool built with [Tinkerdown](https://github.com/livetemplate/tinkerdown).

## Running

```bash
cd {{.ProjectName}}
tinkerdown serve
```

Open http://localhost:8080 in your browser.

## API

Uses the GitHub Search Repositories API. Rate limited to 10 requests/minute for unauthenticated requests.
```

**Step 4: Commit**

```bash
git add cmd/tinkerdown/commands/templates/api-explorer/
git commit -m "feat: add api-explorer template (GitHub search)"
```

---

## Task 7: Create wasm-source template

**Files:**
- Create: `cmd/tinkerdown/commands/templates/wasm-source/source.go`
- Create: `cmd/tinkerdown/commands/templates/wasm-source/Makefile`
- Create: `cmd/tinkerdown/commands/templates/wasm-source/README.md`
- Create: `cmd/tinkerdown/commands/templates/wasm-source/test-app/index.md`
- Create: `cmd/tinkerdown/commands/templates/wasm-source/test-app/tinkerdown.yaml`

**Step 1: Create wasm-source directory structure**

```bash
mkdir -p cmd/tinkerdown/commands/templates/wasm-source/test-app
```

**Step 2: Create source.go**

Create file `cmd/tinkerdown/commands/templates/wasm-source/source.go`:

```go
//go:build tinygo.wasm

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"unsafe"
)

// Response from httpbin.org/get
type HTTPBinResponse struct {
	Args    map[string]string `json:"args"`
	Headers map[string]string `json:"headers"`
	Origin  string            `json:"origin"`
	URL     string            `json:"url"`
}

// Data item returned to Tinkerdown
type DataItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

//export fetch
func fetch() uint64 {
	return fetchWithArgs(0, 0)
}

//export fetchWithArgs
func fetchWithArgs(argsPtr, argsLen uint32) uint64 {
	// Fetch from httpbin.org
	resp, err := http.Get("https://httpbin.org/get")
	if err != nil {
		return encodeError(err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return encodeError(err.Error())
	}

	var httpbinResp HTTPBinResponse
	if err := json.Unmarshal(body, &httpbinResp); err != nil {
		return encodeError(err.Error())
	}

	// Convert to array of key-value pairs
	items := []DataItem{
		{Key: "origin", Value: httpbinResp.Origin},
		{Key: "url", Value: httpbinResp.URL},
	}
	for k, v := range httpbinResp.Headers {
		items = append(items, DataItem{Key: k, Value: v})
	}

	result, _ := json.Marshal(items)
	return encodeResult(result)
}

func encodeResult(data []byte) uint64 {
	ptr := uint32(uintptr(unsafe.Pointer(&data[0])))
	length := uint32(len(data))
	return (uint64(ptr) << 32) | uint64(length)
}

func encodeError(msg string) uint64 {
	errJSON := []byte(`{"error":"` + msg + `"}`)
	return encodeResult(errJSON)
}

func main() {}
```

**Step 3: Create Makefile**

Create file `cmd/tinkerdown/commands/templates/wasm-source/Makefile`:

```makefile
.PHONY: build test clean

build:
	tinygo build -o source.wasm -target=wasi -no-debug source.go

test: build
	cd test-app && tinkerdown serve

clean:
	rm -f source.wasm
```

**Step 4: Create README.md**

Create file `cmd/tinkerdown/commands/templates/wasm-source/README.md`:

```markdown
# {{.Title}} - WASM Source

A custom Tinkerdown data source built with TinyGo and WebAssembly.

## Prerequisites

- [TinyGo](https://tinygo.org/getting-started/install/) installed
- Tinkerdown CLI

## Building

```bash
make build
```

This compiles `source.go` to `source.wasm`.

## Testing

```bash
make test
```

Opens the test app at http://localhost:8080.

## WASM Interface

Your source must export these functions:

### `fetch() uint64`

Fetch data without arguments. Returns pointer+length packed as uint64.

### `fetchWithArgs(argsPtr, argsLen uint32) uint64`

Fetch data with JSON arguments. Returns pointer+length packed as uint64.

### Return Format

Return a JSON array of objects:

```json
[
  {"key": "value", "other": "data"},
  {"key": "value2", "other": "data2"}
]
```

Or return an error:

```json
{"error": "Something went wrong"}
```

## Customizing

1. Edit `source.go` to fetch from your API
2. Run `make build`
3. Test with `make test`
4. Copy `source.wasm` to your Tinkerdown app
```

**Step 5: Create test-app/tinkerdown.yaml**

Create file `cmd/tinkerdown/commands/templates/wasm-source/test-app/tinkerdown.yaml`:

```yaml
sources:
  data:
    type: wasm
    module: ../source.wasm
```

**Step 6: Create test-app/index.md**

Create file `cmd/tinkerdown/commands/templates/wasm-source/test-app/index.md`:

```markdown
---
title: "WASM Source Test"
---

# WASM Source Test

Testing the custom WASM data source.

## Data from WASM

```lvt
<main lvt-source="data">
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>Key</th>
                <th>Value</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>{{.key}}</td>
                <td>{{.value}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}
    <button name="Refresh">Refresh</button>
</main>
```
```

**Step 7: Commit**

```bash
git add cmd/tinkerdown/commands/templates/wasm-source/
git commit -m "feat: add wasm-source template (custom WASM scaffold)"
```

---

## Task 8: Update new.go with --template flag

**Files:**
- Modify: `cmd/tinkerdown/commands/new.go`

**Step 1: Read current implementation**

Review the existing `new.go` to understand the structure.

**Step 2: Update new.go**

Replace `cmd/tinkerdown/commands/new.go` with:

```go
package commands

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*
var templatesFS embed.FS

var validTemplates = []string{
	"basic",
	"tutorial",
	"todo",
	"dashboard",
	"form",
	"api-explorer",
	"wasm-source",
}

// NewCommand implements the new command.
func NewCommand(args []string, templateName string) error {
	if len(args) < 1 {
		return fmt.Errorf("project name required\n\nUsage: tinkerdown new <project-name> [--template=<name>]\n\nAvailable templates: %s", strings.Join(validTemplates, ", "))
	}

	projectName := args[0]

	// Validate project name
	if projectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if strings.Contains(projectName, " ") {
		return fmt.Errorf("project name cannot contain spaces")
	}

	// Default template
	if templateName == "" {
		templateName = "basic"
	}

	// Validate template name
	if !isValidTemplate(templateName) {
		return fmt.Errorf("unknown template '%s'\n\nAvailable templates: %s", templateName, strings.Join(validTemplates, ", "))
	}

	// Check if directory already exists
	if _, err := os.Stat(projectName); !os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' already exists", projectName)
	}

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Template data
	data := map[string]string{
		"Title":       toTitle(projectName),
		"ProjectName": projectName,
	}

	// Process template files
	templateDir := "templates/" + templateName
	err := fs.WalkDir(templatesFS, templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from template root
		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Target path
		targetPath := filepath.Join(projectName, relPath)

		if d.IsDir() {
			// Create directory
			return os.MkdirAll(targetPath, 0755)
		}

		// Read file content
		content, err := templatesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Determine if file should be processed as template
		ext := filepath.Ext(path)
		if ext == ".md" || ext == ".yaml" || ext == ".sh" {
			// Process as Go template
			tmpl, err := template.New(filepath.Base(path)).Parse(string(content))
			if err != nil {
				return fmt.Errorf("failed to parse template %s: %w", path, err)
			}

			f, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create %s: %w", targetPath, err)
			}
			defer f.Close()

			if err := tmpl.Execute(f, data); err != nil {
				return fmt.Errorf("failed to execute template %s: %w", path, err)
			}
		} else {
			// Copy file as-is
			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", targetPath, err)
			}
		}

		// Make .sh files executable
		if ext == ".sh" {
			if err := os.Chmod(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to chmod %s: %w", targetPath, err)
			}
		}

		return nil
	})

	if err != nil {
		// Clean up on error
		os.RemoveAll(projectName)
		return fmt.Errorf("failed to create project: %w", err)
	}

	// Success message
	fmt.Printf("✨ Created new app: %s (template: %s)\n\n", projectName, templateName)
	printProjectStructure(projectName, templateName)
	fmt.Printf("\n🚀 Next steps:\n")
	fmt.Printf("   cd %s\n", projectName)
	fmt.Printf("   tinkerdown serve\n\n")
	fmt.Printf("📚 Your app will be available at http://localhost:8080\n")

	return nil
}

func isValidTemplate(name string) bool {
	for _, t := range validTemplates {
		if t == name {
			return true
		}
	}
	return false
}

func printProjectStructure(projectName, templateName string) {
	fmt.Printf("📁 Project structure:\n")
	fmt.Printf("   %s/\n", projectName)

	// Walk the created directory and print structure
	filepath.WalkDir(projectName, func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == projectName {
			return nil
		}
		relPath, _ := filepath.Rel(projectName, path)
		depth := strings.Count(relPath, string(os.PathSeparator))
		indent := strings.Repeat("   ", depth)
		prefix := "├── "
		fmt.Printf("   %s%s%s\n", indent, prefix, d.Name())
		return nil
	})
}

// toTitle converts a project name to a title case string
// Example: "my-app" -> "My App"
func toTitle(name string) string {
	// Replace hyphens and underscores with spaces
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")

	// Title case each word
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}
```

**Step 3: Update main.go to pass template flag**

Check `cmd/tinkerdown/main.go` for how arguments are passed to NewCommand.

**Step 4: Commit**

```bash
git add cmd/tinkerdown/commands/new.go
git commit -m "feat: add --template flag to new command"
```

---

## Task 9: Update main.go CLI parsing

**Files:**
- Modify: `cmd/tinkerdown/main.go`

**Step 1: Read current main.go**

Review how the CLI currently parses arguments.

**Step 2: Add template flag handling**

Update the `new` command section to extract `--template` or `-t` flag.

**Step 3: Commit**

```bash
git add cmd/tinkerdown/main.go
git commit -m "feat: wire --template flag in CLI"
```

---

## Task 10: Write E2E tests for templates

**Files:**
- Create: `new_command_e2e_test.go`

**Step 1: Create test file**

Create comprehensive E2E tests that verify each template generates correctly and the generated app can run.

**Step 2: Run tests**

```bash
GOWORK=off go test -v -run TestNewCommand ./...
```

**Step 3: Commit**

```bash
git add new_command_e2e_test.go
git commit -m "test: add E2E tests for new command templates"
```

---

## Task 11: Rebuild binary and run full test suite

**Step 1: Rebuild tinkerdown**

```bash
GOWORK=off go build -o tinkerdown ./cmd/tinkerdown
```

**Step 2: Run all tests**

```bash
GOWORK=off go test ./... -count=1
```

**Step 3: Manual verification**

```bash
./tinkerdown new test-basic
./tinkerdown new test-todo --template=todo
./tinkerdown new test-dashboard -t dashboard
```

---

## Task 12: Update ROADMAP and create PR

**Step 1: Update ROADMAP.md**

Mark 3.1 as in progress, update Current Sprint section.

**Step 2: Create PR**

```bash
git push -u origin feature/cli-scaffolding
gh pr create --title "feat: implement enhanced CLI scaffolding (Phase 3.1)" --body "..."
```
