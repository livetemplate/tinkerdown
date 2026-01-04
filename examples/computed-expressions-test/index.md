---
title: "Computed Expressions Test"
---

# Task Dashboard

## Stats

- **Total Tasks:** `=count(tasks)`
- **Completed:** `=count(tasks where done)`
- **Pending:** `=count(tasks where done = false)`
- **Max Priority:** `=max(tasks.priority)`
- **Sum Priority:** `=sum(tasks.priority)`

## Task List

```lvt
<ul lvt-source="tasks">
  <li>{{.title}} ({{if .done}}Done{{else}}Pending{{end}})</li>
</ul>
```
