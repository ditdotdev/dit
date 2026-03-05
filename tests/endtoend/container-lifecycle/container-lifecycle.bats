#!/usr/bin/env bats

# E2E Container Lifecycle Tests
# Tests d3 start, stop, and status commands

# Load shared test helpers
load '../test_helper'

# Cleanup after all tests
teardown_file() {
  "$D3" rm -f lifecycle-test 2>/dev/null || true
}

# ========================================
# Start / Stop / Status workflow
# ========================================

@test "run mongo container for lifecycle test" {
  run "$D3" run -n lifecycle-test -P mongo
  assert_success
  assert_output --partial "Running controlled container lifecycle-test"
  sleep 5
}

@test "d3 status shows running" {
  run "$D3" status lifecycle-test
  assert_success
  # d3 status shows "Status:  <state>" - check for the actual Docker state
  assert_output --partial "running"
}

@test "d3 stop stops the container" {
  run "$D3" stop lifecycle-test
  assert_success
}

@test "docker confirms container is stopped after d3 stop" {
  run docker inspect --type container --format="{{.State.Status}}" lifecycle-test
  assert_success
  assert_output --partial "exited"
}

@test "d3 status shows exited after stop" {
  run "$D3" status lifecycle-test
  assert_success
  assert_output --partial "exited"
}

@test "d3 start restarts the container" {
  run "$D3" start lifecycle-test
  assert_success
  sleep 5
}

@test "docker confirms container is running after d3 start" {
  run docker inspect --type container --format="{{.State.Status}}" lifecycle-test
  assert_success
  assert_output --partial "running"
}

@test "d3 status shows running after start" {
  run "$D3" status lifecycle-test
  assert_success
  assert_output --partial "running"
}

# ========================================
# Commit after stop/start cycle
# ========================================

@test "d3 commit works after stop/start cycle" {
  run "$D3" commit -m "Post-restart commit" lifecycle-test
  assert_success
  assert_output --partial "Commit"

  # Save commit GUID for checkout test
  COMMIT_GUID=$(echo "$output" | grep -o "Commit [a-f0-9]*" | awk '{print $2}')
  echo "$COMMIT_GUID" > "$BATS_TMPDIR/lifecycle_commit.txt"
}

# ========================================
# Stop then checkout
# ========================================

@test "d3 stop before checkout" {
  run "$D3" stop lifecycle-test
  assert_success
}

@test "d3 checkout from stopped state restarts container" {
  [ -f "$BATS_TMPDIR/lifecycle_commit.txt" ] || skip "Commit GUID not saved"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/lifecycle_commit.txt")

  run "$D3" checkout --commit "$COMMIT_GUID" lifecycle-test
  assert_success
  assert_output --partial "$COMMIT_GUID"
  sleep 5
}

@test "container is running after checkout from stopped state" {
  run docker inspect --type container --format="{{.State.Status}}" lifecycle-test
  assert_success
  assert_output --partial "running"
}

# ========================================
# Error paths (d3 stop/start/status return exit 0 even for errors)
# ========================================

@test "d3 stop on non-existent repo prints error" {
  run "$D3" stop nonexistent-lifecycle-xyz
  assert_output --partial "Error"
}

@test "d3 start on non-existent repo prints error" {
  run "$D3" start nonexistent-lifecycle-xyz
  assert_output --partial "Error"
}

@test "d3 status on non-existent repo prints error" {
  run "$D3" status nonexistent-lifecycle-xyz
  assert_failure
  assert_output --partial "Error"
}

# ========================================
# Cleanup
# ========================================

@test "cleanup lifecycle-test" {
  run "$D3" rm -f lifecycle-test
  assert_success
}
