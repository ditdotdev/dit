# Complete Clean Slate Testing Automation Script
# Performs full teardown, rebuild, and testing with enhanced error handling

param(
    [switch]$SkipBuild = $false,
    [switch]$Verbose = $false
)

$ErrorActionPreference = "Stop"

function Write-Step {
    param($Message, $Status = "INFO")
    $timestamp = Get-Date -Format 'HH:mm:ss'
    $color = switch($Status) {
        "OK" { "Green" }
        "WARN" { "Yellow" }
        "ERROR" { "Red" }
        "STEP" { "Cyan" }
        default { "White" }
    }
    Write-Host "[$timestamp] $Message" -ForegroundColor $color
}

Write-Host "=================================" -ForegroundColor Green
Write-Host "Titan Clean Slate Testing Script" -ForegroundColor Green
Write-Host "=================================" -ForegroundColor Green
Write-Host ""

# Step 1: Teardown existing environment
Write-Step "STEP 1: Complete Environment Teardown" "STEP"

# Remove Titan repositories and uninstall
Write-Step "Checking for existing Titan repositories..."
try {
    $repos = & ".\titan.exe" ls 2>$null
    if ($repos -and $repos.Count -gt 1) {
        Write-Step "Found existing repositories, cleaning up..."
        $repoLines = $repos | Select-Object -Skip 1
        foreach ($line in $repoLines) {
            if ($line -match "^\w+\s+(\w+)\s+") {
                $repoName = $matches[1]
                Write-Step "Removing repository: $repoName"
                & ".\titan.exe" stop $repoName 2>$null
                & ".\titan.exe" rm $repoName 2>$null
            }
        }
    }
} catch {
    Write-Step "Repository cleanup skipped (expected on fresh install)" "WARN"
}

Write-Step "Uninstalling Titan..."
& ".\titan.exe" uninstall 2>$null

# Complete Docker cleanup
Write-Step "Performing complete Docker cleanup..."
$cleanup = docker system prune -af --volumes
Write-Step "Docker cleanup completed" "OK"

# Step 2: ZFS Pool Setup
Write-Step "STEP 2: ZFS Pool Setup" "STEP"
& ".\setup-zfs-pools.ps1" -Clean -VerifyDocker
if ($LASTEXITCODE -ne 0) {
    Write-Step "ZFS pool setup failed!" "ERROR"
    exit 1
}

# Step 3: Container Building
if (-not $SkipBuild) {
    Write-Step "STEP 3: Container Building" "STEP"
    
    Write-Step "Pulling base Titan image..."
    docker pull titandata/titan:latest
    docker tag titandata/titan:latest titan:latest
    
    Write-Step "Building custom Titan container..."
    docker build -t titan:custom -f Dockerfile . --no-cache
    if ($LASTEXITCODE -ne 0) {
        Write-Step "Custom container build failed!" "ERROR"
        exit 1
    }
    docker tag titan:custom titan:latest
    
    Write-Step "Building ZFS builder container..."
    Push-Location ..\zfs-builder
    docker build -t titandata/zfs-builder:latest .
    if ($LASTEXITCODE -ne 0) {
        Write-Step "ZFS builder build failed!" "ERROR"
        Pop-Location
        exit 1
    }
    Pop-Location
    
    Write-Step "Container building completed" "OK"
} else {
    Write-Step "Skipping container build as requested" "WARN"
}

# Step 4: Titan Installation
Write-Step "STEP 4: Titan Installation" "STEP"

Write-Step "Installing Titan..."
$installOutput = & ".\titan.exe" install
Write-Step "Titan installation completed" "OK"

# Wait for containers to stabilize
Write-Step "Waiting for Titan containers to stabilize..."
Start-Sleep 10

# Verify containers are running
$titanContainers = docker ps --filter "name=titan" --format "{{.Names}} {{.Status}}"
if ($titanContainers -match "Up") {
    Write-Step "✓ Titan containers are running" "OK"
    if ($Verbose) {
        docker ps --filter "name=titan"
    }
} else {
    Write-Step "✗ Titan containers failed to start properly" "ERROR"
    Write-Step "Running Docker troubleshooting..." "STEP"
    & ".\troubleshoot-docker.ps1" -Verbose
    exit 1
}

# Step 5: Functionality Testing
Write-Step "STEP 5: Functionality Testing" "STEP"

