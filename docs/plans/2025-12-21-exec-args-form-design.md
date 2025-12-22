# Exec Data Source: Auto-Generated Argument Forms

## Progress Tracker
- [ ] Step 1: Create argument parser with type inference (Go)
- [ ] Step 2: Add script introspection (`--help` parsing)
- [ ] Step 3: Extend State struct with Args
- [ ] Step 4: Generate UpdateArgs handler
- [ ] Step 5: Modify command building to use dynamic args
- [ ] Step 6: Extend ExecMeta with arg definitions
- [ ] Step 7: Add form generation to exec toolbar (TypeScript)
- [ ] Step 8: Wire form submission to Run action
- [ ] Step 9: Add E2E test
- [ ] Step 10: Update example

---

## Summary

Add auto-generated input forms for exec data source arguments. Users write:
```yaml
sources:
  hello:
    type: exec
    cmd: ./hello.sh input:string:file.txt --name:string adnaan --count 5 --verbose true
    manual: true
```

The system:
1. Parses `input:string:file.txt` → positional arg, label "input", type string, default "file.txt"
2. Parses `--name:string adnaan` → named arg, label "name", type string (explicit), default "adnaan"
3. Parses `--count 5` → named arg, label "count", type number (inferred), default 5
4. Parses `--verbose true` → named arg, label "verbose", type bool (inferred), default true
5. **Introspects script** via `./hello.sh --help` to get flag descriptions for tooltips
6. **Strips all hints** before executing: `./hello.sh file.txt --name adnaan --count 5 --verbose true`
7. Injects form fields into the exec toolbar, above the Run button

**Type hints are optional** - if provided, they override inference. If omitted, types are inferred from values.

---

## Files to Modify

| File | Purpose |
|------|---------|
| `internal/compiler/execargs.go` | New file: argument parser with type inference |
| `internal/compiler/lvtsource.go` | Generate State with Args, template with form, UpdateArg handler |
| `internal/server/websocket.go` | Minor: pass args in ExecMeta for status display |
| `client/src/blocks/exec-toolbar.css` | CSS for arg form layout (minimal changes) |
| `examples/lvt-source-exec-test/` | Update example to demonstrate args |
| `lvtsource_e2e_test.go` | Add E2E test for arg forms |

**Note:** Form generation is now **server-side** in Go templates, not TypeScript. This preserves full LiveTemplate reactivity.

---

## Implementation Steps

### Step 1: Create Argument Parser with Type Inference (Go)

Create `internal/compiler/execargs.go`:

```go
type ExecArg struct {
    Name        string `json:"name"`        // "name" or "arg1" for positional
    Type        string `json:"type"`        // "string", "number", "bool"
    Default     string `json:"default"`     // Default value as string
    Position    int    `json:"position"`    // -1 for named flags, 0+ for positional
    Description string `json:"description"` // From --help introspection
}

func ParseExecCommand(cmd string) (executable string, args []ExecArg, err error)
func InferType(value string) string  // Returns "number", "bool", or "string"
```

**Parsing rules:**
- First element → executable (not editable)
- `--flag:type value` → named arg with explicit type hint, label = flag name
- `--flag value` → named arg, type inferred from value, label = flag name
- `label:type:value` → positional arg with explicit label and type
- `label:value` → positional arg with label, type inferred
- `value` → positional arg, label = "arg1"/"arg2", type inferred

**Type inference (when no hint provided):**
- `true`/`false` → bool
- Numeric (matches `^-?\d+(\.\d+)?$`) → number
- Otherwise → string

**Hint stripping:** Before execution, all `:type` hints are removed from the command.

**Type inference examples:**
| Value | Inferred Type |
|-------|---------------|
| `5` | number |
| `3.14` | number |
| `-10` | number |
| `true` | bool |
| `false` | bool |
| `hello` | string |
| `file.txt` | string |

### Step 2: Add Script Introspection (`--help` parsing)

