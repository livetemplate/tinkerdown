---
title: "Dev Environment Health Check"
sources:
  disk:
    type: exec
    cmd: df -h / /tmp 2>/dev/null | tail -n +2 | awk '{print "{\"filesystem\":\""$1"\",\"size\":\""$2"\",\"used\":\""$3"\",\"avail\":\""$4"\",\"use_pct\":\""$5"\",\"mount\":\""$6"\"}"}' | jq -s '.'

  memory:
    type: exec
    cmd: |
      if command -v free >/dev/null 2>&1; then
        free -h | awk 'NR==2{print "{\"total\":\""$2"\",\"used\":\""$3"\",\"free\":\""$4"\",\"available\":\""$7"\"}"}' | jq -s '.'
      else
        vm_stat | awk -v pagesize=$(pagesize) '
          /Pages free/ {free=$3}
          /Pages active/ {active=$3}
          /Pages inactive/ {inactive=$3}
          /Pages wired/ {wired=$4}
          END {
            gsub(/\./,"",free); gsub(/\./,"",active); gsub(/\./,"",inactive); gsub(/\./,"",wired);
            total=(free+active+inactive+wired)*pagesize/1073741824;
            used=(active+wired)*pagesize/1073741824;
            printf "[{\"total\":\"%.1fG\",\"used\":\"%.1fG\",\"free\":\"%.1fG\",\"available\":\"%.1fG\"}]", total, used, free, free
          }'
      fi

  ports:
    type: exec
    cmd: |
      (lsof -i -P -n 2>/dev/null || ss -tlnp 2>/dev/null || netstat -tlnp 2>/dev/null) |
      grep -E ':(3000|3001|5000|5173|8000|8080|5432|3306|6379|27017) ' |
      head -10 |
      awk '{
        split($0, a, " ");
        port=""; proc=""; pid="";
        for(i=1;i<=NF;i++) {
          if(a[i] ~ /:[0-9]+/) { gsub(/.*:/, "", a[i]); port=a[i] }
          if(a[i] ~ /^[0-9]+$/ && pid=="") pid=a[i]
        }
        if($NF ~ /[a-zA-Z]/) proc=$NF
        if(port != "") print "{\"port\":\""port"\",\"process\":\""proc"\",\"pid\":\""pid"\"}"
      }' | jq -s 'if length == 0 then [{"port":"—","process":"No services on common ports","pid":"—"}] else . end'

  docker_containers:
    type: exec
    cmd: |
      if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
        docker ps --format '{"name":"{{.Names}}","image":"{{.Image}}","status":"{{.Status}}","ports":"{{.Ports}}"}' | jq -s 'if length == 0 then [{"name":"—","image":"No containers running","status":"—","ports":"—"}] else . end'
      else
        echo '[{"name":"—","image":"Docker not available","status":"—","ports":"—"}]'
      fi

  node_processes:
    type: exec
    cmd: |
      ps aux 2>/dev/null | grep -E 'node|npm|yarn|pnpm' | grep -v grep | head -5 |
      awk '{printf "{\"pid\":\"%s\",\"cpu\":\"%s\",\"mem\":\"%s\",\"command\":\"%s\"}\n", $2, $3, $4, $11}' |
      jq -s 'if length == 0 then [{"pid":"—","cpu":"—","mem":"—","command":"No Node processes"}] else . end'

  recent_errors:
    type: exec
    cmd: |
      if [ -f /var/log/syslog ]; then
        grep -i error /var/log/syslog 2>/dev/null | tail -5 |
        jq -R -s 'split("\n") | map(select(length > 0)) | map({line: .}) | if length == 0 then [{line: "No recent errors"}] else . end'
      elif [ -f /var/log/system.log ]; then
        grep -i error /var/log/system.log 2>/dev/null | tail -5 |
        jq -R -s 'split("\n") | map(select(length > 0)) | map({line: .}) | if length == 0 then [{line: "No recent errors"}] else . end'
      else
        echo '[{"line":"No system log available"}]'
      fi

  git_status:
    type: exec
    cmd: |
      if git rev-parse --git-dir >/dev/null 2>&1; then
        branch=$(git branch --show-current 2>/dev/null || echo "detached")
        dirty=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')
        ahead=$(git rev-list --count @{u}..HEAD 2>/dev/null || echo "0")
        behind=$(git rev-list --count HEAD..@{u} 2>/dev/null || echo "0")
        echo "[{\"branch\":\"$branch\",\"uncommitted\":\"$dirty files\",\"ahead\":\"$ahead\",\"behind\":\"$behind\"}]"
      else
        echo '[{"branch":"—","uncommitted":"Not a git repo","ahead":"—","behind":"—"}]'
      fi
---

# Dev Environment Health Check

A quick runbook to diagnose common development environment issues.

---

## 1. Disk Space

Low disk space causes builds to fail, Docker to crash, and npm install to hang.

```lvt
<table lvt-source="disk" lvt-columns="mount:Mount,size:Size,used:Used,avail:Available,use_pct:Usage">
</table>
```

**Action needed if:** Usage > 90%. Run `docker system prune` or clear `node_modules`.

---

## 2. Memory Usage

```lvt
<table lvt-source="memory" lvt-columns="total:Total,used:Used,free:Free,available:Available">
</table>
```

**Action needed if:** Available < 1GB. Close Chrome tabs or restart Docker.

---

## 3. Common Dev Ports

Check what's running on ports 3000, 5000, 5173, 8000, 8080, and databases.

```lvt
<table lvt-source="ports" lvt-columns="port:Port,process:Process,pid:PID" lvt-empty="No services detected on common ports">
</table>
```

**Common conflicts:**
- Port 3000: React/Next.js dev servers
- Port 5173: Vite
- Port 8080: Various backends
- Port 5432: PostgreSQL

---

## 4. Docker Containers

```lvt
<table lvt-source="docker_containers" lvt-columns="name:Name,image:Image,status:Status" lvt-empty="No containers running">
</table>
```

**Troubleshooting:**
- Container stuck? `docker restart <name>`
- Need logs? `docker logs -f <name>`

---

## 5. Node.js Processes

Zombie Node processes can hold ports and consume memory.

```lvt
<table lvt-source="node_processes" lvt-columns="pid:PID,cpu:CPU%,mem:MEM%,command:Command" lvt-empty="No Node processes running">
</table>
```

**To kill a stuck process:** `kill -9 <PID>`

---

## 6. Git Status

```lvt
<table lvt-source="git_status" lvt-columns="branch:Branch,uncommitted:Uncommitted,ahead:Ahead,behind:Behind">
</table>
```

---

## 7. Quick Fixes

Common commands to resolve issues:

| Problem | Command |
|---------|---------|
| Port already in use | `lsof -ti:3000 \| xargs kill -9` |
| Docker eating disk | `docker system prune -a` |
| node_modules bloat | `find . -name node_modules -type d -prune -exec rm -rf {} +` |
| Clear npm cache | `npm cache clean --force` |
| Reset Docker | `docker-compose down -v && docker-compose up -d` |

---

## 8. System Logs (Recent Errors)

```lvt
<ul lvt-source="recent_errors" lvt-field="line" lvt-empty="No recent errors found">
</ul>
```
