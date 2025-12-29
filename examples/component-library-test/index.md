---
title: "Auto-Rendering Tables"
sources:
  users:
    type: json
    file: users.json
  countries:
    type: json
    file: countries.json
  empty_source:
    type: json
    file: empty.json
---

# Auto-Rendering Tables

This page demonstrates auto-table generation for simple data display.

## Simple Tables

### Test 1: Table with Explicit Columns

```lvt
<table lvt-source="users" lvt-columns="name:Name,email:Email">
</table>
```

### Test 2: Table with Actions

```lvt
<table lvt-source="users" lvt-columns="name:Name,role:Role" lvt-actions="delete:Delete,edit:Edit">
</table>
```

### Test 3: Table with Empty State

```lvt
<table lvt-source="empty_source" lvt-columns="name:Name,email:Email" lvt-empty="No users found">
</table>
```

### Test 4: Table with Auto-Discovery (No Columns)

```lvt
<table lvt-source="users">
</table>
```

---

## Auto Select Dropdown

```lvt
<div class="select-container">
  <label>Select a country:</label>
  <select lvt-source="countries" lvt-value="code" lvt-label="name" class="test-select">
  </select>
</div>
```
