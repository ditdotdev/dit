# Project Development

For general information about contributing changes, see the
[Contributor Guidelines](https://github.com/ditdotdev/.github/blob/master/CONTRIBUTING.md).

## How it Works

Dit is written with GoLang and consists of a CLI that manages Docker containers for data versioning.

## Docker Hub Images

Dit pulls the following images from Docker Hub during installation and operation:

1. **`ditdotdev/dit:v1.0.0`** - Main server container
   - Contains the dit server, ZFS utilities, PostgreSQL database, and docker-volume-proxy
   - Pulled during `dit install` command
   - Handles all data versioning operations and Docker volume management

2. **`ditdotdev/zfs-builder:v1.0.0`** - ZFS module builder
   - Contains build tools and ZFS source code for dynamic kernel module compilation
   - Pulled automatically when ZFS modules need to be built for kernel compatibility
   - Used when precompiled ZFS modules are not available for the current kernel

3. **`ditdotdev/ssh-test-server:v1.0.0`** - SSH testing server
   - Contains SSH server for testing remote repository operations
   - Pulled only during end-to-end testing of SSH remote functionality
   - Not required for normal dit operation

The CLI supports a `--registry` flag during installation to specify alternative Docker registries, but defaults to `ditdotdev`.

## Requirements
*  GoLang 1.13.5
*  Make

### Host ZFS Pool Setup
On a fresh native-Linux or WSL2 development box, provision the host ZFS pools that the dit server expects:

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
Dit testing is handled by a simple e2e framework. Full test suite requires that an SSH Key and AWS CLI are configured.

**Important**: On a fresh dev box, run the ZFS pool setup script once before running tests:

```bash
bash scripts/setup-zfs-pools.sh
make e2e
```

### Database Matrix

The Database Matrix workflow (`.github/workflows/db-matrix-test.yml`) runs
`tests/endtoend/db-matrix/db-matrix.bats` against every database image we
claim to support, one job per image, weekly and on `workflow_dispatch`. For
each image it runs the whole dit flow (run, cp, commit, checkout, push, pull,
clone, migrate) and proves the data survived twice: a marker file planted in
every declared volume, and one object created through the engine's own client
(one row, document, key, node, index entry or bucket object) that is read back
after checkout, clone and migrate.

The workflow only pulls images. It never builds the image it is about to
test. Images that need a change to work with dit (a `VOLUME` declaration, a
container command, or a client the test uses) are wrapped by a Dockerfile in
`tests/endtoend/db-matrix/` and published to Docker Hub under `ditdotdev/`.

To add a database, or a new version of one:

1. **Pick the image.** If the upstream image declares a `VOLUME` for its data
   directory and starts with its default command, use the upstream tag as is.
   Otherwise add `tests/endtoend/db-matrix/<image>-<tag>.Dockerfile` that
   starts `FROM` the upstream image and adds what is missing: `VOLUME` for the
   data directory, `CMD` when the entrypoint needs a subcommand (`dit run`
   cannot pass a container command; its trailing arguments are docker flags),
   and the client package when the image ships none (curl for HTTP APIs, the
   `mysql` client for MySQL-protocol engines). Keep the name
   `<image>-<tag>.Dockerfile`; the local harness finds it that way.

2. **Add the boot table entries** in
   `tests/endtoend/db-matrix/db-matrix-engines.bash`: `db_env_args` for
   environment the engine needs to start and to accept the client without
   interactive setup (passwords, trust auth, single-node mode), `db_dit_flags`
   for flags only `dit run` takes (`--privileged`, `--disable-port-mapping`),
   and `db_docker_flags` for flags only the unmanaged `docker run` in the
   migrate test takes.

3. **Add the client table entries** in the same file: `db_ready` (one cheap
   query that succeeds only when the engine answers; "Up" is not enough),
   `db_write` (create the object, idempotent, value is the image reference),
   and `db_verify` (read it back, non-zero if missing or wrong). Add a
   `db_ready_timeout` entry if a cold boot takes longer than three minutes.
   If the image has no client and no HTTP API reachable from inside the
   container, leave it to the default case and add it to the comment there
   so the gap is visible, and note it in `tests/endtoend/TEST-COVERAGE.md`.

4. **Check it locally** without a dit install. The harness builds the wrapper
   if there is one, starts a plain container with the boot table, waits for
   the engine, writes, restarts the container the way `dit cp` and
   `dit checkout` do, and reads the object back:

   ```bash
   tests/endtoend/db-matrix/check-engines.sh postgres:16 ditdotdev/redis:7
   ```

5. **Publish the wrapper** for both runner architectures once it passes. The
   Docker Hub user is the one in the org's `DOCKER_USER` secret; your personal
   login cannot push to `ditdotdev/`.

   ```bash
   cd tests/endtoend/db-matrix
   docker buildx build --platform linux/amd64,linux/arm64 \
     -t ditdotdev/<image>:<tag> -f <image>-<tag>.Dockerfile --push .
   ```

   Republish the same way whenever the Dockerfile changes; the matrix pulls
   whatever is on Docker Hub under that tag.

6. **Add the matrix entry** in `.github/workflows/db-matrix-test.yml` under
   the vendor's comment block, using the `ditdotdev/` tag if you published a
   wrapper, and run the workflow on your branch with `workflow_dispatch`
   before opening the PR:

   ```bash
   gh workflow run db-matrix-test.yml --ref <branch>
   ```


## Releasing
```bash
make release
```