#requires -version 5.1
<#
 Switch to VirtualBox on Windows 11
 - Stops Docker Desktop and WSL2 cleanly
 - Disables the Windows hypervisor (so VirtualBox runs with its own VT-x)
 - Launches VirtualBox if no reboot is required; otherwise triggers a reboot
 NOTE: On some Windows 11 configs, additional "Core Isolation > Memory integrity"
 must be OFF for best VirtualBox performance. This script leaves security
 features alone; toggle manually if needed.
#>

function Ensure-Elevated {
  $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
  $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
  if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "Re-launching elevated..."
    $args = "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`""
    Start-Process -FilePath "powershell.exe" -ArgumentList $args -Verb RunAs
    exit
  }
}
Ensure-Elevated

$needReboot = $false

# 1) Stop Docker Desktop and related services
Write-Host "Stopping Docker Desktop and related services..."
Get-Process -Name "Docker Desktop","com.docker.backend","com.docker.proxy","vmmem" -ErrorAction SilentlyContinue | Stop-Process -Force
Stop-Service -Name "com.docker.service" -Force -ErrorAction SilentlyContinue
# Stop WSL2 cleanly
try { wsl --shutdown | Out-Null } catch { }
# Stop container networking bits
Stop-Service -Name "vmcompute" -Force -ErrorAction SilentlyContinue
Stop-Service -Name "hns" -Force -ErrorAction SilentlyContinue
Stop-Service -Name "LxssManager" -Force -ErrorAction SilentlyContinue

# 2) Turn the Windows hypervisor OFF (so VirtualBox can use VT-x directly)
try {
  $currentCfg = (bcdedit /enum {current}) 2>$null | Out-String
  if ($currentCfg -notmatch "hypervisorlaunchtype\s+Off") {
    Write-Host "Disabling Windows hypervisor (hypervisorlaunchtype=Off)..."
    bcdedit /set hypervisorlaunchtype Off | Out-Null
    $needReboot = $true
  }
} catch { }

# 3) If no reboot needed, launch VirtualBox
$vbExe = Join-Path $env:ProgramFiles "Oracle\VirtualBox\VirtualBox.exe"
if (-not $needReboot) {
  if (Test-Path $vbExe) {
    Write-Host "Launching VirtualBox..."
    Start-Process -FilePath $vbExe
    Write-Host "Switched to VirtualBox mode."
  } else {
    Write-Host "VirtualBox not found at: $vbExe"
  }
} else {
  Write-Host "A reboot is required to finish switching to VirtualBox. Rebooting in 5 seconds..."
  Start-Process -FilePath "shutdown.exe" -ArgumentList '/r /t 5 /c "Switching to VirtualBox mode"' | Out-Null
}
