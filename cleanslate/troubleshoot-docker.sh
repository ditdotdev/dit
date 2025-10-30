#!/bin/bash
# Datadatdat Docker Troubleshooting Script
# Diagnoses and fixes common Docker container execution issues

# Parse command line arguments
FIX=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --fix)
            FIX=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--fix] [--verbose]"
            exit 1
            ;;
    esac
done

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

echo -e "${GREEN}Datadatdat Docker Troubleshooting Script${NC}"
echo -e "${GREEN}=====================================${NC}"

# Function to write status messages
write_status() {
    local message="$1"
    local status="${2:-INFO}"
    local timestamp=$(date '+%H:%M:%S')
    
    case "$status" in
        OK)
            echo -e "${GREEN}[$timestamp] $message${NC}"
            ;;
        WARN)
            echo -e "${YELLOW}[$timestamp] $message${NC}"
            ;;
        ERROR)
            echo -e "${RED}[$timestamp] $message${NC}"
            ;;
        *)
            echo -e "${WHITE}[$timestamp] $message${NC}"
            ;;
    esac
}

# Check Docker daemon status
write_status "Checking Docker daemon status..."
if docker_version=$(docker version --format json 2>/dev/null); then
    if echo "$docker_version" | grep -q '"Server"'; then
        client_version=$(echo "$docker_version" | grep -o '"Version":"[^"]*"' | head -1 | cut -d'"' -f4)
        server_version=$(echo "$docker_version" | grep -o '"Version":"[^"]*"' | tail -1 | cut -d'"' -f4)
        write_status "Docker daemon is running - Client: $client_version, Server: $server_version" "OK"
    else
        write_status "Docker daemon is not accessible!" "ERROR"
        if [ "$FIX" = true ]; then
            write_status "Attempting to start Docker Desktop..."
            docker_path="/c/Program Files/Docker/Docker/Docker Desktop.exe"
            if [ -f "$docker_path" ]; then
                if ! pgrep -f "Docker Desktop" > /dev/null; then
                    "$docker_path" &
                    write_status "Docker Desktop started. Please wait a moment and try again."
                else
                    write_status "Docker Desktop is running but daemon not ready. Please wait and try again."
                fi
            else
                write_status "Docker Desktop not found. Please install Docker Desktop."
            fi
        else
            write_status "Use --fix parameter to attempt Docker startup"
        fi
        exit 1
    fi
else
    write_status "Docker daemon is not accessible!" "ERROR"
    if [ "$FIX" = true ]; then
        write_status "Attempting to start Docker Desktop..."
        docker_path="/c/Program Files/Docker/Docker/Docker Desktop.exe"
        if [ -f "$docker_path" ]; then
            if ! pgrep -f "Docker Desktop" > /dev/null; then
                "$docker_path" &
                write_status "Docker Desktop started. Please wait a moment and try again."
            else
                write_status "Docker Desktop is running but daemon not ready. Please wait and try again."
            fi
        else
            write_status "Docker Desktop not found. Please install Docker Desktop."
        fi
    else
        write_status "Use --fix parameter to attempt Docker startup"
    fi
    exit 1
fi

# Check existing Datadatdat containers
write_status "Checking Datadatdat container status..."
datadatdat_containers=$(docker ps -a --filter "name=datadatdat" --format "{{.Names}} {{.Status}}" || true)
if [ -n "$datadatdat_containers" ]; then
    write_status "Found Datadatdat containers:" "OK"
    echo "$datadatdat_containers" | while read -r line; do
        write_status "  $line"
    done
else
    write_status "No Datadatdat containers found" "WARN"
fi

# Check for container name conflicts
write_status "Checking for potential container name conflicts..."
all_containers=$(docker ps -a --format "{{.Names}}" || true)
conflicting_names=("testpostgres" "testredis" "testhello")
conflicts=()

for name in "${conflicting_names[@]}"; do
    if echo "$all_containers" | grep -q "^${name}$"; then
        conflicts+=("$name")
        write_status "Found conflicting container: $name" "WARN"
    fi
