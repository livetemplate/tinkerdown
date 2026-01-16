# Tinkerdown Runbooks

**Date:** 2025-12-30
**Status:** Design
**Goal:** Make tinkerdown the best tool for executable, trackable runbooks

---

## Vision

A runbook in tinkerdown is:
1. **A template** - Reusable procedure with live system checks
2. **An instance** - Copy of template used during a specific incident
3. **A record** - The instance becomes the incident record + postmortem

**One file = runbook + timeline + postmortem.**

---

## Why Tinkerdown for Runbooks?

### Current Tools

| Tool Type | What It Does | Limitation |
|-----------|--------------|------------|
| Wiki/docs platforms | Static runbook docs | No execution, no tracking |
| Automation platforms | Automation + logging | Locked in their system |
| Code execution tools | Execute code blocks | No step tracking, no snapshots |

### What's Missing Everywhere

- **Step tracking** - Was step 3 actually run?
- **Timestamps** - When was each action taken?
- **Snapshots** - What did the system show at 14:32?
- **Unified record** - Runbook + timeline + postmortem in one place

### Tinkerdown's Advantage

| Capability | Others | Tinkerdown |
|------------|--------|------------|
| Live system state in runbook | ❌ | ✅ (exec sources) |
| Step execution tracking | Partial | ✅ (execution log) |
| Capture snapshots | ❌ | ✅ (snapshot action) |
| Portable record | ❌ (vendor lock-in) | ✅ (markdown) |
| Git-native | ❌ | ✅ |
| Grep-able history | ❌ | ✅ |
| LLM-analyzable | ❌ | ✅ |

---

## Workflow

### 1. Create Template

Write a runbook with live data sources:

```markdown
---
title: Service Recovery Runbook
sources:
  containers:
    type: exec
    cmd: docker ps --format '{"name":"{{.Names}}","status":"{{.Status}}"}'
  logs:
    type: exec
    cmd: docker logs --tail 20 api-service 2>&1
---

# Service Recovery Runbook

## Step 1: Check Container Status

\`\`\`lvt
<table lvt-source="containers" lvt-columns="name:Container,status:Status">
</table>
\`\`\`

## Step 2: Review Logs

\`\`\`lvt
<ul lvt-source="logs" lvt-field="line">
</ul>
\`\`\`

## Step 3: Restart Service

\`\`\`bash
docker restart api-service
\`\`\`

## Step 4: Verify Recovery

Re-check Step 1. Container should show "healthy".
```

### 2. Incident Starts → Copy Template

No special command needed. Just copy:

```bash
cp templates/service-recovery.md incidents/$(date +%Y-%m-%d-%H%M)-api-down.md
```

Or use a shell alias:

```bash
# Add to .bashrc/.zshrc
incident() {
  local template="templates/$1.md"
  local name="$2"
  local file="incidents/$(date +%Y-%m-%d-%H%M)-${name}.md"
  cp "$template" "$file"
  echo "Created: $file"
  echo "Run: tinkerdown serve $file"
}

# Usage
$ incident service-recovery api-down
Created: incidents/2024-01-15-1432-api-down.md
Run: tinkerdown serve incidents/2024-01-15-1432-api-down.md
```

### 3. During Incident

Operator opens the instance in browser:

```bash
tinkerdown serve incidents/2024-01-15-1432-api-down.md
```

They see:
- Live system state (from exec sources)
- Step-by-step procedure
- Execution log to track actions
- Snapshot buttons to capture state

### 4. Track What Was Done

The instance includes an execution log:

```markdown
## Execution Log {#log}

| time | step | action | result | who |
|------|------|--------|--------|-----|

\`\`\`lvt
<form lvt-submit="add" lvt-source="log">
  <input name="step" placeholder="Step #">
  <select name="action">
    <option value="started">Started</option>
    <option value="completed">Completed</option>
    <option value="failed">Failed</option>
    <option value="skipped">Skipped</option>
    <option value="note">Note</option>
  </select>
  <input name="result" placeholder="What happened?">
  <input name="who" placeholder="Your name">
  <button type="submit">Log</button>
</form>

<table lvt-source="log" lvt-columns="time:Time,step:Step,action:Action,result:Result,who:Who">
</table>
\`\`\`
```

