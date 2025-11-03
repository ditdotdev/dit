# Deep Dive: Understanding `titan install` (d3 install)

## Executive Summary

When you run `titan install` (now `d3 install`), the software performs a sophisticated multi-step process to set up a complete data versioning infrastructure. The goal is to have **two Docker containers running**:

1. **`datadatdat-docker-launch`** - Bootstrap container that sets up the ZFS infrastructure
2. **`datadatdat-docker-server`** - Main server container running the Datadatdat REST API and database

This document provides a comprehensive, low-level walkthrough of every step the software takes during installation.

---

## High-Level Overview

```
User runs: d3 install
    ↓
CLI checks Docker is running
    ↓
CLI pulls Docker image: datadatdat/datadatdat:latest
    ↓
CLI launches datadatdat-docker-launch container
    ↓
Launch container sets up ZFS infrastructure
    ↓
Launch container starts datadatdat-docker-server
    ↓
Server container runs REST API + PostgreSQL
    ↓
Installation complete ✓
```

---

## Detailed Step-by-Step Process

### Phase 1: CLI Command Execution

**Location:** `c:\dev\datadatdat\internal\app\commands\install.go`

#### Step 1.1: Command Parsing
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

**What happens:**
- Parses command-line flags (registry, verbose)
- Default registry: `datadatdat` (Docker Hub)
- Default context type: `docker`
- Default context name: `docker`
- Default port: `5001`

#### Step 1.2: Provider Creation
**Location:** `c:\dev\datadatdat\internal\app\providers\ProviderFactory.go`

Creates a Docker provider instance that will handle the installation process.

---

### Phase 2: Docker Environment Validation

**Location:** `c:\dev\datadatdat\internal\app\providers\local\Install.go`

#### Step 2.1: Docker Version Check
```go
if _, err := docker.Version(); err != nil {
    fmt.Printf("Error checking docker version: %v\n", err)
    os.Exit(1)
}
```

**What happens:**
- Runs: `docker -v`
- Verifies Docker daemon is running and accessible
- **Failure:** Exits with error if Docker is not running

**Success criteria:** Docker responds with version information

---

### Phase 3: Docker Image Management

#### Step 3.1: Check if Image Already Downloaded
**Location:** `c:\dev\datadatdat\internal\app\clients\Docker.go` → `DatadatdatLatestIsDownloaded()`

```go
func (d docker) DatadatdatLatestIsDownloaded(registry string, latest app.Version) bool {
    if registry == "local" {
        localOut, _ := ce.Exec("docker", "images", "datadatdat", "--format", `"{{.Repository}}:{{.Tag}}"`)
        if strings.Contains(localOut, "datadatdat:latest") {
            return true
        }
        registry = "datadatdat"
    }
    
    out, _ := ce.Exec("docker", "images", registry+"/datadatdat", "--format", `"{{.Tag}}"`)
    tags := strings.Split(string(out), EOL)
    for _, item := range tags {
        tag := strings.Trim(item, "\"")
        if tag != "latest" && tag != "" {
            v := app.Version{}.FromString(tag)
            if v.Compare(latest) == 0 {
                return true
            }
        }
    }
    return false
}
```

**What happens:**
- Runs: `docker images datadatdat/datadatdat --format "{{.Tag}}"`
- Checks if the required version is already downloaded
- If found: skips download
- If not found: proceeds to pull

#### Step 3.2: Pull Docker Image
**Location:** `c:\dev\datadatdat\internal\app\providers\local\Install.go`

```go
pullImage := pullRegistry + "/datadatdat:" + latest
if _, err := docker.Pull(pullImage); err != nil {
    fmt.Printf("Error pulling image %s: %v\n", pullImage, err)
    os.Exit(1)
}
```

