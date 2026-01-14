---
title: Daily Standup Bot

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
---

# Daily Standup Bot

This example demonstrates scheduled notifications with filter tokens.

## Schedule

The standup reminder runs at 9am on weekdays only:

Run action:slack-notify @daily:9am @weekdays
> :calendar: *Daily Standup Time!*
>
> Please share your updates:
> - What did you accomplish yesterday?
> - What are you working on today?
> - Any blockers or help needed?

## Questions

When the notification triggers, team members should answer:

1. **Yesterday** - What did you accomplish?
2. **Today** - What are you working on?
3. **Blockers** - Any impediments or help needed?

## How It Works

The `@daily:9am @weekdays` syntax combines:
- `@daily:9am` - Schedule token (fires every day at 9am)
- `@weekdays` - Filter token (restricts to Monday through Friday)

The blockquote content (lines starting with `>`) becomes the notification message sent to Slack.
