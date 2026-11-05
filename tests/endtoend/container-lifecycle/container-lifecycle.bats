#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# E2E Container Lifecycle Tests
# Tests dit start, stop, and status commands

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

# ========================================
# docker cp against a running repository (#260)
#
# docker cp does not read through the container's live mount namespace. For
# every volume it asks the volume driver to Mount the volume for the copy and
# to Unmount it afterwards, reusing the container's own mount ID. The dit
# volume plugin used to forward each of those straight to the server, so the
# second mount failed and the unmount pulled the ZFS dataset out from under
# the running database. The plugin now reference-counts mounts.
# ========================================

@test "docker cp out of a running repository succeeds (#260)" {
  run docker exec lifecycle-test sh -c 'mkdir -p /data/configdb/.dit-cp-check && echo cp-marker > /data/configdb/.dit-cp-check/marker'
  assert_success

  run bash -c 'MSYS_NO_PATHCONV=1 docker cp lifecycle-test:/data/configdb/.dit-cp-check/marker - | tar -xO'
  assert_success
  assert_output "cp-marker"
}

@test "docker cp into a running repository succeeds (#260)" {
  # The host directory lives under the repo (not /tmp): on Windows Docker
  # Desktop only paths under the user's drive are visible to docker cp.
  local src="$REPO_ROOT/.dit-cp-check-lifecycle"
  rm -rf "$src"
  mkdir -p "$src"
  echo "cp-in-marker" > "$src/in-marker"
  run bash -c "MSYS_NO_PATHCONV=1 docker cp '$(get_docker_path "$src")/.' lifecycle-test:/data/configdb/.dit-cp-check/"
  rm -rf "$src"
  assert_success

  run docker exec lifecycle-test cat /data/configdb/.dit-cp-check/in-marker
  assert_success
  assert_output "cp-in-marker"
}

@test "volume is still mounted for the container after docker cp (#260)" {
  # The Unmount that follows docker cp must not deactivate the volume the
  # running container still uses: the file written before the copies is
  # still there, and the database still answers.
  run docker exec lifecycle-test cat /data/configdb/.dit-cp-check/marker
  assert_success
  assert_output "cp-marker"

  run docker exec lifecycle-test mongosh --quiet --eval 'db.runCommand({ping:1}).ok'
  assert_success
  assert_output "1"
}

@test "dit status shows running" {
  run "$D3" status lifecycle-test
  assert_success
  # dit status shows "Status:  <state>" - check for the actual Docker state
  assert_output --partial "running"
}

@test "dit stop stops the container" {
  run "$D3" stop lifecycle-test
  assert_success
}

@test "docker confirms container is stopped after dit stop" {
  run docker inspect --type container --format="{{.State.Status}}" lifecycle-test
  assert_success
  assert_output --partial "exited"
}

@test "dit status shows exited after stop" {
  run "$D3" status lifecycle-test
  assert_success
  assert_output --partial "exited"
}

@test "dit start restarts the container" {
  run "$D3" start lifecycle-test
  assert_success
  sleep 5
}

@test "docker confirms container is running after dit start" {
  run docker inspect --type container --format="{{.State.Status}}" lifecycle-test
  assert_success
  assert_output --partial "running"
}

@test "dit status shows running after start" {
  run "$D3" status lifecycle-test
  assert_success
  assert_output --partial "running"
}

# ========================================
# Commit after stop/start cycle
# ========================================

@test "dit commit works after stop/start cycle" {
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

@test "dit stop before checkout" {
  run "$D3" stop lifecycle-test
  assert_success
}

@test "dit checkout from stopped state restarts container" {
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
# Error paths (dit stop/start/status return exit 0 even for errors)
# ========================================

@test "dit stop on non-existent repo prints error" {
  run "$D3" stop nonexistent-lifecycle-xyz
  assert_output --partial "Error"
}

@test "dit start on non-existent repo prints error" {
  run "$D3" start nonexistent-lifecycle-xyz
  assert_output --partial "Error"
}

@test "dit status on non-existent repo prints error" {
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