done

if [ ${#conflicts[@]} -gt 0 ] && [ "$FIX" = true ]; then
    write_status "Removing conflicting containers..."
    for name in "${conflicts[@]}"; do
        docker rm -f "$name" 2>/dev/null || true
        write_status "Removed container: $name" "OK"
    done
fi

# Check ZFS pool status
write_status "Checking ZFS pool status..."
zfs_status=$(wsl zpool list 2>/dev/null || true)
if [[ "$zfs_status" =~ datadatdat ]]; then
    write_status "ZFS pools are available" "OK"
    if [ "$VERBOSE" = true ]; then
        wsl zpool list
    fi
else
    write_status "ZFS pools not found!" "ERROR"
    if [ "$FIX" = true ]; then
        write_status "Recreating ZFS pools..."
        bash "$(dirname "$0")/setup-zfs-pools.sh" --clean
    fi
fi

# Check Docker volumes
write_status "Checking Docker volumes..."
volumes=$(docker volume ls --filter "name=datadatdat" --format "{{.Name}}" || true)
if [ -n "$volumes" ]; then
    write_status "Found Datadatdat volumes:" "OK"
    echo "$volumes" | while read -r line; do
        write_status "  $line"
    done
else
    write_status "No Datadatdat volumes found" "WARN"
fi

# Test Docker socket from Datadatdat container
if echo "$datadatdat_containers" | grep -q "datadatdat-docker-launch.*Up"; then
    write_status "Testing Docker socket access from Datadatdat container..."
    test_result=$(docker exec datadatdat-docker-launch docker run --rm hello-world 2>/dev/null || true)
    if [[ "$test_result" =~ Hello\ from\ Docker ]]; then
        write_status "Docker socket is accessible from Datadatdat container" "OK"
    else
        write_status "Docker socket test failed" "ERROR"
    fi
fi

# Check for common Docker issues
write_status "Checking for common Docker issues..."

# Check disk space (Windows C: drive)
if command -v wmic &> /dev/null; then
    free_space=$(wmic logicaldisk where "DeviceID='C:'" get FreeSpace /value 2>/dev/null | grep FreeSpace | cut -d= -f2)
    if [ -n "$free_space" ]; then
        free_space_gb=$(echo "scale=2; $free_space / 1073741824" | bc)
        if (( $(echo "$free_space_gb < 5" | bc -l) )); then
            write_status "Low disk space: ${free_space_gb}GB free" "WARN"
        else
            write_status "Disk space OK: ${free_space_gb}GB free" "OK"
        fi
    fi
fi

# Check if Docker Desktop is using WSL2
docker_info=$(docker info --format json 2>/dev/null || echo "{}")
if echo "$docker_info" | grep -q "Docker Desktop"; then
    write_status "Docker Desktop detected" "OK"
    if echo "$docker_info" | grep -q '"runc"'; then
        write_status "Using runc runtime" "OK"
    fi
else
    write_status "Non-Docker Desktop environment detected" "WARN"
fi

# Summary and recommendations
echo ""
echo -e "${GREEN}Summary and Recommendations:${NC}"
echo -e "${GREEN}============================${NC}"

if [ ${#conflicts[@]} -gt 0 ]; then
    write_status "Found ${#conflicts[@]} container name conflicts" "WARN"
    write_status "Recommendation: Run script with --fix flag to resolve" "WARN"
fi

if ! echo "$datadatdat_containers" | grep -q "datadatdat-docker-launch.*Up"; then
    write_status "No Datadatdat infrastructure containers running" "WARN"
    write_status "Recommendation: Run '../d3.exe install' to start infrastructure" "WARN"
fi

write_status "Troubleshooting complete!" "OK"

if [ "$FIX" = true ]; then
    write_status "Applied automatic fixes where possible" "OK"
else
    write_status "Run with --fix flag to automatically resolve issues" "INFO"
fi
