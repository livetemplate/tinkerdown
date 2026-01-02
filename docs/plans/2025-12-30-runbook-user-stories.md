# Tinkerdown Runbooks: User Stories

**Date:** 2025-12-30
**Status:** Draft

---

## Personas

| Persona | Description |
|---------|-------------|
| **On-call Engineer** | First responder to incidents, follows runbooks |
| **Incident Commander** | Coordinates response, tracks progress |
| **Runbook Author** | Creates and maintains runbook templates |
| **Postmortem Reviewer** | Reviews incidents after resolution |
| **New Team Member** | Learning the systems and procedures |
| **Engineering Manager** | Tracks incident patterns, team performance |
| **Approver** | Grants temporary elevated permissions during incidents |
| **Compliance Officer** | Audits access and approvals |

---

## User Stories

### On-Call Engineer

#### Starting an Incident

**US-1: Create incident from template**
> As an on-call engineer, I want to quickly create an incident file from a runbook template so that I can start tracking my actions immediately.

**Acceptance Criteria:**
- Copy template to incidents folder with timestamp in filename
- File opens in browser with live system data
- Execution log is empty and ready for entries

**Git Workflow:**
```bash
# No commit yet - file is being actively edited
cp templates/db-recovery.md incidents/$(date +%Y-%m-%d-%H%M)-db-outage.md
tinkerdown serve incidents/2024-01-15-1432-db-outage.md
```

---

#### During Incident

**US-2: Log actions with timestamps**
> As an on-call engineer, I want to log what I'm doing with automatic timestamps so that I don't waste time typing the time during an incident.

**Acceptance Criteria:**
- Form to add log entry
- Time is auto-filled on submit
- My name is auto-filled (from `--operator` or config)
- Entry appears in table immediately

---

**US-3: Capture system state**
> As an on-call engineer, I want to capture what the system showed at a specific moment so that I have evidence for the postmortem.