**At compile time**, execute `./script.sh --help` and parse output for flag descriptions.

```go
func IntrospectScript(executable, workDir string) (map[string]string, error) {
    // Returns map of flag name → description
    // e.g., {"name": "The name to greet", "count": "Number of times to repeat"}
}
```

**When:** Called during `compileServerBlocks()` in websocket.go, before generating State code. Results are embedded in the generated code as static data.

**Parsing strategy for `--help` output:**
1. Look for lines matching common patterns:
   - `--flag  Description text` (GNU style)
   - `-f, --flag  Description text` (short + long)
   - `--flag=VALUE  Description text` (with placeholder)
2. Extract flag name and description
3. Handle multi-line descriptions (indented continuation lines)

**Error handling:**
- If --help returns non-zero exit code → swallow error, continue without descriptions
- If --help times out (5s limit) → swallow error, continue without descriptions
- If output doesn't match any patterns → continue without descriptions
- Form always works regardless of --help success

### Step 3: Extend State Struct

In `generateExecSourceCode()`, add to State:
```go
type State struct {
    // ... existing fields ...
    Args      []ExecArg         `json:"args"`      // Arg definitions
    ArgValues map[string]string `json:"argValues"` // Current values
}
```

Initialize `Args` and `ArgValues` from parsed command in `NewState()`.

### Step 4: Generate UpdateArgs Handler

Add method to update arg values:
```go
func (s *State) UpdateArgs(ctx *livetemplate.Context) error {
    // ctx.Data contains the new arg values map
    for name, value := range ctx.Data {
        s.ArgValues[name] = value
    }
    return nil
}
```

Modify `Run()` to use `ArgValues` when building the command.

### Step 5: Build Command Dynamically

In `fetchData()`, instead of using static command:
```go
// Build command from executable + current arg values
cmdParts := []string{executable}
for _, arg := range s.Args {
    val := s.ArgValues[arg.Name]
    if arg.Position >= 0 {
        cmdParts = append(cmdParts, val)
    } else {
        cmdParts = append(cmdParts, "--"+arg.Name, val)
    }
}
```

### Step 6: Extend ExecMeta

In `websocket.go`, add args to ExecMeta:
```go
type ExecMeta struct {
    // ... existing fields ...
    Args []compiler.ExecArg `json:"args,omitempty"`
}
```

Populate from State when sending tree updates.

### Step 7: Form Generation (Go HTML Template)

Generate the form **server-side** using `<form lvt-submit="Run">`, preserving full LiveTemplate capabilities.

In `generateExecSourceCode()`, include a template that renders the arg form:

```go
// Generated template includes the form
const execTemplate = `
<div class="exec-block" data-exec-source="true">
  <div class="exec-toolbar">
    <code class="exec-command">{{.Command}}</code>
    <span class="exec-status {{.Status}}">{{.StatusText}}</span>
    {{if .Duration}}<span class="exec-duration">{{.Duration}}ms</span>{{end}}
  </div>

  <form lvt-submit="Run" class="exec-args-form">
    {{range .Args}}
    <label class="exec-arg" {{if .Description}}title="{{.Description}}"{{end}}>
      <span>{{.Label}}</span>
      {{if eq .Type "bool"}}
      <input type="checkbox" name="{{.Name}}" {{if eq .Value "true"}}checked{{end}}>
      {{else if eq .Type "number"}}
      <input type="number" name="{{.Name}}" value="{{.Value}}">
      {{else}}
      <input type="text" name="{{.Name}}" value="{{.Value}}">
      {{end}}
    </label>
    {{end}}

    <button type="submit" {{if eq .Status "running"}}disabled{{end}}>
      {{if eq .Status "running"}}Running...{{else}}Run{{end}}
    </button>
  </form>

  {{if or .Output .Stderr}}
  <details class="exec-output">
    <summary>Output</summary>
    {{if .Output}}<pre class="stdout">{{.Output}}</pre>{{end}}
    {{if .Stderr}}<pre class="stderr">{{.Stderr}}</pre>{{end}}
  </details>
  {{end}}
