# Source Caching

Tinkerdown supports in-memory caching for data sources to reduce API calls and improve performance.

## Configuration

Add a `cache` section to any source in `tinkerdown.yaml`:

```yaml
sources:
  users:
    type: rest
    url: https://api.example.com/users
    cache:
      ttl: 5m
      strategy: simple
```

### Cache Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ttl` | duration | (disabled) | How long to cache data (e.g., "30s", "5m", "1h") |
| `strategy` | string | "simple" | Cache strategy: "simple" or "stale-while-revalidate" |

## Cache Strategies

### Simple (Default)

With `strategy: simple`, data is cached for the TTL duration. When the TTL expires, the next request fetches fresh data.

```yaml
sources:
  products:
    type: rest
    url: https://api.example.com/products
    cache:
      ttl: 5m
      strategy: simple
```

**Behavior:**
- First request fetches from source and caches result
- Subsequent requests within TTL return cached data
- After TTL expires, next request fetches fresh data (may be slow)

### Stale-While-Revalidate

With `strategy: stale-while-revalidate`, stale data is returned immediately while fresh data is fetched in the background.

```yaml
sources:
  metrics:
    type: rest
    url: https://api.example.com/metrics
    cache:
      ttl: 1m
      strategy: stale-while-revalidate
```

**Behavior:**
- First request fetches from source and caches result
- Data becomes "stale" after half the TTL (30s in the example)
- When stale data is requested:
  - Returns stale data immediately (fast response)
  - Triggers background fetch of fresh data
  - Next request gets fresh data
- After full TTL expires, cache entry is removed

**Best for:** Dashboards, metrics, frequently-accessed data where slightly stale data is acceptable.

## Cache Invalidation

### Automatic Invalidation on Write

For writable sources (sqlite, markdown with `readonly: false`), the cache is automatically invalidated when data is modified through Add, Update, Delete, or Toggle actions.

```yaml
sources:
  tasks:
    type: sqlite
    db: ./app.db
    table: tasks
    readonly: false
    cache:
      ttl: 10m
```

When a user adds, updates, or deletes a task, the cache is automatically cleared so the next fetch returns fresh data.

### Manual Cache Refresh

Users can manually refresh cached data using the `Refresh` action:

```html
<button lvt-click="Refresh" lvt-data-source="users">Refresh Users</button>
```

This invalidates the cache for the specified source and triggers a fresh fetch.

## When to Use Caching

### Good Use Cases

- **REST APIs with rate limits**: Reduce API calls
- **Slow database queries**: Cache complex aggregations
- **External services**: Reduce latency for third-party APIs
- **Read-heavy data**: Data that's read frequently but changes rarely

### Not Recommended

- **Real-time data**: Stock prices, live metrics (use short TTL if needed)
- **User-specific data**: Unless scoped per user
- **Frequently changing data**: Very short TTL may not help

## Examples

### Dashboard with REST API

```yaml
sources:
  sales:
    type: rest
    url: https://api.example.com/sales/summary
    cache:
      ttl: 5m
      strategy: stale-while-revalidate

  inventory:
    type: rest
    url: https://api.example.com/inventory
    cache:
      ttl: 1m
```

### SQLite with Caching

```yaml
sources:
  reports:
    type: sqlite
    db: ./analytics.db
    table: monthly_reports
    cache:
      ttl: 1h
```

### PostgreSQL with Caching

```yaml
sources:
  user_stats:
    type: pg
    query: "SELECT * FROM user_statistics"
    cache:
      ttl: 15m
      strategy: stale-while-revalidate
```

## Performance Tips

1. **Start with longer TTLs**: Begin with 5-10 minute TTLs and reduce if needed
2. **Use stale-while-revalidate for dashboards**: Better UX with instant responses
3. **Monitor cache effectiveness**: Check if data is changing faster than your TTL
4. **Consider data freshness requirements**: Don't cache data that must be real-time

## Limitations

- Cache is in-memory and cleared on server restart
- No distributed caching (single server only)
- Cache size is unbounded (be careful with very large datasets)
- Background revalidation requires a running server
