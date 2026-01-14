# Standup Bot Example

Demonstrates scheduled notifications with Slack integration using filter tokens.

## Features

- **Schedule tokens**: `@daily:9am` triggers at 9am every day
- **Filter tokens**: `@weekdays` restricts execution to Monday-Friday
- **Blockquote messages**: Content after `>` becomes the notification body
- **Action-based delivery**: Uses HTTP action to post to Slack webhook

## Setup

### 1. Create a Slack Incoming Webhook

1. Go to [Slack Apps](https://api.slack.com/apps)
2. Create a new app or use an existing one
3. Enable "Incoming Webhooks"
4. Add a webhook to your workspace
5. Copy the webhook URL

### 2. Set Environment Variable

```bash
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
```

### 3. Run the Server

```bash
cd examples/standup-bot
tinkerdown serve
```

## Schedule Syntax

| Token | Description | Example |
|-------|-------------|---------|
| `@daily:9am` | Every day at 9am | `Run action:notify @daily:9am` |
| `@weekdays` | Monday through Friday filter | `@daily:9am @weekdays` |
| `@weekends` | Saturday and Sunday filter | `@daily:10am @weekends` |
| `@weekly:mon,wed:9am` | Specific days at specific time | Fires Mon & Wed at 9am |

## Message Syntax

Blockquote lines following an imperative become the message body:

```markdown
Run action:slack-notify @daily:9am @weekdays
> Line 1 of the message
> Line 2 of the message
```

This sends a message with:
```
Line 1 of the message
Line 2 of the message
```

## Action Configuration

The Slack action uses the HTTP action type:

```yaml
actions:
  slack-notify:
    type: http
    url: ${SLACK_WEBHOOK_URL}
    method: POST
    headers:
      Content-Type: application/json
    body: |
      {
        "text": "{{.Message}}",
        "channel": "#team-standup"
      }
```

The `{{.Message}}` template variable is replaced with the blockquote content.

## Customization

### Different Schedule

For a 10am standup:
```markdown
Run action:slack-notify @daily:10am @weekdays
```

### Weekend Notifications

```markdown
Run action:slack-notify @daily:10am @weekends
> Weekend check-in time!
```

### Multiple Days

For Mon/Wed/Fri standups:
```markdown
Run action:slack-notify @weekly:mon,wed,fri:9am
> Standup time!
```
