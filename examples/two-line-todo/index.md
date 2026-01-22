---
title: "Simple Todo"
sources:
  tasks:
    type: markdown
    file: "./_data/tasks.md"
    readonly: false
---

# Simple Todo

A minimal interactive todo list.

```lvt
<div lvt-source="tasks">
{{range .Data}}
<label style="display: block; padding: 4px 0; cursor: pointer;">
  <input type="checkbox" {{if .Done}}checked{{end}} lvt-click="Toggle" lvt-data-id="{{.Id}}">
  <span {{if .Done}}style="text-decoration: line-through; opacity: 0.6"{{end}}>{{.Text}}</span>
</label>
{{end}}
</div>
```
