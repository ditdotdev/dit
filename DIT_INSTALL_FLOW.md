# Dit Install Flow

## Overview

When you run `dit install`, the CLI sets up a complete data versioning infrastructure backed by ZFS. The end result is two Docker containers and a ZFS storage pool ready to accept API requests.

```
User runs: dit install
         |
         v
   +---------------------+
   | Validate Docker      | <-- docker -v
   +----------+----------+
              v
   +---------------------+
   | Pull Docker image    | <-- ditdotdev/dit:$VERSION (~1GB)
   +----------+----------+
              v
   +---------------------+
   | Start launch         | <-- privileged, --pid=host
   | container            |     dit-docker-launch
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
   | Create/Import ZFS    | <-- dit-docker pool
   | pool + filesystems   |     data (LZ4), db
   +----------+----------+
              v
   +---------------------+
   | Start server         | <-- dit-docker-server
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
    Short: "Install dit infrastructure",
    Run: func(cmd *cobra.Command, args []string) {
        registry, _ := cmd.Flags().GetString("registry")
        provider = providers.Create(context, contextType, providers.GetAvailablePort())
        provider.Install([]string{"registry=" + registry}, verbose)
        providers.AddProvider(provider)
    },
}
```

- Parses `--registry` flag (default: `ditdotdev` on Docker Hub)
- Creates a Docker provider with an available port (default: 5001)
- Calls `provider.Install()` which drives the rest of the process
- Saves provider config to `~/.dit/config` (YAML)

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
out, _ := ce.Exec("docker", "images", registry+"/dit", "--format", `"{{.Tag}}"`)
```

Compares local tags against the required version using semantic versioning. Skips download if a matching version exists.

### Pull image

```go
pullImage := pullRegistry + "/dit:" + latest
docker.Pull(pullImage)
```

Downloads `ditdotdev/dit:$VERSION` from Docker Hub (~1GB).

**Image contents** (see `dit-server/server/docker/server.Dockerfile`):
- Base: Ubuntu 24.04
- Packages: kmod, iproute2, zfsutils-linux, docker.io, postgresql, socat
- Runtime: OpenJDK 17
- Dit components:
  - `/ditdotdev/dit-server.jar` - Server application
  - `/ditdotdev/launch` - ZFS bootstrap script
  - `/ditdotdev/run` - Server startup script
  - `/ditdotdev/teardown` - Cleanup script
  - `/ditdotdev/docker-volume-proxy` - Docker volume driver
  - `/ditdotdev/zfs.sh` - ZFS management functions
  - `/ditdotdev/dit.sh` - Pool and server management
  - `/ditdotdev/util.sh` - Utility functions

### Tag image

Creates local aliases: `dit:latest` and `dit:$VERSION`.

---

## Phase 4: Container Cleanup

Removes any existing containers before starting fresh:

```go
docker.Remove("dit-docker-server", true)   // docker rm -f
docker.Remove("dit-docker-launch", true)
```

---

## Phase 5: Launch Container Startup

**File:** `internal/app/clients/Docker.go` -> `LaunchDitServers()`

```bash
docker run \
    --privileged \                      # ZFS module loading + device nodes
    --pid=host \                        # nsenter into host namespaces
    --network=host \                    # Direct host networking
    -d \                                # Detached mode
    --restart always \                  # Auto-restart
    --name=dit-docker-launch \
    -v /var/lib:/var/lib \              # Host filesystem access
    -v /run/docker:/run/docker \        # Docker socket
    -v /lib:/var/lib/dit-docker/system \  # Host kernel modules
    -v dit-docker-data:/var/lib/dit-docker/data \  # Persistent storage
    -v /var/run/docker.sock:/var/run/docker.sock \
    -e DIT_PORT=5001 \
    -e DIT_IMAGE=dit:latest \
    -e DIT_IDENTITY=dit-docker \
    dit:latest \
    /bin/bash /ditdotdev/launch
```

Both containers use the **same image** but different entry points:
- Launch: `/bin/bash /ditdotdev/launch`
- Server: `/ditdotdev/run`

---

## Phase 6: ZFS Infrastructure Setup

**File:** `dit-server/server/src/scripts/launch` (sources `zfs.sh`, `dit.sh`, `util.sh`)

### ZFS Module Loading (4-step fallback chain)

**File:** `dit-server/server/src/scripts/zfs.sh`

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

**File:** `dit-server/server/src/scripts/dit.sh` -> `bind_mounts()`

```bash
nsenter -m -u -t 1 -n -i sh -c \
  "mkdir -p $mnt_dir && \
   mount --bind $mnt_dir $mnt_dir && \
   mount --make-shared $mnt_dir"
