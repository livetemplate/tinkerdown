---
title: System Status
sources:
  disk:
    type: exec
    cmd: df -h / | tail -1 | awk '{print "[{\"filesystem\":\""$1"\",\"size\":\""$2"\",\"used\":\""$3"\",\"available\":\""$4"\",\"use_percent\":\""$5"\"}]"}'

  memory:
    type: exec
    cmd: |
      if command -v free >/dev/null 2>&1; then
        free -h | awk 'NR==2{print "[{\"total\":\""$2"\",\"used\":\""$3"\",\"free\":\""$4"\"}]"}'
      else
        echo '[{"total":"N/A","used":"N/A","free":"Use Linux for memory stats"}]'
      fi

  processes:
    type: exec
    cmd: ps aux --sort=-%cpu 2>/dev/null | head -6 | tail -5 | awk '{print "{\"user\":\""$1"\",\"pid\":\""$2"\",\"cpu\":\""$3"\",\"mem\":\""$4"\",\"command\":\""$11"\"}"}' | jq -s '.' || echo '[{"user":"N/A","pid":"N/A","cpu":"N/A","mem":"N/A","command":"ps not available"}]'
---

# System Status

Real-time system information from your machine.

## Disk Usage

```lvt
<table lvt-source="disk" lvt-columns="filesystem:Filesystem,size:Size,used:Used,available:Available,use_percent:Usage">
</table>
```

## Memory

```lvt
<table lvt-source="memory" lvt-columns="total:Total,used:Used,free:Free">
</table>
```

## Top Processes (by CPU)

```lvt
<table lvt-source="processes" lvt-columns="user:User,pid:PID,cpu:CPU%,mem:MEM%,command:Command" lvt-empty="Could not fetch processes">
</table>
```

---

*Refresh the page to update stats.*
