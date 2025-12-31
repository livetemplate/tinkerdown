# Prompt Used

```
Create a tinkerdown app to track my expenses.

Each expense has: date, description, amount, category.

Categories should be a dropdown populated from a JSON file:
- Food & Drink
- Transportation
- Software & Subscriptions
- Office Supplies
- Entertainment
- Other

Store expenses in the markdown file itself.
Show expenses in a table with a delete button.
```

# Why This Works

1. **Two data sources** - Markdown (writable) + JSON (read-only categories)
2. **Dropdown from JSON** - Categories populated automatically
3. **Standard CRUD** - Add and delete operations
4. **Typed inputs** - Date picker, number input

# LLM Success Rate

Tested with Claude 3.5 Sonnet: **8/10 successful first attempts**

Common issues:
- Forgets to create the categories.json file
- Sometimes puts categories inline instead of separate file
- May use wrong lvt-value/lvt-label for select

# Multiple Sources Pattern

This example demonstrates combining:
- **Writable markdown source** for user data
- **Read-only JSON source** for reference data (categories)

This pattern is useful for:
- Dropdown options
- Status values
- User lists
- Any lookup table
