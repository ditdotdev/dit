#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# E2E Tag Management Tests
# Tests standalone dit tag, dit delete --tags, dit log --tags, dit checkout --tags

# Load shared test helpers
load '../test_helper'

# Cleanup after all tests
teardown_file() {
  "$D3" rm -f tag-mgmt 2>/dev/null || true
}

# ========================================
# Setup
# ========================================

@test "run postgres for tag management test" {
  run "$D3" run -n tag-mgmt -P postgres
  assert_success
  assert_output --partial "Running controlled container tag-mgmt"
}

@test "wait for tag-mgmt postgres to be ready" {
  run bash -c 'for i in {1..18}; do docker exec tag-mgmt pg_isready && break || sleep 5; done'
  assert_success
}

# ========================================
# Create commits without tags
# ========================================

@test "create first commit without tags" {
  run "$D3" commit -m "baseline commit" tag-mgmt
  assert_success
  assert_output --partial "Commit"

  COMMIT_ONE=$(echo "$output" | grep -o "Commit [a-f0-9]*" | awk '{print $2}')
  echo "$COMMIT_ONE" > "$BATS_TMPDIR/tag_commit_one.txt"
}

@test "create second commit without tags" {
  run "$D3" commit -m "second commit" tag-mgmt
  assert_success
  assert_output --partial "Commit"

  COMMIT_TWO=$(echo "$output" | grep -o "Commit [a-f0-9]*" | awk '{print $2}')
  echo "$COMMIT_TWO" > "$BATS_TMPDIR/tag_commit_two.txt"
}

# ========================================
# Standalone dit tag
# ========================================

@test "dit tag adds env=prod tag to first commit" {
  [ -f "$BATS_TMPDIR/tag_commit_one.txt" ] || skip "COMMIT_ONE not saved"
  COMMIT_ONE=$(cat "$BATS_TMPDIR/tag_commit_one.txt")

  run "$D3" tag -c "$COMMIT_ONE" -t env=prod tag-mgmt
  assert_success
}

@test "dit tag adds env=staging tag to second commit" {
  [ -f "$BATS_TMPDIR/tag_commit_two.txt" ] || skip "COMMIT_TWO not saved"
  COMMIT_TWO=$(cat "$BATS_TMPDIR/tag_commit_two.txt")

  run "$D3" tag -c "$COMMIT_TWO" -t env=staging tag-mgmt
  assert_success
}

# ========================================
# dit log --tags filtering
# ========================================

@test "dit log --tags env=prod shows only first commit" {
  [ -f "$BATS_TMPDIR/tag_commit_one.txt" ] || skip "COMMIT_ONE not saved"
  [ -f "$BATS_TMPDIR/tag_commit_two.txt" ] || skip "COMMIT_TWO not saved"
  COMMIT_ONE=$(cat "$BATS_TMPDIR/tag_commit_one.txt")
  COMMIT_TWO=$(cat "$BATS_TMPDIR/tag_commit_two.txt")

  run "$D3" log -t env=prod tag-mgmt
  assert_success
  assert_output --partial "$COMMIT_ONE"

  # Should NOT contain COMMIT_TWO
  if echo "$output" | grep -q "$COMMIT_TWO"; then
    echo "Expected COMMIT_TWO to be excluded from tag-filtered log"
    return 1
  fi
}

@test "dit log --tags env=staging shows only second commit" {
  [ -f "$BATS_TMPDIR/tag_commit_one.txt" ] || skip "COMMIT_ONE not saved"
  [ -f "$BATS_TMPDIR/tag_commit_two.txt" ] || skip "COMMIT_TWO not saved"
  COMMIT_ONE=$(cat "$BATS_TMPDIR/tag_commit_one.txt")
  COMMIT_TWO=$(cat "$BATS_TMPDIR/tag_commit_two.txt")

  run "$D3" log -t env=staging tag-mgmt
  assert_success
  assert_output --partial "$COMMIT_TWO"

  # Should NOT contain COMMIT_ONE
  if echo "$output" | grep -q "$COMMIT_ONE"; then
    echo "Expected COMMIT_ONE to be excluded from tag-filtered log"
    return 1
  fi
}

# ========================================
# dit checkout --tags
# ========================================

@test "dit checkout --tags env=prod checks out first commit" {
  [ -f "$BATS_TMPDIR/tag_commit_one.txt" ] || skip "COMMIT_ONE not saved"
  COMMIT_ONE=$(cat "$BATS_TMPDIR/tag_commit_one.txt")

  run "$D3" checkout -t env=prod tag-mgmt
  assert_success
  assert_output --partial "$COMMIT_ONE"
  sleep 5
}

@test "dit checkout --tags env=staging checks out second commit" {
  [ -f "$BATS_TMPDIR/tag_commit_two.txt" ] || skip "COMMIT_TWO not saved"
  COMMIT_TWO=$(cat "$BATS_TMPDIR/tag_commit_two.txt")

  run "$D3" checkout -t env=staging tag-mgmt
  assert_success
  assert_output --partial "$COMMIT_TWO"
  sleep 5
}

# ========================================
# dit delete --tags (tag removal)
# ========================================

@test "dit delete --tags removes env=prod from first commit" {
  [ -f "$BATS_TMPDIR/tag_commit_one.txt" ] || skip "COMMIT_ONE not saved"
  COMMIT_ONE=$(cat "$BATS_TMPDIR/tag_commit_one.txt")

  run "$D3" delete -c "$COMMIT_ONE" -t env=prod tag-mgmt
  assert_success
}

@test "dit log --tags env=prod no longer returns first commit" {
  [ -f "$BATS_TMPDIR/tag_commit_one.txt" ] || skip "COMMIT_ONE not saved"
  run "$D3" log -t env=prod tag-mgmt
  COMMIT_ONE=$(cat "$BATS_TMPDIR/tag_commit_one.txt")

  if echo "$output" | grep -q "$COMMIT_ONE"; then
    echo "Expected COMMIT_ONE to be excluded after tag removal"
    return 1
  fi
}

# ========================================
# Error paths
# ========================================

@test "dit checkout --tags with no matching tag fails" {
  run "$D3" checkout -t env=nonexistent tag-mgmt
  assert_failure
}

# ========================================
# Cleanup
# ========================================

@test "cleanup tag-mgmt" {
  run "$D3" rm -f tag-mgmt
  assert_success
}
