# Datadatdat Install Flow

## Overview

When you run `d3 install`, the CLI sets up a complete data versioning infrastructure backed by ZFS. The end result is two Docker containers and a ZFS storage pool ready to accept API requests.

```
User runs: d3 install
         |
         v
   +---------------------+
   | Validate Docker      | <-- docker -v
   +----------+----------+
              v
   +---------------------+
   | Pull Docker image    | <-- datadatdat/datadatdat:$VERSION (~1GB)
   +----------+----------+
              v
   +---------------------+
   | Start launch         | <-- privileged, --pid=host
   | container            |     datadatdat-docker-launch
   +----------+----------+
              v
   +------------------------------------------+
   | ZFS Module Loading (4-step fallback)      |
   |  1. Already loaded? (lsmod + /proc)       |
   |  2. Host system modules? (modprobe)       |
   |  3. Install via package manager?          |
   |     (apt/dnf/pacman)                      |
   |  4. Download prebuilt + insmod from S3?   |
   +----------+-------------------------------+
              v
   +---------------------+
   | Create/Import ZFS    | <-- datadatdat-docker pool
   | pool + filesystems   |     data (LZ4), db
   +----------+----------+
              v
   +---------------------+
   | Start server         | <-- datadatdat-docker-server
   | container            |
   |  * Mount ZFS         |
   |  * Init PostgreSQL   |
   |  * Volume plugin     |
   |  * REST API (:5001)  |
   +----------+----------+
              v
        INSTALL COMPLETE
```

---

## Phase 1: CLI Command Execution

**File:** `internal/app/commands/install.go`

```go
var installCmd = &cobra.Command{
    Use:   "install",
    Short: "Install datadatdat infrastructure",
    Run: func(cmd *cobra.Command, args []string) {
        registry, _ := cmd.Flags().GetString("registry")
        provider = providers.Create(context, contextType, providers.GetAvailablePort())
        provider.Install([]string{"registry=" + registry}, verbose)
        providers.AddProvider(provider)
    },
}
```

- Parses `--registry` flag (default: `datadatdat` on Docker Hub)
- Creates a Docker provider with an available port (default: 5001)
- Calls `provider.Install()` which drives the rest of the process
- Saves provider config to `~/.datadatdat/config` (YAML)

---

## Phase 2: Docker Validation

**File:** `internal/app/providers/local/Install.go`

```go
if _, err := docker.Version(); err != nil {
    fmt.Printf("Error checking docker version: %v\n", err)
    os.Exit(1)
}
```

Runs `docker -v` to verify the Docker daemon is accessible. Exits immediately if Docker is not running.

---

## Phase 3: Docker Image Management

**File:** `internal/app/clients/Docker.go`

### Check if image exists locally

```go
out, _ := ce.Exec("docker", "images", registry+"/datadatdat", "--format", `"{{.Tag}}"`)
```

Compares local tags against the required version using semantic versioning. Skips download if a matching version exists.

### Pull image

```go
pullImage := pullRegistry + "/datadatdat:" + latest
docker.Pull(pullImage)
```

Downloads `datadatdat/datadatdat:$VERSION` from Docker Hub (~1GB).

**Image contents** (see `datadatdat-server/server/docker/server.Dockerfile`):
- Base: Ubuntu 24.04
- Packages: kmod, iproute2, zfsutils-linux, docker.io, postgresql, socat
- Runtime: OpenJDK 17
- Datadatdat components:
  - `/datadatdat/datadatdat-server.jar` - Server application
  - `/datadatdat/launch` - ZFS bootstrap script
  - `/datadatdat/run` - Server startup script
  - `/datadatdat/teardown` - Cleanup script
  - `/datadatdat/docker-volume-proxy` - Docker volume driver
  - `/datadatdat/zfs.sh` - ZFS management functions
  - `/datadatdat/datadatdat.sh` - Pool and server management
  - `/datadatdat/util.sh` - Utility functions

### Tag image

Creates local aliases: `datadatdat:latest` and `datadatdat:$VERSION`.

---

## Phase 4: Container Cleanup

Removes any existing containers before starting fresh:

```go
docker.Remove("datadatdat-docker-server", true)   // docker rm -f
docker.Remove("datadatdat-docker-launch", true)
```

---

## Phase 5: Launch Container Startup

**File:** `internal/app/clients/Docker.go` -> `LaunchDatadatdatServers()`

```bash
docker run \
    --privileged \                      # ZFS module loading + device nodes
    --pid=host \                        # nsenter into host namespaces
    --network=host \                    # Direct host networking
    -d \                                # Detached mode
    --restart always \                  # Auto-restart
    --name=datadatdat-docker-launch \
    -v /var/lib:/var/lib \              # Host filesystem access
    -v /run/docker:/run/docker \        # Docker socket
    -v /lib:/var/lib/datadatdat-docker/system \  # Host kernel modules
    -v datadatdat-docker-data:/var/lib/datadatdat-docker/data \  # Persistent storage
    -v /var/run/docker.sock:/var/run/docker.sock \
    -e DATADATDAT_PORT=5001 \
    -e DATADATDAT_IMAGE=datadatdat:latest \
    -e DATADATDAT_IDENTITY=datadatdat-docker \
    datadatdat:latest \
    /bin/bash /datadatdat/launch
```

