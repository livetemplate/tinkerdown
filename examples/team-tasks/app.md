---
title: "Team Tasks"
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
    statement: "DELETE FROM tasks WHERE status = 'done'"
    confirm: "Delete all completed tasks? This cannot be undone."

  mark-mine-done:
    kind: sql
    source: tasks
    statement: "UPDATE tasks SET status = 'done' WHERE assigned_to = :operator"
    confirm: "Mark all your tasks as done?"
---

# Team Tasks

A collaborative task board for teams with real-time synchronization.

- **Total** `=count(tasks)`
- **Todo** `=count(tasks where status = todo)`
- **In Progress** `=count(tasks where status = in_progress)`
- **Done** `=count(tasks where status = done)`

## [All] | [Mine] assigned_to = operator | [Todo] status = todo | [In Progress] status = in_progress | [Done] status = done

```lvt
<article lvt-source="tasks">
    <header>Add New Task</header>
    <form lvt-submit="Add" lvt-reset-on:success>
        <div class="grid">
            <label>
                Task Title
                <input type="text" name="title" required placeholder="What needs to be done?" maxlength="200">
            </label>
            <label>
                Assigned To
                <input type="text" name="assigned_to" required placeholder="Username" maxlength="50" pattern="[a-zA-Z0-9_-]+">
            </label>
        </div>
        <div class="grid">
            <label>
                Priority
                <select name="priority" required>
                    <option value="low">Low</option>
                    <option value="medium" selected>Medium</option>
                    <option value="high">High</option>
                </select>
            </label>
            <label>
                Status
                <select name="status" required>
                    <option value="todo" selected>Todo</option>
                    <option value="in_progress">In Progress</option>
                    <option value="done">Done</option>
                </select>
            </label>
            <button type="submit">Add Task</button>
        </div>
    </form>

    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th scope="col">Task</th>
                <th scope="col">Assigned To</th>
                <th scope="col">Priority</th>
                <th scope="col">Status</th>
                <th scope="col">Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr data-key="{{.Id}}">
                <td>{{if eq .Status "done"}}<del>{{.Title}}</del>{{else}}{{.Title}}{{end}}</td>
                <td><kbd>@{{.AssignedTo}}</kbd></td>
                <td>{{if eq .Priority "high"}}<mark>High</mark>{{else if eq .Priority "medium"}}Medium{{else}}Low{{end}}</td>
                <td>{{if eq .Status "done"}}<ins>Done</ins>{{else if eq .Status "in_progress"}}<em>In Progress</em>{{else}}Todo{{end}}</td>
                <td><button lvt-click="Delete" lvt-data-id="{{.Id}}" class="outline secondary">Delete</button></td>
            </tr>
            {{else}}
            <tr>
                <td colspan="5"><em>No tasks yet. Create one using the form above!</em></td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}

    <footer>
        <div role="group">
            <button lvt-click="mark-mine-done">Mark My Tasks Done</button>
            <button lvt-click="clear-done" class="secondary">Clear Completed</button>
            <button lvt-click="Refresh" class="contrast">Refresh</button>
        </div>
    </footer>
</article>
```

---

## Multi-User Testing

To test real-time synchronization between multiple users:

1. Open multiple browser windows/tabs
2. Start the server with different operator identities:

```bash
# Terminal 1 - User Alice
tinkerdown serve --operator alice

# Terminal 2 - User Bob
tinkerdown serve --operator bob --port 8081
```

3. Create and modify tasks in different windows
4. Watch changes sync in real-time via WebSocket

## Features Demonstrated

| Feature | Description |
|---------|-------------|
| **Tabbed Filtering** | Switch between All, Mine, Todo, In Progress, and Done views |
| **Live Statistics** | Dashboard shows real-time task counts using computed expressions |
| **CRUD Operations** | Add, delete tasks with instant UI updates |
| **Custom SQL Actions** | Bulk "Clear Completed" and "Mark My Tasks Done" with confirmation |
| **Real-time Sync** | WebSocket-based updates across browser instances |
| **Operator Identity** | "Mine" tab and "Mark My Tasks Done" filter by `--operator` flag |

## Technical Notes

### Field Name Conversion

Form field names use `snake_case` (e.g., `assigned_to`) which SQLite stores as-is. In Go templates, these are accessed using `PascalCase` (e.g., `{{.AssignedTo}}`). This conversion is handled automatically by the runtime.

### Operator Parameter

The `:operator` parameter in SQL actions is automatically populated from the `--operator` CLI flag value:

```yaml
actions:
  mark-mine-done:
    kind: sql
    source: tasks
    statement: "UPDATE tasks SET status = 'done' WHERE assigned_to = :operator"
```

### Database Schema

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Auto-generated primary key |
| title | TEXT | Task description (max 200 chars) |
| assigned_to | TEXT | Username of assignee (max 50 chars) |
| priority | TEXT | low, medium, or high |
| status | TEXT | todo, in_progress, or done |
