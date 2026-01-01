---
title: "{{.Title}}"
sources:
  users:
    type: rest
    url: https://jsonplaceholder.typicode.com/users
  system:
    type: exec
    cmd: ./system-info.sh
---

# {{.Title}}

A multi-source dashboard combining REST API data and local system information.

## API Users

```lvt
<main lvt-source="users">
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
            <tr>
                <td>{{.id}}</td>
                <td>{{.name}}</td>
                <td>{{.email}}</td>
                <td>{{if .company}}{{.company.name}}{{end}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}
    <button lvt-click="Refresh">Refresh Users</button>
</main>
```

## System Disk Usage

```lvt
<main lvt-source="system">
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>Filesystem</th>
                <th>Size</th>
                <th>Used</th>
                <th>Available</th>
                <th>Use %</th>
                <th>Mount</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>{{.filesystem}}</td>
                <td>{{.size}}</td>
                <td>{{.used}}</td>
                <td>{{.available}}</td>
                <td>{{.use_percent}}</td>
                <td>{{.mount}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}
    <button lvt-click="Refresh">Refresh System Info</button>
</main>
```

## About This Dashboard

This demonstrates combining multiple data sources:
- **REST API**: Users from JSONPlaceholder
- **Exec**: Local disk usage via `df -h`
