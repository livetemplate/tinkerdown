# Deployment

Deploy your Tinkerdown app to production.

## Development vs Production

| Mode | Command | Use Case |
|------|---------|----------|
| Development | `tinkerdown serve` | Local development with hot reload |
| Production | `tinkerdown serve --production` | Optimized for production |

## Deployment Options

### Docker

Create a `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o tinkerdown ./cmd/tinkerdown

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/tinkerdown .
COPY . .

EXPOSE 8080
CMD ["./tinkerdown", "serve", "--production", "--port", "8080"]
```

Build and run:

```bash
docker build -t myapp .
docker run -p 8080:8080 myapp
```

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      - DATABASE_URL=sqlite:///app/data/app.db
```

### Fly.io

```bash
# Install flyctl
fly launch
fly deploy
```

### Railway / Render

These platforms auto-detect Go apps. Just connect your repository.

## Environment Variables

Use environment variables for sensitive configuration:

```yaml
# tinkerdown.yaml
sources:
  api:
    type: rest
    url: ${API_URL}
    headers:
      Authorization: Bearer ${API_TOKEN}
```

Set in your deployment:

```bash
export API_URL=https://api.example.com
export API_TOKEN=your-secret-token
```

## Database Persistence

For SQLite databases, ensure the database file is persisted:

```yaml
# docker-compose.yml
volumes:
  - ./data:/app/data
```

Or use a managed database for production.

## SSL/TLS

### Behind a Reverse Proxy (Recommended)

Use nginx, Caddy, or your cloud provider's load balancer:

```nginx
server {
    listen 443 ssl;
    server_name myapp.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Health Checks

Configure health checks for container orchestration:

```yaml
# docker-compose.yml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
  interval: 30s
  timeout: 10s
  retries: 3
```

## Scaling

Tinkerdown maintains WebSocket connections per page. For horizontal scaling:

1. Use sticky sessions for WebSocket connections
2. Use a shared database (PostgreSQL) instead of SQLite
3. Consider Redis for session storage

## Next Steps

- [Configuration Reference](../reference/config.md) - Production settings
- [Error Handling](../error-handling.md) - Reliability in production
