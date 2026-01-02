---
title: "Multi-Source Dashboard"
persist: localstorage
sources:
  tasks:
    type: sqlite
    db: ./data.db
    table: tasks
  users:
    type: rest
    url: https://api.example.com/users
  logs:
    type: exec
    cmd: tail -n 10 /var/log/app.log
    manual: true
---

# Dashboard

This page uses multiple data sources.

## Tasks

```lvt
<div lvt-source="tasks">
  {{range .Data}}
  <p>{{.title}}</p>
  {{end}}
</div>
```
