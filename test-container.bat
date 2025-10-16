@echo off
REM Simple container test runner for Windows

echo 🚀 Running Darrot Container Acceptance Tests
echo =============================================

REM Check if PowerShell is available
where powershell >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo Error: PowerShell is not available
    exit /b 1
)

REM Run the PowerShell test script
powershell.exe -ExecutionPolicy Bypass -File "tests\container\acceptance_test.ps1" %*