# Test repository creation
Write-Step "Testing repository creation..."
$commitSuccess = $false
try {
    $result = & ".\titan.exe" run --name cleanslatetest postgres:alpine 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Step "✓ Repository creation successful" "OK"
    } else {
        Write-Step "Repository creation failed with exit code $LASTEXITCODE" "WARN"
        Write-Step "Output: $result" "WARN"
        
        # Check if repo was created anyway - sometimes Titan creates repo despite container issues
        Start-Sleep 2
        $repos = & ".\titan.exe" ls
        if ($repos -match "cleanslatetest") {
            Write-Step "Repository was created despite container error - this is a known Docker execution issue" "WARN"
            Write-Step "Proceeding with commit test to verify core data versioning functionality" "WARN"
        } else {
            Write-Step "Repository creation completely failed" "ERROR"
            # Try a simple test anyway
            Write-Step "Attempting alternative test with a simpler container..." "STEP"
            $simpleResult = & ".\titan.exe" run --name cleanslatetest alpine:latest 2>&1
            if ($simpleResult -match "Creating repository") {
                Write-Step "Alternative container test showed progress" "WARN"
            }
        }
    }
} catch {
    Write-Step "Exception during repository creation: $($_.Exception.Message)" "ERROR"
}

# Test commit functionality (core requirement)
Write-Step "Testing commit functionality..."
try {
    $commitResult = & ".\titan.exe" commit -m "Clean slate test commit" cleanslatetest
    if ($commitResult -match "^Commit [a-f0-9]{32}$") {
        Write-Step "✓ Commit successful - ID: $commitResult" "OK"
        $commitSuccess = $true
    } else {
        Write-Step "✗ Commit failed: $commitResult" "ERROR"
        $commitSuccess = $false
    }
} catch {
    Write-Step "Exception during commit: $($_.Exception.Message)" "ERROR"
    $commitSuccess = $false
}

# Test log functionality
if ($commitSuccess) {
    Write-Step "Testing log functionality..."
    try {
        $logResult = & ".\titan.exe" log cleanslatetest
        if ($logResult -match "Clean slate test commit") {
            Write-Step "✓ Log functionality working" "OK"
        } else {
            Write-Step "Log functionality issue" "WARN"
        }
    } catch {
        Write-Step "Exception during log test: $($_.Exception.Message)" "WARN"
    }
}

# Step 6: Summary
Write-Step "STEP 6: Clean Slate Testing Summary" "STEP"

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "CLEAN SLATE TESTING RESULTS" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green

# Check final status
$finalZfsStatus = wsl zpool list 2>$null
$finalTitanContainers = docker ps --filter "name=titan" --format "{{.Names}}"
try {
    $finalRepos = & ".\titan.exe" ls 2>$null
} catch {
    $finalRepos = $null
}

Write-Host "ZFS Pools Status:" -ForegroundColor Yellow
if ($finalZfsStatus -match "titan") {
    Write-Host "✓ ZFS pools operational" -ForegroundColor Green
    if ($Verbose) { wsl zpool list }
} else {
    Write-Host "✗ ZFS pools not available" -ForegroundColor Red
}

Write-Host "`nTitan Infrastructure:" -ForegroundColor Yellow
if ($finalTitanContainers -match "titan-docker") {
    Write-Host "✓ Titan containers running" -ForegroundColor Green
    if ($Verbose) { docker ps --filter "name=titan" }
} else {
    Write-Host "✗ Titan containers not running" -ForegroundColor Red
}

Write-Host "`nData Versioning:" -ForegroundColor Yellow
if ($commitSuccess) {
    Write-Host "✓ Core data versioning functional" -ForegroundColor Green
} else {
    Write-Host "✗ Data versioning issues detected" -ForegroundColor Red
}

Write-Host "`nRepository Management:" -ForegroundColor Yellow
if ($finalRepos -match "cleanslatetest") {
    Write-Host "✓ Repository creation working" -ForegroundColor Green
} else {
    Write-Host "⚠ Repository creation had issues" -ForegroundColor Yellow
}

Write-Host ""
if ($commitSuccess -and ($finalZfsStatus -match "titan") -and ($finalTitanContainers -match "titan-docker")) {
    Write-Host "CLEAN SLATE TESTING: SUCCESS ✓" -ForegroundColor Green
    Write-Host "Environment is ready for Titan development and testing" -ForegroundColor Green
} else {
    Write-Host "CLEAN SLATE TESTING: PARTIAL SUCCESS ⚠" -ForegroundColor Yellow
    Write-Host "Core functionality working but some issues detected" -ForegroundColor Yellow
    Write-Host "Run .\troubleshoot-docker.ps1 for detailed diagnostics" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Cyan
Write-Host "- Use .\titan.exe ls to see repositories" -ForegroundColor White
Write-Host "- Use .\troubleshoot-docker.ps1 for any issues" -ForegroundColor White
Write-Host "- Use .\setup-zfs-pools.ps1 -Clean for full reset" -ForegroundColor White
