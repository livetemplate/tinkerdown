# Build stage 1: Build TypeScript client
FROM node:20-alpine AS client-builder

WORKDIR /app/client
COPY client/package*.json ./
RUN npm ci --prefer-offline --no-audit

COPY client/ ./
RUN npm run build

# Build stage 2: Build Go binary
FROM golang:1.24-alpine AS go-builder

# Install git for version info and ca-certificates for HTTPS
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Allow Go to download required toolchain version automatically
ENV GOTOOLCHAIN=auto

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built client assets from previous stage
COPY --from=client-builder /app/client/dist/ ./client/dist/
RUN mkdir -p internal/assets/client && \
    cp client/dist/tinkerdown-client.browser.* internal/assets/client/

# Build with optimizations and version info
ARG VERSION=dev
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o tinkerdown ./cmd/tinkerdown

# Runtime stage: Minimal Alpine image
FROM alpine:3.21

# Add ca-certificates for HTTPS requests and tzdata for timezone support
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=go-builder /app/tinkerdown /usr/local/bin/tinkerdown

# Create non-root user for security
RUN adduser -D -u 1000 tinkerdown && \
    chown -R tinkerdown:tinkerdown /app

USER tinkerdown

EXPOSE 8080

ENTRYPOINT ["tinkerdown"]
CMD ["serve", "--host", "0.0.0.0", "/app"]
