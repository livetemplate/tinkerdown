---
title: "<<.Title>>"
sources:
  tasks:
    type: json
    file: metrics.json
---

# <<.Title>>

A project metrics dashboard powered by a JSON data file.

## Summary

- **Total Tasks:** `=count(tasks)`
- **Completed:** `=count(tasks where status = "done")`
- **In Progress:** `=count(tasks where status = "in-progress")`
- **Remaining:** `=count(tasks where status = "todo")`
- **Max Priority:** `=max(tasks.priority)`
- **Total Effort:** `=sum(tasks.effort)`

## All Tasks

```lvt
<table lvt-source="tasks" lvt-columns="id:ID,task:Task,status:Status,priority:Priority,effort:Effort" lvt-empty="No tasks found.">
</table>
```

---

## How It Works

This app demonstrates **computed expressions** — inline calculations that update automatically.

### Computed Expressions

Write expressions directly in markdown using backtick syntax:

- `` `=count(tasks)` `` — count all items
- `` `=count(tasks where status = "done")` `` — count with filter
- `` `=sum(tasks.effort)` `` — sum a numeric field
- `` `=max(tasks.priority)` `` — find the maximum value

### JSON Source

Data is loaded from `metrics.json`:

```yaml
sources:
  tasks:
    type: json
    file: metrics.json
```

Edit `metrics.json` to change the data. The dashboard updates on next page load.
