#!/bin/bash

# Darrot Container Acceptance Tests
# Tests container functionality without requiring Discord/GCP credentials

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
CONTAINER_NAME="darrot-test-$(date +%s)"
IMAGE_NAME="darrot:test"
TEST_ENV_FILE="tests/container/test.env"

# Cleanup function
cleanup() {
    echo -e "${YELLOW}Cleaning up test containers...${NC}"
    podman stop "$CONTAINER_NAME" 2>/dev/null || true
    podman rm "$CONTAINER_NAME" 2>/dev/null || true
    podman rmi "$IMAGE_NAME" 2>/dev/null || true
    rm -f "$TEST_ENV_FILE"
}

# Set up cleanup trap
trap cleanup EXIT

# Test functions
test_build() {
    echo -e "${YELLOW}Test 1: Building container image...${NC}"
    
    # Try building with pull to ensure we get latest base images
    if podman build --pull -t "$IMAGE_NAME" .; then
        echo -e "${GREEN}✓ Container build successful${NC}"
        return 0
    else
        echo -e "${RED}✗ Container build failed${NC}"
        echo -e "${YELLOW}Tip: If you see registry resolution errors, the Dockerfile uses fully qualified names${NC}"
        echo -e "${YELLOW}You may need to configure /etc/containers/registries.conf${NC}"
        return 1
    fi
}

test_image_properties() {
    echo -e "${YELLOW}Test 2: Verifying image properties...${NC}"
    
    # Check image exists
    if ! podman image exists "$IMAGE_NAME"; then
        echo -e "${RED}✗ Image does not exist${NC}"
        return 1
    fi
    
    # Check image size (should be reasonable)
    local size=$(podman image inspect "$IMAGE_NAME" --format '{{.Size}}')
    local size_mb=$((size / 1024 / 1024))
    
    if [ "$size_mb" -gt 500 ]; then
        echo -e "${YELLOW}⚠ Image size is large: ${size_mb}MB${NC}"
    else
        echo -e "${GREEN}✓ Image size is reasonable: ${size_mb}MB${NC}"
    fi
    
    # Check for non-root user
    local user=$(podman image inspect "$IMAGE_NAME" --format '{{.Config.User}}')
    
    if [ "$user" = "darrot" ]; then
        echo -e "${GREEN}✓ Non-root user configured${NC}"
    else
        echo -e "${RED}✗ Running as root user (got: $user)${NC}"
        return 1
    fi
    
    echo -e "${GREEN}✓ Image properties verified${NC}"
    return 0
}

test_container_startup() {
    echo -e "${YELLOW}Test 3: Testing container startup...${NC}"
    
    # Create test environment file
    mkdir -p "$(dirname "$TEST_ENV_FILE")"
    cat > "$TEST_ENV_FILE" << EOF
DISCORD_TOKEN=test_token_for_validation
LOG_LEVEL=DEBUG
TTS_DEFAULT_VOICE=en-US-Standard-A
EOF
    
    # Start container with test configuration
    if podman run -d \
        --name "$CONTAINER_NAME" \
        --env-file "$TEST_ENV_FILE" \
        "$IMAGE_NAME"; then
        echo -e "${GREEN}✓ Container started successfully${NC}"
    else
        echo -e "${RED}✗ Container failed to start${NC}"
        return 1
    fi
    
    # Wait a moment for startup
    sleep 3
    
    # Check if container started (may exit due to invalid credentials, which is expected)
    local logs=$(podman logs "$CONTAINER_NAME" 2>&1)
    
    if echo "$logs" | grep -q "Starting darrot Discord TTS bot"; then
        echo -e "${GREEN}✓ Application started successfully${NC}"
    else
        echo -e "${RED}✗ Application failed to start${NC}"
        echo "$logs"
        return 1
    fi
    
    # Check for expected credential error (this is normal in test environment)
    if echo "$logs" | grep -q "could not find default credentials"; then
        echo -e "${GREEN}✓ Application correctly handles missing credentials${NC}"
    elif echo "$logs" | grep -q "Configuration loaded successfully"; then
        echo -e "${GREEN}✓ Application configuration loaded successfully${NC}"
    else
        echo -e "${YELLOW}⚠ Unexpected application behavior${NC}"
        echo "$logs"
    fi
    
    return 0
}

