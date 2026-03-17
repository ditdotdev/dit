#!/bin/bash
#
# Run datadatdat E2E tests inside WSL2 in the Vagrant VM.
#
# This script runs inside WSL2 and validates that the prebuilt kernel
# works correctly with the full datadatdat stack.
#

set -euo pipefail

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

D3_REPO="/home/$(whoami)/datadatdat"
if [ ! -d "$D3_REPO" ]; then
    echo "Cloning datadatdat repo..."
    git clone https://github.com/datadatdat/datadatdat.git "$D3_REPO"
fi

cd "$D3_REPO"
git pull
make build
echo "d3 CLI built successfully"
echo ""

# --- ZFS Pool Setup ---
echo "--- ZFS Pool Setup ---"

if [ -f cleanslate/setup-zfs-pools.sh ]; then
    bash cleanslate/setup-zfs-pools.sh --clean || {
        echo "WARNING: ZFS pool setup had issues, continuing anyway..."
    }
fi

echo ""

# --- E2E Tests ---
echo "--- Running E2E Tests (make e2e) ---"

E2E_RESULT=0
make e2e || E2E_RESULT=$?

echo ""

if [ $E2E_RESULT -eq 0 ]; then
    echo "PASS: make e2e completed successfully"
else
    echo "FAIL: make e2e failed with exit code $E2E_RESULT"
fi

# --- Server E2E Tests (optional) ---
echo ""
echo "--- Running Server E2E Tests (make e2e-server) ---"

SERVER_RESULT=0
make e2e-server || SERVER_RESULT=$?

echo ""

if [ $SERVER_RESULT -eq 0 ]; then
    echo "PASS: make e2e-server completed successfully"
else
    echo "FAIL: make e2e-server failed with exit code $SERVER_RESULT"
fi

# --- Summary ---
echo ""
echo "=== E2E Test Summary ==="
echo "Kernel:       $KERNEL"
echo "ZFS:          $(grep zfs /proc/filesystems 2>/dev/null && echo 'built-in' || echo 'NOT FOUND')"
echo "make e2e:     $([ $E2E_RESULT -eq 0 ] && echo 'PASS' || echo 'FAIL')"
echo "make e2e-server: $([ $SERVER_RESULT -eq 0 ] && echo 'PASS' || echo 'FAIL')"
echo ""

if [ $E2E_RESULT -ne 0 ] || [ $SERVER_RESULT -ne 0 ]; then
    echo "OVERALL: FAIL — Do NOT publish this kernel release"
    exit 1
fi

echo "OVERALL: PASS — Safe to promote draft release to published"
exit 0
