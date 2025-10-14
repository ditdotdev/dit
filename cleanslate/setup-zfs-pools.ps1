# ZFS Pool Setup Script for Datadatdat Clean Slate Testing
# Run this script before installing Datadatdat to ensure ZFS pools are ready
# Use -Clean parameter to destroy existing pools first for true clean slate

param(
    [switch]$Clean = $false,
    [switch]$VerifyDocker = $false
)

# Function to ensure Docker is running
function Ensure-DockerRunning {
    Write-Host "Checking Docker status..." -ForegroundColor Cyan
    
    # First, try to connect to Docker
    try {
        $dockerVersion = docker version --format json 2>$null | ConvertFrom-Json
        if ($dockerVersion -and $dockerVersion.Server) {
            Write-Host "✓ Docker is running - Client: $($dockerVersion.Client.Version), Server: $($dockerVersion.Server.Version)" -ForegroundColor Green
            return $true
        }
    } catch {
        # Docker command failed, continue to startup logic
    }
    
    Write-Host "Docker is not running. Attempting to start Docker Desktop..." -ForegroundColor Yellow
    
    # Check if Docker Desktop process is running
    $dockerProcess = Get-Process -Name "Docker Desktop" -ErrorAction SilentlyContinue
    if (-not $dockerProcess) {
        # Start Docker Desktop
        Write-Host "Starting Docker Desktop..." -ForegroundColor Yellow
        try {
            $dockerPath = "C:\Program Files\Docker\Docker\Docker Desktop.exe"
            if (Test-Path $dockerPath) {
                Start-Process $dockerPath -WindowStyle Hidden
                Write-Host "Docker Desktop started. Waiting for initialization..." -ForegroundColor Yellow
            } else {
                Write-Error "Docker Desktop not found at expected location: $dockerPath"
                Write-Host "Please install Docker Desktop or start it manually." -ForegroundColor Red
                return $false
            }
        } catch {
            Write-Error "Failed to start Docker Desktop: $($_.Exception.Message)"
            return $false
        }
    } else {
        Write-Host "Docker Desktop process is running but daemon is not ready. Waiting..." -ForegroundColor Yellow
    }
    
    # Wait for Docker daemon to be ready (up to 60 seconds)
    $timeout = 60
    $elapsed = 0
    $interval = 5
    
    Write-Host "Waiting for Docker daemon to be ready..." -ForegroundColor Yellow
    while ($elapsed -lt $timeout) {
        Start-Sleep $interval
        $elapsed += $interval
        
        try {
            $dockerVersion = docker version --format json 2>$null | ConvertFrom-Json
            if ($dockerVersion -and $dockerVersion.Server) {
                Write-Host "✓ Docker is now ready - Client: $($dockerVersion.Client.Version), Server: $($dockerVersion.Server.Version)" -ForegroundColor Green
                return $true
            }
        } catch {
            # Continue waiting
        }
        
        Write-Host "Still waiting... ($elapsed/$timeout seconds)" -ForegroundColor Gray
    }
    
    Write-Error "Docker failed to start within $timeout seconds. Please check Docker Desktop manually."
    return $false
}

Write-Host "Setting up ZFS pools for Datadatdat..." -ForegroundColor Green

# Always ensure Docker is running
if (-not (Ensure-DockerRunning)) {
    Write-Error "Cannot proceed without Docker. Please ensure Docker Desktop is installed and can start."
    exit 1
}

# Optional additional Docker verification
if ($VerifyDocker) {
    Write-Host "Performing additional Docker environment verification..." -ForegroundColor Cyan
    try {
        # Check for potential container conflicts
        $existingContainers = docker ps -a --format "{{.Names}}"
        $potentialConflicts = @("testpostgres", "testredis", "testhello", "cleanslatetest", "pgtest")
        foreach ($name in $potentialConflicts) {
            if ($existingContainers -contains $name) {
                Write-Host "⚠ Found existing container: $name" -ForegroundColor Yellow
                Write-Host "  Consider running: docker rm -f $name" -ForegroundColor Yellow
            }
        }
    } catch {
        Write-Warning "Could not check for container conflicts: $($_.Exception.Message)"
    }
}

