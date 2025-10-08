#!/bin/bash

# Integration Test Runner for darrot Discord Bot
# This script helps run integration tests with proper setup

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}darrot Discord Bot - Integration Test Runner${NC}"
echo "=============================================="

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed or not in PATH${NC}"
    exit 1
fi

# Check for test token
if [ -z "$DISCORD_TEST_TOKEN" ]; then
    echo -e "${YELLOW}Warning: DISCORD_TEST_TOKEN environment variable not set${NC}"
    echo "Integration tests will be skipped."
    echo ""
    echo "To run integration tests:"
    echo "1. Get a Discord bot token from https://discord.com/developers/applications"
    echo "2. Set the environment variable:"
    echo "   export DISCORD_TEST_TOKEN=\"your_token_here\""
    echo "3. Run this script again"
    echo ""
    echo -e "${YELLOW}Running unit tests only...${NC}"
    go test ./internal/bot -v -short
    exit 0
fi

echo -e "${GREEN}DISCORD_TEST_TOKEN found - running full integration tests${NC}"
echo ""

# Run all tests including integration
echo "Running all tests (unit + integration)..."
go test ./internal/bot -v

echo ""
echo -e "${GREEN}Integration tests completed successfully!${NC}"

# Optional: Run tests with coverage
if [ "$1" = "--coverage" ]; then
    echo ""
    echo "Running tests with coverage..."
    go test ./internal/bot -v -coverprofile=coverage.out
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${GREEN}Coverage report generated: coverage.html${NC}"
fi