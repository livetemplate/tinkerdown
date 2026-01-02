# Runbook Incidents Example

> ⚠️ **NOTE**: This is a research example demonstrating the runbook instance pattern.
> These examples may not work with the current tinkerdown version.

This example demonstrates the **runbook instance** pattern:
- Templates are reusable procedures
- Instances are copies created for specific incidents
- The instance file becomes the incident record AND postmortem

## Directory Structure

```
runbook-incidents/
├── templates/
│   └── service-unhealthy.md    # Reusable runbook template
├── incidents/
│   └── 2024-01-15-1432-api-unhealthy.md   # Specific incident record
└── README.md
```

## The Workflow

### 1. Create Template (once)

```bash
# Write your runbook with live data sources
vim templates/service-unhealthy.md

# Test it
tinkerdown serve templates/service-unhealthy.md
```

### 2. Incident Starts → Create Instance

```bash
# Future: tinkerdown incident new templates/service-unhealthy.md
# For now: copy manually
cp templates/service-unhealthy.md incidents/$(date +%Y-%m-%d-%H%M)-api-unhealthy.md
```

### 3. During Incident

```bash
# Open in browser
tinkerdown serve incidents/2024-01-15-1432-api-unhealthy.md

# Follow steps, log actions, capture snapshots
# All changes saved to the file
```

### 4. Resolve Incident

Add resolution summary and action items to the file.

### 5. Commit to Git

```bash
git add incidents/2024-01-15-1432-api-unhealthy.md
git commit -m "Incident INC-2024-0115: API unhealthy (resolved)"
git push
```

## What Makes This Different

| Traditional Approach | Tinkerdown Approach |
|---------------------|---------------------|
| Runbook in Confluence | Template in git |
| Incident in PagerDuty | Instance file in git |
| Postmortem in Notion | Same instance file |
| 3 separate tools | 1 markdown file |
| Vendor lock-in | Portable markdown |
| Can't search history | `grep` just works |

## Try It

```bash
# See the template (live system data)
tinkerdown serve templates/service-unhealthy.md

# See a resolved incident (historical record)
tinkerdown serve incidents/2024-01-15-1432-api-unhealthy.md
```

## Future Features Needed

1. **`tinkerdown incident new`** - Create instance from template
2. **Snapshot capture** - Button to save exec output at a point in time
3. **Execution log component** - Form to log timestamped actions
4. **Step status tracking** - Checkboxes with auto-timestamps

See [runbook-instances-design.md](../../docs/plans/2025-12-30-runbook-instances-design.md) for full design.
