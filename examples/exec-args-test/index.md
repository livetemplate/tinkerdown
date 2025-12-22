---
title: "Exec Args Form Test"
sources:
  greeting:
    type: exec
    cmd: ./greet.sh --name World --count 3 --uppercase false
    manual: true
---

# Exec Args Form Test

Test for auto-generated argument forms with exec data source.

```lvt
<main lvt-source="greeting">
    <h2>Greeting Generator</h2>

    <form lvt-submit="Run">
        {{range .Args}}
        <div class="form-field">
            {{if eq .Type "bool"}}
            <label>
                <input type="checkbox" name="{{.Name}}" {{if eq .Value "true"}}checked{{end}}>
                {{.Label}}
                {{if .Description}}<small>({{.Description}})</small>{{end}}
            </label>
            {{else if eq .Type "number"}}
            <label>
                {{.Label}}{{if .Description}} <small>({{.Description}})</small>{{end}}
                <input type="number" name="{{.Name}}" value="{{.Value}}">
            </label>
            {{else}}
            <label>
                {{.Label}}{{if .Description}} <small>({{.Description}})</small>{{end}}
                <input type="text" name="{{.Name}}" value="{{.Value}}">
            </label>
            {{end}}
        </div>
        {{end}}
        <button type="submit">Run</button>
    </form>

    <hr>

    <p><strong>Command:</strong> <code>{{.Command}}</code></p>
    <p><strong>Status:</strong> {{.Status}}</p>

    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{end}}

    {{if .Data}}
    <h3>Output</h3>
    <table>
        <thead>
            <tr>
                <th>Index</th>
                <th>Message</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>{{.index}}</td>
                <td>{{.message}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}

    {{if .Stderr}}
    <pre><code>{{.Stderr}}</code></pre>
    {{end}}
</main>
```
