---
title: "[[.Title]]"
sources:
  tasks:
    type: sqlite
    db: "./tasks.db"
    table: tasks
    readonly: false
---

# [[.Title]]

A simple todo list with SQLite persistence.

## My Tasks

```lvt
<main lvt-source="tasks">
    {{if .Error}}
    <p class="error">Error: {{.Error}}</p>
    {{else}}
    <ul class="task-list">
        {{range .Data}}
        <li class="task-item {{if .Done}}done{{end}}">
            <input type="checkbox" {{if .Done}}checked{{end}}
                   lvt-click="Toggle" lvt-data-id="{{.Id}}">
            <span class="task-text">{{.Text}}</span>
            <button class="delete-btn" lvt-click="Delete" lvt-data-id="{{.Id}}">x</button>
        </li>
        {{end}}
    </ul>
    <p class="task-count">{{len .Data}} task(s)</p>
    {{end}}

    <form class="add-form" lvt-submit="Add">
        <input type="text" name="text" placeholder="Add a new task..." required>
        <button type="submit">Add</button>
    </form>
</main>

<style>
.task-list {
    list-style: none;
    padding: 0;
    margin: 0 0 1rem 0;
}

.task-item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem;
    border-bottom: 1px solid #eee;
}

.task-item.done .task-text {
    text-decoration: line-through;
    opacity: 0.6;
}

.task-text {
    flex: 1;
}

.delete-btn {
    padding: 0.25rem 0.5rem;
    color: #dc3545;
    border: 1px solid #dc3545;
    background: transparent;
    border-radius: 4px;
    cursor: pointer;
    opacity: 0.5;
}

.task-item:hover .delete-btn {
    opacity: 1;
}

.task-count {
    color: #666;
    font-size: 0.9em;
    margin: 0.5rem 0;
}

.add-form {
    display: flex;
    gap: 0.5rem;
}

.add-form input {
    flex: 1;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
}

.add-form button {
    padding: 0.5rem 1rem;
    background: #007bff;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
}

.add-form button:hover {
    background: #0056b3;
}

.error {
    color: #dc3545;
    background: #f8d7da;
    padding: 0.75rem;
    border-radius: 4px;
}
</style>
```

## How It Works

- **Add tasks**: Type in the input and click "Add"
- **Complete tasks**: Click the checkbox to toggle done state
- **Delete tasks**: Click the "x" button to remove a task
- **Data persists**: All data is saved in `tasks.db`

## Next Steps

- Add a priority field (see `--template=dashboard` for multi-field example)
- Add categories or tags
- Style with your favorite CSS framework
