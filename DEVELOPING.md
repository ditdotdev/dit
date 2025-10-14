# Project Development

For general information about contributing changes, see the
[Contributor Guidelines](https://github.com/datadatdat/.github/blob/master/CONTRIBUTING.md).

## How it Works

Datadatdat is written with GoLang and consists of a CLI that manages Docker containers for data versioning.

## Docker Hub Images

Datadatdat pulls the following images from Docker Hub during installation and operation:

1. **`datadatdat/datadatdat:v1.0.0`** - Main server container
   - Contains the datadatdat server, ZFS utilities, PostgreSQL database, and docker-volume-proxy
   - Pulled during `d3 install` command
   - Handles all data versioning operations and Docker volume management

2. **`datadatdat/zfs-builder:v1.0.0`** - ZFS module builder
   - Contains build tools and ZFS source code for dynamic kernel module compilation
   - Pulled automatically when ZFS modules need to be built for kernel compatibility
   - Used when precompiled ZFS modules are not available for the current kernel

3. **`datadatdat/ssh-test-server:v1.0.0`** - SSH testing server
   - Contains SSH server for testing remote repository operations
   - Pulled only during end-to-end testing of SSH remote functionality
   - Not required for normal datadatdat operation

The CLI supports a `--registry` flag during installation to specify alternative Docker registries, but defaults to `datadatdat`.

## Requirements
*  GoLang 1.13.5
*  Make

### Windows/WSL2 Additional Requirements
If you are developing on Windows using Docker Desktop with WSL2, you must set up ZFS pools before running end-to-end tests:

```powershell
cd cleanslate
powershell -ExecutionPolicy Bypass -File setup-zfs-pools.ps1
```

This script creates the necessary ZFS infrastructure in WSL2 that the Datadatdat containers require. This step is only needed once per WSL2 environment (or after running the clean slate script).

###Setting up Documentation Building
Please read the details in /docs/README.md. As a prerequisite, you must:

* Ensure that Python3 is installed
* Ensure that virtualenv is installed, if not, execute the following:

```bash
pip install virtualenv
```

## Building
```bash
make build
```

## Testing
Datadatdat testing is handled by a simple e2e framework. Full test suite requires that an SSH Key and AWS CLI are configured.

**Important**: Windows/WSL2 users must run the ZFS pool setup script before running tests:

```powershell
cd cleanslate
powershell -ExecutionPolicy Bypass -File setup-zfs-pools.ps1
```

```bash
make e2e
```


## Releasing
```bash
make release
```