**What happens:**
- Runs: `docker pull datadatdat/datadatdat:latest`
- Downloads the main Datadatdat server image from Docker Hub
- **This is the primary container image** that contains:
  - Ubuntu 20.04 base
  - ZFS userland tools (zfsutils-linux)
  - PostgreSQL 12
  - Java 11 runtime
  - Datadatdat server JAR
  - Datadatdat scripts (launch, run, teardown, etc.)
  - Docker volume proxy binary

**Image size:** Typically 800MB-1.2GB

**What's in the image:**
See: `c:\dev\datadatdat-server\server\docker\server.Dockerfile`
- Operating system: Ubuntu Focal (20.04)
- Key packages: kmod, iproute2, zfsutils-linux, docker.io, postgresql
- Datadatdat components:
  - `/datadatdat/datadatdat-server.jar` - Main server application
  - `/datadatdat/launch` - ZFS setup script
  - `/datadatdat/run` - Server startup script
  - `/datadatdat/teardown` - Cleanup script
  - `/datadatdat/docker-volume-proxy` - Docker volume driver
  - `/datadatdat/zfs.sh` - ZFS management functions
  - `/datadatdat/datadatdat.sh` - Pool and server management
  - `/datadatdat/util.sh` - Utility functions

#### Step 3.3: Tag Image
```go
tagLatest := "datadatdat:" + latest
docker.Tag(pullImage, tagLatest)
docker.Tag(pullImage, "datadatdat")
```

**What happens:**
- Runs: `docker tag datadatdat/datadatdat:latest datadatdat:latest`
- Runs: `docker tag datadatdat/datadatdat:latest datadatdat`
- Creates local aliases for the pulled image

---

### Phase 4: Container Cleanup

#### Step 4.1: Remove Old Server Container
```go
serverAvailable, _ := docker.DatadatdatServerIsAvailable()
if serverAvailable {
    docker.Remove("datadatdat-docker-server", true)
}
```

**What happens:**
- Checks: `docker ps -f name=^/datadatdat-docker-server$ --format "{{.Names}}"`
- If found: `docker rm -f datadatdat-docker-server`
- Ensures clean slate for new installation

#### Step 4.2: Remove Old Launch Container
```go
launchAvailable, _ := docker.DatadatdatLaunchIsAvailable()
if launchAvailable {
    docker.Remove("datadatdat-docker-launch", true)
}
```

**What happens:**
- Checks: `docker ps -f name=^/datadatdat-docker-launch$ --format "{{.Names}}"`
- If found: `docker rm -f datadatdat-docker-launch`

---

### Phase 5: Launch Container Startup

**Location:** `c:\dev\datadatdat\internal\app\clients\Docker.go` → `LaunchDatadatdatServers()`

#### Step 5.1: Construct Launch Arguments
```go
func (d docker) LaunchDatadatdatServers() (string, error) {
    datadatdatImage := d.getImageName("datadatdat:latest")
    args := d.getLocalLaunchArgs()
    args = append(
        args,
        "-e", "DATADATDAT_PORT="+strconv.Itoa(d.port),
        "-e", "DATADATDAT_IMAGE="+datadatdatImage,
        "-e", "DATADATDAT_IDENTITY=datadatdat-"+d.identity,
    )
    return d.Run(datadatdatImage, "/bin/bash /datadatdat/launch", args)
}
```

**Launch container arguments:**
```bash
docker run \
    --privileged \                      # Required for ZFS module loading
    --pid=host \                        # Share host PID namespace
    --network=host \                    # Use host networking
    -d \                                # Run in detached mode
    --restart always \                  # Auto-restart on failure
    --name=datadatdat-docker-launch \   # Container name
    -v /var/lib:/var/lib \             # Access to host filesystem
    -v /run/docker:/run/docker \       # Docker socket access
    -v /lib:/var/lib/datadatdat-docker/system \  # Host kernel modules
    -v datadatdat-docker-data:/var/lib/datadatdat-docker/data \  # Persistent storage
    -v /var/run/docker.sock:/var/run/docker.sock \  # Docker API access
    -e DATADATDAT_PORT=5001 \
    -e DATADATDAT_IMAGE=datadatdat:latest \
    -e DATADATDAT_IDENTITY=datadatdat-docker \
    datadatdat:latest \
    /bin/bash /datadatdat/launch
```

