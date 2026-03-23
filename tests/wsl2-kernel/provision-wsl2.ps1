#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Provisions WSL2 + Ubuntu + Docker Engine + dev tools inside a Vagrant VM.

.DESCRIPTION
    Two-phase provisioning:
      Phase 1 (pre-reboot):  Enable WSL2 + VirtualMachinePlatform Windows features.
      Phase 2 (post-reboot): Install Ubuntu distro, Docker Engine, Go, BATS, build tools.

    The script auto-detects which phase to run. The orchestrator calls this twice:
    once during initial `vagrant up`, then again after `vagrant reload`.
#>

$ErrorActionPreference = "Stop"

Write-Output "=== Provisioning WSL2 Environment ==="

# Detect if WSL is functional (post-reboot)
$wslFunctional = $false
try {
    wsl --status 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        $wslFunctional = $true
    }
} catch {
    # wsl --status throws when WSL features are not yet enabled (pre-reboot).
    # This is expected in Phase 1; we fall through to enable the features.
    Write-Verbose "wsl --status failed: $_"
}

if (-not $wslFunctional) {
    # ========== Phase 1: Enable features (pre-reboot) ==========
    Write-Output "Phase 1: Enabling WSL2 Windows features..."

    Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux -NoRestart -All
    Enable-WindowsOptionalFeature -Online -FeatureName VirtualMachinePlatform -NoRestart -All

    # Install WSL itself (the subsystem, not a distro yet)
    wsl --install --no-distribution 2>&1

    Write-Output ""
    Write-Output "=== Phase 1 complete - reboot required ==="
    exit 0
}

# ========== Phase 2: Install distro + tools (post-reboot) ==========
Write-Output "Phase 2: WSL is functional, installing distro and tools..."

# Install Ubuntu distro
Write-Output "Installing Ubuntu distribution..."
wsl --install -d Ubuntu --no-launch 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Warning "wsl --install -d Ubuntu returned $LASTEXITCODE, retrying..."
    Start-Sleep -Seconds 5
    wsl --install -d Ubuntu --no-launch 2>&1
}

# Set WSL2 as default version
wsl --set-default-version 2

# Copy SSH keys into WSL2 FIRST - fail fast if keys are missing
Write-Output "Configuring SSH keys in WSL2..."
$uploadedKeyDir = "C:\vagrant\.ssh"
if (Test-Path "$uploadedKeyDir\id_ed25519") {
    wsl -d Ubuntu -- bash -c "mkdir -p /root/.ssh && chmod 700 /root/.ssh && cp /mnt/c/vagrant/.ssh/id_ed25519 /root/.ssh/ && cp /mnt/c/vagrant/.ssh/id_ed25519.pub /root/.ssh/ && chmod 600 /root/.ssh/id_ed25519 && chmod 644 /root/.ssh/id_ed25519.pub && ssh-keyscan github.com >> /root/.ssh/known_hosts 2>/dev/null"
    Write-Output "SSH keys configured for root"
} else {
    Write-Error "No SSH key found at $uploadedKeyDir - cannot continue"
    exit 1
}

# Verify SSH to GitHub works
Write-Output "Verifying SSH access to GitHub..."
wsl -d Ubuntu -- bash -c "ssh -T -o StrictHostKeyChecking=accept-new git@github.com 2>&1 || true"
wsl -d Ubuntu -- bash -c "ssh -T git@github.com 2>&1 | grep -q 'successfully authenticated'"
if ($LASTEXITCODE -ne 0) {
    Write-Error "SSH authentication to GitHub failed"
    exit 1
}
Write-Output "GitHub SSH access verified"

# Configure git to use SSH for all GitHub HTTPS URLs (needed for go mod with private repos)
Write-Output "Configuring git to use SSH for GitHub..."
wsl -d Ubuntu -- bash -c 'git config --global url."git@github.com:".insteadOf "https://github.com/"'

# Install Go inside WSL2 (where E2E tests run)
Write-Output "Installing Go in WSL2..."
wsl -d Ubuntu -- bash -c "curl -fsSL https://go.dev/dl/go1.25.1.linux-amd64.tar.gz -o /tmp/go.tar.gz"
wsl -d Ubuntu -- bash -c "sudo tar -C /usr/local -xzf /tmp/go.tar.gz"
wsl -d Ubuntu -- bash -c "sudo ln -sf /usr/local/go/bin/go /usr/local/bin/go"
wsl -d Ubuntu -- bash -c "go version"

# Install Git (ignore cert errors from winget - fall back to WSL git)
Write-Output "Installing Git..."
winget install --id Git.Git -e --silent --accept-package-agreements --accept-source-agreements 2>$null

# Install Docker Engine inside WSL2
Write-Output "Installing Docker Engine in WSL2..."
wsl -d Ubuntu -- bash -c "curl -fsSL https://get.docker.com | sh && sudo usermod -aG docker `$USER"
if ($LASTEXITCODE -ne 0) {
    Write-Error "Docker Engine installation failed in WSL2"
    exit 1
}

# Install BATS inside WSL2
Write-Output "Installing BATS in WSL2..."
wsl -d Ubuntu -- bash -c "sudo apt-get install -y npm && sudo npm install -g bats"

# Install Make + build tools inside WSL2
wsl -d Ubuntu -- bash -c "sudo apt-get install -y make build-essential"

Write-Output ""
Write-Output "=== Phase 2 complete ==="
