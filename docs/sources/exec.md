# Exec Source

Execute shell commands and capture output.

## Configuration

```yaml
sources:
  system_info:
    type: exec
    command: uname -a
```

## Options

| Option | Required | Description |
|--------|----------|-------------|
| `type` | Yes | Must be `exec` |
| `command` | Yes | Shell command to execute |
| `shell` | No | Shell to use (default: /bin/sh) |
| `timeout` | No | Command timeout (default: 10s) |

## Examples

### Basic Command

```yaml
sources:
  uptime:
    type: exec
    command: uptime
```

### Command with Arguments

```yaml
sources:
  disk_usage:
    type: exec
    command: df -h /
```

### Command Producing JSON

```yaml
sources:
  docker_containers:
    type: exec
    command: docker ps --format '{"id":"{{.ID}}","name":"{{.Names}}","status":"{{.Status}}"}'
```

### Complex Pipeline

```yaml
sources:
  recent_logs:
    type: exec
    command: tail -n 100 /var/log/app.log | grep ERROR
```

## Output Handling

### Plain Text

If the output is plain text, it's available as a single string:

```html
<pre>{{.system_info}}</pre>
```

### JSON Output

If the command outputs JSON, it's automatically parsed:

```yaml
sources:
  processes:
    type: exec
    command: ps aux --no-headers | awk '{print "{\"user\":\""$1"\",\"pid\":\""$2"\",\"cpu\":\""$3"\",\"mem\":\""$4"\",\"command\":\""$11"\"}"}'
```

```html
<table lvt-source="processes" lvt-columns="user,pid,cpu,mem">
</table>
```

## Security Considerations

- Commands run with the same permissions as the Tinkerdown server
- Be careful with user input—avoid command injection
- Consider sandboxing for production use

## Environment Variables

Environment variables are inherited from the server process:

```yaml
sources:
  custom_script:
    type: exec
    command: ./scripts/fetch-data.sh
```

Access in script via standard environment variables.

## Use Cases

### System Monitoring

```yaml
sources:
  memory:
    type: exec
    command: free -m | grep Mem | awk '{print "{\"total\":\"" $2 "\",\"used\":\"" $3 "\",\"free\":\"" $4 "\"}"}'

  cpu:
    type: exec
    command: top -bn1 | grep "Cpu(s)" | awk '{print "{\"user\":\"" $2 "\",\"system\":\"" $4 "\",\"idle\":\"" $8 "\"}"}'
```

### Git Information

```yaml
sources:
  git_log:
    type: exec
    command: git log --oneline -10

  git_status:
    type: exec
    command: git status --porcelain
```

### Custom CLI Tools

```yaml
sources:
  custom_data:
    type: exec
    command: ./my-cli-tool --format json
```

## Caching

Cache command output to reduce execution:

```yaml
sources:
  expensive_command:
    type: exec
    command: ./slow-script.sh
    cache:
      ttl: 5m
```

## Full Example

```yaml
# tinkerdown.yaml
sources:
  system:
    type: exec
    command: |
      echo '{"hostname":"'$(hostname)'","uptime":"'$(uptime -p)'","kernel":"'$(uname -r)'"}'
    cache:
      ttl: 1m
```

```html
<!-- index.md -->
<h2>System Information</h2>
<dl>
  <dt>Hostname</dt>
  <dd>{{.system.hostname}}</dd>
  <dt>Uptime</dt>
  <dd>{{.system.uptime}}</dd>
  <dt>Kernel</dt>
  <dd>{{.system.kernel}}</dd>
</dl>
```

## Next Steps

- [JSON Source](json.md) - Static JSON data
- [Data Sources Guide](../guides/data-sources.md) - Overview
