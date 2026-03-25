# Computed Source

The `computed` source type derives data from another source by applying grouping, aggregation, and filtering operations. It's read-only and automatically refreshes when its parent source changes.

## Configuration

```yaml
sources:
  expenses:
    type: sqlite
    db: ./expenses.db
    table: expenses
  by_category:
    type: computed
    from: expenses
    group_by: category
    aggregate:
      total: sum(amount)
      count: count()
    filter: "status = active"  # optional
```

| Field | Required | Description |
|-------|----------|-------------|
| `from` | Yes | Name of the parent source to derive from |
| `group_by` | No | Field to group rows by. If omitted, produces a single aggregate row |
| `aggregate` | Yes | Map of output field name to aggregation expression |
| `filter` | No | Filter expression applied before grouping (e.g., `"status = active"`) |

## Aggregation Functions

| Function | Description | Example |
|----------|-------------|---------|
| `count()` | Count of rows | `count: count()` |
| `sum(field)` | Sum of numeric field | `total: sum(amount)` |
| `avg(field)` | Average of numeric field (skips nil/non-numeric) | `average: avg(amount)` |
| `min(field)` | Minimum value | `lowest: min(amount)` |
| `max(field)` | Maximum value | `highest: max(amount)` |

## Examples

### Category Breakdown

```yaml
sources:
  expenses:
    type: sqlite
    db: ./data.db
    table: expenses
    readonly: false
  by_category:
    type: computed
    from: expenses
    group_by: category
    aggregate:
      total: sum(amount)
      count: count()
```

Output rows:
```
[
  {"category": "Food", "total": 101.70, "count": 3},
  {"category": "Transport", "total": 72.00, "count": 2},
  {"category": "Housing", "total": 1500.00, "count": 1}
]
```

### Summary Statistics (No Group By)

```yaml
sources:
  summary:
    type: computed
    from: expenses
    aggregate:
      total: sum(amount)
      average: avg(amount)
      count: count()
      min: min(amount)
      max: max(amount)
```

Produces a single row with all aggregates.

### Filtered Aggregation

```yaml
sources:
  active_stats:
    type: computed
    from: tasks
    filter: "status = active"
    aggregate:
      count: count()
      total_points: sum(points)
```

Only aggregates rows where `status` equals `active` (case-insensitive).

## Filter Operators

Filters require the full `field operator value` form. Bare truthy checks (like `done` or `not done`) are not supported — use `done = true` instead.

| Operator | Description |
|----------|-------------|
| `=` | Equal (case-insensitive) |
| `!=` | Not equal (case-insensitive) |
| `<` | Less than (numeric) |
| `>` | Greater than (numeric) |
| `<=` | Less than or equal |
| `>=` | Greater than or equal |

## Behavior

- **Auto-refresh:** When the parent source is modified (add, update, delete), all computed sources that depend on it are automatically refreshed.
- **Deterministic ordering:** Group results are sorted alphabetically by group key. Aggregate columns are sorted alphabetically by output field name.
- **Nil handling:** `sum`, `avg`, `min`, `max` skip nil and non-numeric values. `avg` divides by the count of valid values, not total rows.
- **Read-only:** Computed sources cannot be written to. They always reflect the current state of their parent.

## Limitations

- **No chaining:** A computed source cannot reference another computed source as its parent. This is validated at startup.
- **No custom expressions:** Aggregation is limited to the five built-in functions. For complex transformations, use a WASM source.
- **Single parent:** Each computed source derives from exactly one parent source.