**Why these permissions?**
- `--privileged`: Required to load ZFS kernel modules and create device nodes
- `--pid=host`: Needed to use `nsenter` to manipulate host namespaces
- `-v /var/run/docker.sock`: Launch container needs to start the server container
- `-v /var/lib:/var/lib`: Mount propagation for ZFS filesystems

#### Step 5.2: Container Starts Launch Script

The container immediately executes: `/bin/bash /datadatdat/launch`

---

### Phase 6: ZFS Infrastructure Setup (Inside Launch Container)

**Location:** `c:\dev\datadatdat-server\server\src\scripts\launch`

This is the **most complex phase** where ZFS is configured.

#### Step 6.1: Initialize Configuration
**Location:** `c:\dev\datadatdat-server\server\src\scripts\util.sh`

```bash
IDENTITY=${DATADATDAT_IDENTITY:-datadatdat}
PORT=${DATADATDAT_PORT:-5001}
IMAGE=${DATADATDAT_IMAGE:-datadatdat:latest}

POOL=$IDENTITY                          # "datadatdat-docker"
VOLUME=$IDENTITY-data                   # "datadatdat-docker-data"
BASE_DIR=/var/lib/$IDENTITY             # "/var/lib/datadatdat-docker"
POOL_DIR=$(docker volume inspect $VOLUME | jq -r .[0].Mountpoint)/pool
MNT_DIR=$BASE_DIR/mnt                   # "/var/lib/datadatdat-docker/mnt"
SYSTEM_MODULES=$BASE_DIR/system         # "/var/lib/datadatdat-docker/system"
COMPILED_MODULES=$DATA_DIR/modules      # Location for compiled ZFS modules
KERNEL_RELEASE=$(uname -r)              # Current kernel version
```

#### Step 6.2: Docker Desktop Special Case
```bash
uname=$(uname -r)
if [[ $uname == *"linuxkit"* ]]; then
    echo "Installing ZFS for Docker Desktop ($uname)"
    tag="${uname%%-*}"
    docker run --rm --privileged datadatdat/docker-desktop-zfs-kernel:$tag
fi
```

**What happens:**
- Detects if running on Docker Desktop (linuxkit kernel)
- Pulls ZFS kernel module loader for Docker Desktop
- Runs: `docker run --rm --privileged datadatdat/docker-desktop-zfs-kernel:4.9.184`
- Loads ZFS modules into Docker Desktop VM

#### Step 6.3: Create Directories
```bash
mkdir -p $POOL_DIR || log_error "failed to create $POOL_DIR"
mkdir -p $INSTALL_DIR || log_error "failed to create $INSTALL_DIR"
```

Creates:
- `/var/lib/datadatdat-docker/data/pool/` - ZFS pool storage
- `/var/lib/datadatdat-docker/data/install/` - ZFS module storage

#### Step 6.4: Load ZFS Modules (The Critical Step)

**Location:** `c:\dev\datadatdat-server\server\src\scripts\zfs.sh`

This follows a sophisticated fallback chain:

```bash
if ! check_running_zfs &&
   ! load_zfs $SYSTEM_MODULES system $INSTALL_DIR &&
   ! load_zfs $COMPILED_MODULES/$KERNEL_RELEASE compiled $INSTALL_DIR &&
   ! load_precompiled_zfs $COMPILED_MODULES/$KERNEL_RELEASE $INSTALL_DIR &&
   ! compile_and_load_zfs $COMPILED_MODULES/$KERNEL_RELEASE $INSTALL_DIR; then
    log_error "Failed to load ZFS modules"
fi
```

**ZFS Loading Strategy (in order):**

