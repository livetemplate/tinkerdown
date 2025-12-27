---
title: "Project Dashboard"
description: "Dashboard with cross-file markdown data sources"
sources:
  tasks:
    type: markdown
    file: "./_data/tasks.md"
    anchor: "#active"
    readonly: false
  team:
    type: markdown
    file: "./_data/team.md"
    anchor: "#members"
    # readonly: true (default)
---

# {{.Config.Title}}

A dashboard that aggregates data from multiple markdown files.

## Active Tasks

Tasks are stored in `_data/tasks.md` and can be edited here or directly in that file.

```lvt
<main lvt-source="tasks">
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <form lvt-submit="Add" style="display: flex; gap: 8px; margin-bottom: 16px;">
        <input type="text" name="Task" placeholder="New task..." required
               style="flex: 1; padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
        <select name="Priority" style="padding: 8px; border: 1px solid #ccc; border-radius: 4px;">
            <option value="low">Low</option>
            <option value="medium" selected>Medium</option>
            <option value="high">High</option>
        </select>
        <button type="submit"
                style="padding: 8px 16px; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer;">
            Add Task
        </button>
    </form>

    {{if eq (len .Data) 0}}
    <p><em>No active tasks. Add one above!</em></p>
    {{else}}
    <table style="width: 100%; border-collapse: collapse;">
        <thead>
            <tr style="background: #f8f9fa;">
                <th style="text-align: left; padding: 10px; border-bottom: 2px solid #dee2e6;">Task</th>
                <th style="text-align: left; padding: 10px; border-bottom: 2px solid #dee2e6;">Priority</th>
                <th style="width: 60px; padding: 10px; border-bottom: 2px solid #dee2e6;"></th>
            </tr>
        </thead>
        <tbody>
        {{range .Data}}
            <tr style="border-bottom: 1px solid #dee2e6;">
                <td style="padding: 10px;">{{.Task}}</td>
                <td style="padding: 10px;">
                    {{if eq .Priority "high"}}
                    <span style="background: #f8d7da; color: #721c24; padding: 2px 8px; border-radius: 4px;">High</span>
                    {{else if eq .Priority "medium"}}
                    <span style="background: #fff3cd; color: #856404; padding: 2px 8px; border-radius: 4px;">Medium</span>
                    {{else}}
                    <span style="background: #d4edda; color: #155724; padding: 2px 8px; border-radius: 4px;">Low</span>
                    {{end}}
                </td>
                <td style="padding: 10px; text-align: center;">
                    <button lvt-click="Delete" lvt-data-id="{{.Id}}"
                            style="padding: 4px 8px; color: #dc3545; border: 1px solid #dc3545; background: transparent; border-radius: 4px; cursor: pointer;">
                        x
                    </button>
                </td>
            </tr>
        {{end}}
        </tbody>
    </table>
    <p style="color: #666; font-size: 0.9em; margin-top: 8px;">{{len .Data}} active task(s)</p>
    {{end}}
    {{end}}
</main>
```

---

## Team Members

Team data is stored in `_data/team.md` (read-only in this dashboard).

```lvt
<aside lvt-source="team">
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else if eq (len .Data) 0}}
    <p><em>No team members defined.</em></p>
    {{else}}
    <ul style="list-style: none; padding: 0;">
        {{range .Data}}
        <li style="display: flex; align-items: center; gap: 12px; padding: 8px 0; border-bottom: 1px solid #eee;">
            <div style="width: 40px; height: 40px; background: #007bff; color: white; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-weight: bold;">
                {{slice .Name 0 1}}
            </div>
            <div>
                <div style="font-weight: 500;">{{.Name}}</div>
                <div style="color: #666; font-size: 0.9em;">{{.Role}}</div>
            </div>
        </li>
        {{end}}
    </ul>
    <p style="color: #666; font-size: 0.9em; margin-top: 8px;">{{len .Data}} team member(s)</p>
    {{end}}
</aside>
```

---

*Data files are stored in the `_data/` directory for clean separation.*
