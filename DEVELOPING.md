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

### Host ZFS Pool Setup
On a fresh native-Linux or WSL2 development box, provision the host ZFS pools that the d3 server expects:

```bash
bash scripts/setup-zfs-pools.sh
```

This step is only needed once per environment. Pass `--clean` to destroy and recreate the pools.

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

### Building against private ditdotdev Go modules

All Go modules in this org live under the private `github.com/ditdotdev/*` path. To
fetch them locally (`go build`, `go test`, `go get`, `go mod tidy`) you need:

1. Git credentials for the private repos — e.g. `gh auth setup-git` (uses your `gh`
   token as a git credential helper), or an SSH `insteadOf` rewrite.
2. The module-privacy settings passed as **environment variables**, not via
   `go env -w`.

```bash
export GOPRIVATE='github.com/ditdotdev/*'
export GONOSUMDB='github.com/ditdotdev/*'
export GOPROXY=direct
# add GOFLAGS=-mod=mod only when you want `go get`/`go mod tidy` to update go.mod/go.sum
```

**Why env vars, not `go env -w`:** these modules require `go >= 1.26.2`. If your
default `go` is older, the `go` command auto-downloads and re-execs a newer
toolchain — and values set with `go env -w GOPRIVATE=...` do **not** propagate into
that switched toolchain. The new toolchain then treats `github.com/ditdotdev/*` as
public, tries to verify it against `sum.golang.org` (404, since it's private), and
fails with a misleading `could not read Username for 'https://github.com'` error.
Exporting `GOPRIVATE` **and** `GONOSUMDB` as env vars survives the toolchain switch.
Do **not** use `GOSUMDB=off` — that disables checksum verification for all public
modules (including the toolchain download itself).

> When developing against an unreleased version of a sibling module, use a `replace`
> directive in `go.mod` pointing at the local checkout (e.g.
> `replace github.com/ditdotdev/remote-sdk-go => ../remote-sdk-go`). The release
> process strips `replace` directives before tagging (see `release.sh` Phase 0).

## Testing
Datadatdat testing is handled by a simple e2e framework. Full test suite requires that an SSH Key and AWS CLI are configured.

**Important**: On a fresh dev box, run the ZFS pool setup script once before running tests:

```bash
bash scripts/setup-zfs-pools.sh
make e2e
```


## Releasing
```bash
make release
```