1. **check_running_zfs**: Check if ZFS is already loaded in the kernel
   - Runs: `lsmod | grep "^zfs "`
   - Checks: `/sys/module/zfs/version`
   - If compatible version found: SUCCESS, skip to next phase

2. **load_zfs (system modules)**: Try to use host's ZFS modules
   - Location: `/var/lib/datadatdat-docker/system/lib/modules/<kernel>/`
   - Mounted from host `/lib` directory
   - Runs: `modprobe -d /path zfs`
   - If compatible: SUCCESS

3. **load_zfs (compiled modules)**: Try previously compiled modules
   - Location: `/var/lib/datadatdat-docker/data/modules/<kernel>/`
   - Modules compiled on previous run
   - If found and compatible: SUCCESS

4. **load_precompiled_zfs**: Download prebuilt ZFS modules
   - Downloads from: `https://download.datadatdat.com/zfs-releases/zfs-<kernel>-<version>.tar.gz`
   - Extracts to: `/var/lib/datadatdat-docker/data/modules/<kernel>/`
   - If available: SUCCESS

5. **compile_and_load_zfs**: Build ZFS from source (last resort)
   - Runs: `docker run --rm -v /build -e ZFS_VERSION=zfs-2.1.5 -e ZFS_CONFIG=kernel datadatdat/zfs-builder:latest`
   - **This takes 30 minutes** on first run
   - Compiles ZFS kernel modules for current kernel
   - Stores in: `/var/lib/datadatdat-docker/data/modules/<kernel>/`

**After ZFS is loaded:**
- Verifies: `/dev/zfs` device node exists
- Creates if missing: `mknod -m 660 /dev/zfs c <major> <minor>`
- Sanity tests: `zpool list`, `zfs list`

#### Step 6.5: Create Shared Mounts
**Location:** `c:\dev\datadatdat-server\server\src\scripts\datadatdat.sh` → `bind_mounts()`

```bash
nsenter -m -u -t 1 -n -i sh -c \
  "if [ $(mount |grep $mnt_dir | wc -l) -eq 0 ]; then
      mkdir -p $mnt_dir && \
      mount --bind $mnt_dir $mnt_dir && \
      mount --make-shared $mnt_dir;
  fi"
```

**What happens:**
- Enters host mount namespace using `nsenter`
- Creates: `/var/lib/datadatdat-docker/mnt`
- Makes it a bind mount to itself
- Makes it shared: allows mount propagation to host
- **Why?** So ZFS mounts inside container appear on host

#### Step 6.6: Create or Import ZFS Pool
**Location:** `c:\dev\datadatdat-server\server\src\scripts\datadatdat.sh` → `create_import_pool()`

**First Installation (No existing pool):**
```bash
# Calculate pool size (available space - 32MB)
size=$(df -BM --output=avail /pool/dir | tail -1)
size=$(( ${size%%M} - 32 ))

# Create sparse file for pool
truncate -s ${size}M /var/lib/datadatdat-docker/data/pool/data

# Create ZFS pool
zpool create \
    -m /var/lib/datadatdat-docker/mnt \
    -o cachefile=/var/lib/datadatdat-docker/data/pool/cachefile \
    datadatdat-docker \
    /var/lib/datadatdat-docker/data/pool/data

# Create filesystems
zfs create -o mountpoint=legacy -o compression=lz4 datadatdat-docker/data
zfs create -o mountpoint=legacy datadatdat-docker/db
```

**What gets created:**
- **ZFS Pool:** `datadatdat-docker`
  - Backed by sparse file (typically 1GB)
  - Custom cachefile (not auto-imported on boot)
  - Mountpoint: `/var/lib/datadatdat-docker/mnt`
  
- **ZFS Filesystems:**
  - `datadatdat-docker/data` - User data volumes (with LZ4 compression)
  - `datadatdat-docker/db` - PostgreSQL database

**Subsequent Installations (Existing pool):**
```bash
# Import existing pool
zpool import -f -c /cachefile datadatdat-docker

# Update pool structure (migrate from old versions)
update_pool datadatdat-docker
```

