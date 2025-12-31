---
title: Service Unhealthy Runbook
type: runbook-template
version: 1.0
description: Steps to diagnose and recover an unhealthy service
sources:
  # These show LIVE status when the runbook is run
  docker_ps:
    type: exec
    cmd: docker ps --format '{"name":"{{.Names}}","status":"{{.Status}}","ports":"{{.Ports}}"}' 2>/dev/null | jq -s '.' || echo '[]'

  docker_logs:
    type: exec
    cmd: docker logs --tail 20 $(docker ps -q | head -1) 2>&1 | jq -R -s 'split("\n") | map(select(length > 0)) | map({line: .})' || echo '[{"line":"No containers"}]'

  disk_usage:
    type: exec
    cmd: df -h / | tail -1 | awk '{print "[{\"filesystem\":\""$1"\",\"used\":\""$3"\",\"available\":\""$4"\",\"percent\":\""$5"\"}]"}'

  memory:
    type: exec
    cmd: free -h 2>/dev/null | awk 'NR==2{print "[{\"total\":\""$2"\",\"used\":\""$3"\",\"available\":\""$7"\"}]"}' || echo '[{"total":"N/A","used":"N/A","available":"Run on Linux"}]'
---

# Service Unhealthy Runbook

Use this runbook when a service is reporting unhealthy or unresponsive.

---

## Step 1: Identify the Service

**What service is affected?**

Check which containers are running:

```lvt
<table lvt-source="docker_ps" lvt-columns="name:Container,status:Status,ports:Ports" lvt-empty="No containers running">
</table>
```

**Checklist:**
- [ ] Identified affected service
- [ ] Verified service should be running
- [ ] Checked if other services are affected

---

## Step 2: Check Resource Usage

### Disk Space

```lvt
<table lvt-source="disk_usage" lvt-columns="filesystem:Filesystem,used:Used,available:Available,percent:Usage">
</table>
```

⚠️ **If usage > 90%:** Jump to [Disk Full Runbook](./disk-full.md)

### Memory

```lvt
<table lvt-source="memory" lvt-columns="total:Total,used:Used,available:Available">
</table>
```

⚠️ **If available < 500MB:** Consider restarting low-priority services

---

## Step 3: Check Logs

Recent logs from first container:

```lvt
<ul lvt-source="docker_logs" lvt-field="line" lvt-empty="No logs available">
</ul>
```

**Look for:**
- Connection errors
- Out of memory errors
- Dependency failures
- Configuration errors

---

## Step 4: Restart Service

```bash
# Restart specific container
docker restart <container_name>

# Or restart via compose
docker compose restart <service_name>
```

Wait 30 seconds, then verify in Step 5.

---

## Step 5: Verify Recovery

Re-check:
1. Container status (Step 1)
2. Application health endpoint
3. Logs show normal startup

```bash
# Check health endpoint
curl -f http://localhost:8080/health
```

---

## Step 6: If Still Unhealthy

Escalation path:
1. Check dependent services (database, cache)
2. Review recent deployments
3. Page senior on-call

---

## Notes

_Add any service-specific notes here_
