---
title: "Expense Tracker"
sidebar: false
sources:
  expenses:
    type: sqlite
    db: "./expenses.db"
    table: expenses
    readonly: false
---

# Expense Tracker

A simple expense tracking app using SQLite for persistent storage.

## Expenses

```lvt
<div lvt-source="expenses">
    <!-- Add Expense Form -->
    <form lvt-submit="Add" style="display: flex; flex-wrap: wrap; gap: 12px; align-items: flex-end; padding: 16px; background: #f8f9fa; border-radius: 8px; margin-bottom: 24px;">
        <div style="flex: 1; min-width: 120px;">
            <label for="date" style="display: block; font-size: 12px; color: #666; margin-bottom: 4px;">Date</label>
            <input type="date" name="date" id="date" required
                   style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px;">
        </div>
        <div style="flex: 1; min-width: 100px;">
            <label for="amount" style="display: block; font-size: 12px; color: #666; margin-bottom: 4px;">Amount ($)</label>
            <input type="number" name="amount" id="amount" step="0.01" min="0" required placeholder="0.00"
                   style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px;">
        </div>
        <div style="flex: 1; min-width: 120px;">
            <label for="category" style="display: block; font-size: 12px; color: #666; margin-bottom: 4px;">Category</label>
            <select name="category" id="category" required
                    style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px;">
                <option value="Food">Food</option>
                <option value="Transport">Transport</option>
                <option value="Entertainment">Entertainment</option>
                <option value="Utilities">Utilities</option>
                <option value="Shopping">Shopping</option>
                <option value="Other">Other</option>
            </select>
        </div>
        <div style="flex: 2; min-width: 200px;">
            <label for="description" style="display: block; font-size: 12px; color: #666; margin-bottom: 4px;">Description</label>
            <input type="text" name="description" id="description" placeholder="What did you spend on?"
                   style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px;">
        </div>
        <button type="submit"
                style="padding: 8px 20px; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer; font-weight: 500;">
            + Add Expense
        </button>
    </form>

    <!-- Expenses Table -->
    {{if .Error}}
    <p style="color: red; padding: 12px; background: #fee; border-radius: 4px;">Error: {{.Error}}</p>
    {{else}}
    <table style="width: 100%; border-collapse: collapse;">
        <thead>
            <tr style="background: #f8f9fa; border-bottom: 2px solid #ddd;">
                <th style="padding: 12px; text-align: left;">Date</th>
                <th style="padding: 12px; text-align: left;">Category</th>
                <th style="padding: 12px; text-align: right;">Amount</th>
                <th style="padding: 12px; text-align: left;">Description</th>
                <th style="padding: 12px; text-align: center; width: 80px;">Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Data}}
            <tr style="border-bottom: 1px solid #eee;">
                <td style="padding: 12px;">{{.Date}}</td>
                <td style="padding: 12px;">
                    <span style="padding: 2px 8px; background: #e9ecef; border-radius: 12px; font-size: 12px;">
                        {{.Category}}
                    </span>
                </td>
                <td style="padding: 12px; text-align: right; font-family: monospace; font-weight: 500;">
                    ${{.Amount}}
                </td>
                <td style="padding: 12px; color: #666;">{{.Description}}</td>
                <td style="padding: 12px; text-align: center;">
                    <button lvt-click="Delete" lvt-data-id="{{.Id}}"
                            style="padding: 4px 8px; background: transparent; color: #dc3545; border: 1px solid #dc3545; border-radius: 4px; cursor: pointer; font-size: 12px;">
                        Delete
                    </button>
                </td>
            </tr>
            {{else}}
            <tr>
                <td colspan="5" style="padding: 24px; text-align: center; color: #666;">
                    No expenses yet. Add one above!
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
    <div style="margin-top: 16px; text-align: right;">
        <button lvt-click="Refresh" style="padding: 8px 16px; background: #6c757d; color: white; border: none; border-radius: 4px; cursor: pointer;">
            Refresh
        </button>
    </div>
    {{end}}
</div>
```

## Features

This example demonstrates:

- **SQLite Source**: Persistent data storage in a local database file
- **Add Action**: Form submission creates new expense records
- **Delete Action**: Remove expenses with a single click
- **Refresh Action**: Reload data from the database

## Getting Started

```bash
cd examples/expense-tracker
tinkerdown serve
```

The database (`expenses.db`) is created automatically when you add your first expense.
