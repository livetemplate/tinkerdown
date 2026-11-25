# Debugging

Troubleshoot issues with your Tinkerdown apps.

## Server Log

`tinkerdown serve` writes its log to the terminal it runs in. Each line starts with where it comes from, such as `[WS]` for a page's connection and the blocks it sets up, `[API]` for source reads and writes, `[Config]` for config warnings and `[Schedule]` for scheduled actions. Source and action errors show there too. There is no separate debug level: the log is always on.

## Common Issues

### WebSocket Connection Failed

**Symptoms:** Page loads but interactive features don't work.

**Solutions:**

1. Check browser console for WebSocket errors
2. Ensure the server is running
3. Check for proxy/firewall blocking WebSocket connections
4. Verify the page path is correct

### Source Data Not Loading

**Symptoms:** Tables/lists are empty or show errors.

**Debug steps:**

1. Check the source configuration in `tinkerdown.yaml`
2. Verify the data source is accessible (database, API, file)
3. Read the fetch error in the [server log](#server-log).

4. Test the source directly:

```bash
# For SQLite
sqlite3 mydata.db "SELECT * FROM tasks LIMIT 5"

# For REST
curl https://api.example.com/data
```

### Template Errors

**Symptoms:** Page shows template error or blank content.

**Debug steps:**

1. Check for Go template syntax errors
2. Verify variable names match source data
3. Look for unclosed `{{if}}` or `{{range}}` blocks

Common mistakes:

```html
<!-- Wrong: missing dot -->
{{range tasks}}

<!-- Correct -->
{{range .tasks}}
```

### lvt-* Attributes Not Working

**Symptoms:** Clicks/submits don't trigger actions.

**Debug steps:**

1. Check browser console for JavaScript errors
2. Verify the attribute syntax is correct
3. Ensure the action name matches your handler

```html
<!-- Check these are correct -->
<button name="ActionName" data-id="123">
```

## Browser DevTools

### Network Tab

- Check WebSocket connection status
- View WebSocket messages (filter by WS)

### Console Tab

- Look for JavaScript errors
- View client-side logs

### Elements Tab

- Inspect generated HTML
- Check if `lvt-*` attributes are present

## Validation

Catch errors before runtime:

```bash
tinkerdown validate ./myapp
```

This checks:

- Markdown syntax
- Source references
- Configuration validity

## Logging

### Server Logs

The log is plain text, one line per event, as described in [Server Log](#server-log). To keep it, redirect it: `tinkerdown serve > serve.log 2>&1`.

## Performance Issues

### Slow Page Load

1. Enable caching for data sources
2. Check network latency to external APIs
3. Watch the [server log](#server-log) for slow or failing source fetches

### High Memory Usage

1. Check for large data sources
2. Limit result sets with queries
3. Use pagination for large datasets

## Getting Help

If you're still stuck:

1. Check the [GitHub Issues](https://github.com/livetemplate/tinkerdown/issues)
2. Include the server log when reporting issues
3. Provide a minimal reproduction case

## Next Steps

- [Error Handling](../error-handling.md) - Error recovery configuration
- [Caching](../caching.md) - Performance optimization