```

Makes ZFS mounts inside the container visible to the host via mount propagation.

### ZFS Pool Creation

**File:** `dit-server/server/src/scripts/dit.sh` -> `create_import_pool()`

**First install:**

```bash
# Sparse file backing store
truncate -s ${size}M /var/lib/dit-docker/data/pool/data

# Create pool
zpool create \
    -m /var/lib/dit-docker/mnt \
    -o cachefile=/var/lib/dit-docker/data/pool/cachefile \
    dit-docker \
    /var/lib/dit-docker/data/pool/data

# Create filesystems
zfs create -o mountpoint=legacy -o compression=lz4 dit-docker/data
zfs create -o mountpoint=legacy dit-docker/db
```

**Subsequent installs:**

```bash
zpool import -f -c /cachefile dit-docker
```

### Docker Network

```bash
docker network create dit-docker
```

Isolated bridge network for inter-container communication.

---

## Phase 7: Server Container Startup

**File:** `dit-server/server/src/scripts/dit.sh` -> `launch_server()`

```bash
docker run -i --privileged \
    --name=dit-docker-server \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /run/docker/plugins:/run/docker/plugins \
    -v /var/lib/dit-docker/mnt:/var/lib/dit-docker/mnt:rshared \
    -e DIT_CONTEXT=docker-zfs \
    -e DIT_IDENTITY=dit-docker \
    -p 5001:5001 \
    --network dit-docker \
    dit:latest \
    /ditdotdev/run
```

---

## Phase 8: Server Initialization

**File:** `dit-server/server/src/scripts/run`

### Mount database filesystem

```bash
mount -i -t zfs dit-docker/db /var/lib/dit-docker/mnt/_db
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
su - postgres -c "$PG_BIN/createdb dit"
```

### Start Docker Volume Plugin

```bash
/ditdotdev/docker-volume-proxy /run/docker/plugins/dit-docker.sock &
```

Intercepts `docker volume create` commands and maps them to ZFS filesystems:

```
User: docker run -v mydata:/data postgres
  -> Docker daemon: volume driver=dit-docker
  -> Volume plugin: zfs create dit-docker/data/mydata
  -> Container sees /data mounted on ZFS
```

### Start REST API

```bash
exec java \
    -Ddit.context=docker-zfs \
    -Ddit.contextConfig=pool=dit-docker \
    -jar /ditdotdev/dit-server.jar
```

Ktor-based server binds to `0.0.0.0:5001`.

---

## Phase 9: Installation Completion

**File:** `internal/app/providers/local/Install.go`

The CLI polls `docker logs dit-docker-launch` for structured messages:

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
| `dit-docker-launch` | Infrastructure bootstrap, ZFS management | `/bin/bash /ditdotdev/launch` |
| `dit-docker-server` | REST API, PostgreSQL, volume plugin | `/ditdotdev/run` |

### ZFS Pool Structure

```
dit-docker              # Root pool (sparse file backed)
+-- dit-docker/data     # User data volumes (LZ4 compressed)
+-- dit-docker/db       # PostgreSQL database
```

### Persistent Storage

**Volume:** `dit-docker-data`

| Path | Contents |
|------|----------|
| `/pool/data` | ZFS pool backing file (sparse) |
| `/pool/cachefile` | ZFS pool import cache |
| `/modules/<kernel>/` | Compiled ZFS kernel modules |
| `/install/` | Installation metadata |

### Network

**`dit-docker`** - Isolated bridge network connecting server and user containers.

### Configuration

**Location:** `~/.dit/config` (YAML)

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
| `ditdotdev/dit:$VERSION` | Always | ~1.7GB | Main server + ZFS tools |

---

## Common Failures

| Symptom | Cause | Fix |
|---------|-------|-----|
| "Error checking docker version" | Docker not running | Start Docker Desktop or daemon |
| "Failed to load ZFS modules" | Incompatible kernel | Check kernel version, wait for prebuilt modules |
| "Failed to create storage pool" | Disk full or permission issue | Check `docker volume inspect dit-docker-data` |
| "Failed to import existing storage pool" | Corrupted pool | `dit uninstall -f` and reinstall |
| Database version mismatch | PostgreSQL upgrade | Auto-handled by run script (clears and re-inits) |

---

## Debugging

```bash
# Container status
docker ps --filter name=dit

# Server logs
docker logs dit-docker-server

# Launch logs
docker logs dit-docker-launch

# ZFS pool health
docker exec dit-docker-server zpool status dit-docker

# ZFS filesystems
docker exec dit-docker-server zfs list

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
| `dit-server/server/src/scripts/launch` | ZFS bootstrap |
| `dit-server/server/src/scripts/zfs.sh` | ZFS module loading |
| `dit-server/server/src/scripts/dit.sh` | Pool + server management |
| `dit-server/server/src/scripts/run` | Server initialization |
| `dit-server/server/docker/server.Dockerfile` | Container image definition |
