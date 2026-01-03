# Expense Tracker Example

A simple expense tracking application demonstrating Tinkerdown's SQLite integration.

## Quick Start

```bash
cd examples/expense-tracker
tinkerdown serve
```

Open http://localhost:8080 to use the expense tracker.

## Features

- **Add Expenses**: Form with date, amount, category, and description fields
- **View Expenses**: Table showing all recorded expenses
- **Delete Expenses**: Remove expenses with a single click
- **Persistent Storage**: SQLite database (`expenses.db`) stores all data

## How It Works

### SQLite Source Configuration

```yaml
sources:
  expenses:
    type: sqlite
    db: "./expenses.db"
    table: expenses
    readonly: false  # Enable Add/Delete actions
```

### Auto-Created Schema

When you add your first expense, Tinkerdown automatically creates the `expenses` table with columns matching your form fields:

- `id` - Auto-increment primary key
- `date` - Date of expense
- `amount` - Expense amount (numeric)
- `category` - Category text
- `description` - Optional description
- `created_at` - Auto-filled timestamp

### Actions

The template uses three actions:

1. **Add** (`lvt-submit="Add"`) - Creates new expense records
2. **Delete** (`lvt-click="Delete"`) - Removes expense by ID
3. **Refresh** (`lvt-click="Refresh"`) - Reloads data from database

## Project Structure

```
expense-tracker/
  index.md       # Main application file
  README.md      # This file
  expenses.db    # SQLite database (auto-created)
```

## Extending This Example

### Add Categories Table

```yaml
sources:
  categories:
    type: sqlite
    db: "./expenses.db"
    table: categories
    readonly: true
```

### Add Operator Tracking

Use the `--operator` flag to track who added each expense:

```bash
tinkerdown serve --operator alice
```

Then in your form, use a hidden field with `{{.operator}}` to auto-fill the creator.

## Technical Notes

- Data is ordered by `created_at DESC` (newest first)
- Amounts are displayed as stored in SQLite (no additional formatting applied)
- The database file is created in the same directory as `index.md`
