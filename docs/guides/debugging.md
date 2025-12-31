# Debugging

Troubleshoot issues with your Tinkerdown apps.

## Debug Mode

Enable debug logging:

```bash
tinkerdown serve --debug
```

This shows:

- WebSocket messages
- Source fetch operations
- Template rendering times
- State changes

## Verbose Mode

For even more detail:

```bash
tinkerdown serve --verbose
```

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
3. Use debug mode to see fetch errors:

```bash
tinkerdown serve --debug
```

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
<button lvt-click="ActionName" lvt-data-id="123">
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

```bash
# JSON format for log aggregation
tinkerdown serve --log-format=json

# Include request correlation IDs
tinkerdown serve --debug --correlation-ids
```

### Sample Log Output

```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "debug",
  "msg": "source fetch",
  "source": "tasks",
  "duration_ms": 12,
  "rows": 5
}
```

## Performance Issues

### Slow Page Load

1. Enable caching for data sources
2. Check network latency to external APIs
3. Profile source fetch times with debug mode

### High Memory Usage

1. Check for large data sources
2. Limit result sets with queries
3. Use pagination for large datasets

## Getting Help

If you're still stuck:

1. Check the [GitHub Issues](https://github.com/livetemplate/tinkerdown/issues)
2. Include debug output when reporting issues
3. Provide a minimal reproduction case

## Next Steps

- [Error Handling](../error-handling.md) - Error recovery configuration
- [Caching](../caching.md) - Performance optimization
