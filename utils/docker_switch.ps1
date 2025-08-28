#requires -version 5.1
<# 
 Switch to Docker Desktop on Windows 11
 - Gracefully stops any running VirtualBox VMs
 - Enables the Windows hypervisor + WSL2 features needed by Docker Desktop (WSL backend)
 - Starts Docker if no reboot is required; otherwise triggers a reboot
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

# 1) Gracefully stop VirtualBox VMs (then force if needed)
$VBoxManage = Join-Path $env:ProgramFiles "Oracle\VirtualBox\VBoxManage.exe"
if (Test-Path $VBoxManage) {
  try {
    $running = & $VBoxManage list runningvms 2>$null
    if ($running) {
      Write-Host "Requesting ACPI shutdown for running VirtualBox VMs..."
      foreach ($line in $running) {
        if ($line -match '^\s*"(.+?)"\s+\{([0-9a-f\-]+)\}') {
          $uuid = $Matches[2]
          & $VBoxManage controlvm $uuid acpipowerbutton | Out-Null
        }
      }
      $sw = [Diagnostics.Stopwatch]::StartNew()
      while ($sw.Elapsed.TotalSeconds -lt 25) {
        Start-Sleep 2
        $still = & $VBoxManage list runningvms 2>$null
        if (-not $still) { break }
      }
      $still = & $VBoxManage list runningvms 2>$null
      if ($still) {
        Write-Host "Forcing power off for remaining VirtualBox VMs..."
        foreach ($line in $still) {
          if ($line -match '^\s*"(.+?)"\s+\{([0-9a-f\-]+)\}') {
            $uuid = $Matches[2]
            & $VBoxManage controlvm $uuid poweroff | Out-Null
          }
        }
      }
    }
  } catch { }
  Get-Process -Name "VirtualBox","VBoxHeadless","VBoxSVC" -ErrorAction SilentlyContinue | Stop-Process -Force
}

# 2) Turn the Windows hypervisor ON (required for Docker/WSL2)
try {
  $currentCfg = (bcdedit /enum {current}) 2>$null | Out-String
  if ($currentCfg -notmatch "hypervisorlaunchtype\s+Auto") {
    Write-Host "Enabling Windows hypervisor (hypervisorlaunchtype=Auto)..."
    bcdedit /set hypervisorlaunchtype Auto | Out-Null
    $needReboot = $true
  }
} catch { }

# 3) Ensure WSL2 features are enabled (no-restart)
$features = @("Microsoft-Windows-Subsystem-Linux","VirtualMachinePlatform")
foreach ($f in $features) {
  Write-Host "Enabling Windows feature: $f (if not already enabled)..."
  & dism /online /Enable-Feature /FeatureName:$f /All /NoRestart | Out-Null
  if ($LASTEXITCODE -eq 3010) { $needReboot = $true }
}

# 4) If no reboot needed, start services + Docker Desktop
if (-not $needReboot) {
  foreach ($svc in "vmcompute","LxssManager","com.docker.service") {
    $s = Get-Service -Name $svc -ErrorAction SilentlyContinue
    if ($s -and $s.Status -ne "Running") {
      try { Start-Service -Name $svc -ErrorAction Stop } catch { }
    }
  }
  $dockerExe = Join-Path $env:ProgramFiles "Docker\Docker\Docker Desktop.exe"
  if (Test-Path $dockerExe) {
    Write-Host "Starting Docker Desktop..."
    Start-Process -FilePath $dockerExe
  } else {
    Write-Host "Docker Desktop not found at: $dockerExe"
  }
  Write-Host "Switched to Docker mode."
} else {
  Write-Host "A reboot is required to finish switching to Docker. Rebooting in 5 seconds..."
  Start-Process -FilePath "shutdown.exe" -ArgumentList '/r /t 5 /c "Switching to Docker mode"' | Out-Null
}
