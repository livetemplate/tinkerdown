# Prompt Used

```
Create a tinkerdown app for collecting anonymous team feedback.

The form should have:
- Mood (dropdown: great, good, okay, struggling)
- What's going well (text area)
- What could be better (text area)
- Any blockers (text input)

Store submissions in a SQLite database.
Show all submissions in a table below the form.
```

# Why This Works

1. **Form → Database pattern** - Clear data flow
2. **SQLite auto-creates table** - No schema setup needed
3. **Simple field types** - Text and select only
4. **Read-after-write** - Table shows new submissions

# LLM Success Rate

Tested with Claude 3.5 Sonnet: **8/10 successful first attempts**

Common issues:
- Sometimes uses wrong attribute (`lvt-action` instead of `lvt-submit`)
- May forget to add the source reference on form
- Field name mismatches between form and table

# Note

The SQLite database file is created automatically on first submission.
No manual setup required.
