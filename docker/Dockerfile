# Stage 1: Go builder
FROM golang:1.25-bookworm AS builder

WORKDIR /src

# Install Node.js for Vue frontend
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    apt-get install -y nodejs

# Cache Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build frontend
WORKDIR /src/web
RUN npm install && npx vite build

# Build backend (Go binary)
WORKDIR /src
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o lightboot ./cmd/lightboot/

# Stage 2: Runtime
FROM debian:bookworm-slim

# Runtime dependencies (xorriso for ISO extraction fallback)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    xorriso \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary
COPY --from=builder /src/lightboot /app/lightboot

# Copy profiles and bootfiles for runtime use
COPY --from=builder /src/internal/core/profiles /app/profiles
COPY --from=builder /src/bootfiles /app/bootfiles

# Create runtime directories
RUN mkdir -p /app/data /app/iso /app/cache

# Environment defaults (can be overridden at runtime)
ENV LIGHTBOOT_HTTP_PORT=8080 \
    LIGHTBOOT_HTTP_ADDRESS=0.0.0.0 \
    LIGHTBOOT_TFTP_PORT=69 \
    LIGHTBOOT_ISO_DIR=/app/iso \
    LIGHTBOOT_CACHE_DIR=/app/cache \
    LIGHTBOOT_DATA_DIR=/app/data \
    LIGHTBOOT_LOG_LEVEL=info

# Expose ports
# HTTP (web UI + boot files) -> 8080
# TFTP -> 69/udp
# DHCP proxy -> 67/udp
EXPOSE 8080/tcp
EXPOSE 69/udp
EXPOSE 67/udp

VOLUME ["/app/iso", "/app/cache", "/app/data"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/app/lightboot"]