#### Step 6.7: Create Docker Network
```bash
docker network inspect datadatdat-docker || \
    docker network create datadatdat-docker
```

**What happens:**
- Creates isolated bridge network: `datadatdat-docker`
- Used for inter-container communication
- Allows server to talk to test containers

---

### Phase 7: Server Container Startup (Inside Launch Container)

**Location:** `c:\dev\datadatdat-server\server\src\scripts\datadatdat.sh` → `launch_server()`

#### Step 7.1: Cleanup Previous Server
```bash
rm -f /run/docker/plugins/datadatdat-docker.sock
docker rm -f datadatdat-docker-server >/dev/null 2>&1
```

#### Step 7.2: Start Server Container
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

**Server container arguments explained:**
- `--name=datadatdat-docker-server` - Container name
- `-v /var/run/docker.sock` - Docker API access (to manage user containers)
- `-v /run/docker/plugins` - Docker volume plugin socket
- `-v /var/lib/datadatdat-docker/mnt:rshared` - ZFS mount propagation
- `-e DATADATDAT_CONTEXT=docker-zfs` - Operating mode
- `-e DATADATDAT_IDENTITY=datadatdat-docker` - Identity/pool name
- `-p 5001:5001` - Expose REST API
- `--network datadatdat-docker` - Connect to bridge network
- `/datadatdat/run` - Execute run script

---

### Phase 8: Server Initialization (Inside Server Container)

**Location:** `c:\dev\datadatdat-server\server\src\scripts\run`

#### Step 8.1: Mount Database Filesystem
```bash
mount | grep ^datadatdat-docker/db > /dev/null
if [[ $? != 0 ]]; then
    mkdir -p /var/lib/datadatdat-docker/mnt/_db
    mount -i -t zfs datadatdat-docker/db /var/lib/datadatdat-docker/mnt/_db
fi
```

**What happens:**
- Mounts ZFS filesystem: `datadatdat-docker/db`
- Mountpoint: `/var/lib/datadatdat-docker/mnt/_db`
- This is where PostgreSQL stores its data

#### Step 8.2: Fix Database Permissions
```bash
chown -R postgres:postgres /var/lib/datadatdat-docker/mnt/_db

# Handle version incompatibilities
if [[ -f $db_dir/PG_VERSION ]]; then
    local db_version=$(cat $db_dir/PG_VERSION)
    if [[ "$db_version" != "12" ]]; then
        echo "WARNING: Database version $db_version incompatible with server version 12"
        rm -rf $db_dir/*
        chown postgres:postgres $db_dir
    fi
fi
```

#### Step 8.3: Initialize PostgreSQL Database
```bash
if [[ ! -f $db_dir/postgresql.conf ]]; then
    echo "Initializing database"
    su - postgres -c "/usr/lib/postgresql/12/bin/pg_ctl -D $db_dir initdb"
    create_db=true
fi
```

**What happens:**
- Checks if database already initialized
- If not: runs `pg_ctl initdb`
- Creates PostgreSQL cluster in: `/var/lib/datadatdat-docker/mnt/_db`

#### Step 8.4: Start PostgreSQL
```bash
rm -f $db_dir/postmaster.pid
su - postgres -c "/usr/lib/postgresql/12/bin/pg_ctl -D $db_dir -l /var/log/postgresql/logfile start"
[[ $create_db == "true" ]] && su - postgres -c "/usr/lib/postgresql/12/bin/createdb datadatdat"
```

**What happens:**
- Removes stale PID file
- Starts PostgreSQL server
- If new database: creates `datadatdat` database
- PostgreSQL listens on localhost:5432 (inside container)

#### Step 8.5: Start Docker Volume Plugin
```bash
/datadatdat/docker-volume-proxy /run/docker/plugins/datadatdat-docker.sock > /var/log/docker-proxy.log 2>&1 &
```

