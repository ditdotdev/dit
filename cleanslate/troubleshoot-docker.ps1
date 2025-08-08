# Titan Docker Troubleshooting Script
# Diagnoses and fixes common Docker container execution issues

param(
    [switch]$Fix = $false,
    [switch]$Verbose = $false
)

Write-Host "Titan Docker Troubleshooting Script" -ForegroundColor Green
Write-Host "=====================================" -ForegroundColor Green

# Function to write status messages
function Write-Status {
    param($Message, $Status = "INFO")
    $color = switch($Status) {
        "OK" { "Green" }
        "WARN" { "Yellow" }
        "ERROR" { "Red" }
        default { "White" }
    }
    Write-Host "[$(Get-Date -Format 'HH:mm:ss')] $Message" -ForegroundColor $color
}

# Check Docker daemon status
Write-Status "Checking Docker daemon status..."
try {
    $dockerVersion = docker version --format json | ConvertFrom-Json
    Write-Status "Docker daemon is running - Client: $($dockerVersion.Client.Version), Server: $($dockerVersion.Server.Version)" "OK"
} catch {
    Write-Status "Docker daemon is not accessible!" "ERROR"
    if ($Fix) {
        Write-Status "Attempting to restart Docker service..."
        Restart-Service -Name docker -Force
        Start-Sleep 10
    }
    exit 1
}

# Check existing Titan containers
Write-Status "Checking Titan container status..."
$titanContainers = docker ps -a --filter "name=titan" --format "{{.Names}} {{.Status}}"
if ($titanContainers) {
    Write-Status "Found Titan containers:" "OK"
    $titanContainers | ForEach-Object { Write-Status "  $_" }
} else {
    Write-Status "No Titan containers found" "WARN"
}

# Check for container name conflicts
Write-Status "Checking for potential container name conflicts..."
$allContainers = docker ps -a --format "{{.Names}}"
$conflictingNames = @("testpostgres", "testredis", "testhello")
$conflicts = @()

foreach ($name in $conflictingNames) {
    if ($allContainers -contains $name) {
        $conflicts += $name
        Write-Status "Found conflicting container: $name" "WARN"
    }
}

if ($conflicts.Count -gt 0 -and $Fix) {
    Write-Status "Removing conflicting containers..."
    foreach ($name in $conflicts) {
        docker rm -f $name 2>$null
        Write-Status "Removed container: $name" "OK"
    }
}

# Check ZFS pool status
Write-Status "Checking ZFS pool status..."
try {
    $zfsStatus = wsl zpool list 2>$null
    if ($zfsStatus -match "titan") {
        Write-Status "ZFS pools are available" "OK"
        if ($Verbose) {
            wsl zpool list
        }
    } else {
        Write-Status "ZFS pools not found!" "ERROR"
        if ($Fix) {
            Write-Status "Recreating ZFS pools..."
            & "$PSScriptRoot\setup-zfs-pools.ps1" -Clean
        }
    }
} catch {
    Write-Status "Unable to check ZFS status" "ERROR"
}

# Check Docker volumes
Write-Status "Checking Docker volumes..."
$volumes = docker volume ls --filter "name=titan" --format "{{.Name}}"
if ($volumes) {
    Write-Status "Found Titan volumes:" "OK"
    $volumes | ForEach-Object { Write-Status "  $_" }
} else {
    Write-Status "No Titan volumes found" "WARN"
}

# Test Docker socket from Titan container
if ($titanContainers -match "titan-docker-launch.*Up") {
    Write-Status "Testing Docker socket access from Titan container..."
    try {
        $testResult = docker exec titan-docker-launch docker run --rm hello-world 2>$null
        if ($testResult -match "Hello from Docker") {
            Write-Status "Docker socket is accessible from Titan container" "OK"
        } else {
            Write-Status "Docker socket test failed" "ERROR"
        }
    } catch {
        Write-Status "Cannot test Docker socket - Titan container not accessible" "ERROR"
    }
}

# Check for common Docker issues
Write-Status "Checking for common Docker issues..."

# Check disk space
$freeSpace = Get-WmiObject Win32_LogicalDisk | Where-Object {$_.DeviceID -eq "C:"} | Select-Object -ExpandProperty FreeSpace
$freeSpaceGB = [math]::Round($freeSpace / 1GB, 2)
if ($freeSpaceGB -lt 5) {
    Write-Status "Low disk space: ${freeSpaceGB}GB free" "WARN"
} else {
    Write-Status "Disk space OK: ${freeSpaceGB}GB free" "OK"
}

# Check if Docker Desktop is using WSL2
$dockerInfo = docker info --format json | ConvertFrom-Json
if ($dockerInfo.OperatingSystem -match "Docker Desktop") {
    Write-Status "Docker Desktop detected" "OK"
    if ($dockerInfo.ContainerRuntimeInfo.Name -eq "runc") {
        Write-Status "Using runc runtime" "OK"
    }
} else {
    Write-Status "Non-Docker Desktop environment detected" "WARN"
}

# Summary and recommendations
Write-Host "`nSummary and Recommendations:" -ForegroundColor Green
Write-Host "============================" -ForegroundColor Green

if ($conflicts.Count -gt 0) {
    Write-Status "Found $($conflicts.Count) container name conflicts" "WARN"
    Write-Status "Recommendation: Run script with -Fix flag to resolve" "WARN"
}

if (-not ($titanContainers -match "titan-docker-launch.*Up")) {
    Write-Status "Titan infrastructure not running" "WARN"
    Write-Status "Recommendation: Run '.\titan.exe install' to start infrastructure" "WARN"
}

Write-Status "Troubleshooting complete!" "OK"

if ($Fix) {
    Write-Status "Applied automatic fixes where possible" "OK"
} else {
    Write-Status "Run with -Fix flag to automatically resolve issues" "INFO"
}
