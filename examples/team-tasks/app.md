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

## Dashboard

- **Total Tasks:** `=count(tasks)`
- **Todo:** `=count(tasks where status = todo)`
- **In Progress:** `=count(tasks where status = in_progress)`
- **Done:** `=count(tasks where status = done)`

## [All] | [Mine] assigned_to = operator | [Todo] status = todo | [In Progress] status = in_progress | [Done] status = done

```lvt
<div lvt-source="tasks">
    <!-- Add Task Form -->
    <form lvt-submit="Add" lvt-reset-on:success style="display: flex; flex-wrap: wrap; gap: 12px; align-items: flex-end; padding: 16px; background: #f8f9fa; border-radius: 8px; margin-bottom: 24px;">
        <div style="flex: 2; min-width: 200px;">
            <label for="title" style="display: block; font-size: 12px; color: #666; margin-bottom: 4px;">Task Title</label>
            <input type="text" name="title" id="title" required placeholder="What needs to be done?"
                   maxlength="200"
                   style="width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px;">
        </div>
        <div style="flex: 1; min-width: 120px;">
            <label for="assigned_to" style="display: block; font-size: 12px; color: #666; margin-bottom: 4px;">Assigned To</label>
            <input type="text" name="assigned_to" id="assigned_to" required placeholder="Username"
                   maxlength="50" pattern="[a-zA-Z0-9_-]+"
                   title="Username can only contain letters, numbers, underscores, and hyphens"
                   style="width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px;">
        </div>
        <div style="flex: 1; min-width: 100px;">
            <label for="priority" style="display: block; font-size: 12px; color: #666; margin-bottom: 4px;">Priority</label>
            <select name="priority" id="priority" required
                    style="width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px;">
                <option value="low">Low</option>
                <option value="medium" selected>Medium</option>
                <option value="high">High</option>
            </select>
        </div>
        <div style="flex: 1; min-width: 120px;">
            <label for="status" style="display: block; font-size: 12px; color: #666; margin-bottom: 4px;">Status</label>
            <select name="status" id="status" required
                    style="width: 100%; padding: 10px; border: 1px solid #ddd; border-radius: 4px;">
                <option value="todo" selected>Todo</option>
                <option value="in_progress">In Progress</option>
                <option value="done">Done</option>
            </select>
        </div>
        <button type="submit"
                style="padding: 10px 20px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer; font-weight: 500;">
            + Add Task
        </button>
    </form>

    <!-- Tasks Table -->
    {{if .Error}}
    <p style="color: #dc3545; padding: 12px; background: #f8d7da; border-radius: 4px;">Error: {{.Error}}</p>
    {{else}}
    <table style="width: 100%; border-collapse: collapse; margin-bottom: 24px;">
        <thead>
            <tr style="background: #f8f9fa; border-bottom: 2px solid #dee2e6;">
                <th style="padding: 12px; text-align: left;">Task</th>
                <th style="padding: 12px; text-align: left;">Assigned To</th>
                <th style="padding: 12px; text-align: center;">Priority</th>
                <th style="padding: 12px; text-align: center;">Status</th>
                <th style="padding: 12px; text-align: center; width: 100px;">Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr data-key="{{.Id}}" style="border-bottom: 1px solid #eee;">
                <td style="padding: 12px; {{if eq .Status "done"}}text-decoration: line-through; opacity: 0.6;{{end}}">
                    {{.Title}}
                </td>
                <td style="padding: 12px;">
                    <span style="padding: 2px 8px; background: #e9ecef; border-radius: 12px; font-size: 12px;">
                        @{{.AssignedTo}}
                    </span>
                </td>
                <td style="padding: 12px; text-align: center;">
                    {{if eq .Priority "high"}}
                    <span style="padding: 2px 8px; background: #f8d7da; color: #721c24; border-radius: 4px; font-size: 12px; font-weight: 500;">High</span>
                    {{else if eq .Priority "medium"}}
                    <span style="padding: 2px 8px; background: #fff3cd; color: #856404; border-radius: 4px; font-size: 12px; font-weight: 500;">Medium</span>
                    {{else}}
                    <span style="padding: 2px 8px; background: #d4edda; color: #155724; border-radius: 4px; font-size: 12px; font-weight: 500;">Low</span>
                    {{end}}
                </td>
                <td style="padding: 12px; text-align: center;">
                    {{if eq .Status "done"}}
                    <span style="padding: 4px 10px; background: #28a745; color: white; border-radius: 12px; font-size: 12px;">Done</span>
                    {{else if eq .Status "in_progress"}}
                    <span style="padding: 4px 10px; background: #007bff; color: white; border-radius: 12px; font-size: 12px;">In Progress</span>
                    {{else}}
                    <span style="padding: 4px 10px; background: #6c757d; color: white; border-radius: 12px; font-size: 12px;">Todo</span>
                    {{end}}
                </td>
                <td style="padding: 12px; text-align: center;">
                    <button lvt-click="Delete" lvt-data-id="{{.Id}}"
                            aria-label="Delete task: {{.Title}}"
                            style="padding: 4px 8px; background: transparent; color: #dc3545; border: 1px solid #dc3545; border-radius: 4px; cursor: pointer; font-size: 12px;"
                            title="Delete task">
                        Delete
                    </button>
                </td>
            </tr>
            {{else}}
            <tr>
                <td colspan="5" style="padding: 40px; text-align: center; color: #6c757d;">
                    No tasks yet. Create one using the form above!
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}

    <!-- Bulk Actions -->
    <div style="display: flex; gap: 8px; flex-wrap: wrap; padding-top: 16px; border-top: 1px solid #dee2e6;">
        <button lvt-click="mark-mine-done"
                aria-label="Mark all my tasks as done"
                style="padding: 8px 16px; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer;">
            Mark My Tasks Done
        </button>
        <button lvt-click="clear-done"
                aria-label="Delete all completed tasks"
                style="padding: 8px 16px; background: #dc3545; color: white; border: none; border-radius: 4px; cursor: pointer;">
            Clear Completed
        </button>
        <button lvt-click="Refresh"
                aria-label="Refresh task list"
                style="padding: 8px 16px; background: #6c757d; color: white; border: none; border-radius: 4px; cursor: pointer;">
            Refresh
        </button>
    </div>
</div>
```

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