**What happens:**
- Starts Docker volume plugin in background
- Creates socket: `/run/docker/plugins/datadatdat-docker.sock`
- Docker daemon communicates with this socket
- Handles `docker volume create` commands
- Maps volumes to ZFS filesystems

#### Step 8.6: Start Datadatdat Server (Java Application)
```bash
exec java \
    -Ddatadatdat.context=docker-zfs \
    -Ddatadatdat.contextConfig=pool=datadatdat-docker \
    -jar /datadatdat/datadatdat-server.jar
```

**What happens:**
- Launches Ktor-based REST API server
- Connects to PostgreSQL database
- Binds to: `0.0.0.0:5001` (inside container)
- Mapped to: `localhost:5001` (on host)
- Server is now ready to accept API requests

**Server components:**
- REST API endpoints (commit, push, pull, run, etc.)
- Docker volume driver integration
- ZFS snapshot/clone management
- Repository metadata tracking
- PostgreSQL for persistence

---

### Phase 9: Installation Completion

#### Step 9.1: CLI Monitors Launch Logs
**Location:** `c:\dev\datadatdat\internal\app\providers\local\Install.go`

```go
logs := docker.FetchLaunchLogs()
for _, line := range logs {
    if strings.Contains(line, "DATADATDAT START") {
        fmt.Println(strings.Replace(line, "DATADATDAT START", "", 1))
    }
    if strings.Contains(line, "DATADATDAT FINISHED") {
        break
    }
}
```

**What happens:**
- CLI polls: `docker logs datadatdat-docker-launch`
- Parses structured log messages
- Shows progress to user:
  - "Checking if compatible ZFS is running"
  - "Creating storage pool"
  - "Creating shared mounts"
- Waits for: "DATADATDAT FINISHED"

#### Step 9.2: Success Message
```
Datadatdat CLI successfully installed, happy data versioning :)
```

---

## Final State: What's Running?

After successful installation, you have:

### Container 1: datadatdat-docker-launch
```bash
docker ps --filter name=datadatdat-docker-launch
```

**Status:** Running (unless it crashed)
**Purpose:** Infrastructure bootstrap
**Key responsibilities:**
- Loaded ZFS kernel modules
- Created ZFS pool
- Started server container
- Now idle (monitoring)

**Mounts:**
- `/var/lib` - Host filesystem access
- `/run/docker` - Docker socket
- `/var/run/docker.sock` - Docker API
- `datadatdat-docker-data` volume - Persistent storage

### Container 2: datadatdat-docker-server
```bash
docker ps --filter name=datadatdat-docker-server
```

**Status:** Running
**Purpose:** Main application server
**Port:** `5001:5001` (host:container)
**Key responsibilities:**
- REST API server
- Docker volume plugin
- ZFS snapshot management
- PostgreSQL database
- Repository metadata

**Mounts:**
- `/var/run/docker.sock` - Docker API (to manage user containers)
- `/run/docker/plugins` - Volume plugin socket
- `/var/lib/datadatdat-docker/mnt:rshared` - ZFS mounts with propagation

### Volume: datadatdat-docker-data
```bash
docker volume inspect datadatdat-docker-data
```

**Purpose:** Persistent storage
**Contents:**
- `/pool/data` - ZFS pool backing file (sparse, ~1GB)
- `/pool/cachefile` - ZFS pool cache
- `/modules/` - Compiled ZFS kernel modules
- `/install/` - ZFS installation metadata

### ZFS Pool: datadatdat-docker
```bash
docker exec datadatdat-docker-server zpool list
docker exec datadatdat-docker-server zfs list
```

**Structure:**
```
datadatdat-docker              # Root pool
├── datadatdat-docker/data     # User data volumes (compressed)
└── datadatdat-docker/db       # PostgreSQL database
```

### Docker Network: datadatdat-docker
```bash
docker network inspect datadatdat-docker
```

**Purpose:** Isolated bridge network
**Connected containers:**
- datadatdat-docker-server
- (Any test/SSH containers spawned by Datadatdat)

