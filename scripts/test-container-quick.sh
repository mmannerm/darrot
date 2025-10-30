#!/bin/bash

# Quick container validation script
# Tests basic container functionality without full structure tests

set -euo pipefail

# Configuration
CONTAINER_IMAGE=${CONTAINER_IMAGE:-darrot:test}
CONTAINER_RUNTIME=${CONTAINER_RUNTIME:-podman}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check if container runtime is available
if ! command -v "$CONTAINER_RUNTIME" &> /dev/null; then
    log_error "$CONTAINER_RUNTIME is not installed or not in PATH"
    exit 1
fi

log_info "Running quick container validation for $CONTAINER_IMAGE"

# Test 1: Check if image exists
log_info "Checking if container image exists..."
if ! $CONTAINER_RUNTIME image exists "$CONTAINER_IMAGE"; then
    log_error "Container image $CONTAINER_IMAGE does not exist"
    log_info "Run 'make container-build' to build the image"
    exit 1
fi

# Test 2: Check if container can start and show version
log_info "Testing container startup and version command..."
if ! $CONTAINER_RUNTIME run --rm "$CONTAINER_IMAGE" version; then
    log_error "Container failed to start or version command failed"
    exit 1
fi

# Test 3: Check if binary exists and is executable
log_info "Checking if darrot binary exists and is executable..."
if ! $CONTAINER_RUNTIME run --rm --entrypoint /bin/sh "$CONTAINER_IMAGE" -c "ls -la /app/darrot"; then
    log_error "darrot binary not found or not accessible"
    exit 1
fi

# Test 4: Check if required libraries are present
log_info "Checking for required audio libraries..."
if ! $CONTAINER_RUNTIME run --rm --entrypoint /bin/sh "$CONTAINER_IMAGE" -c "ls /usr/lib/libopus.so.0"; then
    log_warn "libopus.so.0 not found - audio functionality may not work"
fi

if ! $CONTAINER_RUNTIME run --rm --entrypoint /bin/sh "$CONTAINER_IMAGE" -c "ls /usr/lib/libopusfile.so.0"; then
    log_warn "libopusfile.so.0 not found - audio functionality may not work"
fi

# Test 5: Check if container runs as non-root user
log_info "Checking if container runs as non-root user..."
USER_ID=$($CONTAINER_RUNTIME run --rm --entrypoint /bin/sh "$CONTAINER_IMAGE" -c "id -u")
if [[ "$USER_ID" == "0" ]]; then
    log_warn "Container is running as root user (security concern)"
else
    log_info "Container is running as user ID: $USER_ID"
fi

# Test 6: Check if data directory exists and is writable
log_info "Checking data directory permissions..."
if ! $CONTAINER_RUNTIME run --rm --entrypoint /bin/sh "$CONTAINER_IMAGE" -c "test -w /app/data"; then
    log_error "Data directory /app/data is not writable"
    exit 1
fi

log_info "Quick container validation completed successfully!"
log_info "All basic functionality tests passed"

exit 0