**Acceptance Criteria:**
- Button next to each live data source
- Clicking captures current output with timestamp
- Snapshot is saved to the file immediately
- Snapshot is immutable (can't be edited)

---

**US-4: Mark step status**
> As an on-call engineer, I want to mark steps as started/done/failed/skipped so that I can track progress and others can see where I am.

**Acceptance Criteria:**
- One-click buttons for each status
- Status change is logged with timestamp
- Visual indicator shows current status

---

**US-5: See live system state**
> As an on-call engineer, I want to see live system data inline with the runbook steps so that I don't have to switch between terminals.

**Acceptance Criteria:**
- Exec sources show current output
- Data refreshes periodically or on demand
- Clear indication that data is live (not cached)

---

**US-6: Work offline**
> As an on-call engineer, I want the runbook to work even if my network is flaky so that I can still follow procedures.

**Acceptance Criteria:**
- Runbook file is local
- Changes are saved to local file
- Can commit to git when network returns

---

### Incident Commander

**US-7: See who's doing what**
> As an incident commander, I want to see the execution log in real-time so that I know what's been tried and who's working on what.

**Acceptance Criteria:**
- Open same incident file in browser
- See log entries as they're added
- See which steps are in progress

**Note:** Real-time sync between browsers is a stretch goal. MVP: refresh to see updates.

---

**US-8: Add participants**
> As an incident commander, I want to record who joined the incident so that we know who was involved for the postmortem.

**Acceptance Criteria:**
- Field to add participant names
- Timestamps when they joined
- Saved to the file

---

### Multiple Responders

**US-9: Concurrent editing**
> As one of multiple responders, I want my changes to not overwrite my colleague's changes so that we don't lose information.

**Acceptance Criteria:**
- Last-write-wins for MVP (simple)
- Future: merge or lock mechanism
- Execution log is append-only (safe for concurrent adds)

**Git Workflow:**
```bash
# During incident: DON'T commit (file is actively edited)
# After incident: one person commits the final state

# If needed during long incident, checkpoint:
git add incidents/2024-01-15-1432-db-outage.md
git commit -m "WIP: incident in progress, checkpoint"
```

---

### Approvals & Elevated Access

**US-9a: Request elevated access**
> As an on-call engineer, I want to request elevated permissions and record the request so that there's an audit trail.

**Acceptance Criteria:**
- Log entry for "access requested" with what access and why
- Clear what permission is needed (e.g., "prod DB write access")
- Timestamp and requester recorded

**Example Log Entry:**
```
| 14:35 | - | access_request | Need prod DB write access to fix data | alice |
```

---

**US-9b: Approve access request**
> As an approver, I want to grant access and have my approval recorded so that there's accountability.

**Acceptance Criteria:**
- Approver adds log entry with "approved" status
- Includes approver name and timestamp
- Optional: expiry time for access

**Example Log Entry:**
```
| 14:37 | - | access_approved | Prod DB write access granted, expires 16:00 | bob (manager) |
```

---

**US-9c: Track access expiry**
> As an on-call engineer, I want to know when my temporary access expires so that I can request extension if needed.

**Acceptance Criteria:**
- Expiry time noted in approval
- Reminder/warning when approaching expiry (stretch goal)
- Log entry when access is revoked

---

**US-9d: Revoke access after incident**
> As an approver, I want to confirm access was revoked after the incident so that we maintain least-privilege.

**Acceptance Criteria:**
- Log entry for "access revoked"
- Timestamp when revoked
- Part of incident resolution checklist

**Example Log Entry:**
```
| 15:45 | - | access_revoked | Prod DB write access removed | alice |
```

---

**US-9e: Audit approvals**
> As a compliance officer, I want to see all access approvals across incidents so that I can audit elevated access.

**Acceptance Criteria:**
- Standard format for approval entries
- Searchable via grep
- Includes: who requested, who approved, what access, when, expiry

**Audit Workflow:**
```bash
# Find all access requests
grep "access_request\|access_approved\|access_revoked" incidents/*.md

# Find approvals by specific approver
grep "access_approved.*bob" incidents/*.md

# Find access that wasn't revoked (potential issue)
# Would need a script to correlate request/approved/revoked
```

---

### Approvals Table (Alternative Format)

Instead of embedding in execution log, a separate approvals table:

```markdown
## Approvals {#approvals}

| time | type | access | requester | approver | expiry | revoked |
|------|------|--------|-----------|----------|--------|---------|
| 14:35 | request | prod DB write | alice | | | |
| 14:37 | approved | prod DB write | alice | bob | 16:00 | |
| 15:45 | revoked | prod DB write | alice | | | alice |
```

**Pros:**
- Cleaner audit trail
- Easy to see all approvals at a glance
- Can enforce structure

**Cons:**
- Separate from execution log timeline
- More complex to implement

**Recommendation:** Start with execution log entries (simpler), consider dedicated table for v2.

---

### After Incident

**US-10: Add resolution details**
> As the on-call engineer, I want to add root cause and resolution details so that the file is complete before committing.

**Acceptance Criteria:**
- Section for resolved timestamp
- Section for root cause (free text)
- Section for action items (checklist)

---

**US-11: Commit incident record**
> As the on-call engineer, I want to commit the incident file to git so that it becomes a permanent record.

**Acceptance Criteria:**
- Standard git workflow
- Commit message includes incident ID
- File is now part of repo history

**Git Workflow:**
```bash
git add incidents/2024-01-15-1432-db-outage.md
git commit -m "Incident: DB outage 2024-01-15 - resolved, disk full"
git push
```

---

### Postmortem Reviewer

**US-12: Review incident via PR**
> As a postmortem reviewer, I want to review the incident record via pull request so that I can comment and request changes.

**Acceptance Criteria:**
- Engineer opens PR with incident file
- Reviewers can comment on specific lines
- Changes requested → engineer updates file
- Approved → merged to main

**Git Workflow:**
```bash
# On-call engineer
git checkout -b postmortem/2024-01-15-db-outage
git add incidents/2024-01-15-1432-db-outage.md
git commit -m "Postmortem: DB outage - disk full root cause"
git push -u origin postmortem/2024-01-15-db-outage
# Open PR

# Reviewer comments, engineer addresses
git commit -m "Address review comments"
git push

# Merge when approved
```

---

**US-13: See what happened during incident**
> As a postmortem reviewer, I want to see the timeline of actions so that I understand what was tried and when.

**Acceptance Criteria:**
- Execution log shows chronological actions
- Snapshots show system state at key moments
- Clear who did what

---

**US-14: Compare to template**
> As a postmortem reviewer, I want to see which steps were followed/skipped so that I can improve the runbook.

**Acceptance Criteria:**
- Instance shows step statuses
- Can diff instance against template
- Identify steps that didn't work

**Git Workflow:**
```bash
# Compare instance to template
diff templates/db-recovery.md incidents/2024-01-15-1432-db-outage.md

# Or use git to see what was added (logs, snapshots, notes)
git diff --no-index templates/db-recovery.md incidents/2024-01-15-1432-db-outage.md
```

---

### Runbook Author

**US-15: Create runbook template**
> As a runbook author, I want to create a template with live data sources so that responders see real system state.

**Acceptance Criteria:**
- Standard tinkerdown markdown file
- Exec sources for system checks
- Clear step-by-step structure
- Template file in `templates/` folder

**Git Workflow:**
```bash
# Create template
vim templates/new-runbook.md

# Test it
tinkerdown serve templates/new-runbook.md

# Commit
git add templates/new-runbook.md
git commit -m "Add new-runbook template"
git push
```

---

**US-16: Update runbook template**
> As a runbook author, I want to update a template without breaking existing incidents so that improvements don't cause confusion.

**Acceptance Criteria:**
- Template changes don't affect existing instances
- Instances are copies, not references
- Can note template version in instance metadata

**Git Workflow:**
```bash
# Update template
vim templates/db-recovery.md
git add templates/db-recovery.md
git commit -m "Update db-recovery: add step for log rotation"
git push

# Existing instances are unchanged (they're copies)
# New instances will get the updated template
```

---

**US-17: Test runbook in dry-run**
> As a runbook author, I want to test a runbook without creating a real incident so that I can verify it works.

**Acceptance Criteria:**
- `tinkerdown serve templates/runbook.md` works
- Can see live data, test forms
- No file created in incidents/

---

### New Team Member

**US-18: Learn from past incidents**
> As a new team member, I want to read past incident files so that I learn how to respond to common issues.

**Acceptance Criteria:**
- Incident files are in git, readable
- Can search for keywords
- Execution logs show what worked

**Git/CLI Workflow:**
```bash
# Find incidents related to database
grep -l "database" incidents/*.md
grep -l "connection refused" incidents/*.md

# Read a specific incident
cat incidents/2024-01-15-1432-db-outage.md

# Or serve it to see in browser
tinkerdown serve incidents/2024-01-15-1432-db-outage.md
```

---

**US-19: Practice with runbook**
> As a new team member, I want to practice following a runbook so that I'm prepared for real incidents.

**Acceptance Criteria:**
- Can run template locally
- Can practice logging actions
- Practice file isn't committed (or deleted after)

---

### Engineering Manager

**US-20: See incident history**
> As an engineering manager, I want to see all incidents over time so that I can identify patterns.

**Acceptance Criteria:**
- All incidents in `incidents/` folder
- Git log shows when each was resolved
- Can filter by date, severity, etc.

**Git/CLI Workflow:**
```bash
# List all incidents
ls -la incidents/

# Count incidents per month
ls incidents/ | cut -d'-' -f1-2 | sort | uniq -c

# Find SEV1 incidents
grep -l "severity: SEV1" incidents/*.md

# Git history
git log --oneline -- incidents/
```

---

**US-21: Track action items**
> As an engineering manager, I want to see open action items from incidents so that I can ensure follow-up.

**Acceptance Criteria:**
- Action items are in standard format in files
- Can grep for unchecked items
- Can create tracking issue from action items

**CLI Workflow:**
```bash
# Find open action items across all incidents
grep -h "^\- \[ \]" incidents/*.md

# Find action items with owners
grep -h "^\- \[ \].*@" incidents/*.md
```

---

**US-22: Measure response times**
> As an engineering manager, I want to see time-to-resolve metrics so that I can track team performance.

**Acceptance Criteria:**
- Incidents have started/resolved timestamps
- Can calculate duration
- Can aggregate across incidents

**CLI Workflow:**
```bash
# Extract started/resolved times (would need a script)
grep -E "^(started|resolved):" incidents/*.md
```

---

## Git Workflow Summary

### During Incident

| Phase | Git Action | Why |
|-------|------------|-----|
| Start incident | None | File being actively edited |
| During incident | None (or WIP commit) | Don't interrupt responders |
| Resolve incident | None yet | Add resolution details first |

### After Incident

| Phase | Git Action | Why |
|-------|------------|-----|
| Add resolution | Edit file | Complete the record |
| Commit | `git commit` | Permanent record |
| Postmortem review | Open PR | Get feedback |
| Merge | Merge PR | Finalize record |

### Template Updates

| Action | Git Workflow |
|--------|--------------|
| Create template | Commit to main (or PR) |
| Update template | Commit to main (or PR) |
| Delete template | Commit deletion |

**Key Principle:** Instances are copies, not references. Updating a template doesn't affect existing instances.

---

## Branch Strategy

```
main
├── templates/
│   ├── db-recovery.md        # Current version
│   ├── api-latency.md        # Current version
│   └── ...
└── incidents/
    ├── 2024-01-15-1432-db-outage.md    # Merged after postmortem
    ├── 2024-01-20-0900-api-slow.md     # Merged after postmortem
    └── ...

feature/update-db-recovery    # Template improvements
postmortem/2024-01-25-network  # Incident under review
```

---

## Conflict Resolution

### Scenario: Two people edit same incident file

**During incident (MVP approach):**
- Last write wins
- Execution log is append-only (low conflict risk)
- Snapshots are append-only (low conflict risk)
- Notes section is free-form (highest conflict risk)

**Mitigation:**
- Keep log entries small and frequent
- Use chat (Slack) for coordination, file for record
- One person owns the file, others contribute verbally

**Future enhancement:**
- File locking during incident
- Real-time sync (CRDT or OT)
- Merge tool for concurrent edits

---

## File Lifecycle

```
┌─────────────────┐
│    Template     │ (in templates/)
│  db-recovery.md │
└────────┬────────┘
         │ cp
         ▼
┌─────────────────┐
│    Instance     │ (in incidents/)
│ 2024-01-15-...  │
│                 │ ← Active incident
│   [EDITING]     │   (not committed)
└────────┬────────┘
         │ resolve
         ▼
┌─────────────────┐
│    Instance     │
│ 2024-01-15-...  │
│                 │ ← Add resolution
│  [RESOLVED]     │
└────────┬────────┘
         │ git commit
         ▼
┌─────────────────┐
│    Instance     │
│ 2024-01-15-...  │
│                 │ ← In git history
│  [COMMITTED]    │
└────────┬────────┘
         │ PR review
         ▼
┌─────────────────┐
│    Instance     │
│ 2024-01-15-...  │
│                 │ ← Merged to main
│   [MERGED]      │
└─────────────────┘
```

---

## Priority

| Priority | User Stories | Why |
|----------|--------------|-----|
| **P0** | US-1, US-2, US-3, US-5 | Core incident workflow |
| **P1** | US-4, US-10, US-11 | Complete incident lifecycle |
| **P2** | US-9a, US-9b, US-9d, US-12, US-13, US-14 | Approvals & postmortem workflow |
| **P3** | US-9c, US-9e, US-15, US-16, US-17 | Expiry tracking, audit, template authoring |
| **P4** | US-18, US-19, US-20, US-21, US-22 | Management & learning |

---

## Open Questions

1. **Should we recommend a .gitignore for WIP incidents?**
   - Option A: Commit everything, mark WIP in filename
   - Option B: `.gitignore` pattern for active incidents
   - Recommendation: Option A (simpler, no special rules)

2. **How to handle long-running incidents (days)?**
   - Periodic commits with "WIP" prefix
   - Handoff notes in execution log
   - Continue in same file across shifts

3. **Should incident files be signed?**
   - For audit/compliance, git commit signing works
   - No special tinkerdown feature needed

4. **Should we auto-generate incident ID?**
   - Current: Use timestamp in filename
   - Alternative: Sequential ID (INC-001, INC-002)
   - Recommendation: Timestamp is sufficient, no central registry needed
