#!/bin/bash

# Simple container test runner for darrot Discord TTS bot

set -e

echo "🚀 Running Darrot Container Acceptance Tests"
echo "============================================="

# Run Linux container acceptance tests
echo "Running Linux container acceptance tests..."
bash "tests/container/acceptance_test.sh" "$@"