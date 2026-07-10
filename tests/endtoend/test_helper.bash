#!/usr/bin/env bash
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# Shared test helper functions for BATS tests
# This file provides fallback implementations of bats-assert and bats-support
# if they are not installed on the system.

# Try to load bats-support and bats-assert from common locations
load_bats_helpers_if_available() {
  for helper_dir in \
    "/usr/lib/bats" \
    "/usr/local/lib/bats" \
    "$HOME/.bats/lib" \
    "$(dirname "$BATS_TEST_DIRNAME")/../node_modules" \
    ; do
    if [ -f "$helper_dir/bats-support/load.bash" ]; then
      load "$helper_dir/bats-support/load.bash"
    fi
    if [ -f "$helper_dir/bats-assert/load.bash" ]; then
      load "$helper_dir/bats-assert/load.bash"
    fi
  done
}

# Fallback implementation of assert_output if not provided by bats-assert
if ! command -v assert_output &> /dev/null; then
  assert_output() {
    if [ "$1" = "--partial" ]; then
      if ! echo "$output" | grep -qF "$2"; then
        echo "Expected output to contain: $2"
        echo "Actual output: $output"
        return 1
      fi
    else
      if [ "$output" != "$1" ]; then
        echo "Expected: $1"
        echo "Actual: $output"
        return 1
      fi
    fi
  }
fi

# Fallback implementation of assert_success if not provided by bats-assert
if ! command -v assert_success &> /dev/null; then
  assert_success() {
    if [ "$status" -ne 0 ]; then
      echo "Expected success (exit 0), got exit $status"
      echo "Output: $output"
      return 1
    fi
  }
fi

# Fallback implementation of assert_failure if not provided by bats-assert
if ! command -v assert_failure &> /dev/null; then
  assert_failure() {
    if [ "$status" -eq 0 ]; then
      echo "Expected failure (non-zero exit), got exit 0"
      echo "Output: $output"
      return 1
    fi
  }
fi

# Set up common paths for all tests
# Get the repository root (two levels up from test_helper.bash location: tests/endtoend)
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
D3="${REPO_ROOT}/dit"

# Helper function to get Docker-compatible path (works on Windows and Linux)
get_docker_path() {
  local path="$1"
  # On Windows Git Bash: convert /c/path to C:/path
  # On Linux: return as-is
  if command -v pwd &> /dev/null && pwd -W &> /dev/null 2>&1; then
    # Windows Git Bash - convert path
    echo "$(cd "$path" && pwd -W)"
  else
    # Linux or other Unix-like systems
    echo "$path"
  fi
}

# Kill stale AWS CLI processes that may hang and block subsequent tests
# Call this in teardown_file() for any test that uses AWS CLI
cleanup_stale_aws_processes() {
  local aws_pids
  aws_pids=$(tasklist 2>/dev/null | grep -i "aws.exe" | awk '{print $2}' || true)
  if [ -n "$aws_pids" ]; then
    echo "# WARNING: Found stale aws.exe processes, killing them" >&3 || true
    echo "$aws_pids" | while read -r pid; do
      taskkill //PID "$pid" //F 2>/dev/null || true
    done
    sleep 1
  fi
}

# Fallback implementation of refute_output if not provided by bats-assert
if ! command -v refute_output &> /dev/null; then
  refute_output() {
    if [ "$1" = "--partial" ]; then
      if echo "$output" | grep -qF "$2"; then
        echo "Expected output NOT to contain: $2"
        echo "Actual output: $output"
        return 1
      fi
    else
      if [ "$output" = "$1" ]; then
        echo "Expected output to differ from: $1"
        return 1
      fi
    fi
  }
fi

# Load helpers on import
load_bats_helpers_if_available
