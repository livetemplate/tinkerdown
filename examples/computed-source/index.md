---
title: Budget Dashboard
sources:
  expenses:
    type: sqlite
    db: ./budget.db
    table: expenses
    readonly: false
  by-category:
    type: computed
    from: expenses
    group_by: category
    aggregate:
      total: sum(amount)
      count: count()
  summary:
    type: computed
    from: expenses
    aggregate:
      total: sum(amount)
      average: avg(amount)
      count: count()
---

# Budget Dashboard

Expenses are tracked in SQLite. The category breakdown and summary are computed sources — derived automatically from the expenses data.

## Expenses
| Description | Category | Amount |
|-------------|----------|--------|

## By Category
| Category | Total | Count |
|----------|-------|-------|

## Summary
| Total | Average | Count |
|-------|---------|-------|