### PostgreSQL Database
**Location:** `/var/lib/datadatdat-docker/mnt/_db` (on ZFS)
**Database:** `datadatdat`
**Version:** PostgreSQL 12
**Schemas:** Repository metadata, commits, volumes, etc.

---

## Success Criteria: What Does d3 Consider Success?

The installation is considered successful when:

1. ✅ **Docker is running** - `docker -v` succeeds
2. ✅ **Image is pulled** - `datadatdat/datadatdat:latest` exists locally
3. ✅ **Launch container starts** - `datadatdat-docker-launch` running
4. ✅ **ZFS loads successfully** - One of the 5 loading strategies succeeds
5. ✅ **ZFS pool created/imported** - `datadatdat-docker` pool exists
6. ✅ **Server container starts** - `datadatdat-docker-server` running
7. ✅ **PostgreSQL initializes** - Database cluster created
8. ✅ **PostgreSQL starts** - `pg_ctl start` succeeds
9. ✅ **Volume plugin starts** - Socket created at `/run/docker/plugins/datadatdat-docker.sock`
10. ✅ **REST API starts** - Java application binds to port 5001
11. ✅ **Launch script completes** - "DATADATDAT FINISHED" logged

The CLI considers installation complete when it sees the "DATADATDAT FINISHED" message in the launch container logs.

---

## Which Docker Containers Are Pulled?

### Primary Image: datadatdat/datadatdat:latest

**Source:** Docker Hub (`docker pull datadatdat/datadatdat:latest`)

**Size:** ~800MB-1.2GB

**Used for:**
- `datadatdat-docker-launch` container
- `datadatdat-docker-server` container

**Note:** Both containers use the **same image** but run different entry points:
- Launch: `/bin/bash /datadatdat/launch`
- Server: `/datadatdat/run`

### Conditional Images

**datadatdat/docker-desktop-zfs-kernel:X.X.XXX**
- **When:** Only on Docker Desktop (linuxkit kernel)
- **Purpose:** Load ZFS modules into Docker Desktop VM
- **Size:** ~50MB
- **Source:** Docker Hub

**datadatdat/zfs-builder:latest**
- **When:** If ZFS compilation is needed (rare)
- **Purpose:** Build ZFS kernel modules from source
- **Size:** ~500MB
- **Source:** Docker Hub
- **Note:** Only pulled if no prebuilt modules available

---

## Common Failure Points

### 1. Docker Not Running
**Error:** "Error checking docker version"
**Solution:** Start Docker Desktop or Docker daemon

### 2. ZFS Load Failure
**Error:** "Failed to load ZFS modules"
**Causes:**
- Incompatible kernel version
- Missing kernel headers
- Compilation failure
**Solution:** Check kernel version, may need to wait for prebuilt modules

### 3. Pool Creation Failure
**Error:** "Failed to create storage pool"
**Causes:**
- Insufficient disk space
- Permission issues
- Existing pool conflicts
**Solution:** Check `docker volume inspect datadatdat-docker-data`

### 4. Server Container Fails to Start
**Error:** "Failed to import existing storage pool"
**Causes:**
- Corrupted ZFS pool
- Version mismatch
**Solution:** `d3 uninstall -f` and reinstall

### 5. PostgreSQL Initialization Failure
**Error:** Database version incompatibility
**Solution:** Clear database directory and reinitialize

---

## Performance Notes

### First Installation
- **Image download:** 2-5 minutes (depends on network)
- **ZFS prebuilt download:** 30 seconds - 2 minutes
- **ZFS compilation (if needed):** **30 minutes**
- **Pool creation:** < 1 second
- **Database initialization:** 5-10 seconds
- **Total (best case):** 3-7 minutes
- **Total (worst case - compilation):** 35-40 minutes

### Subsequent Installations
- **Image check:** < 1 second (already cached)
- **ZFS check:** < 1 second (already loaded)
- **Pool import:** < 1 second
- **Server startup:** 5-10 seconds
- **Total:** 10-20 seconds

