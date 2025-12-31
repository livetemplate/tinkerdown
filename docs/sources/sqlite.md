# SQLite Source

Query SQLite databases for your Tinkerdown apps.

## Configuration

```yaml
sources:
  tasks:
    type: sqlite
    path: ./data.db
    query: SELECT * FROM tasks
```

## Options

| Option | Required | Description |
|--------|----------|-------------|
| `type` | Yes | Must be `sqlite` |
| `path` | Yes | Path to SQLite database file |
| `query` | Yes | SQL query to execute |

## Examples

### Basic Query

```yaml
sources:
  tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks
```

### Filtered Query

```yaml
sources:
  active_tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks WHERE status = 'active'
```

### Ordered Query

```yaml
sources:
  recent_tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks ORDER BY created_at DESC LIMIT 10
```

### Join Query

```yaml
sources:
  tasks_with_users:
    type: sqlite
    path: ./app.db
    query: |
      SELECT t.*, u.name as user_name
      FROM tasks t
      JOIN users u ON t.user_id = u.id
```

## Write Operations

SQLite sources support write operations through actions:

```html
<!-- Insert -->
<form lvt-submit="AddTask">
  <input name="title" placeholder="Task title">
  <button type="submit">Add</button>
</form>

<!-- Update -->
<button lvt-click="ToggleTask" lvt-data-id="{{.id}}">
  Toggle
</button>

<!-- Delete -->
<button lvt-click="DeleteTask" lvt-data-id="{{.id}}">
  Delete
</button>
```

## Database Schema

SQLite sources work with any schema. Example:

```sql
CREATE TABLE tasks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  status TEXT DEFAULT 'pending',
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Caching

Enable caching for read-heavy workloads:

```yaml
sources:
  tasks:
    type: sqlite
    path: ./tasks.db
    query: SELECT * FROM tasks
    cache:
      ttl: 1m
      strategy: simple
```

**Note:** Cache is automatically invalidated on write operations.

## Security

- SQL queries are parameterized to prevent SQL injection
- File paths are validated to prevent path traversal

## Limitations

- SQLite is single-writer; consider PostgreSQL for high-concurrency apps
- Database must be on local filesystem (not remote)

## Full Example

```yaml
# tinkerdown.yaml
sources:
  tasks:
    type: sqlite
    path: ./data/tasks.db
    query: SELECT * FROM tasks ORDER BY created_at DESC

  categories:
    type: sqlite
    path: ./data/tasks.db
    query: SELECT DISTINCT category FROM tasks
```

```html
<!-- index.md -->
<h2>Tasks</h2>
<table lvt-source="tasks" lvt-columns="title,status,category" lvt-actions="Delete">
</table>

<h2>Add Task</h2>
<form lvt-submit="AddTask">
  <input name="title" placeholder="Title">
  <select name="category" lvt-source="categories" lvt-value="category" lvt-label="category">
  </select>
  <button type="submit">Add</button>
</form>
```

## Next Steps

- [REST Source](rest.md) - External API integration
- [Data Sources Guide](../guides/data-sources.md) - Overview of all sources
