# WSL2 Kernel E2E Test Infrastructure

End-to-end validation of prebuilt WSL2 kernels with ZFS using a Vagrant
Hyper-V VM. **No kernel release is published unless ALL tests pass.**

## Prerequisites

- **Vagrant**: `winget install Hashicorp.Vagrant`
- **Hyper-V**: Already active if you use Docker Desktop or WSL2
- **gh CLI**: Authenticated with `gh auth login`

## Quick Start

```powershell
# Test a local bzImage
.\test-wsl2-kernel.ps1 -BzImagePath C:\path\to\bzImage

# Test a draft GitHub Release
.\test-wsl2-kernel.ps1 -ReleaseTag wsl2-kernel-6.6.75.2-zfs-2.3.4

# Keep VM alive for debugging
.\test-wsl2-kernel.ps1 -BzImagePath .\bzImage -KeepVM
```

## What It Does

1. **vagrant up** — Starts a Windows 11 Hyper-V VM with nested virtualization
2. **provision-wsl2.ps1** — Installs WSL2 + Ubuntu + Docker Desktop + Go + BATS
3. **vagrant reload** — Reboots to activate WSL2 features
4. **deploy-kernel.ps1** — Copies bzImage, configures `.wslconfig`, restarts WSL2
5. **run-e2e.sh** — Inside WSL2: validates kernel, sets up ZFS pools, runs
   `make e2e` (12 test suites) and `make e2e-server` (10+ test suites)
6. **vagrant destroy** — Cleans up the VM (unless `-KeepVM`)

## Files

| File | Description |
|------|-------------|
| `Vagrantfile` | Windows 11 Hyper-V VM definition |
| `provision-wsl2.ps1` | Installs WSL2 + Docker + dev tools |
| `deploy-kernel.ps1` | Deploys bzImage and configures .wslconfig |
| `run-e2e.sh` | Runs full E2E suite inside WSL2 |
| `test-wsl2-kernel.ps1` | Orchestrator script |

## Why Hyper-V?

- Hyper-V is already active on your machine (Docker Desktop / WSL2)
- VirtualBox requires disabling Hyper-V, breaking Docker and WSL2
- Hyper-V supports nested virtualization so WSL2 works inside the VM
