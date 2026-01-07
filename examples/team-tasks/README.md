# Team Tasks

A multi-user collaborative task board demonstrating real-time synchronization, tabbed filtering, computed expressions, and custom actions.

## Quick Start

```bash
cd examples/team-tasks
tinkerdown serve
```

Open http://localhost:8080 in your browser.

## Features

### Tabbed Filtering

Switch between different views using the tab bar:

- **All**: Shows all tasks
- **Mine**: Shows tasks assigned to the current operator
- **Todo**: Shows tasks with status "todo"
- **In Progress**: Shows tasks with status "in_progress"
- **Done**: Shows completed tasks

### Live Statistics

The dashboard displays real-time counts that update automatically:

- Total task count
- Todo count
- In Progress count
- Done count

### Task Management

- **Add Task**: Fill out the form with title, assignee, priority, and status
- **Delete Task**: Remove individual tasks with the delete button
- **Bulk Clear**: Clear all completed tasks with confirmation

### Multi-User Collaboration

Test real-time synchronization by opening multiple browser windows:

```bash
# User 1 (default operator)
tinkerdown serve

# User 2 (different port, different operator)
tinkerdown serve --port 8081 --operator bob
```

Changes made in one window appear instantly in others via WebSocket.

## Configuration

### Database

Uses SQLite (`tasks.db`) with **automatic table creation** on first use. No manual schema setup is required - the table is created automatically when you add your first task.

The inferred schema includes:

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER | Auto-generated primary key |
| title | TEXT | Task description (max 200 chars) |
| assigned_to | TEXT | Username of assignee (max 50 chars, alphanumeric) |
| priority | TEXT | low, medium, or high |
| status | TEXT | todo, in_progress, or done |

### Custom Actions

Defined in frontmatter:

- `clear-done`: Deletes all tasks with status "done"
- `mark-mine-done`: Marks all tasks assigned to current operator as done

### Operator Parameter

The `:operator` parameter in SQL actions is automatically populated from the `--operator` CLI flag. This enables user-specific filtering and actions:

```bash
# Start server with operator identity
tinkerdown serve --operator alice

# The "Mine" tab and "Mark My Tasks Done" action will filter by assigned_to = "alice"
```

## Requirements

- tinkerdown v0.3.0 or later
- Milestone 3 features (tabs, computed expressions, SQL actions)
