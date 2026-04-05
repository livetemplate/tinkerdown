---
title: "Data from Files"
sources:
  users:
    type: json
    file: users.json
  products:
    type: csv
    file: products.csv
---

# File Data Sources

Test for `lvt-source` with JSON and CSV files.

## Users (JSON)

```lvt
<main lvt-source="users">
    <h3>Users from JSON</h3>
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Email</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>{{.Id}}</td>
                <td>{{.Name}}</td>
                <td>{{.Email}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}
    <button name="Refresh">Refresh</button>
</main>
```

## Products (CSV)

```lvt
<main lvt-source="products">
    <h3>Products from CSV</h3>
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Price</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>{{.Id}}</td>
                <td>{{.Name}}</td>
                <td>${{.Price}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}
    <button name="Refresh">Refresh</button>
</main>
```
