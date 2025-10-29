#!/bin/bash

# Simple container structure test script for Makefile integration
# This script is called by the Makefile with proper dependency tracking

set -euo pipefail

# Configuration from environment or defaults
CONTAINER_IMAGE=${CONTAINER_IMAGE:-darrot:test}
TEST_CONFIG=${TEST_CONFIG:-tests/container/structure-test.yaml}
RESULTS_DIR=${RESULTS_DIR:-tests/results}
RESULTS_FILE="$RESULTS_DIR/container-structure-test-results.json"
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

# Create results directory
mkdir -p "$RESULTS_DIR"

# Check if container-structure-test is available
if ! command -v container-structure-test &> /dev/null; then
    log_error "container-structure-test is not installed"
    log_info "Run 'make container-test-install' to install it"
    exit 1
fi

# Check if test configuration exists
if [[ ! -f "$TEST_CONFIG" ]]; then
    log_error "Test configuration file not found: $TEST_CONFIG"
    exit 1
fi

log_info "Running container structure tests on $CONTAINER_IMAGE"

# Handle Podman vs Docker
if [[ "$CONTAINER_RUNTIME" == "podman" ]]; then
    # For Podman, we need to start the Docker-compatible API service
    PODMAN_SOCKET="/tmp/podman-$$.sock"
    
    log_info "Starting Podman service for container-structure-test..."
    podman system service --time=30 "unix://$PODMAN_SOCKET" &
    PODMAN_PID=$!
    
    # Wait for service to start
    sleep 3
    
    # Set Docker host for container-structure-test
    export DOCKER_HOST="unix://$PODMAN_SOCKET"
    
    # Cleanup function
    cleanup_podman() {
        if [[ -n "${PODMAN_PID:-}" ]]; then
            kill "$PODMAN_PID" 2>/dev/null || true
            wait "$PODMAN_PID" 2>/dev/null || true
        fi
        rm -f "$PODMAN_SOCKET"
        unset DOCKER_HOST
    }
    
    trap cleanup_podman EXIT
fi

# Run the tests
if container-structure-test test \
    --image "$CONTAINER_IMAGE" \
    --config "$TEST_CONFIG" \
    --output json \
    --test-report "$RESULTS_FILE"; then
    log_info "Container structure tests passed"
    exit_code=0
else
    log_error "Container structure tests failed"
    exit_code=1
fi

# Display results summary if jq is available
if command -v jq &> /dev/null && [[ -f "$RESULTS_FILE" ]]; then
    echo
    log_info "Test Results Summary:"
    
    total=$(jq '.Results | length' "$RESULTS_FILE")
    passed=$(jq '[.Results[] | select(.Pass == true)] | length' "$RESULTS_FILE")
    failed=$(jq '[.Results[] | select(.Pass == false)] | length' "$RESULTS_FILE")
    
    echo "  Total tests: $total"
    echo "  Passed: $passed"
    echo "  Failed: $failed"
    
    if [[ "$failed" -gt 0 ]]; then
        echo
        log_error "Failed tests:"
        jq -r '.Results[] | select(.Pass == false) | "  - \(.Name): \(.Errors // ["Unknown error"] | join(", "))"' "$RESULTS_FILE"
    fi
fi

exit $exit_code