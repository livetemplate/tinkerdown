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

This page demonstrates auto-table generation with two modes: **simple** (default) and **rich** (with `lvt-datatable`).

## Simple Mode (Default)

Simple mode generates inline `<thead>/<tbody>` HTML. Use this for basic data display.

### Test 1: Simple Table with Explicit Columns

```lvt
<table lvt-source="users" lvt-columns="name:Name,email:Email">
</table>
```

### Test 2: Simple Table with Actions

```lvt
<table lvt-source="users" lvt-columns="name:Name,role:Role" lvt-actions="delete:Delete,edit:Edit">
</table>
```

### Test 3: Simple Table with Empty State

```lvt
<table lvt-source="empty_source" lvt-columns="name:Name,email:Email" lvt-empty="No users found">
</table>
```

### Test 4: Simple Table with Auto-Discovery (No Columns)

```lvt
<table lvt-source="users">
</table>
```

---

## Rich Mode (lvt-datatable)

Rich mode uses the datatable component for sorting, pagination, and advanced features.

### Test 5: Rich Datatable with Columns

```lvt
<table lvt-source="users" lvt-columns="name:Name,email:Email" lvt-datatable>
</table>
```

### Test 6: Rich Datatable with Role Column

```lvt
<table lvt-source="users" lvt-columns="name:Name,role:Role" lvt-datatable>
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
