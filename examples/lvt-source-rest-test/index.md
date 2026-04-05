---
title: "Users from REST API"
sources:
  users:
    type: rest
    from: https://jsonplaceholder.typicode.com/users
---

# Users List

Test for `lvt-source` attribute that fetches data from a REST API.

```lvt
<main lvt-source="users">
    <h2>Users from JSONPlaceholder API</h2>

    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Email</th>
                <th>Company</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr data-user-id="{{.Id}}">
                <td>{{.Id}}</td>
                <td>{{.Name}}</td>
                <td>{{.Email}}</td>
                <td>{{if .Company}}{{.Company.Name}}{{end}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}

    <button name="Refresh">Refresh Data</button>
</main>
```
