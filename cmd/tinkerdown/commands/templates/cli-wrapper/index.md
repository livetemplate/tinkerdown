---
title: "<<.Title>>"
sources:
  command:
    type: exec
    cmd: echo "Hello, World! Count:" --count 3
    format: lines
    manual: true
---

# <<.Title>>

Wrap CLI tools with an interactive web interface.

## Command Runner

Click **Run** in the toolbar below to execute the command.

```lvt
<div lvt-source="command">
    {{if eq .Status "ready"}}
    <p style="color: #9ca3af; font-style: italic;">Click Run to execute the command.</p>
    {{else if eq .Status "running"}}
    <p style="color: #f59e0b;">Running...</p>
    {{else if eq .Status "error"}}
    <div style="background: #f8d7da; padding: 1rem; border-radius: 4px;">
        <strong>Error:</strong>
        <pre style="margin: 0.5rem 0 0 0; white-space: pre-wrap;">{{.Error}}</pre>
    </div>
    {{else}}
    <div style="background: #d4edda; padding: 1rem; border-radius: 4px;">
        <strong>Output:</strong>
        <pre style="margin: 0.5rem 0 0 0; white-space: pre-wrap;">{{range .Data}}{{.line}}
{{end}}</pre>
    </div>
    {{end}}
</div>
```

---

## Configuration

Edit the `cmd` in the frontmatter to wrap your own CLI tool:

```yaml
sources:
  command:
    type: exec
    cmd: your-cli-tool --arg1 value1 --arg2 value2
    format: lines
    manual: true
```

The `manual: true` option means the command only runs when you click "Run".

Use `format: lines` for commands that output plain text. Use `format: json` (default) for commands that output JSON.

## Argument Parsing

Arguments are automatically parsed from the command:
- `--flag value` becomes a text input
- `--flag 123` becomes a number input
- `--flag true/false` becomes a checkbox

## Environment Variables

Use `${VAR_NAME}` for secrets:

```yaml
sources:
  command:
    type: exec
    cmd: my-cli --token ${API_TOKEN}
```
