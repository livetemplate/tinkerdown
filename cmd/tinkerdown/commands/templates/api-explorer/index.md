---
title: "[[.Title]]"
sources:
  users:
    type: rest
    url: https://jsonplaceholder.typicode.com/users
  posts:
    type: rest
    url: https://jsonplaceholder.typicode.com/posts
    cache:
      ttl: 60s
---

# [[.Title]]

Explore REST APIs with live data refresh.

## Users

Data from [JSONPlaceholder](https://jsonplaceholder.typicode.com) - a free fake API for testing.

```lvt
<section lvt-source="users">
    <div class="toolbar">
        <h3>Users API</h3>
        <button lvt-click="Refresh" class="refresh-btn">Refresh</button>
    </div>

    {{if .Error}}
    <p class="error">Error: {{.Error}}</p>
    {{else}}
    <table class="data-table">
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
                <td>{{.Id}}</td>
                <td>{{.Name}}</td>
                <td><a href="mailto:{{.Email}}">{{.Email}}</a></td>
                <td>{{if .Company}}{{.Company.Name}}{{end}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
    <p class="count">{{len .Data}} users</p>
    {{end}}
</section>

<style>
.toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
}

.toolbar h3 {
    margin: 0;
}

.refresh-btn {
    padding: 0.5rem 1rem;
    background: #007bff;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
}

.refresh-btn:hover {
    background: #0056b3;
}

.data-table {
    width: 100%;
    border-collapse: collapse;
    margin-bottom: 0.5rem;
}

.data-table th,
.data-table td {
    padding: 0.75rem;
    text-align: left;
    border-bottom: 1px solid #dee2e6;
}

.data-table thead {
    background: #f8f9fa;
}

.data-table a {
    color: #007bff;
    text-decoration: none;
}

.data-table a:hover {
    text-decoration: underline;
}

.count {
    color: #666;
    font-size: 0.9em;
}

.error {
    color: #dc3545;
    background: #f8d7da;
    padding: 0.75rem;
    border-radius: 4px;
}
</style>
```

---

## Posts (Cached)

This source is cached for 60 seconds to reduce API calls.

```lvt
<section lvt-source="posts">
    <div class="toolbar">
        <h3>Posts API</h3>
        <button lvt-click="Refresh" class="refresh-btn">Refresh</button>
    </div>

    {{if .Error}}
    <p class="error">Error: {{.Error}}</p>
    {{else}}
    <div class="posts-grid">
        {{range .Data}}
        {{if le .Id 6}}
        <div class="post-card">
            <div class="post-id">#{{.Id}}</div>
            <h4 class="post-title">{{.Title}}</h4>
            <p class="post-body">{{.Body}}</p>
        </div>
        {{end}}
        {{end}}
    </div>
    <p class="count">Showing first 6 of {{len .Data}} posts</p>
    {{end}}
</section>

<style>
.posts-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 1rem;
    margin-bottom: 1rem;
}

.post-card {
    background: #f8f9fa;
    padding: 1rem;
    border-radius: 8px;
    position: relative;
}

.post-id {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    background: #007bff;
    color: white;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    font-size: 0.8em;
}

.post-title {
    margin: 0 0 0.5rem 0;
    font-size: 1rem;
    padding-right: 2rem;
}

.post-body {
    color: #666;
    font-size: 0.9em;
    margin: 0;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
}
</style>
```

---

## Configuration

Edit the frontmatter to connect to your own APIs:

```yaml
sources:
  myapi:
    type: rest
    url: https://api.example.com/data
    headers:
      Authorization: "Bearer ${API_TOKEN}"
    cache:
      ttl: 30s
```

Set environment variables for secrets:

```bash
export API_TOKEN="your-token-here"
tinkerdown serve
```
