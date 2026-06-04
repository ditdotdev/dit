#!/bin/bash
# ZFS Pool Setup for Dit Linux Development
#
# Provisions the loop-backed ZFS pools that the dit server expects
# (dit-docker, dit, dit-one, dit-two) on
# a fresh native-Linux or WSL2 development box.
#
# Moved from cleanslate/ to scripts/ as part of dit#129. The
# "clean slate testing" workflow it was part of has been retired; this
# script remains the canonical way to provision host ZFS pools for dit
# development.
#
# Usage:
#   bash setup-zfs-pools.sh           # create any missing pools
#   bash setup-zfs-pools.sh --clean   # destroy and recreate all pools

# Prevent Git Bash from converting Unix paths to Windows paths
export MSYS_NO_PATHCONV=1

# Parse command line arguments
CLEAN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --clean)
            CLEAN=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--clean]"
            exit 1
            ;;
    esac
done

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

echo -e "${GREEN}Setting up ZFS pools for Dit...${NC}"

# Detect environment (GitHub Actions, WSL, or native Linux). Must run
# before any block that uses $ZFS_CMD.
if [ -n "$GITHUB_ACTIONS" ]; then
    echo -e "${CYAN}Detected GitHub Actions environment (native Linux)${NC}"
    ENVIRONMENT="github-actions"
    ZFS_CMD=""  # Direct commands, no WSL wrapper
elif command -v wsl &>/dev/null && wsl --status &>/dev/null 2>&1; then
    echo -e "${GREEN}WSL is available${NC}"
    ENVIRONMENT="wsl"
    ZFS_CMD="wsl sudo"  # Commands via WSL
else
    # Assume native Linux (Docker container, bare metal, etc.)
    echo -e "${CYAN}Detected native Linux environment${NC}"
    ENVIRONMENT="native-linux"
    ZFS_CMD="sudo"  # Direct sudo commands
fi

if [ "$CLEAN" = true ]; then
    echo -e "${YELLOW}Clean requested - removing existing ZFS pools...${NC}"

    # Remove existing pools (ignore errors if pools don't exist)
    echo "Destroying existing ZFS pools..."
    $ZFS_CMD zpool destroy dit-docker 2>/dev/null || true
    $ZFS_CMD zpool destroy dit 2>/dev/null || true
    $ZFS_CMD zpool destroy dit-one 2>/dev/null || true
    $ZFS_CMD zpool destroy dit-two 2>/dev/null || true

    # Remove pool image files
    echo "Removing pool image files..."
    $ZFS_CMD rm -rf /dit-pools

    # Remove loop devices
    echo "Removing loop devices..."
    $ZFS_CMD losetup -D

    # Verify cleanup
    pool_check=$($ZFS_CMD zpool list 2>/dev/null || echo "no pools available")
    if [[ "$pool_check" =~ no\ pools\ available ]]; then
        echo -e "${GREEN}OK All ZFS pools successfully removed${NC}"
    else
        echo -e "${YELLOW}Some pools may still exist:${NC}"
        $ZFS_CMD zpool list
    fi
fi

# Check ZFS kernel support
echo "Checking ZFS kernel support..."
case $ENVIRONMENT in
    wsl)
        zfs_support=$(wsl cat /proc/filesystems | grep zfs || true)
        ;;
    *)
        zfs_support=$(cat /proc/filesystems 2>/dev/null | grep zfs || true)
        ;;
esac

if [ -n "$zfs_support" ]; then
    echo -e "${GREEN}OK ZFS kernel support detected${NC}"
else
    echo -e "${RED}X ZFS kernel support not found${NC}"
    if [ "$ENVIRONMENT" = "github-actions" ]; then
        echo -e "${YELLOW}Note: ZFS may need to be installed in GitHub Actions runners${NC}"
        echo -e "${YELLOW}Consider using ZFS modules or Docker-based testing instead${NC}"
    fi
    exit 1
fi

# Check if pools already exist
echo "Checking existing ZFS pools..."
existing_pools=$($ZFS_CMD zpool list 2>/dev/null || true)
if [ -n "$existing_pools" ] && ! [[ "$existing_pools" =~ no\ pools\ available ]]; then
    echo -e "${YELLOW}Existing pools found:${NC}"
    $ZFS_CMD zpool list

    # Handle hostid mismatches
    echo "Fixing any hostid mismatches..."
    $ZFS_CMD zpool export dit 2>/dev/null || true
    $ZFS_CMD zpool export dit-docker 2>/dev/null || true
    $ZFS_CMD zpool export dit-one 2>/dev/null || true
    $ZFS_CMD zpool export dit-two 2>/dev/null || true
    sleep 2
    $ZFS_CMD zpool import dit 2>/dev/null || true
    $ZFS_CMD zpool import dit-docker 2>/dev/null || true
    $ZFS_CMD zpool import dit-one 2>/dev/null || true
    $ZFS_CMD zpool import dit-two 2>/dev/null || true
