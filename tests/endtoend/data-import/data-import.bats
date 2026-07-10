#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# E2E Data Import Tests
# Tests dit cp (copy) and dit migrate commands

# Load shared test helpers
load '../test_helper'

# Cleanup after all tests
teardown_file() {
  "$D3" rm -f cp-test 2>/dev/null || true
  "$D3" rm -f migrate-dest 2>/dev/null || true
  docker rm -f migrate-src 2>/dev/null || true
}

# ========================================
# dit cp tests
# ========================================

@test "run mongo for cp test" {
  run "$D3" run -n cp-test -P mongo
  assert_success
  assert_output --partial "Running controlled container cp-test"
  sleep 5
}

@test "dit cp copies directory into repository" {
  # dit cp copies a directory's contents into a volume mount.
  # Use repo-relative path since Git Bash /tmp is not visible to Docker Desktop on Windows.
  mkdir -p "${REPO_ROOT}/cptest_data"
  echo "test content for cp" > "${REPO_ROOT}/cptest_data/cptest_file.txt"
  run "$D3" cp -s "${REPO_ROOT}/cptest_data" -d /data/configdb cp-test
  rm -rf "${REPO_ROOT}/cptest_data"
  assert_success
}

@test "docker exec confirms copied file exists in container" {
  run docker exec cp-test ls /data/configdb/cptest_file.txt
  assert_success
}

@test "dit commit after cp succeeds" {
  run "$D3" commit -m "After file copy" cp-test
  assert_success
  assert_output --partial "Commit"
}

@test "dit cp on non-existent repo fails" {
  echo "test" > "$BATS_TMPDIR/cptest_file2.txt"
  run "$D3" cp -s "$BATS_TMPDIR/cptest_file2.txt" -d /tmp/ nonexistent-cp-xyz
  assert_failure
}

@test "cleanup cp-test" {
  run "$D3" rm -f cp-test
  assert_success
}

# ========================================
# dit migrate tests
# ========================================

@test "run unmanaged docker container for migrate" {
  run docker run -d --name migrate-src mongo:4
  assert_success
  sleep 5
}

@test "stop unmanaged container before migrate" {
  run docker stop migrate-src
  assert_success
}

@test "dit migrate captures unmanaged container" {
  run "$D3" migrate -s migrate-src migrate-dest
  assert_success
}

@test "dit ls shows migrated repository" {
  run "$D3" ls
  assert_success
  assert_output --partial "migrate-dest"
}

@test "dit status on migrated repo shows running" {
  run "$D3" status migrate-dest
  assert_success
  assert_output --partial "running"
}

@test "dit commit on migrated repo succeeds" {
  run "$D3" commit -m "Migrated repo commit" migrate-dest
  assert_success
  assert_output --partial "Commit"
}

@test "cleanup migrated repo" {
  "$D3" rm -f migrate-dest 2>/dev/null || true
  docker rm -f migrate-src 2>/dev/null || true
  true
}