</div>
`
```

**Benefits of `<form lvt-submit="Run">`:**
- Single request on submit (not on every keystroke)
- All arg values collected via FormData automatically
- Standard HTML form semantics
- No separate `UpdateArg` handler needed
- Less network traffic

### Step 8: Handle Form Submission in Run Action

The `Run` action receives all form data at once:

```go
func (s *State) Run(ctx *livetemplate.Context) error {
    // Update arg values from form submission
    for _, arg := range s.Args {
        if val := ctx.GetString(arg.Name); val != "" {
            s.ArgValues[arg.Name] = val
        } else if arg.Type == "bool" {
            // Unchecked checkbox won't be in form data
            s.ArgValues[arg.Name] = "false"
        }
    }

    // Build and execute command with current arg values
    s.Status = "running"
    if err := s.fetchData(); err != nil {
        s.Error = err.Error()
        s.Status = "error"
    } else {
        s.Status = "success"
    }
    return nil
}
```

**Flow:**
1. User fills in form inputs
2. User clicks Run (submit button)
3. `lvt-submit="Run"` collects all inputs via FormData
4. Server receives all arg values in single request
5. `Run` updates `ArgValues`, builds command, executes

### Step 9: Add E2E Test

Create test in `lvtsource_e2e_test.go`:
1. Load page with exec source that has args
2. Verify form inputs are rendered with correct types
3. Verify tooltips show descriptions from --help
4. Modify an input value
5. Click Run
6. Verify command executed with updated args

### Step 10: Update Example

Update `examples/lvt-source-exec-test/index.md`:
```yaml
sources:
  greeting:
    type: exec
    # Positional with label: input:string:greeting.txt
    # Named with type hint: --name:string World
    # Named with inference: --count 3 (infers number), --uppercase false (infers bool)
    cmd: ./greet.sh input:string:greeting.txt --name:string World --count 3 --uppercase false
    manual: true
```

**Executed as:** `./greet.sh greeting.txt --name World --count 3 --uppercase false` (hints stripped)

Create `greet.sh` with --help support:
```bash
#!/bin/bash

# Handle --help flag
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat << 'EOF'
Usage: greet.sh [options]

Options:
  --name NAME      The name to greet (default: World)
  --count N        Number of times to repeat (default: 1)
  --uppercase      Output in uppercase (default: false)
  --help           Show this help message
EOF
    exit 0
fi

# Parse arguments
name="World"
count=1
uppercase="false"

while [[ $# -gt 0 ]]; do
    case $1 in
        --name) name="$2"; shift 2 ;;
        --count) count="$2"; shift 2 ;;
        --uppercase) uppercase="$2"; shift 2 ;;
        *) shift ;;
    esac
done

# Generate output
for i in $(seq 1 $count); do
    if [ "$uppercase" = "true" ]; then
        echo "HELLO, $name!" | tr '[:lower:]' '[:upper:]'
    else
        echo "Hello, $name!"
    fi
done
```

---

## Design Decisions

1. **Server-side form generation** - Go HTML templates, full LiveTemplate reactivity preserved
2. **`<form lvt-submit="Run">`** - Standard form, single request on submit, FormData collection
3. **Type inference from values** - `5` → number, `true`/`false` → bool, else string
4. **Optional type hints** - Named: `--name:string`, Positional: `label:type:value`
5. **Positional arg labels** - Custom via `label:type:value` or auto as "arg1"/"arg2"
6. **Hint stripping** - All `:type` and `label:type:` prefixes removed before script execution
7. **Script introspection** - Run `--help` to get flag descriptions for tooltips (fails silently)
8. **All args are strings internally** - Script handles type conversion
9. **Graceful fallback** - If --help fails, swallow error, form still works (just no tooltips)
