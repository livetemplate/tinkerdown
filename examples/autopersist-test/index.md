---
title: "Auto-Persist Test (with Delete)"
---

# Items Test

Test for `lvt-data-*` attributes with delete buttons.

```lvt
<main>
    <h2>Add Item</h2>

    <form lvt-submit="save" lvt-persist="items" lvt-reset-on:success>
        <fieldset role="group">
            <input type="text" name="title" required placeholder="Item title">
            <button type="submit">Add</button>
        </fieldset>
    </form>

    <h2>Items</h2>

    {{if .Items}}
    {{range .Items}}
    <fieldset role="group" data-item-id="{{.Id}}">
        <input type="text" value="{{.Title}}" readonly>
        <button lvt-click="Delete" lvt-data-id="{{.Id}}" lvt-confirm="Are you sure you want to delete this item?">Delete</button>
    </fieldset>
    {{end}}
    {{else}}
    <p><em>No items yet.</em></p>
    {{end}}
</main>
```
