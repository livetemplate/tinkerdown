---
title: "<<.Title>>"
sources:
  pods:
    type: exec
    cmd: ./get-pods.sh
---

# <<.Title>>

A simple Kubernetes pods dashboard built with Tinkerdown.

## Pods

```lvt
<main lvt-source="pods">
    {{if .Error}}
    <p><mark>Error: {{.Error}}</mark></p>
    <p>Make sure kubectl is configured and accessible.</p>
    {{else if not .Data}}
    <p>No pods found. Run <code>kubectl get pods</code> to verify your cluster connection.</p>
    {{else}}
    <table>
        <thead>
            <tr>
                <th>Name</th>
                <th>Namespace</th>
                <th>Status</th>
                <th>Ready</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr>
                <td>{{.name}}</td>
                <td>{{.namespace}}</td>
                <td>{{.status}}</td>
                <td>{{if .ready}}✓{{else}}✗{{end}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    {{end}}

    <button name="Refresh" style="margin-top: 16px;">Refresh</button>
</main>
```

## Customizing

Edit `get-pods.sh` to change what data is displayed. For example:

- Show deployments: `kubectl get deployments -o json`
- Filter by namespace: `kubectl get pods -n my-namespace -o json`
- Show services: `kubectl get services -o json`