---

## Technical Architecture Notes

### Why Two Containers?

**Launch Container:**
- Needs `--privileged` and `--pid=host` to load kernel modules
- Handles one-time infrastructure setup
- Manipulates host filesystem and namespaces
- Not suitable for long-running application

**Server Container:**
- Runs the actual application code
- Cleaner separation of concerns
- Can be restarted without re-running infrastructure setup
- Still needs `--privileged` for ZFS mount operations

### ZFS Architecture

**Why ZFS?**
- Copy-on-write snapshots (instant backups)
- Zero-copy clones (instant rollbacks)
- Compression (LZ4 - ~2x space savings)
- Data integrity guarantees

**How Data Versioning Works:**
1. User commits data: Creates ZFS snapshot
2. User rolls back: Clones from snapshot
3. User pushes: Snapshots sent to remote
4. User pulls: Snapshots received and cloned

### Docker Volume Plugin

**Purpose:** Intercept `docker volume create` commands

**Flow:**
```
User: docker run -v mydata:/data postgres
    ↓
Docker Daemon: volume driver=datadatdat-docker
    ↓
Volume Plugin: Create ZFS filesystem datadatdat-docker/data/mydata
    ↓
Docker: Mount at /var/lib/datadatdat-docker/mnt/mydata
    ↓
Container: See /data mounted
```

---

## Debugging Commands

### Check Installation Status
```bash
# Verify containers running
docker ps --filter name=datadatdat

# Check server logs
docker logs datadatdat-docker-server

# Check launch logs
docker logs datadatdat-docker-launch

# Verify ZFS pool
docker exec datadatdat-docker-server zpool list
docker exec datadatdat-docker-server zpool status datadatdat-docker

# Check ZFS filesystems
docker exec datadatdat-docker-server zfs list

# Verify database
docker exec datadatdat-docker-server su - postgres -c "psql -l"

# Test REST API
curl http://localhost:5001/api/v1/repositories

# Check volume plugin socket
ls -la /run/docker/plugins/datadatdat-docker.sock
```

### Manual ZFS Operations
```bash
# Enter server container
docker exec -it datadatdat-docker-server /bin/bash

# List all snapshots
zfs list -t snapshot

# Check pool health
zpool status -v datadatdat-docker

# Monitor pool I/O
zpool iostat datadatdat-docker 1
```

---

## Configuration Files

### Persistent Configuration
**Location:** `~/.config/datadatdat/providers.json` (on host)

**Contents:**
```json
{
  "default": "docker",
  "docker": {
    "name": "docker",
    "type": "docker",
    "port": 5001
  }
}
```

### ZFS Pool Configuration
**Location:** `/var/lib/datadatdat-docker/data/pool/cachefile` (in volume)

**Purpose:** Stores ZFS pool import cache

---

## Summary

The `titan install` (d3 install) command orchestrates a complex multi-phase installation:

1. **Validates** Docker environment
2. **Pulls** the datadatdat/datadatdat:latest image (~1GB)
3. **Starts** datadatdat-docker-launch container
4. **Loads** ZFS kernel modules (5 fallback strategies)
5. **Creates** ZFS pool and filesystems
6. **Starts** datadatdat-docker-server container
7. **Initializes** PostgreSQL database
8. **Launches** Docker volume plugin
9. **Starts** REST API server on port 5001

**Result:** Two containers running:
- `datadatdat-docker-launch` - Infrastructure bootstrap
- `datadatdat-docker-server` - Application server

**Success:** When server responds to API requests at `http://localhost:5001`

---

## Document Metadata

- **Created:** November 3, 2025
- **Purpose:** Comprehensive technical documentation of d3 install process
- **Audience:** Developers and advanced users
- **Scope:** Complete end-to-end process from CLI command to running containers
- **Version:** Based on datadatdat v0.8.x codebase analysis
