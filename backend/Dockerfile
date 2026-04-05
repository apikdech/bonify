# Multi-stage build for Receipt Manager Backend
# Stage 1: Builder
FROM golang:1.26.1-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build both binaries with optimizations
# CGO_ENABLED=0 is required for static binary (distroless compatibility)
# -ldflags="-w -s" strips debug info to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o bin/api \
    ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o bin/worker \
    ./cmd/worker

# Stage 2: Final - Distroless Static
# Using Google's distroless static image for minimal attack surface
FROM gcr.io/distroless/static:nonroot

# Copy CA certificates for HTTPS requests (needed for LLM APIs, webhooks, etc.)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data for proper time handling
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binaries from builder
COPY --from=builder /build/bin/api /usr/local/bin/api
COPY --from=builder /build/bin/worker /usr/local/bin/worker

# Use nonroot user (UID 65532) for security
USER nonroot:nonroot

# Expose API port (only relevant when running API mode)
EXPOSE 8080

# Default entrypoint - can be overridden in docker-compose
# Usage:
#   docker run <image> api     -> runs API server
#   docker run <image> worker  -> runs Temporal worker
ENTRYPOINT ["/usr/local/bin/api"]
