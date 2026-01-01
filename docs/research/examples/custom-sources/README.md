# Custom Sources Examples

> ⚠️ **SECURITY WARNING**: These examples execute arbitrary code on your system.
> Custom sources run without sandboxing. Before using in production:
> - Validate all inputs to prevent command injection
> - Run sources with minimal required permissions
> - Never pass untrusted user input directly to shell commands
> - Consider containerizing sources for isolation

Example policy-encoded sources for tinkerdown in multiple languages.

## The Contract

Every custom source follows the same interface:

```
INPUT:  JSON on stdin
OUTPUT: JSON on stdout
EXIT:   0 = success, non-zero = error
```

### Input Format
```json
{
  "query": { ... },      // Parameters from markdown config
  "env": {               // Execution context
    "operator": "alice",
    "incident_id": "..."
  }
}
```

### Output Format
```json
{
  "columns": ["col1", "col2"],
  "rows": [
    {"col1": "value1", "col2": "value2"}
  ]
}
```

## Examples

| Language | File | Purpose |
|----------|------|---------|
| Python | `python/check-permission.py` | Check operator permissions |
| Shell | `shell/system-health.sh` | System disk/memory/load check |
| Node.js | `node/api-status.js` | Check API endpoint health |
| Go | `go/change-freeze.go` | Check if in change freeze period |

## Usage in Markdown

```yaml
sources:
  perm:
    type: exec
    command: "./sources/check-permission.py"
    query:
      permission: "prod-db-write"
```

```html
<div lvt-if="perm[0].allowed">✅ Access granted</div>
<div lvt-if="!perm[0].allowed">❌ Access denied</div>
```

## Testing Sources

```bash
# Test a source directly
echo '{"query":{"permission":"prod-db-write"},"env":{"operator":"alice"}}' | python3 python/check-permission.py

# Test shell source (no input needed)
echo '{}' | bash shell/system-health.sh

# Build and test Go source
cd go && go build -o change-freeze change-freeze.go
echo '{}' | ./change-freeze
```

## Writing Your Own

1. Pick your language
2. Read JSON from stdin
3. Do your thing (API calls, file reads, policy checks)
4. Write JSON to stdout
5. Exit 0 on success, non-zero on error

That's it. 20 lines of code to encode organizational policy.
