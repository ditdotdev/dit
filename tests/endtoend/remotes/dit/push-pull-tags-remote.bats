#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# E2E Push/Pull Tags on Dit Remote Tests
# Tests dit push --tags and dit pull --tags on the dit remote server

# Load shared test helpers
load '../../test_helper'
load 'env'

# Setup: Verify server is running
setup_file() {
  run curl -s ${GATEWAY}/health
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || {
    echo "dit-remote-server is not running"
    return 1
  }
}

# Cleanup after all tests
teardown_file() {
  "$D3" rm -f tagremote 2>/dev/null || true
  DIT_API_KEY="${DIT_API_KEY}" "$D3" repo delete "${TEST_ORG}" tag-remote-test \
    --server "${GATEWAY}" 2>/dev/null || true
}

# ========================================
# Setup
# ========================================

@test "push-pull-tags: run mongo container" {
  run "$D3" run -n tagremote -P mongo
  assert_success
  assert_output --partial "Running controlled container tagremote"
  sleep 5
}

# ========================================
# Create tagged commits
# ========================================

@test "push-pull-tags: create commit with tag env=prod" {
  run "$D3" commit -t env=prod tagremote
  assert_success
  assert_output --partial "Commit"

  COMMIT_PROD=$(echo "$output" | grep -o "Commit [a-f0-9]*" | awk '{print $2}')
  echo "$COMMIT_PROD" > "$BATS_TMPDIR/tag_remote_commit_prod.txt"
}

@test "push-pull-tags: create commit with tag env=staging" {
  run "$D3" commit -t env=staging tagremote
  assert_success
  assert_output --partial "Commit"

  COMMIT_STAGING=$(echo "$output" | grep -o "Commit [a-f0-9]*" | awk '{print $2}')
  echo "$COMMIT_STAGING" > "$BATS_TMPDIR/tag_remote_commit_staging.txt"
}

@test "push-pull-tags: add dit remote" {
  run "$D3" remote add ${REMOTE_URL}/${TEST_ORG}/tag-remote-test tagremote
  assert_success
}

# ========================================
# Tag-filtered push
# ========================================

@test "push-pull-tags: push only env=prod commit" {
  run "$D3" push -t env=prod tagremote
  assert_success
}

@test "push-pull-tags: remote log shows prod commit only" {
  [ -f "$BATS_TMPDIR/tag_remote_commit_prod.txt" ] || skip "COMMIT_PROD not saved"
  [ -f "$BATS_TMPDIR/tag_remote_commit_staging.txt" ] || skip "COMMIT_STAGING not saved"
  COMMIT_PROD=$(cat "$BATS_TMPDIR/tag_remote_commit_prod.txt")
  COMMIT_STAGING=$(cat "$BATS_TMPDIR/tag_remote_commit_staging.txt")

  run "$D3" remote log tagremote
  assert_success
  assert_output --partial "$COMMIT_PROD"

  # Staging commit should NOT be in remote yet
  if echo "$output" | grep -q "$COMMIT_STAGING"; then
    echo "Expected COMMIT_STAGING to be excluded from remote log"
    return 1
  fi
}

@test "push-pull-tags: push env=staging commit" {
  run "$D3" push -t env=staging tagremote
  assert_success
}

@test "push-pull-tags: remote log shows both commits" {
  [ -f "$BATS_TMPDIR/tag_remote_commit_prod.txt" ] || skip "COMMIT_PROD not saved"
  [ -f "$BATS_TMPDIR/tag_remote_commit_staging.txt" ] || skip "COMMIT_STAGING not saved"
  COMMIT_PROD=$(cat "$BATS_TMPDIR/tag_remote_commit_prod.txt")
  COMMIT_STAGING=$(cat "$BATS_TMPDIR/tag_remote_commit_staging.txt")

  run "$D3" remote log tagremote
  assert_success
  assert_output --partial "$COMMIT_PROD"
  assert_output --partial "$COMMIT_STAGING"
}

# ========================================
# Tag-filtered pull
# ========================================

@test "push-pull-tags: delete both local commits" {
  [ -f "$BATS_TMPDIR/tag_remote_commit_prod.txt" ] || skip "COMMIT_PROD not saved"
  [ -f "$BATS_TMPDIR/tag_remote_commit_staging.txt" ] || skip "COMMIT_STAGING not saved"
  COMMIT_PROD=$(cat "$BATS_TMPDIR/tag_remote_commit_prod.txt")
  COMMIT_STAGING=$(cat "$BATS_TMPDIR/tag_remote_commit_staging.txt")

  run "$D3" delete -c "$COMMIT_STAGING" tagremote
  assert_success
  run "$D3" delete -c "$COMMIT_PROD" tagremote
  assert_success
}

@test "push-pull-tags: local log is empty after delete" {
  run "$D3" log tagremote
  assert_success
}

@test "push-pull-tags: pull only env=prod commit" {
  run "$D3" pull -t env=prod tagremote
  assert_success
}

@test "push-pull-tags: local log shows prod commit" {
  [ -f "$BATS_TMPDIR/tag_remote_commit_prod.txt" ] || skip "COMMIT_PROD not saved"
  COMMIT_PROD=$(cat "$BATS_TMPDIR/tag_remote_commit_prod.txt")

  run "$D3" log tagremote
  assert_success
  assert_output --partial "$COMMIT_PROD"
}

@test "push-pull-tags: pull env=staging commit" {
  run "$D3" pull -t env=staging tagremote
  assert_success
}

@test "push-pull-tags: local log shows both commits after pulls" {
  [ -f "$BATS_TMPDIR/tag_remote_commit_prod.txt" ] || skip "COMMIT_PROD not saved"
  [ -f "$BATS_TMPDIR/tag_remote_commit_staging.txt" ] || skip "COMMIT_STAGING not saved"
  COMMIT_PROD=$(cat "$BATS_TMPDIR/tag_remote_commit_prod.txt")
  COMMIT_STAGING=$(cat "$BATS_TMPDIR/tag_remote_commit_staging.txt")

  run "$D3" log tagremote
  assert_success
  assert_output --partial "$COMMIT_PROD"
  assert_output --partial "$COMMIT_STAGING"
}

# ========================================
# Cleanup
# ========================================

@test "push-pull-tags: remove local tagremote repository" {
  run "$D3" rm -f tagremote
  assert_success
}

@test "push-pull-tags: delete remote repository" {
  run env DIT_API_KEY="${DIT_API_KEY}" "$D3" repo delete "${TEST_ORG}" tag-remote-test \
    --server "${GATEWAY}"
  assert_success
}
