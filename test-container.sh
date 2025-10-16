#!/bin/bash

# Simple container test runner for darrot Discord TTS bot

set -e

echo "🚀 Running Darrot Container Acceptance Tests"
echo "============================================="

# Check if we're on Windows (Git Bash/WSL) or Unix
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
    echo "Detected Windows environment, using PowerShell test runner..."
    powershell.exe -ExecutionPolicy Bypass -File "tests/container/acceptance_test.ps1" "$@"
else
    echo "Detected Unix environment, using Bash test runner..."
    bash "tests/container/acceptance_test.sh" "$@"
fi