---
title: "LVT Block Test"
sources:
  items:
    type: sqlite
    db: ./test.db
    table: items
---

# Interactive Block

```lvt
<main lvt-source="items">
  <h2>Items</h2>
  {{if .Error}}
    <p class="error">{{.Error}}</p>
  {{else}}
    <ul>
      {{range .Data}}
      <li>{{.name}} - ${{.price}}</li>
      {{end}}
    </ul>
  {{end}}
  <button lvt-click="Refresh">Reload</button>
</main>
```