- **Tabbed Filtering**: Switch between All, Mine, Todo, In Progress, and Done views
- **Live Statistics**: Dashboard shows real-time task counts using computed expressions
- **CRUD Operations**: Add, delete tasks with instant UI updates
- **Custom SQL Actions**: Bulk "Clear Completed" and "Mark My Tasks Done" with confirmation dialogs
- **Real-time Sync**: WebSocket-based updates across browser instances
- **Operator Identity**: "Mine" tab and "Mark My Tasks Done" filter by `--operator` flag value

## Technical Notes

### Field Name Conversion

Form field names use `snake_case` (e.g., `assigned_to`) which SQLite stores as-is. In Go templates, these are accessed using `PascalCase` (e.g., `{{.AssignedTo}}`). This conversion is handled automatically by the runtime.

### Operator Parameter

The `:operator` parameter in SQL actions (like `mark-mine-done`) is automatically populated from the `--operator` CLI flag value. This enables user-specific operations without manual input:

```yaml
actions:
  mark-mine-done:
    kind: sql
    source: tasks
    statement: "UPDATE tasks SET status = 'done' WHERE assigned_to = :operator"
```

### Database Schema

The SQLite database (`tasks.db`) is auto-created on first use. The schema is inferred from form submissions:

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Auto-generated primary key |
| title | TEXT | Task description (max 200 chars) |
| assigned_to | TEXT | Username of assignee (max 50 chars) |
| priority | TEXT | low, medium, or high |
| status | TEXT | todo, in_progress, or done |
