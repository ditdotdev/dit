#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Deploys a bzImage into the VM and configures .wslconfig.
#>

$ErrorActionPreference = "Stop"

Write-Output "=== Deploying WSL2 Kernel ==="

$KernelDir = "C:\Users\vagrant\.datadatdat\kernels"
$BzImageSource = "C:\vagrant\bzImage"
$WslConfig = "C:\Users\vagrant\.wslconfig"

# Validate bzImage exists
if (-not (Test-Path $BzImageSource)) {
    Write-Error "bzImage not found at $BzImageSource. Place bzImage in the Vagrant synced folder."
    exit 1
}

# Create kernel directory
if (-not (Test-Path $KernelDir)) {
    New-Item -ItemType Directory -Path $KernelDir -Force | Out-Null
}

# Copy bzImage
$BzImageDest = Join-Path $KernelDir "bzImage"
Copy-Item $BzImageSource $BzImageDest -Force
Write-Output "Copied bzImage to $BzImageDest"

# Validate file
$size = (Get-Item $BzImageDest).Length / 1MB
Write-Output "bzImage size: $([math]::Round($size, 1))MB"
if ($size -lt 5 -or $size -gt 50) {
    Write-Warning "bzImage size $([math]::Round($size, 1))MB is outside expected range (5-50MB)"
}

# Configure .wslconfig
$kernelPathEscaped = $BzImageDest.Replace("\", "\\")
$content = @"
[wsl2]
kernel=$kernelPathEscaped
"@

Set-Content $WslConfig $content
Write-Output "Updated $WslConfig"
Get-Content $WslConfig

# Restart WSL2
Write-Output "Restarting WSL2..."
wsl --shutdown
Start-Sleep -Seconds 3

# Verify kernel
Write-Output "Verifying kernel..."
$kernelVersion = wsl -d Ubuntu -- uname -r
Write-Output "Kernel version: $kernelVersion"

$zfsCheck = wsl -d Ubuntu -- grep zfs /proc/filesystems 2>&1
if ($zfsCheck -match "zfs") {
    Write-Output "ZFS support: DETECTED"
} else {
    Write-Error "ZFS support NOT detected in /proc/filesystems"
    exit 1
}

Write-Output ""
Write-Output "=== Kernel deployment successful ==="
