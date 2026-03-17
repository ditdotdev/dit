#Requires -Version 5.1
<#
.SYNOPSIS
    Orchestrates end-to-end testing of a prebuilt WSL2 kernel with ZFS.

.DESCRIPTION
    Spins up a Windows 11 Hyper-V VM via Vagrant, provisions WSL2 + Docker,
    deploys the bzImage, and runs the full datadatdat E2E test suite.

.PARAMETER BzImagePath
    Local path to a bzImage file to test.

.PARAMETER ReleaseTag
    GitHub release tag to download and test (e.g., wsl2-kernel-6.6.75.2-zfs-2.3.4).

.PARAMETER KeepVM
    Don't destroy the VM after testing (for debugging).

.PARAMETER SkipServerTests
    Skip make e2e-server (if no server available).

.EXAMPLE
    .\test-wsl2-kernel.ps1 -BzImagePath C:\path\to\bzImage

.EXAMPLE
    .\test-wsl2-kernel.ps1 -ReleaseTag wsl2-kernel-6.6.75.2-zfs-2.3.4
#>

[CmdletBinding()]
param(
    [string]$BzImagePath,
    [string]$ReleaseTag,
    [switch]$KeepVM,
    [switch]$SkipServerTests
)

$ErrorActionPreference = "Stop"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

Write-Host "`n=== WSL2 Kernel E2E Test Orchestrator ===" -ForegroundColor White
Write-Host ""

# Validate inputs
if (-not $BzImagePath -and -not $ReleaseTag) {
    Write-Error "Specify either -BzImagePath or -ReleaseTag"
    exit 1
}

# Download from GitHub Release if needed
if ($ReleaseTag) {
    Write-Host "Downloading bzImage from release: $ReleaseTag" -ForegroundColor Cyan
    $BzImagePath = Join-Path $ScriptDir "bzImage"
    gh release download $ReleaseTag --repo datadatdat/zfs-releases --pattern "bzImage" --dir $ScriptDir --clobber
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to download bzImage from release $ReleaseTag"
        exit 1
    }
}

# Validate bzImage exists
if (-not (Test-Path $BzImagePath)) {
    Write-Error "bzImage not found at: $BzImagePath"
    exit 1
}

# Copy bzImage to Vagrant synced folder
$vagrantBzImage = Join-Path $ScriptDir "bzImage"
if ($BzImagePath -ne $vagrantBzImage) {
    Copy-Item $BzImagePath $vagrantBzImage -Force
}

$size = [math]::Round((Get-Item $vagrantBzImage).Length / 1MB, 1)
Write-Host "bzImage: $vagrantBzImage (${size}MB)" -ForegroundColor Cyan
Write-Host ""

# Step 1: Bring up the VM
Write-Host "--- Step 1: Starting Vagrant VM ---" -ForegroundColor Yellow
Push-Location $ScriptDir
try {
    vagrant up --provider=hyperv
    if ($LASTEXITCODE -ne 0) {
        Write-Error "vagrant up failed"
        exit 1
    }

    # The initial provision installs WSL2 + Docker. A reboot is usually needed.
    Write-Host "Reloading VM after WSL2 provisioning..." -ForegroundColor Cyan
    vagrant reload
    if ($LASTEXITCODE -ne 0) {
        Write-Error "vagrant reload failed"
        exit 1
    }

    # Step 2: Deploy kernel
    Write-Host "`n--- Step 2: Deploying kernel ---" -ForegroundColor Yellow
    vagrant provision --provision-with deploy-kernel
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Kernel deployment failed"
        exit 1
    }

    # Step 3: Run E2E tests
    Write-Host "`n--- Step 3: Running E2E tests ---" -ForegroundColor Yellow
    vagrant provision --provision-with e2e
    $e2eResult = $LASTEXITCODE

    # Report results
    Write-Host ""
    Write-Host "=== Results ===" -ForegroundColor White
    if ($e2eResult -eq 0) {
        Write-Host "ALL TESTS PASSED" -ForegroundColor Green
        Write-Host "Safe to promote draft release to published." -ForegroundColor Green
    } else {
        Write-Host "TESTS FAILED (exit code: $e2eResult)" -ForegroundColor Red
        Write-Host "Do NOT publish this kernel release." -ForegroundColor Red
    }

} finally {
    if (-not $KeepVM) {
        Write-Host "`nDestroying VM..." -ForegroundColor Cyan
        vagrant destroy -f 2>$null
    } else {
        Write-Host "`nVM kept alive for debugging. Run 'vagrant destroy -f' when done." -ForegroundColor Yellow
    }
    Pop-Location
}

exit $e2eResult
