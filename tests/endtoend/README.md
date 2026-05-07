## End to End Tests

### Prerequisites

On a fresh dev box (native Linux or WSL2), provision the host ZFS pools once:

```bash
bash scripts/setup-zfs-pools.sh
```

This creates the loop-backed ZFS pools that the Datadatdat containers require. Pass `--clean` to destroy and recreate them.

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

