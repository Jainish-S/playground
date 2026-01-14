# URL Shortener Go - High Performance Multi-stage Build
# Build: docker build --target server -t url-shortener-server -f Dockerfile ../..
# Or:    docker build --target worker -t url-shortener-worker -f Dockerfile ../..

# ============================================
# Stage 1: Build Stage
# ============================================
FROM golang:1.24-alpine AS builder

# Install git for go mod download (some deps may need it)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go-common package first (for caching)
COPY packages/go-common/ ./packages/go-common/

# Copy go.mod and go.sum for dependency caching
COPY apps/url-shortener-go/go.mod apps/url-shortener-go/go.sum ./apps/url-shortener-go/

# Download dependencies (cached layer)
RUN cd apps/url-shortener-go && \
    GOWORK=off go mod download

# Copy source code
COPY apps/url-shortener-go/ ./apps/url-shortener-go/

# Build flags for smallest binary:
# - CGO_ENABLED=0: Static binary, no C dependencies
# - -ldflags="-s -w": Strip debug info and symbol table (~30% smaller)
# - -trimpath: Remove file paths from binary
# - GOWORK=off: Ignore go.work file in monorepo

# Build server binary
RUN cd apps/url-shortener-go && \
    GOWORK=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -trimpath \
    -o /server ./cmd/server

# Build worker binary
RUN cd apps/url-shortener-go && \
    GOWORK=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -trimpath \
    -o /worker ./cmd/worker

# ============================================
# Stage 2: Server Runtime (scratch = 0 bytes base!)
# ============================================
FROM scratch AS server

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /server /server

# Expose port
EXPOSE 8000

# Health check isn't available in scratch, handle at orchestration layer

# Run the server
ENTRYPOINT ["/server"]

# ============================================
# Stage 3: Worker Runtime (scratch = 0 bytes base!)
# ============================================
FROM scratch AS worker

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /worker /worker

# Run the worker
ENTRYPOINT ["/worker"]
