# Clean Slate Testing for Titan

> **Note**: All clean slate testing scripts are located in this `cleanslate` folder. Run scripts from within this directory or use the relative path `.\cleanslate\script-name.ps1` from the main Titan directory.

## Quick Start

For complete automated clean slate testing:

```powershell
cd cleanslate
.\clean-slate-automation.ps1 -Verbose
```

**For troubleshooting existing Docker issues:**
```powershell
.\troubleshoot-docker.ps1 -Verbose -Fix
```

**For ZFS pool management with Docker verification:**
```powershell
.\setup-zfs-pools.ps1 -Clean -VerifyDocker
```

## Available Scripts

1. **`clean-slate-automation.ps1`** - Complete automation for the entire clean slate process
   - Full environment teardown and rebuild
   - PostgreSQL testing
   - Enhanced error handling and verbose logging
   - Usage: `.\clean-slate-automation.ps1 -Verbose`

2. **`setup-zfs-pools.ps1`** - Enhanced ZFS pool management with Docker verification
   - `-Clean` parameter for complete pool reset
   - `-VerifyDocker` parameter for container conflict detection
   - Automated troubleshooting guidance
   - Usage: `.\setup-zfs-pools.ps1 -Clean -VerifyDocker`

3. **`troubleshoot-docker.ps1`** - Comprehensive Docker diagnostic tool
   - Container name conflict detection and resolution
   - Docker socket testing and verification
   - Automatic fix capabilities with `-Fix` parameter
   - Usage: `.\troubleshoot-docker.ps1 -Verbose -Fix`

## Prerequisites

Before running these scripts, ensure you have:
- Windows Subsystem for Linux 2 (WSL2) installed and running
- Docker Desktop installed (scripts will automatically start it if not running)
- PowerShell 5.1 or higher
- Administrative privileges for ZFS operations
- Custom ZFS-enabled WSL2 kernel
- Git repositories: `titan` and `zfs-builder`

> **Note**: The scripts now automatically detect and start Docker Desktop if it's not running, so you don't need to manually start Docker before running the clean slate tests.

## Key Fixes Implemented

- **ZFS Integration**: Fixed `custom-zfs.sh` to properly detect built-in ZFS in WSL2 kernel
- **Container Naming**: Added detection and prevention of Docker container name conflicts
- **Error Diagnostics**: Created comprehensive troubleshooting tools for Docker execution issues
- **Process Automation**: Complete clean slate process can now be run with a single command
- **Docker Execution**: Fixed "exit status 127" during container creation by adding socat package to Dockerfile - volume driver now works correctly for all container types including PostgreSQL
- **Docker Auto-Start**: Scripts automatically detect and start Docker Desktop if not running, eliminating manual startup requirements

## Overview

A clean slate test involves:
1. Complete removal of existing Titan infrastructure
2. Docker environment cleanup
3. ZFS pool preparation
4. Custom container building
5. Fresh Titan installation
6. Database repository creation and testing
7. Data versioning and rollback verification

## Manual Step-by-Step Process

### 1. Environment Preparation

#### Uninstall Existing Titan
```powershell
cd c:\dev\titan
.\titan.exe uninstall -f
```

#### Clean Docker Environment
```powershell
# Stop and remove all containers
docker stop $(docker ps -aq) 2>$null
docker rm $(docker ps -aq) 2>$null

# Remove all volumes
docker volume prune -f

# Remove all networks
docker network prune -f

# Remove all images and build cache
docker system prune -a -f
```

#### Clean ZFS Pools (Complete Clean Slate)
```powershell
# Use the setup script with -Clean parameter to remove existing pools
.\setup-zfs-pools.ps1 -Clean
```

#### Restart Docker Desktop
```powershell
Stop-Process -Name "Docker Desktop" -Force -ErrorAction SilentlyContinue
Start-Sleep 5
Start-Process "C:\Program Files\Docker\Docker\Docker Desktop.exe"
Start-Sleep 30  # Wait for Docker to start
```

### 2. ZFS Pool Setup

#### Automated Setup (Recommended)
For normal setup:
```powershell
.\setup-zfs-pools.ps1
```

For clean slate setup (includes cleanup):
```powershell
.\setup-zfs-pools.ps1 -Clean
```

#### Manual Setup (Alternative)
```powershell
# Create pool storage directory
wsl sudo mkdir -p /titan-pools

# Create titan-docker pool (required by Titan)
wsl sudo dd if=/dev/zero of=/titan-pools/titan-docker.img bs=1M count=1024
wsl sudo losetup -f /titan-pools/titan-docker.img
wsl sudo zpool create titan-docker /dev/loop3  # adjust loop device as needed

# Create main titan pool (optional but recommended)
wsl sudo dd if=/dev/zero of=/titan-pools/titan.img bs=1M count=1024
wsl sudo losetup -f /titan-pools/titan.img
wsl sudo zpool create titan /dev/loop2  # adjust loop device as needed

# Verify pools
wsl zpool list
wsl zpool status
```

