---
title: "{{.Title}}"
sources:
  tasks:
    type: sqlite
    db: "./tasks.db"
    table: tasks
    readonly: false
---

# {{.Title}}

A simple task manager with SQLite persistence.

## Tasks

```lvt
<main lvt-source="tasks">
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
                           lvt-click="Toggle" lvt-data-id="{{.Id}}">
                </td>
                <td {{if .Done}}style="text-decoration: line-through; opacity: 0.6"{{end}}>
                    {{.Text}}
                </td>
                <td>{{.Priority}}</td>
                <td>
                    <button lvt-click="Delete" lvt-data-id="{{.Id}}"
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

    <h3>Add New Task</h3>
    <form lvt-submit="Add" style="display: flex; gap: 8px; flex-wrap: wrap; align-items: center;">
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
</main>
```

## How It Works

- **Add**: Submit the form to create a new task
- **Toggle**: Click checkbox to mark done/undone
- **Delete**: Remove a task permanently
- Data persists in `tasks.db` SQLite database
