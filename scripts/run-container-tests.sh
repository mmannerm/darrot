#!/bin/bash

# Script to run container structure tests locally

set -euo pipefail

# Configuration
CONTAINER_IMAGE="darrot:test"
TEST_CONFIG="tests/container/structure-test.yaml"
RESULTS_DIR="tests/results"
RESULTS_FILE="$RESULTS_DIR/container-structure-test-results.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Determine container runtime
    CONTAINER_RUNTIME=${CONTAINER_RUNTIME:-podman}
    
    # Check if container runtime is available
    if ! command -v "$CONTAINER_RUNTIME" &> /dev/null; then
        log_error "$CONTAINER_RUNTIME is not installed or not in PATH"
        if [[ "$CONTAINER_RUNTIME" == "podman" ]]; then
            log_info "Trying to fall back to docker..."
            if command -v docker &> /dev/null; then
                CONTAINER_RUNTIME="docker"
                log_info "Using docker as container runtime"
            else
                log_error "Neither podman nor docker is available"
                exit 1
            fi
        else
            exit 1
        fi
    fi
    
    log_info "Using container runtime: $CONTAINER_RUNTIME"
    
    # Check if container-structure-test is available
    if ! command -v container-structure-test &> /dev/null; then
        log_error "container-structure-test is not installed"
        log_info "Run 'make container-test-install' or './scripts/install-container-structure-test.sh' to install it"
        exit 1
    fi
    
    # Check if test configuration exists
    if [[ ! -f "$TEST_CONFIG" ]]; then
        log_error "Test configuration file not found: $TEST_CONFIG"
        exit 1
    fi
    
    log_info "Prerequisites check passed"
}

# Build container image
build_container() {
    log_info "Building container image: $CONTAINER_IMAGE"
    
    if ! "$CONTAINER_RUNTIME" build -t "$CONTAINER_IMAGE" .; then
        log_error "Failed to build container image"
        exit 1
    fi
    
    log_info "Container image built successfully"
}

# Start Podman service if using Podman
start_podman_service() {
    if [[ "$CONTAINER_RUNTIME" == "podman" ]]; then
        log_info "Starting Podman Docker-compatible API service..."
        
        # Use a temporary socket file
        PODMAN_SOCKET="/tmp/podman-$$.sock"
        
        # Check if service is already running
        if pgrep -f "podman.*system.*service" > /dev/null; then
            log_info "Podman service already running"
            return 0
        fi
        
        # Start Podman service in background
        podman system service --time=0 "unix://$PODMAN_SOCKET" &
        PODMAN_SERVICE_PID=$!
        
        # Wait a moment for service to start
        sleep 3
        
        # Verify service is running and socket exists
        if [[ ! -S "$PODMAN_SOCKET" ]]; then
            log_error "Failed to start Podman service - socket not created"
            return 1
        fi
        
        # Set Docker host for container-structure-test
        export DOCKER_HOST="unix://$PODMAN_SOCKET"
        
        log_info "Podman service started successfully at $PODMAN_SOCKET"
    fi
}

# Stop Podman service if we started it
stop_podman_service() {
    if [[ -n "${PODMAN_SERVICE_PID:-}" ]]; then
        log_info "Stopping Podman service..."
        kill "$PODMAN_SERVICE_PID" 2>/dev/null || true
        wait "$PODMAN_SERVICE_PID" 2>/dev/null || true
        unset PODMAN_SERVICE_PID
        
        # Clean up socket file
        if [[ -n "${PODMAN_SOCKET:-}" && -S "$PODMAN_SOCKET" ]]; then
            rm -f "$PODMAN_SOCKET"
        fi
        
        # Unset Docker host
        unset DOCKER_HOST
    fi
}

# Run container structure tests
run_tests() {
    log_info "Running container structure tests..."
    
    # Create results directory
    mkdir -p "$RESULTS_DIR"
    
    # Start Podman service if needed
    start_podman_service
    
    # Run the tests
    local test_cmd=(
        "container-structure-test" "test"
        "--image" "$CONTAINER_IMAGE"
        "--config" "$TEST_CONFIG"
        "--output" "json"
        "--test-report" "$RESULTS_FILE"
    )
    
    log_debug "Running command: ${test_cmd[*]}"
    
    if "${test_cmd[@]}"; then
        log_info "Container structure tests completed"
        return 0
    else
        log_error "Container structure tests failed"
        return 1
    fi
}

# Display test results
display_results() {
    if [[ -f "$RESULTS_FILE" ]]; then
        log_info "Test results:"
        
        # Check if jq is available for pretty printing
        if command -v jq &> /dev/null; then
            cat "$RESULTS_FILE" | jq '.'
        else
            cat "$RESULTS_FILE"
        fi
        
        # Parse results to show summary
        if command -v jq &> /dev/null; then
            local total_tests passed_tests failed_tests
            
            # Count total tests
            total_tests=$(cat "$RESULTS_FILE" | jq '.Results | length')
            
            # Count passed tests
            passed_tests=$(cat "$RESULTS_FILE" | jq '[.Results[] | select(.Pass == true)] | length')
            
            # Count failed tests
            failed_tests=$(cat "$RESULTS_FILE" | jq '[.Results[] | select(.Pass == false)] | length')
            
            echo
            log_info "Test Summary:"
            echo "  Total tests: $total_tests"
            echo "  Passed: $passed_tests"
            echo "  Failed: $failed_tests"
            
            # Show failed tests details
            if [[ "$failed_tests" -gt 0 ]]; then
                echo
                log_error "Failed tests:"
                cat "$RESULTS_FILE" | jq -r '.Results[] | select(.Pass == false) | "  - \(.Name): \(.Errors // ["Unknown error"] | join(", "))"'
            fi
        fi
    else
        log_warn "No test results file found: $RESULTS_FILE"
    fi
}

# Cleanup function
cleanup() {
    log_debug "Cleaning up..."
    stop_podman_service
}

# Main execution
main() {
    local exit_code=0
    
    # Set up cleanup trap
    trap cleanup EXIT
    
    log_info "Starting container structure tests..."
    
    check_prerequisites
    build_container
    
    if run_tests; then
        log_info "All container structure tests passed!"
    else
        log_error "Some container structure tests failed!"
        exit_code=1
    fi
    
    display_results
    
    if [[ $exit_code -eq 0 ]]; then
        log_info "Container testing completed successfully!"
    else
        log_error "Container testing completed with failures!"
    fi
    
    exit $exit_code
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [options]"
        echo
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --verbose, -v  Enable verbose output"
        echo "  --no-build     Skip container build step"
        echo
        echo "Environment variables:"
        echo "  CONTAINER_IMAGE     Container image name (default: $CONTAINER_IMAGE)"
        echo "  TEST_CONFIG         Test configuration file (default: $TEST_CONFIG)"
        echo "  CONTAINER_RUNTIME   Container runtime to use (default: podman)"
        echo
        exit 0
        ;;
    --verbose|-v)
        set -x
        shift
        ;;
    --no-build)
        build_container() {
            log_info "Skipping container build (--no-build specified)"
        }
        shift
        ;;
esac

# Run main function
main "$@"