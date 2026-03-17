#Requires -Version 5.1
<#
.SYNOPSIS
    Downloads and installs a prebuilt WSL2 kernel with ZFS support for datadatdat.

.DESCRIPTION
    This script downloads a prebuilt WSL2 kernel image (bzImage) with ZFS
    statically compiled from the datadatdat/zfs-releases GitHub Releases.
    It configures .wslconfig to use the custom kernel.

    The default WSL2 kernel ships with CONFIG_MODULES=n, which means kernel
    modules cannot be loaded. ZFS must be statically compiled into the kernel.

.PARAMETER CheckOnly
    Only check for available kernels, don't install.

.PARAMETER ZfsVersion
    ZFS version to install (default: 2.3.4).

.PARAMETER KernelRelease
    Specific kernel release to install. If not specified, auto-detects from
    the current WSL2 kernel version.

.PARAMETER Force
    Overwrite existing kernel without prompting.

.EXAMPLE
    .\Install-D3Kernel.ps1
    # Auto-detect WSL2 kernel version and install matching prebuilt kernel

.EXAMPLE
    .\Install-D3Kernel.ps1 -CheckOnly
    # Check what prebuilt kernels are available

.EXAMPLE
    .\Install-D3Kernel.ps1 -KernelRelease "6.6.75.2-microsoft-standard-WSL2" -ZfsVersion "2.3.4"
    # Install a specific kernel + ZFS version
#>

[CmdletBinding()]
param(
    [switch]$CheckOnly,
    [string]$ZfsVersion = "2.3.4",
    [string]$KernelRelease,
    [switch]$Force
)

$ErrorActionPreference = "Stop"

$KernelDir = Join-Path $env:USERPROFILE ".datadatdat\kernels"
$GithubRepo = "datadatdat/zfs-releases"

function Get-CurrentWSL2Kernel {
    try {
        $output = wsl uname -r 2>&1
        if ($LASTEXITCODE -eq 0) {
            return $output.Trim()
        }
    } catch {
        # WSL not running or not installed
    }
    return $null
}

function Get-AvailableReleases {
    param([string]$Filter)

    Write-Host "Querying GitHub for available WSL2 kernel releases..." -ForegroundColor Cyan

    try {
        $releases = gh api "repos/$GithubRepo/releases" --paginate 2>&1 | ConvertFrom-Json
    } catch {
        Write-Error "Failed to query GitHub releases. Ensure 'gh' CLI is installed and authenticated."
        return @()
    }

    $wsl2Releases = $releases | Where-Object {
        $_.tag_name -match "^wsl2-kernel-" -and -not $_.draft
    }

    if ($Filter) {
        $wsl2Releases = $wsl2Releases | Where-Object {
            $_.tag_name -match [regex]::Escape($Filter)
        }
    }

    return $wsl2Releases
}

function Get-ReleaseTag {
    param(
        [string]$KernelRelease,
        [string]$ZfsVersion
    )
    return "wsl2-kernel-$KernelRelease-zfs-$ZfsVersion"
}

function Install-Kernel {
    param(
        [object]$Release,
        [string]$DestDir
    )

    # Find bzImage asset
    $bzImageAsset = $Release.assets | Where-Object { $_.name -eq "bzImage" }
    if (-not $bzImageAsset) {
        Write-Error "No bzImage found in release $($Release.tag_name)"
        return $false
    }

    # Find checksum asset
    $checksumAsset = $Release.assets | Where-Object { $_.name -eq "CHECKSUMS.sha256" }

    # Create destination directory
    if (-not (Test-Path $DestDir)) {
        New-Item -ItemType Directory -Path $DestDir -Force | Out-Null
    }

    $bzImagePath = Join-Path $DestDir "bzImage"
    $checksumPath = Join-Path $DestDir "CHECKSUMS.sha256"

    # Check if kernel already exists
    if ((Test-Path $bzImagePath) -and -not $Force) {
        $existing = Get-Item $bzImagePath
        Write-Host "Existing kernel found: $bzImagePath ($([math]::Round($existing.Length / 1MB, 1))MB)" -ForegroundColor Yellow
        $confirm = Read-Host "Overwrite? (y/N)"
        if ($confirm -ne "y") {
            Write-Host "Aborted." -ForegroundColor Yellow
            return $false
        }
    }

    # Download bzImage
    Write-Host "Downloading bzImage ($([math]::Round($bzImageAsset.size / 1MB, 1))MB)..." -ForegroundColor Cyan
    gh api "$($bzImageAsset.url)" -H "Accept: application/octet-stream" > $bzImagePath 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to download bzImage"
        return $false
    }

    # Download and verify checksum
    if ($checksumAsset) {
        Write-Host "Verifying checksum..." -ForegroundColor Cyan
        gh api "$($checksumAsset.url)" -H "Accept: application/octet-stream" > $checksumPath 2>&1

        $expectedHash = (Get-Content $checksumPath | Where-Object { $_ -match "bzImage$" } | Select-Object -First 1) -split '\s+' | Select-Object -First 1
        $actualHash = (Get-FileHash -Path $bzImagePath -Algorithm SHA256).Hash.ToLower()

        if ($expectedHash -and $actualHash -ne $expectedHash) {
            Remove-Item $bzImagePath -Force
            Write-Error "Checksum verification failed!`nExpected: $expectedHash`nActual:   $actualHash"
            return $false
        }
        Write-Host "Checksum verified." -ForegroundColor Green
    }

    return $true
}

