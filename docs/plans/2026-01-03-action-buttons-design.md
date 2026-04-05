# Action Buttons Design (Issue #44)

## Overview

Add support for custom actions declared in frontmatter. Actions can execute SQL statements, HTTP requests, or shell commands when triggered via `name` attribute on buttons.

## Frontmatter Schema

Actions are declared alongside sources in the page's YAML frontmatter:

```yaml
---
title: Task Manager
sources:
  tasks:
    type: sqlite
    db: tasks.db
    table: tasks

actions:
  clear-done:
    kind: sql
    source: tasks
    statement: "DELETE FROM tasks WHERE done = 1"
    confirm: "Delete all completed tasks?"

  archive-old:
    kind: sql
    source: tasks
    statement: "UPDATE tasks SET archived = 1 WHERE created_at < :cutoff"
    params:
      cutoff:
        type: date
        required: true

  notify-slack:
    kind: http
    url: "https://hooks.slack.com/services/xxx"
    method: POST
    body: '{"text": "Task update: {{.message}}"}'

  run-backup:
    kind: exec
    cmd: "./scripts/backup.sh"
---
```

### Action Struct

```go
type Action struct {
    Kind      string              `yaml:"kind"`      // sql, http, exec
    Source    string              `yaml:"source"`    // for sql kind
    Statement string              `yaml:"statement"` // SQL query
    URL       string              `yaml:"url"`       // for http kind
    Method    string              `yaml:"method"`    // GET, POST, etc.
    Body      string              `yaml:"body"`      // HTTP body template
    Cmd       string              `yaml:"cmd"`       // for exec kind
    Params    map[string]ParamDef `yaml:"params"`
    Confirm   string              `yaml:"confirm"`
}

type ParamDef struct {
    Type     string `yaml:"type"`     // string, number, date, bool
    Required bool   `yaml:"required"`
    Default  string `yaml:"default"`
}
```

## Server-side Integration

### Data Flow

```
Frontmatter YAML
    ↓ (parsed by parser.go)
PageConfig.Actions map[string]*Action
    ↓ (passed to NewGenericState)
GenericState.actions map[string]*Action
    ↓ (checked in HandleAction)
Execute action if declared, else "unknown action" error
```

### GenericState Changes

```go
type GenericState struct {
    // ... existing fields ...
    actions map[string]*config.Action
}

func NewGenericState(cfg *config.PageConfig, ...) *GenericState {
    return &GenericState{
        // ... existing ...
        actions: cfg.Actions,
    }
}
```

### HandleAction Changes

```go
func (s *GenericState) HandleAction(action string, data map[string]interface{}) error {
    actionLower := strings.ToLower(action)

    // Check built-in actions first (existing switch)
    switch actionLower {
    case "refresh":
        return s.refresh()
    case "run":
        return s.runExec(data)
    case "add", "toggle", "delete", "update":
        return s.handleWriteAction(action, data)
    }

    // Check datatable actions (existing code)
    if strings.HasPrefix(actionLower, "sort") || ... {
        return s.handleDatatableAction(action, data)
    }

    // NEW: Check custom declared actions
    if customAction, ok := s.actions[action]; ok {
        return s.executeCustomAction(customAction, data)
    }

    return fmt.Errorf("action %q not declared in frontmatter", action)
}
```

## Action Execution

### Dispatcher

```go
func (s *GenericState) executeCustomAction(action *config.Action, data map[string]interface{}) error {
    // Validate required params
    if err := validateParams(action, data); err != nil {
        return err
    }

    switch action.Kind {
    case "sql":
        return s.executeSQLAction(action, data)
    case "http":
        return s.executeHTTPAction(action, data)
    case "exec":
        return s.executeExecAction(action, data)
    default:
        return fmt.Errorf("unknown action kind: %s", action.Kind)
    }
}
```

### SQL Actions

```go
func (s *GenericState) executeSQLAction(action *config.Action, data map[string]interface{}) error {
    src, ok := s.registry.Get(action.Source)
    if !ok {
        return fmt.Errorf("source %q not found", action.Source)
    }

    executor, ok := src.(SQLExecutor)
    if !ok {
        return fmt.Errorf("source %q does not support SQL execution", action.Source)
    }

    // Parameterized query (prevents SQL injection)
    query, args := substituteParams(action.Statement, data)

    _, err := executor.Exec(ctx, query, args...)
    if err != nil {
        return err
    }

    return s.refresh()  // Reload data after mutation
}
```

