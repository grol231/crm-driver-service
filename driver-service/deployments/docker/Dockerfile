# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed to be able to call HTTPS)
RUN apk --no-cache add ca-certificates git tzdata
WORKDIR /build

# Copy go.mod and go.sum for dependency download
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o driver-service \
    ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS calls
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /build/driver-service .

# Copy migrations
COPY --from=builder /build/internal/infrastructure/database/migrations ./internal/infrastructure/database/migrations

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8001 9001 9002

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8001/health || exit 1

# Run the binary
CMD ["./driver-service"]