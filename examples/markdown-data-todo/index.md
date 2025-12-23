---
title: "Markdown Data Todo"
sources:
  tasks:
    type: markdown
    anchor: "#data-section"
---

# Todo List from Markdown

This example demonstrates reading task list data from a markdown section in the same file.

## Interactive Todo Display

```lvt
<main lvt-source="tasks">
    <h3>My Tasks</h3>
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <ul>
        {{range .Data}}
        <li>
            <input type="checkbox" {{if .Done}}checked{{end}} disabled>
            <span {{if .Done}}style="text-decoration: line-through; opacity: 0.7"{{end}}>{{.Text}}</span>
        </li>
        {{end}}
    </ul>
    <p><small>Total: {{len .Data}} tasks</small></p>
    {{end}}
    <button lvt-click="Refresh">Refresh</button>
</main>
```

---

## Data Section {#data-section}

Edit this task list directly in the markdown file:

- [ ] Buy groceries <!-- id:task1 -->
- [x] Clean the house <!-- id:task2 -->
- [ ] Walk the dog <!-- id:task3 -->
- [ ] Send emails <!-- id:task4 -->
- [x] Finish project report <!-- id:task5 -->
