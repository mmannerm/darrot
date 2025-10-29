#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="darrot-acceptance-tests"
RESULTS_DIR="./test-results"
TIMEOUT=${TEST_TIMEOUT:-300}

# Container runtime commands (will be set by detect_container_runtime)
CONTAINER_CMD=""
COMPOSE_CMD=""
COMPOSE_FILE=""

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to cleanup containers
cleanup() {
    print_status "Cleaning up containers..."
    $COMPOSE_CMD -f $COMPOSE_FILE -p $PROJECT_NAME down -v --remove-orphans 2>/dev/null || true
    $CONTAINER_CMD system prune -f 2>/dev/null || true
}

# Function to detect and set container runtime
detect_container_runtime() {
    if command -v podman >/dev/null 2>&1 && podman info >/dev/null 2>&1; then
        CONTAINER_CMD="podman"
        COMPOSE_FILE="podman-compose.test.yml"
        print_status "Using Podman as container runtime"
        
        # Check if podman-compose is available, fallback to docker-compose with podman
        if command -v podman-compose >/dev/null 2>&1; then
            COMPOSE_CMD="podman-compose"
            print_status "Using podman-compose"
        elif command -v docker-compose >/dev/null 2>&1; then
            COMPOSE_CMD="docker-compose"
            COMPOSE_FILE="docker-compose.test.yml"
            export DOCKER_HOST="unix:///run/user/$(id -u)/podman/podman.sock"
            print_status "Using docker-compose with Podman socket"
        else
            print_error "Neither podman-compose nor docker-compose found. Please install one of them."
            exit 1
        fi
    elif command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
        CONTAINER_CMD="docker"
        COMPOSE_CMD="docker-compose"
        COMPOSE_FILE="docker-compose.test.yml"
        print_status "Using Docker as container runtime"
    else
        print_error "Neither Docker nor Podman is available or running."
        print_error "Please install and start either Docker or Podman."
        print_error ""
        print_error "For Podman (rootless):"
        print_error "  - Install: sudo apt install podman podman-compose"
        print_error "  - Start: systemctl --user start podman.socket"
        print_error ""
        print_error "For Docker:"
        print_error "  - Install Docker Desktop or Docker Engine"
        print_error "  - Start the Docker daemon"
        exit 1
    fi
}

# Function to build required images
build_images() {
    print_status "Building required container images..."
    
    # Build darrot bot image
    print_status "Building darrot bot image..."
    $CONTAINER_CMD build -t darrot:test -f ../../Dockerfile ../../ || {
        print_error "Failed to build darrot bot image"
        exit 1
    }
    
    # Build mock Discord server image
    print_status "Building mock Discord server image..."
    $CONTAINER_CMD build -t mock-discord:test -f ../mock-discord/Dockerfile ../mock-discord/ || {
        print_error "Failed to build mock Discord server image"
        exit 1
    }
    
    # Build acceptance test runner image
    print_status "Building acceptance test runner image..."
    $CONTAINER_CMD build -t acceptance-tests:test -f Dockerfile.test . || {
        print_error "Failed to build acceptance test runner image"
        exit 1
    }
}

# Function to wait for a specific service
wait_for_service() {
    local service_name=$1
    local max_attempts=20
    local attempt=1
    
    print_status "Waiting for $service_name to be ready..."
    
    while [ $attempt -le $max_attempts ]; do
        if $COMPOSE_CMD -f $COMPOSE_FILE -p $PROJECT_NAME ps $service_name | grep -q "Up"; then
            print_status "$service_name is ready"
            return 0
        fi
        
        print_status "Waiting for $service_name... (attempt $attempt/$max_attempts)"
        sleep 3
        attempt=$((attempt + 1))
    done
    
    print_error "$service_name failed to become ready within timeout"
    return 1
}

