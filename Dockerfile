# Multi-stage build for minimal container size
FROM docker.io/golang:1.23-alpine AS builder

# Install build dependencies for Opus audio library
RUN apk add --no-cache \
    gcc \
    musl-dev \
    opus-dev \
    opusfile-dev \
    pkgconfig

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -X main.version=container -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o darrot ./cmd/darrot

# Final stage - minimal runtime image
FROM docker.io/alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    opus \
    opusfile \
    ca-certificates \
    tzdata

# Create non-root user for security
RUN addgroup -g 1001 -S darrot && \
    adduser -u 1001 -S darrot -G darrot

# Create directories with proper permissions
RUN mkdir -p /app/data && \
    chown -R darrot:darrot /app

# Copy binary from builder stage
COPY --from=builder /app/darrot /app/darrot

# Switch to non-root user
USER darrot

# Set working directory
WORKDIR /app

# Expose health check port (if needed in future)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep darrot || exit 1

# Run the application
ENTRYPOINT ["/app/darrot"]