fi

# Create pool storage directory
echo "Creating pool storage directory..."
$ZFS_CMD mkdir -p /dit-pools

# Create dit-docker pool if it does not exist
echo "Checking for dit-docker pool..."
dit_docker_exists=$($ZFS_CMD zpool list dit-docker 2>/dev/null || true)
if [ -z "$dit_docker_exists" ]; then
    echo -e "${YELLOW}Creating dit-docker pool (2GB)...${NC}"

    # Create image file (2GB = 2048MB)
    $ZFS_CMD dd if=/dev/zero of=/dit-pools/dit-docker.img bs=1M count=2048 2>/dev/null

    # Create loop device and get its path
    loop_device=$($ZFS_CMD losetup --show -f /dit-pools/dit-docker.img)

    if [ -n "$loop_device" ]; then
        # Create the pool
        $ZFS_CMD zpool create dit-docker "$loop_device"
        echo -e "${GREEN}OK dit-docker pool created successfully${NC}"
    else
        echo -e "${RED}X Failed to create loop device for dit-docker${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}OK dit-docker pool already exists${NC}"
fi

# Create main dit pool if it does not exist
echo "Checking for dit pool..."
dit_exists=$($ZFS_CMD zpool list dit 2>/dev/null || true)
if [ -z "$dit_exists" ]; then
    echo -e "${YELLOW}Creating dit pool (2GB)...${NC}"

    # Create image file (2GB = 2048MB)
    $ZFS_CMD dd if=/dev/zero of=/dit-pools/dit.img bs=1M count=2048 2>/dev/null

    # Create loop device and get its path
    loop_device=$($ZFS_CMD losetup --show -f /dit-pools/dit.img)

    if [ -n "$loop_device" ]; then
        # Create the pool
        $ZFS_CMD zpool create dit "$loop_device"
        echo -e "${GREEN}OK dit pool created successfully${NC}"
    else
        echo -e "${RED}X Failed to create loop device for dit${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}OK dit pool already exists${NC}"
fi

# Create dit-one pool if it does not exist (for multi-context tests)
echo "Checking for dit-one pool..."
dit_one_exists=$($ZFS_CMD zpool list dit-one 2>/dev/null || true)
if [ -z "$dit_one_exists" ]; then
    echo -e "${YELLOW}Creating dit-one pool (2GB for multi-context tests)...${NC}"

    # Create image file (2GB = 2048MB)
    $ZFS_CMD dd if=/dev/zero of=/dit-pools/dit-one.img bs=1M count=2048 2>/dev/null

    # Create loop device and get its path
    loop_device=$($ZFS_CMD losetup --show -f /dit-pools/dit-one.img)

    if [ -n "$loop_device" ]; then
        # Create the pool
        $ZFS_CMD zpool create dit-one "$loop_device"
        echo -e "${GREEN}OK dit-one pool created successfully${NC}"
    else
        echo -e "${RED}X Failed to create loop device for dit-one${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}OK dit-one pool already exists${NC}"
fi

# Create dit-two pool if it does not exist (for multi-context tests)
echo "Checking for dit-two pool..."
dit_two_exists=$($ZFS_CMD zpool list dit-two 2>/dev/null || true)
if [ -z "$dit_two_exists" ]; then
    echo -e "${YELLOW}Creating dit-two pool (2GB for multi-context tests)...${NC}"

    # Create image file (2GB = 2048MB)
    $ZFS_CMD dd if=/dev/zero of=/dit-pools/dit-two.img bs=1M count=2048 2>/dev/null

    # Create loop device and get its path
    loop_device=$($ZFS_CMD losetup --show -f /dit-pools/dit-two.img)

    if [ -n "$loop_device" ]; then
        # Create the pool
        $ZFS_CMD zpool create dit-two "$loop_device"
        echo -e "${GREEN}OK dit-two pool created successfully${NC}"
    else
        echo -e "${RED}X Failed to create loop device for dit-two${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}OK dit-two pool already exists${NC}"
fi

# Final verification
echo ""
echo -e "${GREEN}Final ZFS pool status:${NC}"
$ZFS_CMD zpool list

echo ""
echo -e "${GREEN}Pool health check:${NC}"
$ZFS_CMD zpool status

echo ""
echo -e "${GREEN}ZFS pools are ready for Dit${NC}"
echo -e "${CYAN}Standard pools: dit, dit-docker${NC}"
echo -e "${CYAN}Multi-context test pools: dit-one, dit-two${NC}"
echo -e "${CYAN}You can now run: dit install${NC}"
echo ""
echo -e "${WHITE}For a full reset, run: bash setup-zfs-pools.sh --clean${NC}"
