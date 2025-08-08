# ZFS Pool Setup Script for Titan Clean Slate Testing
# Run this script before installing Titan to ensure ZFS pools are ready
# Use -Clean parameter to destroy existing pools first for true clean slate

param(
    [switch]$Clean = $false,
    [switch]$VerifyDocker = $false
)

Write-Host "Setting up ZFS pools for Titan..." -ForegroundColor Green

# Optional Docker verification
if ($VerifyDocker) {
    Write-Host "Verifying Docker environment..." -ForegroundColor Cyan
    try {
        $dockerVersion = docker version --format json | ConvertFrom-Json
        Write-Host "✓ Docker is accessible - Client: $($dockerVersion.Client.Version)" -ForegroundColor Green
        
        # Check for potential container conflicts
        $existingContainers = docker ps -a --format "{{.Names}}"
        $potentialConflicts = @("testpostgres", "testredis", "testhello")
        foreach ($name in $potentialConflicts) {
            if ($existingContainers -contains $name) {
                Write-Host "⚠ Found existing container: $name" -ForegroundColor Yellow
                Write-Host "  Consider running: docker rm -f $name" -ForegroundColor Yellow
            }
        }
    } catch {
        Write-Error "Docker is not accessible. Please ensure Docker Desktop is running."
        exit 1
    }
}

if ($Clean) {
    Write-Host "Clean slate requested - removing existing ZFS pools..." -ForegroundColor Yellow
    
    # Remove existing pools
    Write-Host "Destroying existing ZFS pools..."
    wsl sudo zpool destroy titan-docker 2>$null
    wsl sudo zpool destroy titan 2>$null
    
    # Remove pool image files
    Write-Host "Removing pool image files..."
    wsl sudo rm -rf /titan-pools
    
    # Remove loop devices
    Write-Host "Removing loop devices..."
    wsl sudo losetup -D
    
    # Verify cleanup
    $poolCheck = wsl zpool list 2>$null
    if ($poolCheck -match "no pools available") {
        Write-Host "OK All ZFS pools successfully removed" -ForegroundColor Green
    } else {
        Write-Host "Some pools may still exist:" -ForegroundColor Yellow
        wsl zpool list
    }
}

# Check if WSL is available
try {
    $wslCheck = wsl --status
    Write-Host "WSL is available" -ForegroundColor Green
} catch {
    Write-Error "WSL is not available or not running"
    exit 1
}

# Check ZFS kernel support
Write-Host "Checking ZFS kernel support..."
$zfsSupport = wsl cat /proc/filesystems | Select-String "zfs"
if ($zfsSupport) {
    Write-Host "OK ZFS kernel support detected" -ForegroundColor Green
} else {
    Write-Error "X ZFS kernel support not found"
    exit 1
}

# Check if pools already exist
Write-Host "Checking existing ZFS pools..."
$existingPools = wsl zpool list 2>$null
if ($existingPools) {
    Write-Host "Existing pools found:" -ForegroundColor Yellow
    wsl zpool list
    
    # Handle hostid mismatches
    Write-Host "Fixing any hostid mismatches..."
    wsl sudo zpool export titan 2>$null
    wsl sudo zpool export titan-docker 2>$null
    Start-Sleep 2
    wsl sudo zpool import titan 2>$null
    wsl sudo zpool import titan-docker 2>$null
}

# Create pool storage directory
Write-Host "Creating pool storage directory..."
wsl sudo mkdir -p /titan-pools

# Create titan-docker pool if it does not exist
Write-Host "Checking for titan-docker pool..."
$titanDockerExists = wsl zpool list titan-docker 2>$null
if (-not $titanDockerExists) {
    Write-Host "Creating titan-docker pool..." -ForegroundColor Yellow
    
    # Create image file
    wsl sudo dd if=/dev/zero of=/titan-pools/titan-docker.img bs=1M count=1024 2>$null
    
    # Create loop device
    wsl sudo losetup -f /titan-pools/titan-docker.img
    
    # Find the loop device
    $loopDeviceOutput = wsl losetup -a
    $loopDevice = ($loopDeviceOutput | Select-String "titan-docker.img" | ForEach-Object { $_.Line.Split(":")[0] })
    if ($loopDevice) {
        # Create the pool
        wsl sudo zpool create titan-docker $loopDevice
        Write-Host "OK titan-docker pool created successfully" -ForegroundColor Green
    } else {
        Write-Error "X Failed to create loop device for titan-docker"
        exit 1
    }
} else {
    Write-Host "OK titan-docker pool already exists" -ForegroundColor Green
}

# Create main titan pool if it does not exist
Write-Host "Checking for titan pool..."
$titanExists = wsl zpool list titan 2>$null
if (-not $titanExists) {
    Write-Host "Creating titan pool..." -ForegroundColor Yellow
    
    # Create image file
    wsl sudo dd if=/dev/zero of=/titan-pools/titan.img bs=1M count=1024 2>$null
    
    # Create loop device
    wsl sudo losetup -f /titan-pools/titan.img
    
    # Find the loop device
    $loopDeviceOutput = wsl losetup -a
    $loopDevice = ($loopDeviceOutput | Select-String "titan.img" | ForEach-Object { $_.Line.Split(":")[0] })
    if ($loopDevice) {
        # Create the pool
        wsl sudo zpool create titan $loopDevice
        Write-Host "OK titan pool created successfully" -ForegroundColor Green
    } else {
        Write-Error "X Failed to create loop device for titan"
        exit 1
    }
} else {
    Write-Host "OK titan pool already exists" -ForegroundColor Green
}

# Final verification
Write-Host ""
Write-Host "Final ZFS pool status:" -ForegroundColor Green
wsl zpool list

Write-Host ""
Write-Host "Pool health check:" -ForegroundColor Green
wsl zpool status

Write-Host ""
Write-Host "✓ ZFS pools are ready for Titan!" -ForegroundColor Green
Write-Host "You can now run: .\titan.exe install" -ForegroundColor Cyan

Write-Host ""
Write-Host "Troubleshooting Tips:" -ForegroundColor Yellow
Write-Host "- If container creation fails with 'exit status 127':" -ForegroundColor White
Write-Host "  Run: .\troubleshoot-docker.ps1 -Fix" -ForegroundColor White
Write-Host "- To verify Docker integration:" -ForegroundColor White
Write-Host "  Run: .\setup-zfs-pools.ps1 -VerifyDocker" -ForegroundColor White
Write-Host "- For complete reset:" -ForegroundColor White
Write-Host "  Run: .\setup-zfs-pools.ps1 -Clean" -ForegroundColor White
