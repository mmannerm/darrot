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
        # Check if project .env file exists
        $envFileArg = ""
        if (Test-Path ".env") {
            Write-Host "${Green}✓ Using existing .env file from project root${Reset}"
            $envFileArg = "--env-file .env"
        }
        else {
            # Create minimal test environment file
            $testEnvDir = Split-Path $TestEnvFile -Parent
            if (!(Test-Path $testEnvDir)) {
                New-Item -ItemType Directory -Path $testEnvDir -Force | Out-Null
            }
            
            @"
DISCORD_TOKEN=test_token_for_validation
LOG_LEVEL=DEBUG
TTS_DEFAULT_VOICE=en-US-Standard-A
"@ | Out-File -FilePath $TestEnvFile -Encoding UTF8
            
            Write-Host "${Yellow}⚠ No .env file found, using test configuration${Reset}"
            $envFileArg = "--env-file $TestEnvFile"
        }
        
        # Add Google Cloud credentials from host environment if available
        $extraEnvArgs = ""
        if ($env:GOOGLE_CLOUD_CREDENTIALS_PATH -and (Test-Path $env:GOOGLE_CLOUD_CREDENTIALS_PATH)) {
            Write-Host "${Green}✓ Using Google Cloud credentials from host environment${Reset}"
            $extraEnvArgs = "-e GOOGLE_CLOUD_CREDENTIALS_PATH=/app/credentials/credentials.json -v `"$($env:GOOGLE_CLOUD_CREDENTIALS_PATH):/app/credentials/credentials.json:ro,Z`""
        }
        elseif ($env:GOOGLE_APPLICATION_CREDENTIALS -and (Test-Path $env:GOOGLE_APPLICATION_CREDENTIALS)) {
            Write-Host "${Green}✓ Using Google Application Default Credentials from host${Reset}"
            $extraEnvArgs = "-e GOOGLE_CLOUD_CREDENTIALS_PATH=/app/credentials/credentials.json -v `"$($env:GOOGLE_APPLICATION_CREDENTIALS):/app/credentials/credentials.json:ro,Z`""
        }
        
        # Start container with configuration
        $cmdArgs = @("run", "-d", "--name", $ContainerName) + $envFileArg.Split(" ") + $extraEnvArgs.Split(" ") + @($ImageName)
        $result = & podman $cmdArgs 2>&1
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
        
        # Check application behavior based on credentials availability
        if ($logs -like "*could not find default credentials*") {
            Write-Host "${Green}✓ Application correctly handles missing Google Cloud credentials${Reset}"
        }
        elseif ($logs -like "*TTS system initialized successfully*") {
            Write-Host "${Green}✓ Application started with Google Cloud TTS enabled${Reset}"
        }
        elseif ($logs -like "*Configuration loaded successfully*") {
            Write-Host "${Green}✓ Application configuration loaded successfully${Reset}"
        }
        else {
            Write-Host "${Yellow}⚠ Check application logs for details${Reset}"
            if ($logs -like "*DISCORD_TOKEN*required*") {
                Write-Host "${Red}✗ Missing Discord token${Reset}"
                return $false
            }
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
        # Check that Discord token is loaded (either from .env or test config)
        $discordToken = podman exec $ContainerName printenv DISCORD_TOKEN 2>$null
        if ($discordToken) {
            Write-Host "${Green}✓ Discord token environment variable loaded${Reset}"
        }
        else {
            Write-Host "${Red}✗ Discord token environment variable not found${Reset}"
            return $false
        }
        
        # Check log level (should have default or configured value)
        $logLevel = podman exec $ContainerName printenv LOG_LEVEL 2>$null
        if ($logLevel) {
            Write-Host "${Green}✓ Log level configuration: $logLevel${Reset}"
        }
        else {
            Write-Host "${Yellow}⚠ Log level not set, using default${Reset}"
        }
        
        # Check TTS configuration (should have defaults even if not explicitly set)
        $ttsVoice = podman exec $ContainerName printenv TTS_DEFAULT_VOICE 2>$null
        if (-not $ttsVoice) { $ttsVoice = "en-US-Standard-A" }
        Write-Host "${Green}✓ TTS voice configuration: $ttsVoice${Reset}"
        
        # Check Google Cloud credentials path if mounted
        $gcPath = podman exec $ContainerName printenv GOOGLE_CLOUD_CREDENTIALS_PATH 2>$null
        if ($gcPath) {
            Write-Host "${Green}✓ Google Cloud credentials path configured: $gcPath${Reset}"
            # Verify the file exists if path is set
            $fileExists = podman exec $ContainerName test -f $gcPath 2>$null
            if ($LASTEXITCODE -eq 0) {
                Write-Host "${Green}✓ Google Cloud credentials file accessible${Reset}"
            }
            else {
                Write-Host "${Yellow}⚠ Google Cloud credentials path set but file not accessible${Reset}"
            }
        }
        else {
            Write-Host "${Yellow}⚠ No Google Cloud credentials configured (TTS will use defaults)${Reset}"
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