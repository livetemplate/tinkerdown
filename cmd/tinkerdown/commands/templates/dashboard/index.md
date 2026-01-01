---
title: "[[.Title]]"
sources:
  tasks:
    type: markdown
    file: "./_data/tasks.md"
    anchor: "#tasks"
    readonly: false
  team:
    type: markdown
    file: "./_data/team.md"
    anchor: "#team"
---

# [[.Title]]

A dashboard that aggregates data from multiple sources.

## Overview

```lvt
<div class="stats-grid">
    <div class="stat-card" lvt-source="tasks">
        <div class="stat-value">{{len .Data}}</div>
        <div class="stat-label">Active Tasks</div>
    </div>
    <div class="stat-card" lvt-source="team">
        <div class="stat-value">{{len .Data}}</div>
        <div class="stat-label">Team Members</div>
    </div>
</div>

<style>
.stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 1rem;
    margin-bottom: 2rem;
}

.stat-card {
    background: #f8f9fa;
    padding: 1.5rem;
    border-radius: 8px;
    text-align: center;
}

.stat-value {
    font-size: 2rem;
    font-weight: bold;
    color: #007bff;
}

.stat-label {
    color: #666;
    font-size: 0.9rem;
}
</style>
```

---

## Active Tasks

```lvt
<section lvt-source="tasks">
    {{if .Error}}
    <p class="error">Error: {{.Error}}</p>
    {{else if eq (len .Data) 0}}
    <p class="empty">No active tasks. Add one below!</p>
    {{else}}
    <table class="data-table">
        <thead>
            <tr>
                <th>Task</th>
                <th>Priority</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>{{.Task}}</td>
                <td>
                    {{if eq .Priority "high"}}
                    <span class="badge badge-high">High</span>
                    {{else if eq .Priority "medium"}}
                    <span class="badge badge-medium">Medium</span>
                    {{else}}
                    <span class="badge badge-low">Low</span>
                    {{end}}
                </td>
                <td>
                    <button class="btn-delete" lvt-click="Delete" lvt-data-id="{{.Id}}">x</button>
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}

    <form class="add-form" lvt-submit="Add">
        <input type="text" name="Task" placeholder="New task..." required>
        <select name="Priority">
            <option value="low">Low</option>
            <option value="medium" selected>Medium</option>
            <option value="high">High</option>
        </select>
        <button type="submit">Add Task</button>
    </form>
</section>

<style>
.data-table {
    width: 100%;
    border-collapse: collapse;
    margin-bottom: 1rem;
}

.data-table th,
.data-table td {
    padding: 0.75rem;
    text-align: left;
    border-bottom: 1px solid #dee2e6;
}

.data-table thead {
    background: #f8f9fa;
}

.badge {
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.85em;
}

.badge-high { background: #f8d7da; color: #721c24; }
.badge-medium { background: #fff3cd; color: #856404; }
.badge-low { background: #d4edda; color: #155724; }

.btn-delete {
    padding: 0.25rem 0.5rem;
    color: #dc3545;
    border: 1px solid #dc3545;
    background: transparent;
    border-radius: 4px;
    cursor: pointer;
}

.add-form {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
}

.add-form input {
    flex: 1;
    min-width: 200px;
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
}

.add-form select {
    padding: 0.5rem;
    border: 1px solid #ddd;
    border-radius: 4px;
}

.add-form button {
    padding: 0.5rem 1rem;
    background: #28a745;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
}

.error {
    color: #dc3545;
    background: #f8d7da;
    padding: 0.75rem;
    border-radius: 4px;
}

.empty {
    color: #666;
    font-style: italic;
}
</style>
```

---

## Team Members

```lvt
<section lvt-source="team">
    {{if .Error}}
    <p class="error">Error: {{.Error}}</p>
    {{else if eq (len .Data) 0}}
    <p class="empty">No team members defined.</p>
    {{else}}
    <div class="team-grid">
        {{range .Data}}
        <div class="team-card">
            <div class="avatar">{{slice .Name 0 1}}</div>
            <div class="member-info">
                <div class="member-name">{{.Name}}</div>
                <div class="member-role">{{.Role}}</div>
            </div>
        </div>
        {{end}}
    </div>
    {{end}}
</section>

<style>
.team-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 1rem;
}

.team-card {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1rem;
    background: #f8f9fa;
    border-radius: 8px;
}

.avatar {
    width: 40px;
    height: 40px;
    background: #007bff;
    color: white;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: bold;
}

.member-name {
    font-weight: 500;
}

.member-role {
    color: #666;
    font-size: 0.9em;
}
</style>
```

---

*Data files are stored in the `_data/` directory.*
