# Darrot Container Acceptance Tests - PowerShell Version
# Tests container functionality without requiring Discord/GCP credentials

param(
    [switch]$Verbose = $false
)

# Test configuration
$ContainerName = "darrot-test-$(Get-Date -Format 'yyyyMMddHHmmss')"
$ImageName = "darrot:test"
$TestEnvFile = "tests/container/test.env"

# Colors for output
$Red = "`e[31m"
$Green = "`e[32m"
$Yellow = "`e[33m"
$Reset = "`e[0m"

# Cleanup function
function Cleanup {
    Write-Host "${Yellow}Cleaning up test containers...${Reset}"
    
    try {
        podman stop $ContainerName 2>$null
        podman rm $ContainerName 2>$null
        podman rmi $ImageName 2>$null
        if (Test-Path $TestEnvFile) {
            Remove-Item $TestEnvFile -Force
        }
    }
    catch {
        # Ignore cleanup errors
    }
}

# Test functions
function Test-Build {
    Write-Host "${Yellow}Test 1: Building container image...${Reset}"
    
    try {
        $result = podman build --pull -t $ImageName . 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "${Green}✓ Container build successful${Reset}"
            return $true
        }
        else {
            Write-Host "${Red}✗ Container build failed${Reset}"
            Write-Host "${Yellow}Tip: If you see registry resolution errors, the Dockerfile uses fully qualified names${Reset}"
            Write-Host "${Yellow}You may need to configure registries.conf${Reset}"
            if ($Verbose) { Write-Host $result }
            return $false
        }
    }
    catch {
        Write-Host "${Red}✗ Container build failed: $($_.Exception.Message)${Reset}"
        return $false
    }
}

function Test-ImageProperties {
    Write-Host "${Yellow}Test 2: Verifying image properties...${Reset}"
    
    try {
        # Check image exists
        $imageExists = podman image exists $ImageName
        if ($LASTEXITCODE -ne 0) {
            Write-Host "${Red}✗ Image does not exist${Reset}"
            return $false
        }
        
        # Check image size
        $sizeBytes = podman image inspect $ImageName --format '{{.Size}}' | ConvertTo-Json | ConvertFrom-Json
        $sizeMB = [math]::Round($sizeBytes / 1024 / 1024, 2)
        
        if ($sizeMB -gt 500) {
            Write-Host "${Yellow}⚠ Image size is large: ${sizeMB}MB${Reset}"
        }
        else {
            Write-Host "${Green}✓ Image size is reasonable: ${sizeMB}MB${Reset}"
        }
        
        # Check for non-root user
        $config = podman image inspect $ImageName --format '{{.Config.User}}'
        if ($config -eq "darrot") {
            Write-Host "${Green}✓ Non-root user configured${Reset}"
        }
        else {
            Write-Host "${Red}✗ Running as root user${Reset}"
            return $false
        }
        
        Write-Host "${Green}✓ Image properties verified${Reset}"
        return $true
    }
    catch {
        Write-Host "${Red}✗ Image property verification failed: $($_.Exception.Message)${Reset}"
        return $false
    }
}

function Test-ContainerStartup {
    Write-Host "${Yellow}Test 3: Testing container startup...${Reset}"
    
    try {
        # Create test environment file
        $testEnvDir = Split-Path $TestEnvFile -Parent
        if (!(Test-Path $testEnvDir)) {
            New-Item -ItemType Directory -Path $testEnvDir -Force | Out-Null
        }
        
        @"
DISCORD_TOKEN=test_token_for_validation
LOG_LEVEL=DEBUG
TTS_DEFAULT_VOICE=en-US-Standard-A
"@ | Out-File -FilePath $TestEnvFile -Encoding UTF8
        
        # Start container
        $result = podman run -d --name $ContainerName --env-file $TestEnvFile $ImageName 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Host "${Red}✗ Container failed to start${Reset}"
            if ($Verbose) { Write-Host $result }
            return $false
        }
        
        Write-Host "${Green}✓ Container started successfully${Reset}"
        
        # Wait for startup
        Start-Sleep -Seconds 3
        
        # Check if container started (may exit due to invalid credentials, which is expected)
        $logs = podman logs $ContainerName 2>&1
        
        if ($logs -like "*Starting darrot Discord TTS bot*") {
            Write-Host "${Green}✓ Application started successfully${Reset}"
        }
        else {
            Write-Host "${Red}✗ Application failed to start${Reset}"
            if ($Verbose) { Write-Host $logs }
            return $false
        }
        
        # Check for expected credential error (this is normal in test environment)
        if ($logs -like "*could not find default credentials*") {
            Write-Host "${Green}✓ Application correctly handles missing credentials${Reset}"
        }
        elseif ($logs -like "*Configuration loaded successfully*") {
            Write-Host "${Green}✓ Application configuration loaded successfully${Reset}"
        }
        else {
            Write-Host "${Yellow}⚠ Unexpected application behavior${Reset}"
            if ($Verbose) { Write-Host $logs }
        }
        
        return $true
    }
    catch {
        Write-Host "${Red}✗ Container startup test failed: $($_.Exception.Message)${Reset}"
        return $false
    }
}

