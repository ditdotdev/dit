## End to End Tests

### Prerequisites

**Windows/WSL2 users**: Before running end-to-end tests, you must set up ZFS pools:

```powershell
cd cleanslate
powershell -ExecutionPolicy Bypass -File setup-zfs-pools.ps1
```

This creates the necessary ZFS infrastructure in WSL2 that Titan containers require. This step is only needed once per WSL2 environment.

### Running Tests

```
make e2e
```

## Manual Install

*   Download runner from [here](https://github.com/datadatdat/vexrun/releases)
*   `alias vexrun="java -jar vexrun-VERSION.jar"`
*   Make sure titan and docker are both in PATH

## Getting Started Tests
```
vexrun -d ./src/endtoend-test/getting-started
```

## S3 Tests
The following environment variables must be set:

* AWS_ACCESS_KEY_ID
* AWS_SECRET_ACCESS_KEY
* AWS_REGION

Alternately, `aws configure` can be used to set up AWS access. 

```bash
titan clone s3web://demo.titan-data.io/hello-world/postgres hello-world 
vexrun -f ./src/endtoend-test/remotes/RemoteWorkflowTests.yml -p REMOTE s3 -p URI s3://datadatdat-testdata/e2etest -p REPO hello-world
```

## SSH Tests
An SSH Keyfile must be created. The script `generateKey.sh` in the ssh test directory can assist with this. 