Both containers use the **same image** but different entry points:
- Launch: `/bin/bash /datadatdat/launch`
- Server: `/datadatdat/run`

---

## Phase 6: ZFS Infrastructure Setup

**File:** `datadatdat-server/server/src/scripts/launch` (sources `zfs.sh`, `datadatdat.sh`, `util.sh`)

### Docker Desktop Special Case

```bash
if [[ $(uname -r) == *"linuxkit"* ]]; then
    docker run --rm --privileged datadatdat/docker-desktop-zfs-kernel:$tag
fi
```

Pulls a small (~50MB) image that loads ZFS modules into the Docker Desktop VM.

### ZFS Module Loading (4-step fallback chain)

**File:** `datadatdat-server/server/src/scripts/zfs.sh`

```bash
if ! check_running_zfs &&
   ! load_zfs "$SYSTEM_MODULES" system "$INSTALL_DIR" &&
   ! install_zfs_packages &&
   ! insmod_prebuilt_zfs "$KERNEL_RELEASE" "$INSTALL_DIR"; then
    log_error "Failed to load ZFS"
fi
```

| Step | Method | What it does |
|------|--------|-------------|
| 1 | `check_running_zfs` | Checks `lsmod` + `/sys/module/zfs/version` for compatible ZFS (>= 2.0.0) |
| 2 | `load_zfs` (system) | Tries host kernel modules via `modprobe -d /path zfs` |
| 3 | `install_zfs_packages` | Detects package manager (apt/dnf/pacman) and installs `zfsutils-linux` + `modprobe zfs` |
| 4 | `insmod_prebuilt_zfs` | Downloads prebuilt `spl.ko` + `zfs.ko` from S3 and loads via `insmod` |

After loading, verifies `/dev/zfs` exists (creates via `mknod` if needed).

**WSL2 note:** On WSL2 with stock Microsoft kernel (>= 6.6.36.3), step 3 installs userland tools but `modprobe` fails (no kernel headers for DKMS). Step 4 provides prebuilt modules compiled against the WSL2 kernel source. Pool creation uses loop devices as a workaround for WSL2's file vdev limitation.

### Shared Bind Mounts

**File:** `datadatdat-server/server/src/scripts/datadatdat.sh` -> `bind_mounts()`

```bash
nsenter -m -u -t 1 -n -i sh -c \
  "mkdir -p $mnt_dir && \
   mount --bind $mnt_dir $mnt_dir && \
   mount --make-shared $mnt_dir"
```

Makes ZFS mounts inside the container visible to the host via mount propagation.

### ZFS Pool Creation

**File:** `datadatdat-server/server/src/scripts/datadatdat.sh` -> `create_import_pool()`

**First install:**

```bash
# Sparse file backing store
truncate -s ${size}M /var/lib/datadatdat-docker/data/pool/data

# Create pool
zpool create \
    -m /var/lib/datadatdat-docker/mnt \
    -o cachefile=/var/lib/datadatdat-docker/data/pool/cachefile \
    datadatdat-docker \
    /var/lib/datadatdat-docker/data/pool/data

# Create filesystems
zfs create -o mountpoint=legacy -o compression=lz4 datadatdat-docker/data
zfs create -o mountpoint=legacy datadatdat-docker/db
```

**Subsequent installs:**

```bash
zpool import -f -c /cachefile datadatdat-docker
```

### Docker Network

```bash
docker network create datadatdat-docker
```

Isolated bridge network for inter-container communication.

---

## Phase 7: Server Container Startup

**File:** `datadatdat-server/server/src/scripts/datadatdat.sh` -> `launch_server()`

```bash
docker run -i --privileged \
    --name=datadatdat-docker-server \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /run/docker/plugins:/run/docker/plugins \
    -v /var/lib/datadatdat-docker/mnt:/var/lib/datadatdat-docker/mnt:rshared \
    -e DATADATDAT_CONTEXT=docker-zfs \
    -e DATADATDAT_IDENTITY=datadatdat-docker \
    -p 5001:5001 \
    --network datadatdat-docker \
    datadatdat:latest \
    /datadatdat/run
```

---

## Phase 8: Server Initialization

**File:** `datadatdat-server/server/src/scripts/run`

### Mount database filesystem

```bash
mount -i -t zfs datadatdat-docker/db /var/lib/datadatdat-docker/mnt/_db
```

### Initialize PostgreSQL

PostgreSQL version is auto-detected at runtime:

```bash
PG_VERSION_INSTALLED=$(ls /usr/lib/postgresql/ | sort -n | tail -1)
PG_BIN="/usr/lib/postgresql/${PG_VERSION_INSTALLED}/bin"
```

If a database exists from an incompatible version, the data directory is cleared and re-initialized.