Each log entry is saved to the markdown file.

### 5. Capture Snapshots

When the operator wants to record what the system showed:

```html
<button lvt-click="snapshot" lvt-data-source="containers">
  📸 Capture
</button>
```

This saves the current output to a snapshots section:

```markdown
## Snapshots {#snapshots}

### 14:32 - containers
\`\`\`
name: api-service, status: Up 2 hours (unhealthy)
name: postgres, status: Up 3 days (healthy)
\`\`\`

### 14:45 - containers
\`\`\`
name: api-service, status: Up 1 minute (healthy)
name: postgres, status: Up 3 days (healthy)
\`\`\`
```

### 6. Resolve & Commit

After incident:

```bash
# Add resolution notes to the file
# Then commit
git add incidents/2024-01-15-1432-api-down.md
git commit -m "Incident: API down - resolved, disk full root cause"
```

The file is now:
- Permanent record
- Searchable (`grep "disk full" incidents/`)
- Diffable (see what changed)
- Ready for postmortem review

---

## Required Features

### Feature 1: Auto-Timestamp on Form Submit

**Problem:** Operator has to type the time manually.

**Solution:** Auto-fill timestamp when form is submitted.

```yaml
sources:
  log:
    type: markdown
    anchor: "#log"
    readonly: false
    auto_fields:
      time: "{{now:15:04}}"  # HH:MM format
```

Form no longer needs time input - it's added automatically:

```html
<form lvt-submit="add" lvt-source="log">
  <!-- no time field needed -->
  <input name="step" placeholder="Step #">
  <input name="action" placeholder="What happened">
  <button type="submit">Log</button>
</form>
```

**Implementation:**
- On form submit, check source config for `auto_fields`
- For each auto field, evaluate template and add to form data
- `{{now:FORMAT}}` uses Go time format

**Effort:** Small

---

### Feature 2: Snapshot Capture

**Problem:** Exec sources show live data. Operator wants to freeze it at a point in time.

**Solution:** A `snapshot` action that saves exec output to the file.

```html
<button lvt-click="snapshot" lvt-data-source="containers" lvt-data-label="Step 1">
  📸 Capture
</button>
```

Clicking this:
1. Runs the exec command
2. Formats output as markdown code block
3. Appends to `#snapshots` section with timestamp

```markdown
## Snapshots {#snapshots}

### 14:32 - Step 1
\`\`\`
container output here...
\`\`\`
```

**Implementation:**
- New action handler: `snapshot`
- Takes `source` (which exec source) and `label` (description)
- Appends formatted block to snapshots section

**Effort:** Medium

---

### Feature 3: Step Status Buttons

**Problem:** Operator wants to mark steps as done/failed/skipped with one click.

**Solution:** Pre-built step status buttons that log + update display.

```html
<div class="step-controls" lvt-step="1" lvt-log-source="log">
  <button lvt-click="step_start">⏳ Start</button>
  <button lvt-click="step_done">✅ Done</button>
  <button lvt-click="step_failed">❌ Failed</button>
  <button lvt-click="step_skip">⏭️ Skip</button>
</div>
```

Clicking any button:
1. Adds entry to execution log with timestamp
2. Updates visual status indicator

**Implementation:**
- New action handlers: `step_start`, `step_done`, `step_failed`, `step_skip`
- Each logs to the specified source
- Optionally updates a status field on the step

**Effort:** Medium

---

### Feature 4: Operator Identity

**Problem:** Who did what? Currently operator types their name each time.

**Solution:** Set operator once, auto-fill on actions.

```bash
tinkerdown serve incidents/file.md --operator alice
```

Or in config:
```yaml
# tinkerdown.yaml
operator: alice  # Or from env: ${USER}
```