test_application_binary() {
    echo -e "${YELLOW}Test 4: Testing application binary...${NC}"
    
    # Test version command
    if podman exec "$CONTAINER_NAME" /app/darrot -version 2>/dev/null; then
        echo -e "${GREEN}✓ Application binary responds to version flag${NC}"
    else
        echo -e "${RED}✗ Application binary version check failed${NC}"
        return 1
    fi
    
    # Check if binary has required dependencies
    if podman exec "$CONTAINER_NAME" ldd /app/darrot | grep -q "opus"; then
        echo -e "${GREEN}✓ Opus library dependency found${NC}"
    else
        echo -e "${YELLOW}⚠ Opus library dependency not found (may affect audio)${NC}"
    fi
    
    # Check if opusfile is available (common build issue)
    if podman exec "$CONTAINER_NAME" sh -c "pkg-config --exists opusfile" 2>/dev/null; then
        echo -e "${GREEN}✓ Opusfile library available${NC}"
    else
        echo -e "${YELLOW}⚠ Opusfile library not available (build may have issues)${NC}"
    fi
    
    return 0
}

test_filesystem_permissions() {
    echo -e "${YELLOW}Test 5: Testing filesystem permissions...${NC}"
    
    # Test data directory access
    if podman exec "$CONTAINER_NAME" test -w /app/data; then
        echo -e "${GREEN}✓ Data directory is writable${NC}"
    else
        echo -e "${RED}✗ Data directory is not writable${NC}"
        return 1
    fi
    
    # Test that application directory is not writable (security)
    if podman exec "$CONTAINER_NAME" test -w /app/darrot; then
        echo -e "${RED}✗ Application binary is writable (security risk)${NC}"
        return 1
    else
        echo -e "${GREEN}✓ Application binary is not writable${NC}"
    fi
    
    # Test user context
    local user_id=$(podman exec "$CONTAINER_NAME" id -u)
    if [ "$user_id" = "1001" ]; then
        echo -e "${GREEN}✓ Running as correct non-root user (1001)${NC}"
    else
        echo -e "${RED}✗ Not running as expected user (got $user_id, expected 1001)${NC}"
        return 1
    fi
    
    return 0
}

test_environment_variables() {
    echo -e "${YELLOW}Test 6: Testing environment variable handling...${NC}"
    
    # Check that environment variables are loaded
    local discord_token=$(podman exec "$CONTAINER_NAME" printenv DISCORD_TOKEN)
    if [ "$discord_token" = "test_token_for_validation" ]; then
        echo -e "${GREEN}✓ Environment variables loaded correctly${NC}"
    else
        echo -e "${RED}✗ Environment variables not loaded correctly${NC}"
        return 1
    fi
    
    # Check log level
    local log_level=$(podman exec "$CONTAINER_NAME" printenv LOG_LEVEL)
    if [ "$log_level" = "DEBUG" ]; then
        echo -e "${GREEN}✓ Log level configuration working${NC}"
    else
        echo -e "${RED}✗ Log level configuration not working${NC}"
        return 1
    fi
    
    return 0
}

test_health_check() {
    echo -e "${YELLOW}Test 7: Testing health check...${NC}"
    
    # Wait for health check to initialize
    sleep 10
    
    # Check health status
    local health_status=$(podman inspect "$CONTAINER_NAME" --format '{{.State.Health.Status}}' 2>/dev/null || echo "no-health")
    
    if [ "$health_status" = "healthy" ]; then
        echo -e "${GREEN}✓ Container health check is healthy${NC}"
    elif [ "$health_status" = "starting" ]; then
        echo -e "${YELLOW}⚠ Container health check is still starting${NC}"
    else
        echo -e "${YELLOW}⚠ Health check status: $health_status${NC}"
    fi
    
    # Manual health check
    if podman exec "$CONTAINER_NAME" pgrep darrot >/dev/null; then
        echo -e "${GREEN}✓ Application process is running${NC}"
    else
        echo -e "${RED}✗ Application process is not running${NC}"
        return 1
    fi
    
    return 0
}