# Function to run acceptance tests
run_tests() {
    local suite=${1:-"all"}
    
    print_status "Running acceptance tests (suite: $suite)..."
    
    # Create results directory
    mkdir -p $RESULTS_DIR
    
    # For Podman, we need to handle container startup order manually
    if [ "$CONTAINER_CMD" = "podman" ]; then
        print_status "Starting containers in sequence for Podman..."
        
        # Start mock-discord first
        $COMPOSE_CMD -f $COMPOSE_FILE -p $PROJECT_NAME up -d mock-discord
        wait_for_service mock-discord
        
        # Start darrot-bot
        $COMPOSE_CMD -f $COMPOSE_FILE -p $PROJECT_NAME up -d darrot-bot
        wait_for_service darrot-bot
        
        # Run acceptance tests
        $COMPOSE_CMD -f $COMPOSE_FILE -p $PROJECT_NAME run --rm \
            -e TEST_SUITE=$suite \
            -e TEST_TIMEOUT=${TIMEOUT}s \
            acceptance-tests || {
            print_error "Acceptance tests failed"
            return 1
        }
    else
        # Docker can handle depends_on properly
        $COMPOSE_CMD -f $COMPOSE_FILE -p $PROJECT_NAME up -d
        
        # Wait for services
        sleep 10
        
        $COMPOSE_CMD -f $COMPOSE_FILE -p $PROJECT_NAME run --rm \
            -e TEST_SUITE=$suite \
            -e TEST_TIMEOUT=${TIMEOUT}s \
            acceptance-tests || {
            print_error "Acceptance tests failed"
            return 1
        }
    fi
    
    print_status "Acceptance tests completed"
}

# Function to collect results
collect_results() {
    print_status "Collecting test results..."
    
    # Copy results from container volume
    $CONTAINER_CMD run --rm \
        -v ${PROJECT_NAME}_test-results:/source \
        -v $(pwd)/$RESULTS_DIR:/dest \
        alpine:latest \
        sh -c "cp -r /source/* /dest/ 2>/dev/null || true"
    
    # Display results summary
    if [ -f "$RESULTS_DIR/acceptance-test-results-"*.json ]; then
        local result_file=$(ls $RESULTS_DIR/acceptance-test-results-*.json | head -1)
        local passed=$(jq -r '.passed_tests' "$result_file" 2>/dev/null || echo "0")
        local failed=$(jq -r '.failed_tests' "$result_file" 2>/dev/null || echo "0")
        local total=$(jq -r '.total_tests' "$result_file" 2>/dev/null || echo "0")
        
        print_status "Test Results Summary:"
        echo "  Total Tests: $total"
        echo "  Passed: $passed"
        echo "  Failed: $failed"
        
        if [ "$failed" -gt 0 ]; then
            print_error "Some tests failed. Check results in $RESULTS_DIR/"
            return 1
        else
            print_status "All tests passed!"
        fi
    else
        print_warning "No test results found"
    fi
}

# Function to show logs
show_logs() {
    print_status "Showing container logs..."
    $COMPOSE_CMD -f $COMPOSE_FILE -p $PROJECT_NAME logs
}

# Main execution
main() {
    local command=${1:-"run"}
    local suite=${2:-"all"}
    
    case $command in
        "build")
            detect_container_runtime
            build_images
            ;;
        "run")
            detect_container_runtime
            
            # Cleanup any existing containers
            cleanup
            
            # Build images
            build_images
            
            # Start services (handled in run_tests for proper ordering)
            print_status "Starting services..."
            
            # Run tests (which will start services in correct order)
            if run_tests $suite; then
                collect_results
                cleanup
                print_status "Acceptance tests completed successfully!"
            else
                print_error "Acceptance tests failed"
                show_logs
                collect_results
                cleanup
                exit 1
            fi
            ;;
        "logs")
            show_logs
            ;;
        "cleanup")
            cleanup
            ;;
        "help"|*)
            echo "Usage: $0 [command] [suite]"
            echo ""
            echo "Commands:"
            echo "  build     - Build container images only"
            echo "  run       - Run acceptance tests (default)"
            echo "  logs      - Show container logs"
            echo "  cleanup   - Clean up containers and volumes"
            echo "  help      - Show this help message"
            echo ""
            echo "Test Suites:"
            echo "  all       - Run all test suites (default)"
            echo "  core      - Run core functionality tests"
            echo "  concurrent - Run concurrent testing"
            echo "  error     - Run error resilience tests"
            echo ""
            echo "Container Runtime:"
            echo "  Supports both Docker and Podman (rootless)"
            echo "  Automatically detects available runtime"
            echo ""
            echo "Environment Variables:"
            echo "  TEST_TIMEOUT - Test timeout in seconds (default: 300)"
            echo ""
            echo "Examples:"
            echo "  $0 run all"
            echo "  $0 run core"
            echo "  TEST_TIMEOUT=600 $0 run all"
            echo ""
            echo "Podman Setup (rootless):"
            echo "  sudo apt install podman podman-compose"
            echo "  systemctl --user start podman.socket"
            ;;
    esac
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Run main function
main "$@"