Then in auto_fields:
```yaml
auto_fields:
  who: "{{operator}}"
```

**Implementation:**
- Add `--operator` flag to serve command
- Make `{{operator}}` available in templates

**Effort:** Small

---

## File Structure

```
runbooks/
├── templates/
│   ├── service-recovery.md
│   ├── database-failover.md
│   └── disk-full.md
├── incidents/
│   ├── 2024-01-15-1432-api-down.md
│   ├── 2024-01-20-0900-db-slow.md
│   └── 2024-02-01-2230-disk-full.md
└── tinkerdown.yaml  # Optional global config
```

---

## Instance Template

A good instance template structure:

```markdown
---
title: "Incident: {{name}}"
started: {{timestamp}}
status: active
severity:
on_call:
sources:
  log:
    type: markdown
    anchor: "#execution-log"
    readonly: false
    auto_fields:
      time: "{{now:15:04}}"
  # ... exec sources from template ...
---

# Incident: {{name}}

| Field | Value |
|-------|-------|
| Started | {{date}} {{time}} |
| Status | 🔴 Active |
| Severity | |
| On-call | |
| Participants | |

---

## Execution Log {#execution-log}

| time | step | action | result | who |
|------|------|--------|--------|-----|

\`\`\`lvt
<form lvt-submit="add" lvt-source="log">
  <input name="step" placeholder="Step">
  <select name="action">
    <option value="started">⏳ Started</option>
    <option value="completed">✅ Completed</option>
    <option value="failed">❌ Failed</option>
    <option value="skipped">⏭️ Skipped</option>
    <option value="note">📝 Note</option>
  </select>
  <input name="result" placeholder="Details">
  <input name="who" placeholder="Who">
  <button type="submit">Log</button>
</form>

<table lvt-source="log" lvt-columns="time,step,action,result,who">
</table>
\`\`\`

---

## Steps

(Paste steps from template, or include inline)

---

## Snapshots {#snapshots}

(Captured automatically via snapshot buttons)

---

## Resolution

**Resolved:**
**Duration:**
**Root Cause:**

## Action Items

- [ ]
```

---

## Priority Order

| Priority | Feature | Why |
|----------|---------|-----|
| **P0** | Auto-timestamp on form submit | Core logging usability |
| **P0** | Snapshot capture action | Core value prop |
| **P1** | Step status buttons | Faster logging |
| **P1** | Operator identity | Less typing |
| **P2** | Step status display | Visual progress |

---

## Success Criteria

A runbook system is successful if:

1. **Operator can log actions in <5 seconds** - Fast form, auto-timestamp
2. **System state is captured** - Snapshots preserve what was seen
3. **File is self-contained** - Anyone can read it and understand what happened
4. **Grep works** - `grep "connection refused" incidents/*.md` finds incidents
5. **Postmortem is easy** - Just add analysis to existing file

---

## What Tinkerdown Does NOT Do

- **Real-time collaboration** - Use dedicated incident management tools for war rooms
- **Automated remediation** - Use dedicated automation platforms
- **Paging/alerting** - Use dedicated alerting tools
- **Dashboards** - Use dedicated monitoring/dashboard tools
- **Authentication/Authorization** - Delegate to sources and CLI tools

Tinkerdown is for: **executable documentation that becomes the incident record.**

---

## Authentication & Authorization

**Principle: Tinkerdown does NOT implement auth. It delegates to existing infrastructure.**

### Why No Built-in Auth?

1. **Orgs already have auth** - SSO, LDAP, IAM roles
2. **CLI tools have auth** - kubectl uses kubeconfig, aws uses credentials, gh uses tokens
3. **Sources can check permissions** - Call your existing permission APIs
4. **Less to maintain** - No token management, no user database

### How Auth Works in Practice

#### Operator Identity

Comes from the environment, not tinkerdown:

