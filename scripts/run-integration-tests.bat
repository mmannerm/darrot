@echo off
REM Integration Test Runner for darrot Discord Bot
REM This script helps run integration tests with proper setup

echo darrot Discord Bot - Integration Test Runner
echo ==============================================

REM Check if Go is installed
where go >nul 2>nul
if %errorlevel% neq 0 (
    echo Error: Go is not installed or not in PATH
    exit /b 1
)

REM Check for test token
if "%DISCORD_TEST_TOKEN%"=="" (
    echo Warning: DISCORD_TEST_TOKEN environment variable not set
    echo Integration tests will be skipped.
    echo.
    echo To run integration tests:
    echo 1. Get a Discord bot token from https://discord.com/developers/applications
    echo 2. Set the environment variable:
    echo    set DISCORD_TEST_TOKEN=your_token_here
    echo 3. Run this script again
    echo.
    echo Running unit tests only...
    go test ./internal/bot -v -short
    exit /b 0
)

echo DISCORD_TEST_TOKEN found - running full integration tests
echo.

REM Run all tests including integration
echo Running all tests (unit + integration)...
go test ./internal/bot -v

if %errorlevel% neq 0 (
    echo.
    echo Integration tests failed!
    exit /b 1
)

echo.
echo Integration tests completed successfully!

REM Optional: Run tests with coverage
if "%1"=="--coverage" (
    echo.
    echo Running tests with coverage...
    go test ./internal/bot -v -coverprofile=coverage.out
    go tool cover -html=coverage.out -o coverage.html
    echo Coverage report generated: coverage.html
)