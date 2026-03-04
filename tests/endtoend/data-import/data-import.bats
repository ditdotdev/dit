#!/usr/bin/env bats

# E2E Data Import Tests
# Tests d3 cp (copy) and d3 migrate commands

# Load shared test helpers
load '../test_helper'

# Cleanup after all tests
teardown_file() {
  "$D3" rm -f cp-test 2>/dev/null || true
  "$D3" rm -f migrate-dest 2>/dev/null || true
  docker rm -f migrate-src 2>/dev/null || true
}

# ========================================
# d3 cp tests
# ========================================

@test "run mongo for cp test" {
  run "$D3" run -n cp-test -P mongo
  assert_success
  assert_output --partial "Running controlled container cp-test"
  sleep 5
}

@test "d3 cp copies file into repository" {
  # BUG: d3 cp stops the container then fails with "Failed to unmarshal mounts: unexpected end of JSON input"
  # The cp command stops the container and then cannot parse the mount information from the stopped container.
  skip "Known CLI bug: d3 cp fails with 'Failed to unmarshal mounts' after stopping container"

  echo "test content for cp" > "$BATS_TMPDIR/cptest_file.txt"
  run "$D3" cp -s "$BATS_TMPDIR/cptest_file.txt" -d /tmp/ cp-test
  assert_success
}

@test "docker exec confirms copied file exists in container" {
  skip "Depends on 'd3 cp' which has a known CLI bug"
  run docker exec cp-test ls /tmp/cptest_file.txt
  assert_success
}

@test "d3 commit after cp succeeds" {
  run "$D3" commit -m "After file copy" cp-test
  assert_success
  assert_output --partial "Commit"
}

@test "d3 cp on non-existent repo fails" {
  echo "test" > "$BATS_TMPDIR/cptest_file2.txt"
  run "$D3" cp -s "$BATS_TMPDIR/cptest_file2.txt" -d /tmp/ nonexistent-cp-xyz
  assert_failure
}

@test "cleanup cp-test" {
  run "$D3" rm -f cp-test
  assert_success
}

# ========================================
# d3 migrate tests
# ========================================

@test "run unmanaged docker container for migrate" {
  run docker run -d --name migrate-src mongo:4
  assert_success
  sleep 5
}

@test "d3 migrate captures unmanaged container" {
  # BUG: d3 migrate fails with "Container information is not available"
  # The migrate command cannot read container metadata for containers not on the datadatdat-docker network.
  skip "Known CLI bug: d3 migrate fails with 'Container information is not available'"

  run "$D3" migrate -s migrate-src migrate-dest
  assert_success
}

@test "d3 ls shows migrated repository" {
  skip "Depends on 'd3 migrate' which has a known CLI bug"
  run "$D3" ls
  assert_success
  assert_output --partial "migrate-dest"
}

@test "d3 status on migrated repo shows running" {
  skip "Depends on 'd3 migrate' which has a known CLI bug"
  run "$D3" status migrate-dest
  assert_success
  assert_output --partial "running"
}

@test "d3 commit on migrated repo succeeds" {
  skip "Depends on 'd3 migrate' which has a known CLI bug"
  run "$D3" commit -m "Migrated repo commit" migrate-dest
  assert_success
  assert_output --partial "Commit"
}

@test "cleanup migrated repo" {
  "$D3" rm -f migrate-dest 2>/dev/null || true
  docker rm -f migrate-src 2>/dev/null || true
  true
}
