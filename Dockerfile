FROM golang:1.24.4-alpine AS builder

# Install git and ca-certificates (needed for dependencies)
RUN apk add --no-cache git ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main cmd/api/main.go

# Production stage
FROM alpine:3.20.1 AS prod

# Install necessary runtime dependencies including shell and curl
RUN apk add --no-cache ca-certificates tzdata bash curl

# Create non-root user in alpine
RUN adduser -D -g '' appuser

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /app/main /main

USER appuser

# Expose port (informational)
EXPOSE 8080

# Health check via HTTP endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -fsS "http://localhost:${PORT:-8080}/health" || exit 1

# Run the binary
ENTRYPOINT ["/main"]





