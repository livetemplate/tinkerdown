---
title: "Custom Action Buttons"
sidebar: false
sources:
  tasks:
    type: sqlite
    db: "./tasks.db"
    table: tasks
    readonly: false

actions:
  clear-done:
    kind: sql
    source: tasks
    statement: "DELETE FROM tasks WHERE done = 1"
    confirm: "Delete all completed tasks?"

  mark-all-done:
    kind: sql
    source: tasks
    statement: "UPDATE tasks SET done = 1"
    confirm: "Mark all tasks as complete?"
---

# Custom Action Buttons

This example demonstrates **custom actions** declared in frontmatter. Actions can execute SQL statements, HTTP requests, or shell commands when triggered via `name` attribute on buttons.

## Task List with Custom Actions

```lvt
<main lvt-source="tasks">
    <h3>My Tasks</h3>
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>Done</th>
                <th>Task</th>
                <th>Priority</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>
                    <input type="checkbox" {{if .Done}}checked{{end}}
                           lvt-on:click="Toggle" data-id="{{.Id}}">
                </td>
                <td {{if .Done}}style="text-decoration: line-through; opacity: 0.7"{{end}}>
                    {{.Text}}
                </td>
                <td>{{.Priority}}</td>
                <td>
                    <button name="Delete" data-id="{{.Id}}"
                            style="color: red; border: 1px solid red; background: transparent; border-radius: 4px; cursor: pointer; padding: 2px 8px;">
                        Delete
                    </button>
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
    <p><small>Total: {{len .Data}} tasks</small></p>
    {{end}}

    <hr style="margin: 16px 0;">

    <h4>Add New Task</h4>
    <form name="Add" style="display: flex; gap: 8px; flex-wrap: wrap; align-items: center;">
        <input type="text" name="text" placeholder="Task description..." required
               style="flex: 1; min-width: 200px; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
        <select name="priority" style="padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
            <option value="low">Low</option>
            <option value="medium" selected>Medium</option>
            <option value="high">High</option>
        </select>
        <button type="submit"
                style="padding: 8px 16px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;">
            Add Task
        </button>
    </form>

    <hr style="margin: 16px 0;">

    <h4>Bulk Actions (Custom Actions)</h4>
    <div style="display: flex; gap: 8px; flex-wrap: wrap;">
        <button name="clear-done"
                style="padding: 8px 16px; background: #dc3545; color: white; border: none; border-radius: 4px; cursor: pointer;">
            Clear Completed
        </button>
        <button name="mark-all-done"
                style="padding: 8px 16px; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer;">
            Mark All Done
        </button>
        <button name="Refresh" style="padding: 8px 16px;">Refresh</button>
    </div>
</main>
```

## Configuration

Custom actions are declared in frontmatter alongside sources:

```yaml
sources:
  tasks:
    type: sqlite
    db: "./tasks.db"
    table: tasks
    readonly: false

actions:
  clear-done:
    kind: sql
    source: tasks
    statement: "DELETE FROM tasks WHERE done = 1"
    confirm: "Delete all completed tasks?"

  archive-old:
    kind: sql
    source: tasks
    statement: "UPDATE tasks SET archived = 1 WHERE created_at < :cutoff"
    params:
      cutoff:
        type: date
        required: true

  mark-all-done:
    kind: sql
    source: tasks
    statement: "UPDATE tasks SET done = 1"
    confirm: "Mark all tasks as complete?"
```

## Action Kinds

| Kind | Description |
|------|-------------|
| `sql` | Execute SQL statement against a source (parameterized) |
| `http` | Send HTTP request (POST, GET, etc.) |
| `exec` | Run shell command (requires `--allow-exec` flag) |

## SQL Actions

SQL actions use parameterized queries with `:param` placeholders:

```yaml
archive-old:
  kind: sql
  source: tasks
  statement: "UPDATE tasks SET archived = 1 WHERE created_at < :cutoff"
  params:
    cutoff:
      type: date
      required: true
```

Parameters are substituted safely (no SQL injection risk).

## Security

- SQL actions use parameterized queries only
- Exec actions require `--allow-exec` CLI flag
- Confirmation dialogs can be added with `confirm:` field