```yaml
# tinkerdown.yaml
identity:
  # Environment variable
  operator: ${USER}
  # Or from SSO
  operator: ${SSO_USER}
  # Or from git
  operator: $(git config user.email)
```

#### Permission Checks via Sources

```python
#!/usr/bin/env python3
# sources/check-permission.py
# Calls your existing permission system

import sys, json, requests

def main():
    input_data = json.load(sys.stdin)
    operator = input_data["env"]["operator"]
    permission = input_data["query"]["permission"]

    # Call your existing auth system
    resp = requests.get(
        "https://auth.internal/check",
        params={"user": operator, "permission": permission},
        headers={"Authorization": f"Bearer {os.environ['AUTH_TOKEN']}"}
    )

    allowed = resp.json().get("allowed", False)
    print(json.dumps({
        "columns": ["allowed", "reason"],
        "rows": [{"allowed": allowed, "reason": resp.json().get("reason", "")}]
    }))

if __name__ == "__main__":
    main()
```

Use in runbook:

```markdown
```yaml
sources:
  perm:
    type: exec
    command: "./sources/check-permission.py"
    query:
      permission: "prod-db-write"
```​

<div lvt-if="!perm[0].allowed" class="error">
  ❌ You don't have permission: {{perm[0].reason}}
</div>
```

#### CLI Tools Enforce Their Own Auth

```markdown
## Step 3: Scale Down Service

```bash
# kubectl uses ~/.kube/config - no tinkerdown auth needed
kubectl scale deployment api --replicas=0
```​

```bash
# aws cli uses ~/.aws/credentials - no tinkerdown auth needed
aws ecs update-service --desired-count 0
```​
```

If the operator doesn't have permission, the command fails. Tinkerdown just captures the output.

#### Approval Workflow Without Auth System

Approvals are **log entries + notifications**, not an auth system:

```markdown
## Approvals Required

**Prod DB access requires manager approval.**

<button lvt-on:click="exec:request-approval.sh --access=prod-db-write">
  Request Access
</button>

This:
1. Adds log entry: `| 14:35 | access_request | prod-db-write | alice |`
2. Sends notification to approvers
3. Approver clicks button in notification (or CLI)
4. Adds log entry: `| 14:37 | access_approved | prod-db-write | alice | bob |`
```

The approval is **recorded**, not **enforced**. Enforcement happens when:
- The operator tries to run `kubectl exec` (k8s RBAC blocks them)
- The source checks permission before showing sensitive data
- The CLI tool checks credentials

### What About the API?

API auth uses your existing patterns:

```yaml
# tinkerdown.yaml
api:
  auth:
    type: bearer
    # Token from environment - you manage token lifecycle
    token: ${TINKERDOWN_API_TOKEN}

    # Or use your auth proxy
    type: proxy
    # Nginx/Envoy/etc. handles auth, passes X-User header
    user_header: X-Authenticated-User
```

### Summary

| Auth Concern | Who Handles It |
|--------------|----------------|
| Operator identity | Environment ($USER, SSO) |
| Permission checks | Custom sources (call your APIs) |
| Command authorization | CLI tools (kubectl, aws, etc.) |
| Approval tracking | Log entries + notifications |
| API auth | Bearer token or auth proxy |

**Tinkerdown is a UI and record-keeper, not a security boundary.**

---

## Appendix: Tinkerdown vs Code Execution Tools

Code execution tools run code blocks in markdown. How is tinkerdown different?

| Feature | Code Execution Tools | Tinkerdown |
|---------|---------------------|------------|
| Execute code blocks | ✅ | ✅ (exec source) |
| **Live data display** | ❌ | ✅ (tables, lists) |
| **Execution logging** | ❌ | ✅ (markdown source) |
| **Snapshot capture** | ❌ | ✅ |
| **Forms for input** | ❌ | ✅ |
| **Step tracking** | ❌ | ✅ |
| IDE integration | ✅ | ❌ (browser) |

Code execution tools are great for running commands. Tinkerdown is for **tracking what was done.**
