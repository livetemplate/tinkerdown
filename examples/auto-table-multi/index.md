---
title: Project Dashboard
sources:
  tasks:
    type: sqlite
    db: ./dashboard.db
    table: tasks
    readonly: false
  team:
    type: rest
    from: https://jsonplaceholder.typicode.com/users
---

# Project Dashboard

Multiple data sources on one page. Tasks are writable (add/delete). Team is read-only from an API.

## Tasks
| Title | Status | Priority |
|-------|--------|----------|

## Team
| Name | Email |
|------|-------|
