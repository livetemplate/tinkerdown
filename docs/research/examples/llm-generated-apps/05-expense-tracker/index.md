---
title: Expense Tracker
sources:
  expenses:
    type: markdown
    anchor: "#expenses"
    readonly: false
  categories:
    type: json
    file: categories.json
---

# Expense Tracker

Track your expenses. Data is stored in this file.

## Add Expense

```lvt
<form lvt-submit="add" lvt-source="expenses">
  <input name="date" type="date" required>
  <input name="description" placeholder="What did you buy?" required>
  <input name="amount" type="number" step="0.01" placeholder="Amount" required>
  <select lvt-source="categories" lvt-value="id" lvt-label="name" name="category">
  </select>
  <button type="submit">Add Expense</button>
</form>
```

## Recent Expenses

```lvt
<table lvt-source="expenses" lvt-columns="date:Date,description:Description,amount:Amount,category:Category" lvt-actions="delete:×" lvt-empty="No expenses yet">
</table>
```

---

## Expenses {#expenses}

| date | description | amount | category |
|------|-------------|--------|----------|
| 2024-01-15 | Coffee | 4.50 | food | <!-- id:exp_001 -->
| 2024-01-15 | Uber to airport | 35.00 | transport | <!-- id:exp_002 -->
| 2024-01-14 | Monthly subscription | 9.99 | software | <!-- id:exp_003 -->