### HTTP Actions

```go
func (s *GenericState) executeHTTPAction(action *config.Action, data map[string]interface{}) error {
    url := expandTemplate(action.URL, data)
    body := expandTemplate(action.Body, data)

    method := action.Method
    if method == "" {
        method = "POST"
    }

    req, _ := http.NewRequest(method, url, strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }
    return nil
}
```

### Exec Actions

```go
// sanitizeExecCommand validates commands for shell safety
func sanitizeExecCommand(cmd string) error {
    cmd = strings.TrimSpace(cmd)
    if cmd == "" {
        return fmt.Errorf("exec command is empty after templating")
    }
    // Reject shell metacharacters to prevent command injection
    if strings.ContainsAny(cmd, "&;|$><`\\\n\r") {
        return fmt.Errorf("exec command contains disallowed shell characters")
    }
    return nil
}

func (s *GenericState) executeExecAction(action *config.Action, data map[string]interface{}) error {
    if !config.IsExecAllowed() {
        return fmt.Errorf("exec actions disabled (use --allow-exec)")
    }

    cmd := expandTemplate(action.Cmd, data)

    // Validate command for shell safety
    if err := sanitizeExecCommand(cmd); err != nil {
        return err
    }

    return runCommand(ctx, cmd, s.siteDir)
}
```

**Security Note**: The `sanitizeExecCommand` function rejects shell metacharacters (`&;|$><\`\\`) to prevent command injection when user-supplied parameters are expanded into commands via templates.

## Error Handling & Security

### Parameter Validation

```go
func validateParams(action *config.Action, data map[string]interface{}) error {
    for name, def := range action.Params {
        val, exists := data[name]
        if def.Required && (!exists || val == "") {
            return &ValidationError{
                Field:  name,
                Reason: "required parameter missing",
            }
        }
    }
    return nil
}
```

### Security Measures

| Kind | Security Measure |
|------|-----------------|
| `sql` | Parameterized queries only (`:param` → `?` substitution) |
| `http` | Standard HTTP client, URLs from config only |
| `exec` | Requires `--allow-exec` flag (same as exec sources) |

### SQL Parameter Substitution

```go
// Converts :name placeholders to positional args
// Input:  "DELETE FROM tasks WHERE id = :id", {"id": "123"}
// Output: "DELETE FROM tasks WHERE id = ?", ["123"]
func substituteParams(stmt string, data map[string]interface{}) (string, []interface{}) {
    var args []interface{}
    result := paramRegex.ReplaceAllStringFunc(stmt, func(match string) string {
        name := match[1:] // strip leading ":"
        args = append(args, data[name])
        return "?"
    })
    return result, args
}
```

## Files to Modify

1. **`internal/config/config.go`** - Add `Action` and `ParamDef` structs, add `Actions` field to page config
2. **`internal/runtime/state.go`** - Add `actions` field to GenericState, pass from config
3. **`internal/runtime/actions.go`** - Add `executeCustomAction`, `executeSQLAction`, `executeHTTPAction`, `executeExecAction`
4. **`internal/source/sqlite.go`** - Add `Exec` method to SQLiteSource (implements `SQLExecutor` interface)
5. **`parser.go`** - Parse `actions:` from frontmatter

## Testing

### Unit Tests

- `TestExecuteCustomAction_SQL` - Execute SQL action, verify data changes
- `TestExecuteCustomAction_HTTP` - Execute HTTP action with mock server
- `TestExecuteCustomAction_Exec` - Execute with/without `--allow-exec`
- `TestExecuteCustomAction_UndeclaredAction` - Error for unknown action
- `TestValidateParams` - Required param validation
- `TestSubstituteParams` - SQL parameter substitution

### E2E Test

- Create test page with SQLite source + `clear-done` action
- Render with chromedp
- Click `name="clear-done"` button
- Verify data refreshed and completed tasks removed

## Definition of Done

- [ ] `actions:` parsed from frontmatter
- [ ] Actions accessible in GenericState
- [ ] HandleAction routes to custom actions
- [ ] SQL actions execute parameterized queries
- [ ] HTTP actions send requests
- [ ] Exec actions respect `--allow-exec`
- [ ] Required params validated
- [ ] Undeclared actions return clear error
- [ ] Unit tests for all action kinds
- [ ] E2E test for button click → SQL action