test_resource_limits() {
    echo -e "${YELLOW}Test 8: Testing resource limits...${NC}"
    
    # Stop current container
    podman stop "$CONTAINER_NAME"
    podman rm "$CONTAINER_NAME"
    
    # Start with resource limits
    if podman run -d \
        --name "$CONTAINER_NAME" \
        --memory=128m \
        --cpus=0.25 \
        --env-file "$TEST_ENV_FILE" \
        "$IMAGE_NAME"; then
        echo -e "${GREEN}✓ Container starts with resource limits${NC}"
    else
        echo -e "${RED}✗ Container failed to start with resource limits${NC}"
        return 1
    fi
    
    sleep 3
    
    # Check if still running with limits
    if podman ps --filter "name=$CONTAINER_NAME" --format "{{.Status}}" | grep -q "Up"; then
        echo -e "${GREEN}✓ Container runs within resource limits${NC}"
    else
        echo -e "${RED}✗ Container failed with resource limits${NC}"
        podman logs "$CONTAINER_NAME"
        return 1
    fi
    
    return 0
}

test_volume_mounts() {
    echo -e "${YELLOW}Test 9: Testing volume mounts...${NC}"
    
    # Create test data directory
    local test_data_dir="/tmp/darrot-test-data-$$"
    mkdir -p "$test_data_dir"
    
    # Stop current container
    podman stop "$CONTAINER_NAME"
    podman rm "$CONTAINER_NAME"
    
    # Start with volume mount
    if podman run -d \
        --name "$CONTAINER_NAME" \
        --env-file "$TEST_ENV_FILE" \
        -v "$test_data_dir:/app/data:Z" \
        "$IMAGE_NAME"; then
        echo -e "${GREEN}✓ Container starts with volume mount${NC}"
    else
        echo -e "${RED}✗ Container failed to start with volume mount${NC}"
        rm -rf "$test_data_dir"
        return 1
    fi
    
    sleep 3
    
    # Test file creation in mounted volume
    if podman exec "$CONTAINER_NAME" touch /app/data/test-file; then
        if [ -f "$test_data_dir/test-file" ]; then
            echo -e "${GREEN}✓ Volume mount is working correctly${NC}"
        else
            echo -e "${RED}✗ Volume mount not persisting files${NC}"
            rm -rf "$test_data_dir"
            return 1
        fi
    else
        echo -e "${RED}✗ Cannot write to mounted volume${NC}"
        rm -rf "$test_data_dir"
        return 1
    fi
    
    # Cleanup test data
    rm -rf "$test_data_dir"
    return 0
}

# Main test execution
main() {
    echo -e "${GREEN}Starting Darrot Container Acceptance Tests${NC}"
    echo "=========================================="
    
    local failed_tests=0
    local total_tests=9
    
    # Run all tests
    test_build || ((failed_tests++))
    test_image_properties || ((failed_tests++))
    test_container_startup || ((failed_tests++))
    test_application_binary || ((failed_tests++))
    test_filesystem_permissions || ((failed_tests++))
    test_environment_variables || ((failed_tests++))
    test_health_check || ((failed_tests++))
    test_resource_limits || ((failed_tests++))
    test_volume_mounts || ((failed_tests++))
    
    echo "=========================================="
    
    if [ $failed_tests -eq 0 ]; then
        echo -e "${GREEN}All $total_tests tests passed! ✓${NC}"
        echo -e "${GREEN}Container is ready for deployment.${NC}"
        exit 0
    else
        echo -e "${RED}$failed_tests out of $total_tests tests failed! ✗${NC}"
        echo -e "${RED}Please fix the issues before deploying.${NC}"
        exit 1
    fi
}

# Check prerequisites
check_prerequisites() {
    if ! command -v podman &> /dev/null; then
        echo -e "${RED}Error: Podman is not installed or not in PATH${NC}"
        exit 1
    fi
    
    if [ ! -f "Dockerfile" ]; then
        echo -e "${RED}Error: Dockerfile not found. Run this script from the project root.${NC}"
        exit 1
    fi
}

# Run prerequisites check and main tests
check_prerequisites
main "$@"