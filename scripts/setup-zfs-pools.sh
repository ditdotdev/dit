#!/bin/bash
# ZFS Pool Setup for Datadatdat Linux Development
#
# Provisions the loop-backed ZFS pools that the d3 server expects
# (datadatdat-docker, datadatdat, datadatdat-one, datadatdat-two) on
# a fresh native-Linux or WSL2 development box.
#
# Moved from cleanslate/ to scripts/ as part of datadatdat#129. The
# "clean slate testing" workflow it was part of has been retired; this
# script remains the canonical way to provision host ZFS pools for d3
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

echo -e "${GREEN}Setting up ZFS pools for Datadatdat...${NC}"

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
    $ZFS_CMD zpool destroy datadatdat-docker 2>/dev/null || true
    $ZFS_CMD zpool destroy datadatdat 2>/dev/null || true
    $ZFS_CMD zpool destroy datadatdat-one 2>/dev/null || true
    $ZFS_CMD zpool destroy datadatdat-two 2>/dev/null || true

    # Remove pool image files
    echo "Removing pool image files..."
    $ZFS_CMD rm -rf /datadatdat-pools

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
    $ZFS_CMD zpool export datadatdat 2>/dev/null || true
    $ZFS_CMD zpool export datadatdat-docker 2>/dev/null || true
    $ZFS_CMD zpool export datadatdat-one 2>/dev/null || true
    $ZFS_CMD zpool export datadatdat-two 2>/dev/null || true
    sleep 2
    $ZFS_CMD zpool import datadatdat 2>/dev/null || true
    $ZFS_CMD zpool import datadatdat-docker 2>/dev/null || true
    $ZFS_CMD zpool import datadatdat-one 2>/dev/null || true
    $ZFS_CMD zpool import datadatdat-two 2>/dev/null || true
fi

# Create pool storage directory
echo "Creating pool storage directory..."
$ZFS_CMD mkdir -p /datadatdat-pools

# Create datadatdat-docker pool if it does not exist
echo "Checking for datadatdat-docker pool..."
datadatdat_docker_exists=$($ZFS_CMD zpool list datadatdat-docker 2>/dev/null || true)
if [ -z "$datadatdat_docker_exists" ]; then
    echo -e "${YELLOW}Creating datadatdat-docker pool (2GB)...${NC}"

    # Create image file (2GB = 2048MB)
    $ZFS_CMD dd if=/dev/zero of=/datadatdat-pools/datadatdat-docker.img bs=1M count=2048 2>/dev/null

    # Create loop device and get its path
    loop_device=$($ZFS_CMD losetup --show -f /datadatdat-pools/datadatdat-docker.img)

    if [ -n "$loop_device" ]; then
        # Create the pool
        $ZFS_CMD zpool create datadatdat-docker "$loop_device"
        echo -e "${GREEN}OK datadatdat-docker pool created successfully${NC}"
    else
        echo -e "${RED}X Failed to create loop device for datadatdat-docker${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}OK datadatdat-docker pool already exists${NC}"
fi

# Create main datadatdat pool if it does not exist
echo "Checking for datadatdat pool..."
datadatdat_exists=$($ZFS_CMD zpool list datadatdat 2>/dev/null || true)
if [ -z "$datadatdat_exists" ]; then
    echo -e "${YELLOW}Creating datadatdat pool (2GB)...${NC}"

    # Create image file (2GB = 2048MB)
    $ZFS_CMD dd if=/dev/zero of=/datadatdat-pools/datadatdat.img bs=1M count=2048 2>/dev/null

    # Create loop device and get its path
    loop_device=$($ZFS_CMD losetup --show -f /datadatdat-pools/datadatdat.img)

    if [ -n "$loop_device" ]; then
        # Create the pool
        $ZFS_CMD zpool create datadatdat "$loop_device"
        echo -e "${GREEN}OK datadatdat pool created successfully${NC}"
    else
        echo -e "${RED}X Failed to create loop device for datadatdat${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}OK datadatdat pool already exists${NC}"
fi

# Create datadatdat-one pool if it does not exist (for multi-context tests)
echo "Checking for datadatdat-one pool..."
datadatdat_one_exists=$($ZFS_CMD zpool list datadatdat-one 2>/dev/null || true)
if [ -z "$datadatdat_one_exists" ]; then
    echo -e "${YELLOW}Creating datadatdat-one pool (2GB for multi-context tests)...${NC}"

    # Create image file (2GB = 2048MB)
    $ZFS_CMD dd if=/dev/zero of=/datadatdat-pools/datadatdat-one.img bs=1M count=2048 2>/dev/null

    # Create loop device and get its path
    loop_device=$($ZFS_CMD losetup --show -f /datadatdat-pools/datadatdat-one.img)

    if [ -n "$loop_device" ]; then
        # Create the pool
        $ZFS_CMD zpool create datadatdat-one "$loop_device"
        echo -e "${GREEN}OK datadatdat-one pool created successfully${NC}"
    else
        echo -e "${RED}X Failed to create loop device for datadatdat-one${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}OK datadatdat-one pool already exists${NC}"
fi

# Create datadatdat-two pool if it does not exist (for multi-context tests)
echo "Checking for datadatdat-two pool..."
datadatdat_two_exists=$($ZFS_CMD zpool list datadatdat-two 2>/dev/null || true)
if [ -z "$datadatdat_two_exists" ]; then
    echo -e "${YELLOW}Creating datadatdat-two pool (2GB for multi-context tests)...${NC}"

    # Create image file (2GB = 2048MB)
    $ZFS_CMD dd if=/dev/zero of=/datadatdat-pools/datadatdat-two.img bs=1M count=2048 2>/dev/null

    # Create loop device and get its path
    loop_device=$($ZFS_CMD losetup --show -f /datadatdat-pools/datadatdat-two.img)

    if [ -n "$loop_device" ]; then
        # Create the pool
        $ZFS_CMD zpool create datadatdat-two "$loop_device"
        echo -e "${GREEN}OK datadatdat-two pool created successfully${NC}"
    else
        echo -e "${RED}X Failed to create loop device for datadatdat-two${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}OK datadatdat-two pool already exists${NC}"
fi

# Final verification
echo ""
echo -e "${GREEN}Final ZFS pool status:${NC}"
$ZFS_CMD zpool list

echo ""
echo -e "${GREEN}Pool health check:${NC}"
$ZFS_CMD zpool status

echo ""
echo -e "${GREEN}ZFS pools are ready for Datadatdat${NC}"
echo -e "${CYAN}Standard pools: datadatdat, datadatdat-docker${NC}"
echo -e "${CYAN}Multi-context test pools: datadatdat-one, datadatdat-two${NC}"
echo -e "${CYAN}You can now run: d3 install${NC}"
echo ""
echo -e "${WHITE}For a full reset, run: bash setup-zfs-pools.sh --clean${NC}"
