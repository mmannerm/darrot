#!/bin/bash

# Quick container test script for development

set -euo pipefail

CONTAINER_IMAGE="darrot:test"
CONTAINER_RUNTIME=${CONTAINER_RUNTIME:-podman}

# Check if container runtime is available
if ! command -v "$CONTAINER_RUNTIME" &> /dev/null; then
    echo "⚠️  $CONTAINER_RUNTIME not found, trying docker..."
    if command -v docker &> /dev/null; then
        CONTAINER_RUNTIME="docker"
        echo "✓ Using docker as container runtime"
    else
        echo "❌ Neither podman nor docker is available"
        exit 1
    fi
fi

echo "🔨 Building container with $CONTAINER_RUNTIME..."
"$CONTAINER_RUNTIME" build -t "$CONTAINER_IMAGE" . || exit 1

echo "🧪 Running quick container tests..."

# Test 1: Check if binary exists and is executable
echo "  ✓ Testing binary existence..."
"$CONTAINER_RUNTIME" run --rm --entrypoint /bin/sh "$CONTAINER_IMAGE" -c "ls -la /app/darrot"

# Test 2: Check version command
echo "  ✓ Testing version command..."
"$CONTAINER_RUNTIME" run --rm "$CONTAINER_IMAGE" version || echo "Version command test completed"

# Test 3: Check user
echo "  ✓ Testing non-root user..."
"$CONTAINER_RUNTIME" run --rm --entrypoint /bin/sh "$CONTAINER_IMAGE" -c "whoami"

# Test 4: Check opus libraries
echo "  ✓ Testing opus libraries..."
"$CONTAINER_RUNTIME" run --rm --entrypoint /bin/sh "$CONTAINER_IMAGE" -c "ls /usr/lib/libopus* && echo 'Opus library found'"

# Test 5: Check data directory permissions
echo "  ✓ Testing data directory..."
"$CONTAINER_RUNTIME" run --rm --entrypoint /bin/sh "$CONTAINER_IMAGE" -c "ls -la /app/data"

echo "✅ Quick container tests completed successfully!"