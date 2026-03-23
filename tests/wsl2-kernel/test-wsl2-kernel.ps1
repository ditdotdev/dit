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
$startTime = Get-Date

function Log($msg, $color = "White") {
    $ts = (Get-Date).ToString("HH:mm:ss")
    # $color parameter accepted for interface compatibility but Write-Output does not support colors
    Write-Output "[$ts] $msg"
}

function LogElapsed($label) {
    $elapsed = (Get-Date) - $startTime
    Log "$label (elapsed: $($elapsed.ToString('hh\:mm\:ss')))" Cyan
}

function Invoke-Vagrant {
    param([Parameter(ValueFromRemainingArguments)]$VagrantArgs)
    # Run vagrant directly - no piping, so $LASTEXITCODE is preserved.
    # Per-step timestamps come from LogElapsed calls between steps.
    Push-Location $ScriptDir
    try {
        & vagrant @VagrantArgs
    } finally {
        Pop-Location
    }
}

Log "=== WSL2 Kernel E2E Test Orchestrator ===" White
Write-Output ""

# Validate inputs
if (-not $BzImagePath -and -not $ReleaseTag) {
    Write-Error "Specify either -BzImagePath or -ReleaseTag"
    exit 1
}

# Download from GitHub Release if needed
if ($ReleaseTag) {
    Write-Output "Downloading bzImage from release: $ReleaseTag"
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
Write-Output "bzImage: $vagrantBzImage (${size}MB)"
Write-Output ""

# Step 1: Bring up the VM
LogElapsed "--- Step 1: Starting Vagrant VM ---"
Push-Location $ScriptDir
try {
    # vagrant up may write to stderr during Phase 1 provisioning (e.g. wsl --set-default-version
    # fails pre-reboot). With ErrorActionPreference=Stop, stderr output becomes a terminating
    # error. Temporarily set to Continue so we can handle it gracefully.
    $ErrorActionPreference = "Continue"
    Invoke-Vagrant up --provider=hyperv
    $vagrantUpExit = $LASTEXITCODE
    $ErrorActionPreference = "Stop"
    if ($vagrantUpExit -ne 0) {
        Write-Output "WARNING: vagrant up returned exit code $vagrantUpExit (may be expected for Phase 1)"
    }

    # Phase 1 enabled WSL2 features - reboot required before distro install
    LogElapsed "Reloading VM after enabling WSL2 features..."
    Invoke-Vagrant reload
    if ($LASTEXITCODE -ne 0) {
        Write-Error "vagrant reload failed"
        exit 1
    }

    # Re-upload files and SSH keys (reload loses file provisioner state)
    LogElapsed "Re-uploading files and SSH keys to VM..."
    Invoke-Vagrant provision --provision-with upload-files
    if ($LASTEXITCODE -ne 0) {
        Write-Error "File upload failed"
        exit 1
    }
    Invoke-Vagrant provision --provision-with upload-ssh-keys
    if ($LASTEXITCODE -ne 0) {
        Write-Error "SSH key upload failed"
        exit 1
    }

    # Phase 2: Install Ubuntu distro + Docker Engine + tools (WSL now functional)
    LogElapsed "--- Step 1b: Installing WSL2 distro + tools ---"
    Invoke-Vagrant provision --provision-with wsl2
    if ($LASTEXITCODE -ne 0) {
        Write-Error "WSL2 Phase 2 provisioning failed"
        exit 1
    }

    # Step 2: Deploy kernel
    LogElapsed "--- Step 2: Deploying kernel ---"
    Invoke-Vagrant provision --provision-with deploy-kernel
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Kernel deployment failed"
        exit 1
    }

    # Step 3: Run E2E tests
    LogElapsed "--- Step 3: Running E2E tests ---"
    if ($SkipServerTests) {
        Write-Output "SkipServerTests flag set, skipping e2e-server tests"
    }
    Invoke-Vagrant provision --provision-with e2e
    $e2eResult = $LASTEXITCODE

    # Guard against false positives: verify WSL2 was actually functional
    Invoke-Vagrant winrm -c "wsl -d Ubuntu -- echo ok"
    if ($LASTEXITCODE -ne 0) {
        Log "ERROR: WSL2 Ubuntu distro not functional - results are invalid" Red
        $e2eResult = 1
    }

    # Report results
    Write-Output ""
    LogElapsed "=== Results ==="
    if ($e2eResult -eq 0) {
        Log "ALL TESTS PASSED" Green
        Log "Safe to promote draft release to published." Green
    } else {
        Log "TESTS FAILED (exit code: $e2eResult)" Red
        Log "Do NOT publish this kernel release." Red
    }

} finally {
    if (-not $KeepVM) {
        Log "Destroying VM..." Cyan
        Invoke-Vagrant destroy -f
    } else {
        Log "VM kept alive for debugging. Run 'vagrant destroy -f' when done." Yellow
    }
    Pop-Location
}

exit $e2eResult
