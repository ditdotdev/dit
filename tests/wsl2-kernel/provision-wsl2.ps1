#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Provisions WSL2 + Ubuntu + Docker Desktop + dev tools inside a Vagrant VM.
#>

$ErrorActionPreference = "Stop"

Write-Host "=== Provisioning WSL2 Environment ===" -ForegroundColor Cyan

# Enable Windows features for WSL2
Write-Host "Enabling WSL2 features..."
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Windows-Subsystem-Linux -NoRestart -All
Enable-WindowsOptionalFeature -Online -FeatureName VirtualMachinePlatform -NoRestart -All

# Set WSL2 as default
wsl --set-default-version 2

# Install Ubuntu
Write-Host "Installing Ubuntu distribution..."
wsl --install -d Ubuntu --no-launch

# Install Docker Desktop (silent)
Write-Host "Installing Docker Desktop..."
$dockerInstaller = "$env:TEMP\DockerDesktop.exe"
if (-not (Test-Path $dockerInstaller)) {
    Invoke-WebRequest -Uri "https://desktop.docker.com/win/main/amd64/Docker%20Desktop%20Installer.exe" -OutFile $dockerInstaller
}
Start-Process -FilePath $dockerInstaller -Args "install --quiet --accept-license" -Wait

# Install Go
Write-Host "Installing Go..."
$goInstaller = "$env:TEMP\go.msi"
if (-not (Test-Path $goInstaller)) {
    Invoke-WebRequest -Uri "https://go.dev/dl/go1.25.1.windows-amd64.msi" -OutFile $goInstaller
}
Start-Process -FilePath "msiexec.exe" -Args "/i `"$goInstaller`" /quiet" -Wait

# Install Git
Write-Host "Installing Git..."
winget install --id Git.Git -e --silent --accept-package-agreements --accept-source-agreements 2>$null

# Install BATS inside WSL2
Write-Host "Installing BATS in WSL2..."
wsl -d Ubuntu -- bash -c "sudo apt-get update && sudo apt-get install -y npm && sudo npm install -g bats"

# Install Make inside WSL2
wsl -d Ubuntu -- bash -c "sudo apt-get install -y make build-essential"

Write-Host ""
Write-Host "=== Provisioning complete ===" -ForegroundColor Green
Write-Host "A reboot may be required for WSL2 to function properly."
Write-Host "Run: vagrant reload && vagrant provision --provision-with deploy-kernel"
