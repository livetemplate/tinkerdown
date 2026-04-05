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
