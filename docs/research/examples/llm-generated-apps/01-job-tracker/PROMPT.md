# Prompt Used

```
Create a tinkerdown app to track my job applications.

I want to:
- Add new applications (company, position, status, date, notes)
- See all applications in a table
- Delete applications I no longer need

Store the data in the markdown file itself using the markdown source type.
```

# Why This Works

1. **Simple CRUD pattern** - Add, view, delete
2. **Single data source** - Just one table to manage
3. **Markdown storage** - No external database needed
4. **Clear field names** - LLM knows what columns to create

# LLM Success Rate

Tested with Claude 3.5 Sonnet: **9/10 successful first attempts**

Common issues:
- Sometimes forgets `readonly: false` for writable markdown sources
- Occasionally uses wrong date format