function Test-ApplicationBinary {
    Write-Host "${Yellow}Test 4: Testing application binary...${Reset}"
    
    try {
        # Test version command
        $versionOutput = podman exec $ContainerName /app/darrot -version 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "${Green}✓ Application binary responds to version flag${Reset}"
        }
        else {
            Write-Host "${Red}✗ Application binary version check failed${Reset}"
            if ($Verbose) { Write-Host $versionOutput }
            return $false
        }
        
        # Check dependencies
        $lddOutput = podman exec $ContainerName ldd /app/darrot 2>&1
        if ($lddOutput -like "*opus*") {
            Write-Host "${Green}✓ Opus library dependency found${Reset}"
        }
        else {
            Write-Host "${Yellow}⚠ Opus library dependency not found (may affect audio)${Reset}"
        }
        
        # Check if opusfile is available (common build issue)
        $opusfileCheck = podman exec $ContainerName sh -c "pkg-config --exists opusfile && echo 'found' || echo 'missing'" 2>&1
        if ($opusfileCheck -eq "found") {
            Write-Host "${Green}✓ Opusfile library available${Reset}"
        }
        else {
            Write-Host "${Yellow}⚠ Opusfile library not available (build may have issues)${Reset}"
        }
        
        return $true
    }
    catch {
        Write-Host "${Red}✗ Application binary test failed: $($_.Exception.Message)${Reset}"
        return $false
    }
}

function Test-FilesystemPermissions {
    Write-Host "${Yellow}Test 5: Testing filesystem permissions...${Reset}"
    
    try {
        # Test data directory access
        $dataWritable = podman exec $ContainerName test -w /app/data
        if ($LASTEXITCODE -eq 0) {
            Write-Host "${Green}✓ Data directory is writable${Reset}"
        }
        else {
            Write-Host "${Red}✗ Data directory is not writable${Reset}"
            return $false
        }
        
        # Test application binary is not writable
        $binaryWritable = podman exec $ContainerName test -w /app/darrot
        if ($LASTEXITCODE -ne 0) {
            Write-Host "${Green}✓ Application binary is not writable${Reset}"
        }
        else {
            Write-Host "${Red}✗ Application binary is writable (security risk)${Reset}"
            return $false
        }
        
        # Test user context
        $userId = podman exec $ContainerName id -u
        if ($userId -eq "1001") {
            Write-Host "${Green}✓ Running as correct non-root user (1001)${Reset}"
        }
        else {
            Write-Host "${Red}✗ Not running as expected user (got $userId, expected 1001)${Reset}"
            return $false
        }
        
        return $true
    }
    catch {
        Write-Host "${Red}✗ Filesystem permissions test failed: $($_.Exception.Message)${Reset}"
        return $false
    }
}

function Test-EnvironmentVariables {
    Write-Host "${Yellow}Test 6: Testing environment variable handling...${Reset}"
    
    try {
        # Check Discord token
        $discordToken = podman exec $ContainerName printenv DISCORD_TOKEN
        if ($discordToken -eq "test_token_for_validation") {
            Write-Host "${Green}✓ Environment variables loaded correctly${Reset}"
        }
        else {
            Write-Host "${Red}✗ Environment variables not loaded correctly${Reset}"
            return $false
        }
        
        # Check log level
        $logLevel = podman exec $ContainerName printenv LOG_LEVEL
        if ($logLevel -eq "DEBUG") {
            Write-Host "${Green}✓ Log level configuration working${Reset}"
        }
        else {
            Write-Host "${Red}✗ Log level configuration not working${Reset}"
            return $false
        }
        
        return $true
    }
    catch {
        Write-Host "${Red}✗ Environment variables test failed: $($_.Exception.Message)${Reset}"
        return $false
    }
}

# Main test execution
function Main {
    Write-Host "${Green}Starting Darrot Container Acceptance Tests${Reset}"
    Write-Host "=========================================="
    
    $failedTests = 0
    $totalTests = 6
    
    # Run tests
    if (!(Test-Build)) { $failedTests++ }
    if (!(Test-ImageProperties)) { $failedTests++ }
    if (!(Test-ContainerStartup)) { $failedTests++ }
    if (!(Test-ApplicationBinary)) { $failedTests++ }
    if (!(Test-FilesystemPermissions)) { $failedTests++ }
    if (!(Test-EnvironmentVariables)) { $failedTests++ }
    
    Write-Host "=========================================="
    
    if ($failedTests -eq 0) {
        Write-Host "${Green}All $totalTests tests passed! ✓${Reset}"
        Write-Host "${Green}Container is ready for deployment.${Reset}"
        exit 0
    }
    else {
        Write-Host "${Red}$failedTests out of $totalTests tests failed! ✗${Reset}"
        Write-Host "${Red}Please fix the issues before deploying.${Reset}"
        exit 1
    }
}

# Check prerequisites
function Test-Prerequisites {
    if (!(Get-Command podman -ErrorAction SilentlyContinue)) {
        Write-Host "${Red}Error: Podman is not installed or not in PATH${Reset}"
        exit 1
    }
    
    if (!(Test-Path "Dockerfile")) {
        Write-Host "${Red}Error: Dockerfile not found. Run this script from the project root.${Reset}"
        exit 1
    }
}

# Set up cleanup trap
try {
    Test-Prerequisites
    Main
}
finally {
    Cleanup
}