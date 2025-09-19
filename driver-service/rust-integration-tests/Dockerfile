# Multi-stage Dockerfile for Rust Integration Tests

# Build stage
FROM rust:1.75-slim-bullseye AS builder

# Install system dependencies
RUN apt-get update && apt-get install -y \
    pkg-config \
    libssl-dev \
    libpq-dev \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Create app directory
WORKDIR /app

# Copy manifests
COPY Cargo.toml Cargo.lock ./

# Create a dummy main.rs to build dependencies
RUN mkdir src && \
    echo "fn main() {}" > src/main.rs

# Build dependencies
RUN cargo build --release && \
    rm -rf src

# Copy source code
COPY src ./src

# Build application
RUN touch src/main.rs && \
    cargo build --release

# Runtime stage
FROM debian:bullseye-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    libssl1.1 \
    libpq5 \
    curl \
    postgresql-client \
    redis-tools \
    && rm -rf /var/lib/apt/lists/*

# Create app user
RUN useradd -m -u 1000 -s /bin/bash testuser

# Create app directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/target/release/test_runner /usr/local/bin/integration-tests

# Copy test configuration
COPY .env.example .env.example

# Create directories for test results
RUN mkdir -p /app/test-results && \
    chown -R testuser:testuser /app

# Switch to non-root user
USER testuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command
CMD ["integration-tests", "--mode", "all", "--output", "json"]

# Add labels
LABEL org.opencontainers.image.title="Driver Service Integration Tests"
LABEL org.opencontainers.image.description="Comprehensive integration tests for Driver Service written in Rust"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.source="https://github.com/your-org/driver-service"