**Expected Output:**
```
NAME           SIZE  ALLOC   FREE  CKPOINT  EXPANDSZ   FRAG    CAP  DEDUP    HEALTH  ALTROOT
titan          960M  53.0M   907M        -         -     0%     5%  1.00x    ONLINE  -
titan-docker   960M   104K   960M        -         -     0%     0%  1.00x    ONLINE  -
```

### 3. Container Building

Build the custom Titan container with ZFS support:

```powershell
cd c:\dev\titan
docker build -t titan:latest -f Dockerfile . --no-cache
```

### 4. Titan Installation

Install Titan using the custom container:

```powershell
.\titan.exe install
```

### 5. Database Testing

#### Create PostgreSQL Repository
```powershell
.\titan.exe run --name pgtest -e POSTGRES_PASSWORD=password postgres
```

#### Verify Database Connectivity
```powershell
# Check container is running
docker exec titan-docker-launch docker ps

# Test database connection
docker exec titan-docker-launch docker exec pgtest psql -U postgres -c "SELECT version();"
```

#### Test Data Versioning
```powershell
# Create some test data
docker exec titan-docker-launch docker exec pgtest psql -U postgres -c "CREATE TABLE test (id SERIAL PRIMARY KEY, name VARCHAR(100));"
docker exec titan-docker-launch docker exec pgtest psql -U postgres -c "INSERT INTO test (name) VALUES ('Test Entry 1'), ('Test Entry 2');"

# Commit the changes
.\titan.exe commit -m "Initial test data" pgtest

# Add more data
docker exec titan-docker-launch docker exec pgtest psql -U postgres -c "INSERT INTO test (name) VALUES ('Test Entry 3'), ('Test Entry 4');"

# Create another commit
.\titan.exe commit -m "Additional test data" pgtest

# Verify commits
.\titan.exe log pgtest
```

## Troubleshooting

### Common Issues

#### Exit Status 127 Errors
**Issue**: Container creation fails with "exit status 127"
**Solution**: This was caused by missing socat package and has been fixed in the Dockerfile

#### Docker Container Name Conflicts
**Issue**: "container name already in use"
**Solution**: Run `.\troubleshoot-docker.ps1 -Fix` to detect and resolve conflicts

#### ZFS Pool Issues
**Issue**: Pools not found or corrupted
**Solution**: Run `.\setup-zfs-pools.ps1 -Clean` to recreate pools

#### Docker Desktop Not Running
**Issue**: "Cannot connect to the Docker daemon"
**Solution**: Scripts automatically detect and start Docker Desktop. If manual intervention is needed, ensure Docker Desktop is installed and WSL2 integration is enabled. Use `.\troubleshoot-docker.ps1 -Fix` for automatic startup.

### Diagnostic Commands

```powershell
# Check Docker status
docker version
docker info

# Check ZFS pools
wsl zpool list
wsl zpool status

# Check Titan status
.\titan.exe status

# Check running containers
docker ps -a

# Check Titan logs
docker logs titan-docker-server
```

## Running from Root Directory

If you want to run these scripts from the main Titan directory, use:

```powershell
# From c:\dev\titan\
.\cleanslate\clean-slate-automation.ps1 -Verbose
.\cleanslate\setup-zfs-pools.ps1 -Clean -VerifyDocker
.\cleanslate\troubleshoot-docker.ps1 -Verbose -Fix
```

## Verified Working Components

✅ **ZFS Integration**: Custom kernel with built-in ZFS support  
✅ **Container Building**: Docker builds complete successfully with all dependencies  
✅ **Pool Management**: Automated ZFS pool creation and management  
✅ **Titan Installation**: Clean installation process works reliably  
✅ **PostgreSQL Support**: Database containers start and run correctly  
✅ **Data Versioning**: Commit and rollback operations function properly  
✅ **Volume Driver**: Fixed socat dependency for proper container plugin functionality  

## Current Status

The clean slate testing process is fully functional with the following resolution:

- **Docker Execution**: Previously failing "exit status 127" errors have been resolved by adding the socat package to the Dockerfile
- **PostgreSQL Testing**: Database containers now start correctly and accept connections
- **Complete Automation**: The entire clean slate process can be run with a single command
- **Troubleshooting Tools**: Comprehensive diagnostic and repair capabilities available

The system is ready for production database testing and development work.
