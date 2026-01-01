---
title: "[[.Title]]"
sources:
  command:
    type: exec
    cmd: echo "Hello, World! Count:" --count 3
    manual: true
---

# [[.Title]]

Wrap CLI tools with an interactive web interface.

## Command Runner

```lvt
<main lvt-source="command">
    <div class="command-panel">
        <h3>Command Configuration</h3>

        <form lvt-submit="Run" class="args-form">
            {{range .Args}}
            <div class="form-group">
                {{if eq .Type "bool"}}
                <label class="checkbox-label">
                    <input type="checkbox" name="{{.Name}}" {{if eq .Value "true"}}checked{{end}}>
                    <span>{{.Label}}</span>
                    {{if .Description}}<small>{{.Description}}</small>{{end}}
                </label>
                {{else if eq .Type "number"}}
                <label>
                    {{.Label}}
                    {{if .Description}}<small>{{.Description}}</small>{{end}}
                    <input type="number" name="{{.Name}}" value="{{.Value}}">
                </label>
                {{else}}
                <label>
                    {{.Label}}
                    {{if .Description}}<small>{{.Description}}</small>{{end}}
                    <input type="text" name="{{.Name}}" value="{{.Value}}">
                </label>
                {{end}}
            </div>
            {{end}}

            <button type="submit" class="run-btn">Run Command</button>
        </form>

        <div class="command-preview">
            <strong>Command:</strong>
            <code>{{.Command}}</code>
        </div>
    </div>

    <div class="output-panel">
        <h3>Output</h3>

        <div class="status-bar">
            <span class="status {{.Status}}">{{.Status}}</span>
        </div>

        {{if .Error}}
        <div class="error-output">
            <strong>Error:</strong>
            <pre>{{.Error}}</pre>
        </div>
        {{end}}

        {{if .Data}}
        <div class="data-output">
            <strong>Result:</strong>
            <pre>{{range .Data}}{{.}}
{{end}}</pre>
        </div>
        {{else if .Stdout}}
        <div class="stdout-output">
            <strong>stdout:</strong>
            <pre>{{.Stdout}}</pre>
        </div>
        {{end}}

        {{if .Stderr}}
        <div class="stderr-output">
            <strong>stderr:</strong>
            <pre>{{.Stderr}}</pre>
        </div>
        {{end}}
    </div>
</main>

<style>
.command-panel {
    background: #f8f9fa;
    padding: 1.5rem;
    border-radius: 8px;
    margin-bottom: 1.5rem;
}

.command-panel h3 {
    margin-top: 0;
    margin-bottom: 1rem;
}

.args-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.form-group label {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
}

.form-group small {
    color: #666;
    font-size: 0.85em;
}

.form-group input[type="text"],
.form-group input[type="number"] {
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 1rem;
}

.checkbox-label {
    flex-direction: row !important;
    align-items: center;
    gap: 0.5rem !important;
}

.checkbox-label input {
    width: auto;
}

.run-btn {
    padding: 0.75rem 1.5rem;
    background: #28a745;
    color: white;
    border: none;
    border-radius: 4px;
    font-size: 1rem;
    cursor: pointer;
    align-self: flex-start;
}

.run-btn:hover {
    background: #218838;
}

.command-preview {
    margin-top: 1rem;
    padding: 0.75rem;
    background: #e9ecef;
    border-radius: 4px;
}

.command-preview code {
    font-family: monospace;
    word-break: break-all;
}

.output-panel h3 {
    margin-top: 0;
}

.status-bar {
    margin-bottom: 1rem;
}

.status {
    display: inline-block;
    padding: 0.25rem 0.75rem;
    border-radius: 4px;
    font-size: 0.9em;
    font-weight: 500;
}

.status.pending { background: #fff3cd; color: #856404; }
.status.running { background: #cce5ff; color: #004085; }
.status.success { background: #d4edda; color: #155724; }
.status.error { background: #f8d7da; color: #721c24; }

.error-output,
.stderr-output {
    background: #f8d7da;
    padding: 1rem;
    border-radius: 4px;
    margin-bottom: 1rem;
}

.data-output,
.stdout-output {
    background: #d4edda;
    padding: 1rem;
    border-radius: 4px;
    margin-bottom: 1rem;
}

pre {
    margin: 0.5rem 0 0 0;
    white-space: pre-wrap;
    word-break: break-word;
    font-family: monospace;
}
</style>
```

---

## Configuration

Edit the `cmd` in the frontmatter to wrap your own CLI tool:

```yaml
sources:
  command:
    type: exec
    cmd: your-cli-tool --arg1 value1 --arg2 value2
    manual: true
```

The `manual: true` option means the command only runs when you click "Run".

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
