#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Registers Windows Scheduled Tasks for autonomous Vagrant E2E testing.

.DESCRIPTION
    Creates one Scheduled Task per WSL2 kernel version, plus a cleanup task.
    Tasks run with highest privileges so Claude Code can trigger them from
    a non-elevated shell via 'schtasks /run /tn <TaskName>'.

    Run this script ONCE in an elevated PowerShell. Re-run when adding
    new kernel versions to the build matrix.

.EXAMPLE
    .\Register-TestTasks.ps1
#>

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$TestScript = Join-Path $ScriptDir "test-wsl2-kernel.ps1"
$LogDir = Join-Path $env:USERPROFILE ".datadatdat\test-logs"
$ArtifactBase = "C:\dev\datadatdat\zfs-releases\test-artifacts"

# Kernel versions matching the wsl2.yml build matrix
$versions = @(
    @{ Name = "5.15"; BzImage = "$ArtifactBase\5.15\bzImage" }
    @{ Name = "6.1";  BzImage = "$ArtifactBase\6.1\bzImage" }
    @{ Name = "6.6";  BzImage = "$ArtifactBase\6.6\bzImage" }
)

Write-Output "=== Registering WSL2 Kernel Test Tasks ==="
Write-Output "Log directory: $LogDir"
Write-Output "Test script:   $TestScript"
Write-Output ""

if (-not (Test-Path $LogDir)) {
    New-Item -ItemType Directory -Path $LogDir -Force | Out-Null
    Write-Output "Created log directory: $LogDir"
}

foreach ($v in $versions) {
    $taskName = "D3-Test-WSL2-$($v.Name)"
    $bzImage = $v.BzImage

    # Build the PowerShell command that the task will execute
    # Set PSDefaultParameterValues for UTF8 output, then redirect with *>
    $cmd = @"
-ExecutionPolicy Bypass -NoProfile -Command "& {
    `$PSDefaultParameterValues['Out-File:Encoding'] = 'utf8'
    `$logDir = '$LogDir'
    if (-not (Test-Path `$logDir)) { New-Item -ItemType Directory -Path `$logDir -Force | Out-Null }
    `$ts = Get-Date -Format 'yyyyMMdd-HHmmss'
    `$log = Join-Path `$logDir ('wsl2-$($v.Name)-' + `$ts + '.log')
    & '$TestScript' -BzImagePath '$bzImage' *> `$log
    `$result = `$LASTEXITCODE
    Add-Content `$log ('EXIT_CODE=' + `$result)
    exit `$result
}"
"@

    $action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument $cmd -WorkingDirectory $ScriptDir
    $principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -RunLevel Highest -LogonType S4U
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit (New-TimeSpan -Hours 2)

    Register-ScheduledTask -TaskName $taskName -Action $action -Principal $principal -Settings $settings -Force | Out-Null
    Write-Output "Registered: $taskName -> $bzImage"
}

# Register cleanup task - force kills processes, stops VM via Hyper-V, then vagrant destroy
$destroyCmd = @"
-ExecutionPolicy Bypass -NoProfile -Command "& {
    Write-Output 'Killing vagrant/ruby processes...'
    Get-Process -Name ruby, vagrant -ErrorAction SilentlyContinue | Stop-Process -Force
    Start-Sleep -Seconds 2
    Write-Output 'Force-stopping Hyper-V VMs...'
    Get-VM | Where-Object { `$_.Name -match 'wsl2' } | Stop-VM -TurnOff -Force -ErrorAction SilentlyContinue
    Get-VM | Where-Object { `$_.Name -match 'wsl2' } | Remove-VM -Force -ErrorAction SilentlyContinue
    Write-Output 'Running vagrant destroy...'
    Set-Location '$ScriptDir'
    vagrant destroy -f 2>&1
    Write-Output 'Cleanup complete.'
}"
"@
$destroyAction = New-ScheduledTaskAction -Execute "powershell.exe" -Argument $destroyCmd -WorkingDirectory $ScriptDir
$destroyPrincipal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -RunLevel Highest -LogonType S4U
$destroySettings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -ExecutionTimeLimit (New-TimeSpan -Minutes 10)

Register-ScheduledTask -TaskName "D3-Vagrant-Destroy" -Action $destroyAction -Principal $destroyPrincipal -Settings $destroySettings -Force | Out-Null
Write-Output "Registered: D3-Vagrant-Destroy"

Write-Output ""
Write-Output "=== Registration complete ==="
Write-Output ""
Write-Output "Usage from Claude Code (non-elevated):"
Write-Output "  schtasks /run /tn `"D3-Test-WSL2-6.6`""
Write-Output "  schtasks /query /tn `"D3-Test-WSL2-6.6`" /fo LIST"
Write-Output "  tail -f ~/.datadatdat/test-logs/wsl2-6.6-*.log"
Write-Output ""
Write-Output "Cleanup:"
Write-Output "  schtasks /run /tn `"D3-Vagrant-Destroy`""
