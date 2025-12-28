---
title: "SQLite Data Source"
sidebar: false
sources:
  tasks:
    type: sqlite
    db: "./tasks.db"
    table: tasks
    readonly: false
---

# SQLite Data Source

This example demonstrates reading and writing data from a SQLite database.

The `sqlite` source type provides:
- Automatic table creation from form fields
- CRUD operations (Add, Update, Delete, Toggle)
- Persistent storage in a local `.db` file

## Task List

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
                           lvt-click="Toggle" lvt-data-id="{{.Id}}">
                </td>
                <td {{if .Done}}style="text-decoration: line-through; opacity: 0.7"{{end}}>
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

    <h4>Add New Task</h4>
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

    <button lvt-click="Refresh" style="margin-top: 16px;">Refresh</button>
</main>
```

## Configuration

The source is configured in the frontmatter:

```yaml
sources:
  tasks:
    type: sqlite
    db: "./tasks.db"      # SQLite database file
    table: tasks          # Table name
    readonly: false       # Enable Add/Update/Delete
```

## Supported Actions

- **Add** - Insert new record from form fields
- **Update** - Update record by id
- **Delete** - Delete record by id
- **Toggle** - Toggle boolean field (like `done`)
- **Refresh** - Reload data from database