if ($Clean) {
    Write-Host "Clean slate requested - removing existing ZFS pools..." -ForegroundColor Yellow
    
    # Remove existing pools (ignore errors if pools don't exist)
    Write-Host "Destroying existing ZFS pools..."
    wsl sudo zpool destroy datadatdat-docker 2>$null
    wsl sudo zpool destroy datadatdat 2>$null
    
    # Remove pool image files
    Write-Host "Removing pool image files..."
    wsl sudo rm -rf /datadatdat-pools
    
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
    wsl sudo zpool export datadatdat 2>$null
    wsl sudo zpool export datadatdat-docker 2>$null
    Start-Sleep 2
    wsl sudo zpool import datadatdat 2>$null
    wsl sudo zpool import datadatdat-docker 2>$null
}

# Create pool storage directory
Write-Host "Creating pool storage directory..."
wsl sudo mkdir -p /datadatdat-pools

# Create datadatdat-docker pool if it does not exist
Write-Host "Checking for datadatdat-docker pool..."
$datadatdatDockerExists = wsl zpool list datadatdat-docker 2>$null
if (-not $datadatdatDockerExists) {
    Write-Host "Creating datadatdat-docker pool..." -ForegroundColor Yellow
    
    # Create image file
    wsl sudo dd if=/dev/zero of=/datadatdat-pools/datadatdat-docker.img bs=1M count=1024 2>$null
    
    # Create loop device
    wsl sudo losetup -f /datadatdat-pools/datadatdat-docker.img
    
    # Find the loop device
    $loopDeviceOutput = wsl losetup -a
    $loopDevice = ($loopDeviceOutput | Select-String "datadatdat-docker.img" | ForEach-Object { $_.Line.Split(":")[0] })
    if ($loopDevice) {
        # Create the pool
        wsl sudo zpool create datadatdat-docker $loopDevice
        Write-Host "OK datadatdat-docker pool created successfully" -ForegroundColor Green
    } else {
        Write-Error "X Failed to create loop device for datadatdat-docker"
        exit 1
    }
} else {
    Write-Host "OK datadatdat-docker pool already exists" -ForegroundColor Green
}

# Create main datadatdat pool if it does not exist
Write-Host "Checking for datadatdat pool..."
$datadatdatExists = wsl zpool list datadatdat 2>$null
if (-not $datadatdatExists) {
    Write-Host "Creating datadatdat pool..." -ForegroundColor Yellow
    
    # Create image file
    wsl sudo dd if=/dev/zero of=/datadatdat-pools/datadatdat.img bs=1M count=1024 2>$null
    
    # Create loop device
    wsl sudo losetup -f /datadatdat-pools/datadatdat.img
    
    # Find the loop device
    $loopDeviceOutput = wsl losetup -a
    $loopDevice = ($loopDeviceOutput | Select-String "datadatdat.img" | ForEach-Object { $_.Line.Split(":")[0] })
    if ($loopDevice) {
        # Create the pool
        wsl sudo zpool create datadatdat $loopDevice
        Write-Host "OK datadatdat pool created successfully" -ForegroundColor Green
    } else {
        Write-Error "X Failed to create loop device for datadatdat"
        exit 1
    }
} else {
    Write-Host "OK datadatdat pool already exists" -ForegroundColor Green
}

# Final verification
Write-Host ""
Write-Host "Final ZFS pool status:" -ForegroundColor Green
wsl zpool list

Write-Host ""
Write-Host "Pool health check:" -ForegroundColor Green
wsl zpool status

Write-Host ""
Write-Host "✓ ZFS pools are ready for Datadatdat!" -ForegroundColor Green
Write-Host "You can now run: .\datadatdat.exe install" -ForegroundColor Cyan

Write-Host ""
Write-Host "Troubleshooting Tips:" -ForegroundColor Yellow
Write-Host "- If container creation fails with 'exit status 127':" -ForegroundColor White
Write-Host "  Run: .\troubleshoot-docker.ps1 -Fix" -ForegroundColor White
Write-Host "- To verify Docker integration:" -ForegroundColor White
Write-Host "  Run: .\setup-zfs-pools.ps1 -VerifyDocker" -ForegroundColor White
Write-Host "- For complete reset:" -ForegroundColor White
Write-Host "  Run: .\setup-zfs-pools.ps1 -Clean" -ForegroundColor White
