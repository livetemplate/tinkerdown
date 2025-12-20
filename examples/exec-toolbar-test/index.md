---
title: "Exec Toolbar Test"
sources:
  system-info:
    type: exec
    cmd: ./get-info.sh
    manual: true
  auto-info:
    type: exec
    cmd: ./auto-info.sh
---

# Exec Toolbar Test

This example demonstrates the exec toolbar feature for runbook-style command execution.

## Manual Execution (with Run Button)

This block uses `manual: true` so it won't execute until you click Run:

```lvt
<div lvt-source="system-info">
  {{if eq .Status "idle"}}
  <p style="color: #9ca3af; font-style: italic;">Click Run to see system information</p>
  {{else if eq .Status "running"}}
  <p style="color: #f59e0b;">Loading...</p>
  {{else if eq .Status "error"}}
  <p style="color: #ef4444;">Error: {{.Error}}</p>
  {{else}}
  <ul>
    {{range .Data}}
    <li><strong>{{.key}}</strong>: {{.value}}</li>
    {{end}}
  </ul>
  {{end}}
</div>
```

## Auto Execution (default)

This block auto-executes on page load (no `manual: true`):

```lvt
<div lvt-source="auto-info">
  {{if eq .Status "success"}}
  <p>Status: {{range .Data}}{{.status}}{{end}}</p>
  {{else if eq .Status "error"}}
  <p style="color: #ef4444;">Error: {{.Error}}</p>
  {{else}}
  <p>Loading...</p>
  {{end}}
</div>
```
