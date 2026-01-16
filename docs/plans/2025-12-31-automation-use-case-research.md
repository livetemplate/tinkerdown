# Tinkerdown for Automation: Bots & Scheduled Tasks

**Date:** 2025-12-31
**Status:** Research
**Goal:** Explore automation as the third major use case for tinkerdown

---

## Table of Contents

1. [The Opportunity](#the-opportunity)
2. [Automation Patterns](#automation-patterns)
3. [Architecture](#architecture)
4. [Bot Catalog](#bot-catalog)
5. [LLM Generation Strategy](#llm-generation-strategy)
6. [Required Features](#required-features)
7. [Comparison with Alternatives](#comparison-with-alternatives)
8. [Implementation Plan](#implementation-plan)

---

## The Opportunity

### The Problem

Teams need simple automations constantly:

```
"Send a standup reminder every morning at 9am"
"When a GitHub issue is opened, add it to our tracker"
"Check if the API is healthy every 5 minutes"
"Process expense receipts dropped in a folder"
"Generate a weekly report from our data"
```

**Current solutions fail:**

| Solution | Problem |
|----------|---------|
| **Zapier/Make** | $20+/mo, vendor lock-in, limited logic |
| **n8n/Temporal** | Complex setup, heavyweight |
| **Cron + scripts** | Scattered, no state management, hard to maintain |
| **Custom code** | Takes days to build, overkill for simple tasks |
| **GitHub Actions** | Only for repo events, YAML complexity |

### The Tinkerdown Solution

**Markdown defines the automation. Same file = logic + state + history.**

```
┌─────────────────────────────────────────────────────────────────┐
│                    TINKERDOWN AUTOMATION                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                     standup-bot.md                       │   │
│  │                                                          │   │
│  │  triggers:                                               │   │
│  │    - schedule: "0 9 * * 1-5"    # 9am weekdays          │   │
│  │    - schedule: "0 10 * * 1-5"   # 10am weekdays         │   │
│  │                                                          │   │
│  │  sources:                                                │   │
│  │    team: ...                                             │   │
│  │    updates: ...                                          │   │
│  │                                                          │   │
│  │  on:                                                     │   │
│  │    9am: send_reminder                                    │   │
│  │    10am: send_summary                                    │   │
│  │                                                          │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│                    tinkerdown serve standup-bot.md              │
│                              │                                  │
│                              ▼                                  │
│                   Serves UI + runs triggers in background       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Why Tinkerdown for Automation?

| Value | How Tinkerdown Delivers |
|-------|-------------------------|
| **LLM-friendly** | Markdown is easy to generate |
| **Self-documenting** | Bot logic readable by humans |
| **Stateful** | State tracked in same file |
| **Auditable** | Git history shows all changes |
| **Composable** | Reuse sources from other apps |
| **Free** | No per-execution pricing |
| **Local-first** | Run on your machine or server |

---

## Automation Patterns

### Pattern 1: Scheduled Task (Cron-like)

**When:** Run actions on a time schedule.

```yaml
triggers:
  - schedule: "0 9 * * 1-5"     # 9am Mon-Fri
  - schedule: "0 0 * * 0"       # Midnight Sunday
  - schedule: "*/5 * * * *"     # Every 5 minutes
```

**Examples:**
- Daily standup reminder
- Hourly health check
- Weekly report generation
- Periodic data cleanup

---

### Pattern 2: Webhook Receiver

**When:** React to external events via HTTP.

```yaml
triggers:
  - webhook:
      path: /github
      secret: ${GITHUB_WEBHOOK_SECRET}
  - webhook:
      path: /stripe
      secret: ${STRIPE_WEBHOOK_SECRET}
```

**Examples:**
- GitHub issue → add to tracker
- Payment webhook → update records
- Alert webhook → start runbook
- Chat command → execute action

---

### Pattern 3: File Watcher

**When:** React to file system changes.

```yaml
triggers:
  - watch:
      path: ./inbox/*.pdf
      events: [create]
  - watch:
      path: ./data/*.csv
      events: [create, modify]
```

**Examples:**
- New receipt → extract and add to expenses
- New CSV → import into database
- Config change → notify team

---

### Pattern 4: Source Polling

**When:** React to data changes by polling sources.

```yaml
triggers:
  - poll:
      source: api_health
      interval: 5m
      condition: "status != 'healthy'"
  - poll:
      source: queue_depth
      interval: 1m
      condition: "depth > 100"
```

**Examples:**
- API unhealthy → alert and start runbook
- Queue too deep → scale up workers
- Low inventory → reorder

---

### Pattern 5: Chained Automation

**When:** One bot triggers another.

```yaml
triggers:
  - event: standup.completed
    from: standup-bot
```

**Examples:**
- Standup completed → generate summary → post to Slack
- Incident resolved → run postmortem bot
- Daily data collected → weekly aggregation

---

## Architecture

### Serve with Triggers

```
┌─────────────────────────────────────────────────────────────────┐
│                      TINKERDOWN SERVE                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  tinkerdown serve bot.md              # UI + triggers           │
│  tinkerdown serve bot.md --headless   # triggers only           │
│  tinkerdown serve ./bots/             # multiple files          │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Scheduler                             │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │   │
│  │  │ standup-bot │  │ health-bot  │  │ report-bot  │      │   │
│  │  │ 9am, 10am   │  │ every 5min  │  │ weekly      │      │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                  Webhook Server                          │   │
│  │  POST /webhook/github      → github-bot.md               │   │
│  │  POST /webhook/stripe      → payments-bot.md             │   │
│  │  POST /webhook/slack       → slack-bot.md                │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   File Watcher                           │   │
│  │  ./inbox/*.pdf             → receipt-processor.md        │   │
│  │  ./data/*.csv              → data-importer.md            │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Execution Model

```
┌─────────────────────────────────────────────────────────────────┐
│                     EXECUTION MODEL                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Trigger fires                                                  │
│       │                                                         │
│       ▼                                                         │
│  ┌─────────────────┐                                           │
│  │  Load bot.md    │                                           │
│  └────────┬────────┘                                           │
│           │                                                     │
│           ▼                                                     │
│  ┌─────────────────┐                                           │
│  │  Fetch sources  │  ← Same as interactive mode               │
│  └────────┬────────┘                                           │
│           │                                                     │
│           ▼                                                     │
│  ┌─────────────────┐                                           │
│  │ Evaluate        │  ← Check conditions                       │
│  │ conditions      │                                           │
│  └────────┬────────┘                                           │
│           │                                                     │
│           ▼                                                     │
│  ┌─────────────────┐                                           │
│  │ Execute actions │  ← Run configured actions                 │
│  └────────┬────────┘                                           │
│           │                                                     │
│           ▼                                                     │
│  ┌─────────────────┐                                           │
│  │ Update state    │  ← Write to markdown                      │
│  └────────┬────────┘                                           │
│           │                                                     │
│           ▼                                                     │
│  ┌─────────────────┐                                           │
│  │ Send outputs    │  ← Slack, email, webhook                  │
│  └─────────────────┘                                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### State Management

Bot state lives in the markdown file:

```markdown
---
triggers:
  - schedule: "0 9 * * 1-5"
sources:
  runs:
    type: markdown
    anchor: "#runs"
---

# Daily Health Check Bot

## Configuration
- Check: API health endpoint
- Alert: Slack #ops-alerts
- Threshold: 3 consecutive failures

## Runs {#runs}

| timestamp | status | latency | alerted |
|-----------|--------|---------|---------|
| 2025-01-15 09:00 | healthy | 45ms | no |
| 2025-01-15 09:05 | healthy | 52ms | no |
| 2025-01-15 09:10 | unhealthy | timeout | yes |
```

---

## Bot Catalog

### 1. Standup Reminder Bot

**What:** Posts reminder, collects updates, summarizes.

```markdown
---
triggers:
  - schedule: "0 9 * * 1-5"    # 9am reminder
    action: send_reminder
  - schedule: "0 10 * * 1-5"   # 10am summary
    action: send_summary

sources:
  team:
    type: markdown
    anchor: "#team"
  updates:
    type: markdown
    anchor: "#updates"

outputs:
  slack:
    channel: "#team-standup"
    token: ${SLACK_BOT_TOKEN}
---

# Standup Bot

## Team {#team}

| name | slack_id |
|------|----------|
| Alice | U123 |
| Bob | U456 |

## Today's Updates {#updates}

| date | who | yesterday | today | blockers |
|------|-----|-----------|-------|----------|

## Actions

### send_reminder
Post to Slack:
```
Good morning team! 👋
Time for standup. Reply with:
- What you did yesterday
- What you're doing today
- Any blockers
```

### send_summary
Compile updates and post summary to Slack.
```

---

### 2. GitHub Issue Tracker Bot

**What:** Receives GitHub webhooks, adds issues to tracker.

```markdown
---
triggers:
  - webhook:
      path: /github
      secret: ${GITHUB_WEBHOOK_SECRET}
      events: [issues.opened, issues.closed]

sources:
  issues:
    type: markdown
    anchor: "#issues"
---

# GitHub Issue Tracker

## Open Issues {#issues}

| number | title | author | opened | status |
|--------|-------|--------|--------|--------|

## Webhook Handler

### on issues.opened
Add row to issues:
- number: {{payload.issue.number}}
- title: {{payload.issue.title}}
- author: {{payload.issue.user.login}}
- opened: {{now}}
- status: open

### on issues.closed
Update row where number = {{payload.issue.number}}:
- status: closed
```

---

### 3. Health Check Bot

**What:** Periodic health check with alerting.

```markdown
---
triggers:
  - schedule: "*/5 * * * *"   # Every 5 minutes
    action: check_health

sources:
  health:
    type: rest
    url: https://api.example.com/health
  history:
    type: markdown
    anchor: "#history"
  state:
    type: markdown
    anchor: "#state"

outputs:
  slack:
    channel: "#ops-alerts"
---

# API Health Check Bot

## Current State {#state}

| consecutive_failures | last_alert |
|---------------------|------------|
| 0 | never |

## History {#history}

| timestamp | status | latency_ms |
|-----------|--------|------------|

## Actions

### check_health
1. Fetch health source
2. Log to history with timestamp
3. If status != healthy:
   - Increment consecutive_failures
   - If consecutive_failures >= 3 AND last_alert > 10min ago:
     - Alert to Slack
     - Update last_alert
4. If status == healthy:
   - Reset consecutive_failures to 0
```

---

### 4. Expense Receipt Processor

**What:** Watches folder, extracts receipt data, adds to tracker.

```markdown
---
triggers:
  - watch:
      path: ./receipts/*.pdf
      events: [create]
    action: process_receipt

sources:
  expenses:
    type: markdown
    anchor: "#expenses"

outputs:
  notification:
    type: exec
    cmd: "say 'Receipt processed'"
---

# Receipt Processor Bot

## Expenses {#expenses}

| date | vendor | amount | category | receipt |
|------|--------|--------|----------|---------|

## Actions

### process_receipt
1. Extract text from PDF (pdftotext)
2. Parse for:
   - date (regex for date patterns)
   - vendor (first line usually)
   - amount (regex for currency)
3. Add row to expenses
4. Move PDF to ./receipts/processed/
5. Notify
```

---

### 5. Weekly Report Generator

**What:** Aggregates data, generates report, sends email.

```markdown
---
triggers:
  - schedule: "0 9 * * 1"    # Monday 9am
    action: generate_report

sources:
  tasks:
    type: sqlite
    db: ./data/tasks.db
    query: |
      SELECT status, COUNT(*) as count
      FROM tasks
      WHERE completed_at >= date('now', '-7 days')
      GROUP BY status
  commits:
    type: exec
    cmd: git log --oneline --since="1 week ago" | wc -l

outputs:
  email:
    to: team@example.com
    subject: "Weekly Report - {{now:2006-01-02}}"
---

# Weekly Report Bot

## Actions

### generate_report
Generate markdown report:
```
# Weekly Report - {{now:2006-01-02}}

## Tasks
- Completed: {{tasks | where status="done" | count}}
- In Progress: {{tasks | where status="doing" | count}}

## Commits
- {{commits}} commits this week

## Highlights
(manually added)
```

Send via email.
```

---

### 6. Queue Monitor Bot

**What:** Monitors queue depth, alerts and scales.

```markdown
---
triggers:
  - poll:
      source: queue
      interval: 1m
      condition: "depth > 100"
    action: alert_and_scale

sources:
  queue:
    type: rest
    url: https://sqs.amazonaws.com/queue/stats
    headers:
      Authorization: "Bearer ${AWS_TOKEN}"
  history:
    type: markdown
    anchor: "#history"
---

# Queue Monitor Bot

## Thresholds
- Alert: depth > 100
- Scale up: depth > 500
- Scale down: depth < 10 for 10 minutes

## History {#history}

| timestamp | depth | action |
|-----------|-------|--------|

## Actions

### alert_and_scale
1. Log current depth to history
2. If depth > 500:
   - Scale up workers (aws ecs update-service)
   - Alert Slack
3. Else if depth > 100:
   - Alert Slack only
```

---

### 7. Slack Command Bot

**What:** Responds to Slack slash commands.

```markdown
---
triggers:
  - webhook:
      path: /slack/commands
      verify: slack_signature

sources:
  todos:
    type: markdown
    anchor: "#todos"

outputs:
  slack:
    response_type: in_channel
---

# Slack Todo Bot

## Todos {#todos}

| id | task | owner | done |
|----|------|-------|------|

## Commands

### /todo list
Reply with current todos table formatted for Slack.

### /todo add <task>
Add new todo with:
- task: {{text}}
- owner: {{user_name}}
- done: no

Reply: "Added: {{text}}"

### /todo done <id>
Update todo where id = {{id}}:
- done: yes

Reply: "Completed: {{task}}"
```

---

### 8. Database Backup Bot

**What:** Daily database backup with rotation.

```markdown
---
triggers:
  - schedule: "0 2 * * *"    # 2am daily
    action: backup

sources:
  backups:
    type: markdown
    anchor: "#backups"
---

# Database Backup Bot

## Configuration
- Database: postgres://prod-db
- Destination: s3://backups/db/
- Retention: 30 days

## Backups {#backups}

| timestamp | file | size | status |
|-----------|------|------|--------|

## Actions

### backup
1. Run pg_dump
2. Compress with gzip
3. Upload to S3
4. Log to backups table
5. Delete backups older than 30 days
6. Verify backup count >= 7 (alert if not)
```

---

### 9. Deployment Pipeline Bot

**What:** Triggered by git push, runs deployment.

```markdown
---
triggers:
  - webhook:
      path: /github
      events: [push]
      filter: "ref == 'refs/heads/main'"
    action: deploy

sources:
  deployments:
    type: markdown
    anchor: "#deployments"

outputs:
  slack:
    channel: "#deploys"
---

# Deployment Bot

## Deployments {#deployments}

| timestamp | commit | author | status | duration |
|-----------|--------|--------|--------|----------|

## Actions

### deploy
1. Log deployment started
2. Notify Slack: "Deploying {{commit_sha}}"
3. Run:
   - git pull
   - npm install
   - npm run build
   - pm2 restart app
4. Health check (curl /health)
5. Update deployment status
6. Notify Slack: "Deploy complete" or "Deploy failed"
```

---

### 10. Meeting Scheduler Bot

**What:** Finds available times, sends calendar invites.

```markdown
---
triggers:
  - webhook:
      path: /schedule

sources:
  requests:
    type: markdown
    anchor: "#requests"

outputs:
  calendar:
    type: google_calendar
    credentials: ${GOOGLE_CREDENTIALS}
  email:
    smtp: smtp.gmail.com
---

# Meeting Scheduler Bot

## Pending Requests {#requests}

| id | attendees | duration | preferred_times | status |
|----|-----------|----------|-----------------|--------|

## Actions

### on webhook
1. Parse attendees and duration from request
2. Fetch calendar availability for each attendee
3. Find overlapping free slots
4. Add to requests with status=pending
5. Email organizer with available times

### confirm
1. Create calendar event
2. Send invites
3. Update status to scheduled
```

---

## LLM Generation Strategy

### Prompt Template

```
Create a tinkerdown automation bot that:
- Trigger: [schedule/webhook/file watch/poll]
- Action: [what to do when triggered]
- State: [what to track between runs]
- Outputs: [where to send results - Slack/email/webhook]

The bot should be defined in a single markdown file with:
1. YAML frontmatter with triggers and sources
2. State tables for tracking
3. Action definitions in markdown
```

### Example Prompts

**Standup bot:**
```
Create a tinkerdown bot that:
- Posts a standup reminder to Slack at 9am on weekdays
- Collects responses via Slack
- Posts a summary at 10am
- Tracks who has responded
```

**GitHub tracker:**
```
Create a tinkerdown bot that:
- Receives GitHub issue webhooks
- Adds new issues to a markdown table
- Updates status when issues are closed
- Posts to Slack when high-priority issues are opened
```

**Health check:**
```
Create a tinkerdown bot that:
- Checks API health every 5 minutes
- Tracks consecutive failures
- Alerts Slack after 3 failures
- Logs all checks to a history table
```

---

## Required Features

### Must Have (P0)

| Feature | Description | Effort |
|---------|-------------|--------|
| **Trigger runner** | Detect triggers in serve | Medium |
| **Schedule triggers** | Cron syntax | Small |
| **Action execution** | Run defined actions | Medium |
| **State persistence** | Write results to markdown | Small |

### Should Have (P1)

| Feature | Description | Effort |
|---------|-------------|--------|
| **Webhook triggers** | HTTP endpoint with secret | Medium |
| **Outputs (Slack)** | Post to Slack | Medium |
| **Outputs (email)** | Send email | Small |
| **Condition evaluation** | Only run if condition met | Medium |

### Nice to Have (P2)

| Feature | Description | Effort |
|---------|-------------|--------|
| **File watch triggers** | React to file changes | Medium |
| **Poll triggers** | Check source periodically | Medium |
| **Outputs (webhook)** | POST to external URL | Small |
| **Bot chaining** | One bot triggers another | Medium |
| **Dashboard** | See all bots and status | Large |

---

## Comparison with Alternatives

### vs. Zapier/Make

| Aspect | Zapier | Tinkerdown |
|--------|--------|------------|
| Price | $20-600/mo | Free |
| Vendor lock-in | Yes | No (markdown files) |
| Custom logic | Limited | Full (exec source) |
| State management | Limited | Built-in (markdown) |
| LLM-friendly | No | Yes |
| Self-hosted | No | Yes |
| Version control | No | Git native |

### vs. n8n/Temporal

| Aspect | n8n/Temporal | Tinkerdown |
|--------|--------------|------------|
| Complexity | High | Low |
| Setup | Docker/K8s | Single binary |
| Learning curve | Steep | Minimal (markdown) |
| Visual editor | Yes | No (text-based) |
| Enterprise features | Yes | Basic |

### vs. Cron + Scripts

| Aspect | Cron + Scripts | Tinkerdown |
|--------|----------------|------------|
| State management | Manual | Built-in |
| Documentation | Separate | Same file |
| History/audit | Manual | Automatic |
| LLM generation | Hard | Easy |
| Composability | Hard | Sources reusable |

### vs. GitHub Actions

| Aspect | GH Actions | Tinkerdown |
|--------|------------|------------|
| Triggers | Repo events only | Any trigger |
| Runs on | GitHub runners | Your machine |
| State | Artifacts only | Markdown |
| Cost | Limited free mins | Free |
| Local dev | Complex | Simple |

---

## Implementation Plan

### Phase 1: Core Trigger Runner (1 week)

```
Week 1:
├── Day 1-2: Trigger detection in serve
│   ├── Parse triggers: from frontmatter
│   ├── Start trigger runner alongside web server
│   ├── --headless flag for triggers only
│   └── Scheduler (robfig/cron)
│
├── Day 3-4: Action execution
│   ├── Parse action blocks from markdown
│   ├── Execute actions (same as interactive)
│   ├── Write state back to file
│   └── Logging
│
└── Day 5: State management
    ├── Lock file during execution
    ├── Atomic writes
    └── Error recovery
```

### Phase 2: Triggers (1 week)

```
Week 2:
├── Day 1-2: Webhook triggers
│   ├── HTTP server for webhooks
│   ├── Path routing
│   ├── Secret validation (HMAC)
│   └── Payload available in templates
│
├── Day 3: Condition evaluation
│   ├── Parse condition expressions
│   ├── Evaluate against source data
│   └── Skip if condition false
│
└── Day 4-5: File watch triggers
    ├── fsnotify integration
    ├── Glob patterns
    └── File info in templates
```

### Phase 3: Outputs (1 week)

```
Week 3:
├── Day 1-2: Slack output
│   ├── Slack API integration
│   ├── Message formatting
│   ├── Blocks support
│   └── Response handling
│
├── Day 3: Email output
│   ├── SMTP client
│   ├── Template rendering
│   └── Attachments
│
└── Day 4-5: Webhook output
    ├── HTTP POST to external URLs
    ├── Retry with backoff
    └── Response logging
```

### Phase 4: Polish (1 week)

```
Week 4:
├── Day 1-2: Dashboard
│   ├── Web UI showing all bots
│   ├── Run history
│   ├── Manual trigger button
│   └── Log viewer
│
├── Day 3: Poll triggers
│   ├── Periodic source fetch
│   ├── Change detection
│   └── Condition evaluation
│
└── Day 4-5: Documentation
    ├── Bot authoring guide
    ├── Example bots
    └── LLM prompts
```

---

## Summary

### The Third Pillar

| Use Case | Interactive | Trigger | State |
|----------|-------------|---------|-------|
| **Runbooks** | Yes | Manual | Incident record |
| **Productivity** | Yes | Manual | Personal data |
| **Automation** | No | Schedule/Event | Bot history |

### Why It Works

1. **Same markdown format** - Reuse everything from interactive apps
2. **Same sources** - exec, rest, markdown, sqlite all work
3. **State in file** - Bot history is just another markdown table
4. **LLM-friendly** - Same patterns, easy to generate
5. **Git-native** - Bot definitions and history in version control

### The Pitch

```
Before:
"I need a health check bot"
→ Research Zapier → Pay $20/mo → Configure UI → Limited customization

After:
"I need a health check bot"
→ Ask LLM → Get health-bot.md → tinkerdown daemon → Free forever
→ Logic is readable → State tracked → Git backup
```

### Key Differentiator

**Zapier:** Visual editor, pay per execution
**Tinkerdown:** Markdown files, free, LLM-generated, self-documented

---

## Next Steps

1. Add `triggers` and `outputs` to YAML frontmatter schema
2. Implement trigger runner in serve command
3. Implement --headless flag
4. Implement webhook server
5. Implement Slack output
6. Create 5 example bots
7. Write LLM prompt templates
