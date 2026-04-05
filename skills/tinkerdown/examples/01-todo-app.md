---
title: "Todo App"
---

# Todo App

A simple todo list demonstrating `lvt-persist` for automatic CRUD operations.

**Features demonstrated:**
- `lvt-persist` - Auto-save to SQLite
- `name` (on form) - Form handling
- `name` (on button) - Button actions
- `data-id` - Pass data with actions
- Conditional rendering with `{{if}}`/`{{else}}`
- **No CSS classes needed** - PicoCSS styles semantic HTML automatically

```lvt
<main>
    <h1>Todo List</h1>

    <!-- Add Todo Form -->
    <form name="save" lvt-persist="todos">
        <fieldset role="group">
            <input type="text" name="title" required placeholder="What needs to be done?">
            <button type="submit">Add</button>
        </fieldset>
    </form>

    <!-- Todo List -->
    {{if .Todos}}
    <table>
        <thead>
            <tr>
                <th>Done</th>
                <th>Task</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Todos}}
            <tr>
                <td>
                    <input type="checkbox" {{if .Completed}}checked{{end}} lvt-on:click="ToggleComplete" data-id="{{.Id}}">
                </td>
                <td>{{if .Completed}}<s>{{.Title}}</s>{{else}}{{.Title}}{{end}}</td>
                <td><button name="Delete" data-id="{{.Id}}" >Delete</button></td>
            </tr>
            {{end}}
        </tbody>
    </table>
    <small>{{len .Todos}} items total</small>
    {{else}}
    <p><em>No todos yet. Add one above to get started!</em></p>
    {{end}}
</main>
```

## How It Works

1. **Form submission** - When you submit the form, `name="save"` triggers the save action
2. **Auto-persistence** - `lvt-persist="todos"` automatically:
   - Creates a SQLite table named `todos`
   - Generates `Title` column from the form field
   - Adds `Id`, `CreatedAt` columns automatically
   - Loads existing todos into `.Todos` on page load
3. **Toggle completion** - `lvt-on:click="ToggleComplete"` on checkbox with `data-id` passes the todo ID
4. **Delete** - `name="Delete"` on button removes the item from the database

## Prompt to Generate This

> Build a todo app with Livemdtools. I want to add todos, mark them complete with a checkbox, and delete them. Use semantic HTML - no CSS classes needed.
