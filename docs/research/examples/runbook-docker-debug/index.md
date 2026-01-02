---
title: "Docker Debug Runbook"
sources:
  containers:
    type: exec
    cmd: docker ps -a --format '{"id":"{{.ID}}","name":"{{.Names}}","image":"{{.Image}}","status":"{{.Status}}","ports":"{{.Ports}}","state":"{{.State}}"}' 2>/dev/null | jq -s '.' || echo '[]'

  unhealthy:
    type: exec
    cmd: |
      docker ps -a --filter "health=unhealthy" --format '{"name":"{{.Names}}","status":"{{.Status}}"}' 2>/dev/null |
      jq -s 'if length == 0 then [{"name":"✓","status":"All containers healthy"}] else . end'

  exited:
    type: exec
    cmd: |
      docker ps -a --filter "status=exited" --format '{"name":"{{.Names}}","image":"{{.Image}}","status":"{{.Status}}"}' 2>/dev/null |
      jq -s 'if length == 0 then [{"name":"✓","image":"—","status":"No exited containers"}] else . end'

  resource_usage:
    type: exec
    cmd: |
      docker stats --no-stream --format '{"name":"{{.Name}}","cpu":"{{.CPUPerc}}","mem":"{{.MemUsage}}","net":"{{.NetIO}}"}' 2>/dev/null |
      jq -s 'if length == 0 then [{"name":"—","cpu":"—","mem":"No running containers","net":"—"}] else . end'

  disk_usage:
    type: exec
    cmd: |
      docker system df --format '{"type":"{{.Type}}","total":"{{.TotalCount}}","size":"{{.Size}}","reclaimable":"{{.Reclaimable}}"}' 2>/dev/null |
      jq -s '.' || echo '[{"type":"Error","total":"—","size":"Docker not running","reclaimable":"—"}]'

  networks:
    type: exec
    cmd: docker network ls --format '{"name":"{{.Name}}","driver":"{{.Driver}}","scope":"{{.Scope}}"}' 2>/dev/null | jq -s '.' || echo '[]'

  volumes:
    type: exec
    cmd: |
      docker volume ls --format '{"name":"{{.Name}}","driver":"{{.Driver}}"}' 2>/dev/null |
      head -10 |
      jq -s 'if length == 0 then [{"name":"—","driver":"No volumes"}] else . end'

  recent_logs:
    type: exec
    cmd: |
      container=$(docker ps -q 2>/dev/null | head -1)
      if [ -n "$container" ]; then
        name=$(docker inspect --format '{{.Name}}' $container | sed 's/\///')
        docker logs --tail 10 $container 2>&1 | jq -R -s --arg name "$name" 'split("\n") | map(select(length > 0)) | map({container: $name, line: .}) | if length == 0 then [{container: $name, line: "No recent logs"}] else .[:10] end'
      else
        echo '[{"container":"—","line":"No running containers"}]'
      fi

  dangling_images:
    type: exec
    cmd: |
      docker images -f "dangling=true" --format '{"id":"{{.ID}}","size":"{{.Size}}","created":"{{.CreatedSince}}"}' 2>/dev/null |
      jq -s 'if length == 0 then [{"id":"✓","size":"—","created":"No dangling images"}] else . end'

  compose_services:
    type: exec
    cmd: |
      if [ -f docker-compose.yml ] || [ -f docker-compose.yaml ] || [ -f compose.yml ]; then
        docker compose ps --format '{"service":"{{.Service}}","status":"{{.Status}}","ports":"{{.Ports}}"}' 2>/dev/null |
        jq -s 'if length == 0 then [{"service":"—","status":"No services running","ports":"—"}] else . end'
      else
        echo '[{"service":"—","status":"No compose file found","ports":"—"}]'
      fi
---

# Docker Debug Runbook

Quick diagnostics for Docker issues in development.

---

## Container Overview

### All Containers

```lvt
<table lvt-source="containers" lvt-columns="name:Name,image:Image,state:State,status:Status" lvt-empty="No containers found">
</table>
```

### Unhealthy Containers

```lvt
<table lvt-source="unhealthy" lvt-columns="name:Container,status:Status">
</table>
```

### Exited Containers

```lvt
<table lvt-source="exited" lvt-columns="name:Container,image:Image,status:Exit Status">
</table>
```

**Troubleshoot exited containers:**
```bash
docker logs <container_name>
docker inspect <container_name> | jq '.[0].State'
```

---

## Resource Usage

### Live CPU & Memory

```lvt
<table lvt-source="resource_usage" lvt-columns="name:Container,cpu:CPU,mem:Memory,net:Network I/O">
</table>
```

### Disk Usage

```lvt
<table lvt-source="disk_usage" lvt-columns="type:Type,total:Count,size:Size,reclaimable:Reclaimable">
</table>
```

**Reclaim space:**
```bash
# Remove unused containers, networks, images
docker system prune

# Also remove unused volumes (⚠️ data loss)
docker system prune --volumes

# Nuclear option - remove everything
docker system prune -a --volumes
```

---

## Docker Compose Services

```lvt
<table lvt-source="compose_services" lvt-columns="service:Service,status:Status,ports:Ports" lvt-empty="No compose file in current directory">
</table>
```

**Common compose commands:**
```bash
docker compose up -d          # Start services
docker compose down           # Stop services
docker compose down -v        # Stop + remove volumes
docker compose logs -f        # Follow all logs
docker compose restart <svc>  # Restart one service
```

---

## Networks & Volumes

### Networks

```lvt
<table lvt-source="networks" lvt-columns="name:Name,driver:Driver,scope:Scope">
</table>
```

### Volumes

```lvt
<table lvt-source="volumes" lvt-columns="name:Volume Name,driver:Driver">
</table>
```

---

## Recent Logs (First Running Container)

```lvt
<ul lvt-source="recent_logs" lvt-field="line" lvt-empty="No logs available">
</ul>
```

**Get logs for specific container:**
```bash
docker logs -f --tail 100 <container_name>
```

---

## Cleanup: Dangling Images

```lvt
<table lvt-source="dangling_images" lvt-columns="id:Image ID,size:Size,created:Created">
</table>
```

**Remove dangling images:**
```bash
docker image prune
```

---

## Quick Reference

| Problem | Solution |
|---------|----------|
| Container won't start | `docker logs <name>` then `docker inspect <name>` |
| Port already in use | `docker ps -a` then `docker rm -f <conflicting>` |
| Out of disk space | `docker system prune -a --volumes` |
| Can't connect to container | `docker network inspect bridge` |
| Container keeps restarting | Check logs, then `docker update --restart=no <name>` |
| Need a shell inside | `docker exec -it <name> /bin/sh` |
| Rebuild without cache | `docker compose build --no-cache` |
| Reset everything | `docker compose down -v && docker compose up -d --build` |