```bash
su - postgres -c "$PG_BIN/pg_ctl -D $db_dir initdb"
su - postgres -c "$PG_BIN/pg_ctl -D $db_dir start"
su - postgres -c "$PG_BIN/createdb datadatdat"
```

### Start Docker Volume Plugin

```bash
/datadatdat/docker-volume-proxy /run/docker/plugins/datadatdat-docker.sock &
```

Intercepts `docker volume create` commands and maps them to ZFS filesystems:

```
User: docker run -v mydata:/data postgres
  -> Docker daemon: volume driver=datadatdat-docker
  -> Volume plugin: zfs create datadatdat-docker/data/mydata
  -> Container sees /data mounted on ZFS
```

### Start REST API

```bash
exec java \
    -Ddatadatdat.context=docker-zfs \
    -Ddatadatdat.contextConfig=pool=datadatdat-docker \
    -jar /datadatdat/datadatdat-server.jar
```

Ktor-based server binds to `0.0.0.0:5001`.

---

## Phase 9: Installation Completion

**File:** `internal/app/providers/local/Install.go`

The CLI polls `docker logs datadatdat-docker-launch` for structured messages:

```go
for _, line := range logs {
    if strings.Contains(line, "DATADATDAT START") {
        fmt.Println(line)
    }
    if strings.Contains(line, "DATADATDAT FINISHED") {
        break
    }
}
```

Installation is complete when "DATADATDAT FINISHED" appears in the launch logs.

---

## Final State

### Running Containers

| Container | Purpose | Entry Point |
|-----------|---------|-------------|
| `datadatdat-docker-launch` | Infrastructure bootstrap, ZFS management | `/bin/bash /datadatdat/launch` |
| `datadatdat-docker-server` | REST API, PostgreSQL, volume plugin | `/datadatdat/run` |

### ZFS Pool Structure

```
datadatdat-docker              # Root pool (sparse file backed)
+-- datadatdat-docker/data     # User data volumes (LZ4 compressed)
+-- datadatdat-docker/db       # PostgreSQL database
```

### Persistent Storage

**Volume:** `datadatdat-docker-data`

| Path | Contents |
|------|----------|
| `/pool/data` | ZFS pool backing file (sparse) |
| `/pool/cachefile` | ZFS pool import cache |
| `/modules/<kernel>/` | Compiled ZFS kernel modules |
| `/install/` | Installation metadata |

### Network

**`datadatdat-docker`** - Isolated bridge network connecting server and user containers.

### Configuration

**Location:** `~/.datadatdat/config` (YAML)

```yaml
contexts:
  docker:
    default: true
    host: localhost
    port: 5001
    type: docker
```

---

## Performance

| Scenario | Time |
|----------|------|
| First install (prebuilt ZFS available) | 3-7 minutes |
| First install (ZFS compilation needed) | 35-40 minutes |
| Subsequent installs (cached) | 10-20 seconds |

---

## Docker Images

| Image | When Pulled | Size | Purpose |
|-------|-------------|------|---------|
| `datadatdat/datadatdat:$VERSION` | Always | ~1.7GB | Main server + ZFS tools |
| `datadatdat/docker-desktop-zfs-kernel:$TAG` | Docker Desktop only | ~50MB | ZFS modules for linuxkit |

---

## Common Failures

| Symptom | Cause | Fix |
|---------|-------|-----|
| "Error checking docker version" | Docker not running | Start Docker Desktop or daemon |
| "Failed to load ZFS modules" | Incompatible kernel | Check kernel version, wait for prebuilt modules |
| "Failed to create storage pool" | Disk full or permission issue | Check `docker volume inspect datadatdat-docker-data` |
| "Failed to import existing storage pool" | Corrupted pool | `d3 uninstall -f` and reinstall |
| Database version mismatch | PostgreSQL upgrade | Auto-handled by run script (clears and re-inits) |

---

## Debugging

```bash
# Container status
docker ps --filter name=datadatdat

# Server logs
docker logs datadatdat-docker-server

# Launch logs
docker logs datadatdat-docker-launch

# ZFS pool health
docker exec datadatdat-docker-server zpool status datadatdat-docker

# ZFS filesystems
docker exec datadatdat-docker-server zfs list

# Test REST API
curl http://localhost:5001/api/v1/repositories
```

---

## Key Source Files

| File | Purpose |
|------|---------|
| `internal/app/commands/install.go` | CLI install command |
| `internal/app/providers/local/Install.go` | Install orchestration |
| `internal/app/clients/Docker.go` | Docker client (pull, run, logs) |
| `internal/app/providers/ProviderFactory.go` | Provider creation + config |
| `datadatdat-server/server/src/scripts/launch` | ZFS bootstrap |
| `datadatdat-server/server/src/scripts/zfs.sh` | ZFS module loading |
| `datadatdat-server/server/src/scripts/datadatdat.sh` | Pool + server management |
| `datadatdat-server/server/src/scripts/run` | Server initialization |
| `datadatdat-server/server/docker/server.Dockerfile` | Container image definition |
