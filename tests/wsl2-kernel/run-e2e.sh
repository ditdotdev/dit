#!/bin/bash
#
# Run datadatdat E2E tests inside WSL2 in the Vagrant VM.
#
# This script runs inside WSL2 and validates that the prebuilt kernel
# works correctly with the full datadatdat stack.
#

set -euo pipefail

# Ensure Go is in PATH (installed to /usr/local/go by provisioner)
export PATH=$PATH:/usr/local/go/bin

# Skip Go module proxy for private datadatdat repos (fetch directly via SSH)
export GOPRIVATE=github.com/datadatdat/*

echo "=== WSL2 Kernel E2E Tests ==="
echo ""

# --- Kernel Validation ---
echo "--- Kernel Validation ---"

KERNEL=$(uname -r)
echo "Kernel: $KERNEL"

if grep -q "zfs" /proc/filesystems; then
    echo "PASS: ZFS detected in /proc/filesystems"
else
    echo "FAIL: ZFS not found in /proc/filesystems"
    exit 1
fi

# Check ZFS version if available
if command -v zfs &>/dev/null; then
    echo "ZFS version: $(zfs version 2>/dev/null | head -1)"
fi

echo ""

# --- Build d3 CLI ---
echo "--- Building d3 CLI ---"

# Verify SSH auth to GitHub before cloning
echo "Verifying GitHub SSH access..."
ssh -T git@github.com 2>&1 || true

D3_REPO="/home/$(whoami)/datadatdat"
if [ ! -d "$D3_REPO" ]; then
    echo "Cloning datadatdat repo..."
    GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=accept-new" git clone git@github.com:datadatdat/datadatdat.git "$D3_REPO"
fi

cd "$D3_REPO"
git pull

# Query Docker Hub for the latest datadatdat image tag so d3 install pulls a real image
echo "Querying Docker Hub for latest datadatdat/datadatdat image tag..."
LATEST_VERSION=$(curl -s "https://hub.docker.com/v2/repositories/datadatdat/datadatdat/tags/?page_size=10&ordering=last_updated" \
    | grep -o '"name":"v[0-9]*\.[0-9]*\.[0-9]*"' \
    | head -1 \
    | grep -o 'v[0-9]*\.[0-9]*\.[0-9]*' || echo "")

if [ -z "$LATEST_VERSION" ]; then
    echo "WARNING: Could not determine latest Docker Hub version, using 'dev'"
    LATEST_VERSION="dev"
else
    echo "Using version: $LATEST_VERSION"
fi

VERSION="$LATEST_VERSION" make build
cp build/d3 "$D3_REPO/d3"
chmod +x "$D3_REPO/d3"
echo "d3 CLI built and installed to $D3_REPO/d3"
./d3 --version
echo ""

# --- Install ZFS userspace tools ---
echo "--- Installing ZFS userspace tools ---"
sudo apt-get update -qq
sudo apt-get install -y zfsutils-linux
echo ""

# --- ZFS Pool Setup ---
echo "--- ZFS Pool Setup ---"

if [ -f cleanslate/setup-zfs-pools.sh ]; then
    bash cleanslate/setup-zfs-pools.sh --clean || {
        echo "WARNING: ZFS pool setup had issues, continuing anyway..."
    }
fi

echo ""

# --- D3 Installation Tests ---
echo "--- Running D3 Installation Tests (make test-install) ---"

INSTALL_RESULT=0
make test-install || INSTALL_RESULT=$?

echo ""

if [ $INSTALL_RESULT -eq 0 ]; then
    echo "PASS: make test-install completed successfully"
else
    echo "FAIL: make test-install failed with exit code $INSTALL_RESULT"
fi

echo ""


# --- Summary ---
echo ""
echo "=== Installation Test Summary ==="
echo "Kernel:       $KERNEL"
echo "ZFS:          $(grep zfs /proc/filesystems 2>/dev/null && echo 'built-in' || echo 'NOT FOUND')"
echo "make test-install: $([ $INSTALL_RESULT -eq 0 ] && echo 'PASS' || echo 'FAIL')"
echo ""

if [ $INSTALL_RESULT -ne 0 ]; then
    echo "OVERALL: FAIL - Do NOT publish this kernel release"
    exit 1
fi

echo "OVERALL: PASS - Safe to promote draft release to published"
exit 0
