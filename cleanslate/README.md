# Clean Slate Testing for Datadatdat

> **Note**: All clean slate testing scripts are located in this `cleanslate` folder. Run scripts from within this directory or use the relative path `./cleanslate/script-name.sh` (Bash) from the main Datadatdat directory.

## Quick Start

For complete automated clean slate testing:

```bash
cd cleanslate
bash clean-slate-automation.sh --verbose
```

**For troubleshooting existing Docker issues:**
```bash
bash troubleshoot-docker.sh --verbose --fix
```

**For ZFS pool management with Docker verification:**
```bash
bash setup-zfs-pools.sh --clean --verify-docker
```

## Available Scripts

1. **`clean-slate-automation.sh`** - Complete automation for the entire clean slate process
   - Full environment teardown and rebuild
   - PostgreSQL testing
   - Enhanced error handling and verbose logging
   - Usage: `bash clean-slate-automation.sh --verbose`

2. **`setup-zfs-pools.sh`** - Enhanced ZFS pool management with Docker verification
   - `--clean` parameter for complete pool reset
   - `--verify-docker` parameter for container conflict detection
   - Automated troubleshooting guidance
   - Usage: `bash setup-zfs-pools.sh --clean --verify-docker`

3. **`troubleshoot-docker.sh`** - Comprehensive Docker diagnostic tool
   - Container name conflict detection and resolution
   - Docker socket testing and verification
   - Automatic fix capabilities with `--fix` parameter
   - Usage: `bash troubleshoot-docker.sh --verbose --fix`

## Prerequisites

Before running these scripts, ensure you have:
- Windows Subsystem for Linux 2 (WSL2) installed and running
- Docker Desktop installed (scripts will automatically start it if not running)
- Bash shell (Git Bash, WSL Bash, or similar)
- Administrative privileges for ZFS operations
- Custom ZFS-enabled WSL2 kernel
- Git repositories: `datadatdat` and `zfs-builder`

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
1. Complete removal of existing Datadatdat infrastructure
2. Docker environment cleanup
3. ZFS pool preparation
4. Custom container building
5. Fresh Datadatdat installation
6. Database repository creation and testing
7. Data versioning and rollback verification

## Manual Step-by-Step Process

### 1. Environment Preparation

#### Uninstall Existing Datadatdat
```bash
cd c:/dev/datadatdat
./d3.exe uninstall -f
```

#### Clean Docker Environment
```bash
# Stop and remove all containers
docker stop $(docker ps -aq) 2>/dev/null || true
docker rm $(docker ps -aq) 2>/dev/null || true

# Remove all volumes
docker volume prune -f

# Remove all networks
docker network prune -f

# Remove all images and build cache
docker system prune -a -f
```

#### Clean ZFS Pools (Complete Clean Slate)
```bash
# Use the setup script with --clean parameter to remove existing pools
bash setup-zfs-pools.sh --clean
```

#### Restart Docker Desktop
```bash
# Stop Docker Desktop
taskkill //F //IM "Docker Desktop.exe" 2>/dev/null || true
sleep 5
# Start Docker Desktop
"/c/Program Files/Docker/Docker/Docker Desktop.exe" &
sleep 30  # Wait for Docker to start
```

### 2. ZFS Pool Setup

#### Automated Setup (Recommended)
For normal setup:
```bash
bash setup-zfs-pools.sh
```

For clean slate setup (includes cleanup):
```bash
bash setup-zfs-pools.sh --clean
```

#### Manual Setup (Alternative)
```bash
# Create pool storage directory
wsl sudo mkdir -p /datadatdat-pools

# Create datadatdat-docker pool (required by Datadatdat)
wsl sudo dd if=/dev/zero of=/datadatdat-pools/datadatdat-docker.img bs=1M count=1024
wsl sudo losetup -f /datadatdat-pools/datadatdat-docker.img
wsl sudo zpool create datadatdat-docker /dev/loop3  # adjust loop device as needed

# Create main datadatdat pool (optional but recommended)
wsl sudo dd if=/dev/zero of=/datadatdat-pools/datadatdat.img bs=1M count=1024
wsl sudo losetup -f /datadatdat-pools/datadatdat.img
wsl sudo zpool create datadatdat /dev/loop2  # adjust loop device as needed

# Verify pools
wsl zpool list
wsl zpool status
```

**Expected Output:**
```
NAME           SIZE  ALLOC   FREE  CKPOINT  EXPANDSZ   FRAG    CAP  DEDUP    HEALTH  ALTROOT
datadatdat          960M  53.0M   907M        -         -     0%     5%  1.00x    ONLINE  -
datadatdat-docker   960M   104K   960M        -         -     0%     0%  1.00x    ONLINE  -
```

### 3. Container Building

Build the custom Datadatdat container with ZFS support:

```bash
cd c:/dev/datadatdat
docker build -t datadatdat:latest -f Dockerfile . --no-cache
```

### 4. Datadatdat Installation

Install Datadatdat using the custom container:

```bash
./d3.exe install
```

### 5. Database Testing

#### Create PostgreSQL Repository
```bash
./d3.exe run --name pgtest -e POSTGRES_PASSWORD=password postgres
```

#### Verify Database Connectivity
```bash
# Check container is running
docker exec datadatdat-docker-launch docker ps

# Test database connection
docker exec datadatdat-docker-launch docker exec pgtest psql -U postgres -c "SELECT version();"
```

#### Test Data Versioning
```bash
# Create some test data
docker exec datadatdat-docker-launch docker exec pgtest psql -U postgres -c "CREATE TABLE test (id SERIAL PRIMARY KEY, name VARCHAR(100));"
docker exec datadatdat-docker-launch docker exec pgtest psql -U postgres -c "INSERT INTO test (name) VALUES ('Test Entry 1'), ('Test Entry 2');"

# Commit the changes
./d3.exe commit -m "Initial test data" pgtest

# Add more data
docker exec datadatdat-docker-launch docker exec pgtest psql -U postgres -c "INSERT INTO test (name) VALUES ('Test Entry 3'), ('Test Entry 4');"

# Create another commit
./d3.exe commit -m "Additional test data" pgtest

# Verify commits
./d3.exe log pgtest
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
**Solution**: Run `bash setup-zfs-pools.sh --clean` to recreate pools

#### Docker Desktop Not Running
**Issue**: "Cannot connect to the Docker daemon"
**Solution**: Scripts automatically detect and start Docker Desktop. If manual intervention is needed, ensure Docker Desktop is installed and WSL2 integration is enabled. Use `bash troubleshoot-docker.sh --fix` for automatic startup.

### Diagnostic Commands

```bash
# Check Docker status
docker version
docker info

# Check ZFS pools
wsl zpool list
wsl zpool status

# Check Datadatdat status
./d3.exe status

# Check running containers
docker ps -a

# Check Datadatdat logs
docker logs datadatdat-docker-server
```

## Running from Root Directory

If you want to run these scripts from the main Datadatdat directory, use:

```bash
# From c:/dev/datadatdat/
bash cleanslate/clean-slate-automation.sh --verbose
bash cleanslate/setup-zfs-pools.sh --clean --verify-docker
bash cleanslate/troubleshoot-docker.sh --verbose --fix
```

## Verified Working Components

✅ **ZFS Integration**: Custom kernel with built-in ZFS support  
✅ **Container Building**: Docker builds complete successfully with all dependencies  
✅ **Pool Management**: Automated ZFS pool creation and management  
✅ **Datadatdat Installation**: Clean installation process works reliably  
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
