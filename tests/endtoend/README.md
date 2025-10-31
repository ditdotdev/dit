## End to End Tests

### Prerequisites

**Windows/WSL2 users**: Before running end-to-end tests, you must set up ZFS pools:

```powershell
cd cleanslate
powershell -ExecutionPolicy Bypass -File setup-zfs-pools.ps1
```

This creates the necessary ZFS infrastructure in WSL2 that Datadatdat containers require. This step is only needed once per WSL2 environment.

### Running Tests

```bash
# Run all end-to-end tests
make e2e

# Or run individual test suites
make test-install
make test-getting-started
make test-s3-workflow
make test-ssh-workflow
```

## Prerequisites

*   Install BATS: `npm install -g bats`
*   Make sure datadatdat (d3) and docker are both in PATH

## Getting Started Tests
```bash
bats tests/endtoend/getting-started/getting-started.bats
# Or via Makefile
make test-getting-started
```

## S3 Tests
The following environment variables must be set:

* AWS_ACCESS_KEY_ID
* AWS_SECRET_ACCESS_KEY
* AWS_REGION

Alternately, `aws configure` can be used to set up AWS access. 

```bash
# Run S3 workflow tests
make test-s3-workflow

# Or run directly with BATS
bats tests/endtoend/remotes/s3/s3-workflow.bats
```

## SSH Tests
An SSH Keyfile must be created. The script `generateKey.sh` in the ssh test directory can assist with this. 

