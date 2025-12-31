# Prompt Used

```
Create a tinkerdown app that shows my system status:
- Disk usage (filesystem, size, used, available, percentage)
- Memory usage (total, used, free)
- Top 5 processes by CPU

Use shell commands via the exec source type.
Output JSON from each command so tinkerdown can parse it.
```

# Why This Works

1. **Exec source** - Runs any shell command
2. **JSON output** - awk/jq convert command output to JSON
3. **Read-only** - Just displays data
4. **Universal commands** - df, free, ps work on most Unix systems

# LLM Success Rate

Tested with Claude 3.5 Sonnet: **7/10 successful first attempts**

Common issues:
- awk syntax errors (quoting is tricky)
- macOS vs Linux command differences
- Missing jq on some systems

# Tips for Better Prompts

Include in your prompt:
- "Output valid JSON from each command"
- "Handle both macOS and Linux"
- "Use jq -s to combine multiple JSON objects into an array"