function Update-WSLConfig {
    param([string]$KernelPath)

    $wslConfigPath = Join-Path $env:USERPROFILE ".wslconfig"
    # Windows paths in .wslconfig need double backslashes
    $kernelPathEscaped = $KernelPath.Replace("\", "\\")

    if (Test-Path $wslConfigPath) {
        # Back up existing config
        $backupPath = "$wslConfigPath.bak"
        Copy-Item $wslConfigPath $backupPath -Force
        Write-Host "Backed up existing .wslconfig to $backupPath" -ForegroundColor Cyan

        $content = Get-Content $wslConfigPath -Raw

        if ($content -match "(?m)^kernel\s*=") {
            # Update existing kernel line
            $content = $content -replace "(?m)^kernel\s*=.*$", "kernel=$kernelPathEscaped"
        } elseif ($content -match "(?m)^\[wsl2\]") {
            # Add kernel line under [wsl2] section
            $content = $content -replace "(?m)^(\[wsl2\])", "`$1`nkernel=$kernelPathEscaped"
        } else {
            # Add [wsl2] section with kernel line
            $content = "$content`n[wsl2]`nkernel=$kernelPathEscaped`n"
        }

        Set-Content $wslConfigPath $content -NoNewline
    } else {
        # Create new .wslconfig
        $content = "[wsl2]`nkernel=$kernelPathEscaped`n"
        Set-Content $wslConfigPath $content -NoNewline
    }

    Write-Host "Updated $wslConfigPath with kernel=$kernelPathEscaped" -ForegroundColor Green
}

# --- Main ---

Write-Host "`n=== Datadatdat WSL2 Kernel Installer ===" -ForegroundColor White
Write-Host ""

# Detect current WSL2 kernel
$currentKernel = Get-CurrentWSL2Kernel
if ($currentKernel) {
    Write-Host "Current WSL2 kernel: $currentKernel" -ForegroundColor Cyan
} else {
    Write-Host "WSL2 not detected or not running." -ForegroundColor Yellow
    if (-not $KernelRelease) {
        Write-Host "Use -KernelRelease to specify a kernel version." -ForegroundColor Yellow
        exit 1
    }
}

# Determine target kernel release
if (-not $KernelRelease) {
    # Strip any custom suffix from the current kernel release
    # e.g., "5.15.167.4-microsoft-standard-WSL2-Datadatdat-zfs" -> "5.15.167.4-microsoft-standard-WSL2"
    $KernelRelease = $currentKernel -replace '-Datadatdat-zfs$', ''
    # Also handle other custom suffixes by matching the standard pattern
    if ($KernelRelease -match '^(\d+\.\d+\.\d+\.\d+-microsoft-standard-WSL2)') {
        $KernelRelease = $Matches[1]
    }
}

Write-Host "Target kernel: $KernelRelease" -ForegroundColor Cyan
Write-Host "Target ZFS version: $ZfsVersion" -ForegroundColor Cyan
Write-Host ""

$releaseTag = Get-ReleaseTag -KernelRelease $KernelRelease -ZfsVersion $ZfsVersion

if ($CheckOnly) {
    Write-Host "Checking for available releases..." -ForegroundColor Cyan
    $releases = Get-AvailableReleases
    if ($releases.Count -eq 0) {
        Write-Host "No published WSL2 kernel releases found." -ForegroundColor Yellow
    } else {
        Write-Host "`nAvailable prebuilt WSL2 kernels:" -ForegroundColor Green
        foreach ($r in $releases) {
            $assets = ($r.assets | Measure-Object).Count
            Write-Host "  $($r.tag_name) ($assets assets, published $($r.published_at))"
        }
    }

    # Check if our target is available
    $target = $releases | Where-Object { $_.tag_name -eq $releaseTag }
    if ($target) {
        Write-Host "`nTarget release $releaseTag is available." -ForegroundColor Green
    } else {
        Write-Host "`nTarget release $releaseTag is NOT available." -ForegroundColor Yellow
        Write-Host "You may need to build the kernel manually. See: wsl-kernel-zfs.md" -ForegroundColor Yellow
    }
    exit 0
}

# Download and install
Write-Host "Looking for release: $releaseTag" -ForegroundColor Cyan

$releases = Get-AvailableReleases -Filter $KernelRelease
$targetRelease = $releases | Where-Object { $_.tag_name -eq $releaseTag }

if (-not $targetRelease) {
    Write-Error "Release $releaseTag not found. Run with -CheckOnly to see available releases."
    exit 1
}

$success = Install-Kernel -Release $targetRelease -DestDir $KernelDir
if (-not $success) {
    exit 1
}

$bzImagePath = Join-Path $KernelDir "bzImage"
Update-WSLConfig -KernelPath $bzImagePath

Write-Host ""
Write-Host "Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor White
Write-Host "  1. Run: wsl --shutdown" -ForegroundColor Cyan
Write-Host "  2. Restart your WSL2 terminal" -ForegroundColor Cyan
Write-Host "  3. Verify: grep zfs /proc/filesystems" -ForegroundColor Cyan
Write-Host ""
