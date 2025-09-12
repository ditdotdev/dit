# Project Development

For general information about contributing changes, see the
[Contributor Guidelines](https://github.com/titan-data/.github/blob/master/CONTRIBUTING.md).

## How it Works

Titan is written with GoLang,

## Requirements
*  GoLang 1.13.5
*  Make

### Windows/WSL2 Additional Requirements
If you are developing on Windows using Docker Desktop with WSL2, you must set up ZFS pools before running end-to-end tests:

```powershell
cd cleanslate
powershell -ExecutionPolicy Bypass -File setup-zfs-pools.ps1
```

This script creates the necessary ZFS infrastructure in WSL2 that the Titan containers require. This step is only needed once per WSL2 environment (or after running the clean slate script).

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
Titan testing is handled by a simple e2e framework. Full test suite requires that an SSH Key and AWS CLI are configured.

**Windows/WSL2 users**: Ensure you have run the ZFS pool setup script first (see Requirements section above).

```bash
make e2e
```


## Releasing
```bash
make release
```