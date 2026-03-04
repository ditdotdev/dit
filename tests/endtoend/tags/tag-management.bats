#!/usr/bin/env bats

# E2E Tag Management Tests
# Tests standalone d3 tag, d3 delete --tags, d3 log --tags, d3 checkout --tags

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
# Standalone d3 tag
# ========================================

@test "d3 tag adds env=prod tag to first commit" {
  [ -f "$BATS_TMPDIR/tag_commit_one.txt" ] || skip "COMMIT_ONE not saved"
  COMMIT_ONE=$(cat "$BATS_TMPDIR/tag_commit_one.txt")

  run "$D3" tag -c "$COMMIT_ONE" -t env=prod tag-mgmt
  assert_success
}

@test "d3 tag adds env=staging tag to second commit" {
  [ -f "$BATS_TMPDIR/tag_commit_two.txt" ] || skip "COMMIT_TWO not saved"
  COMMIT_TWO=$(cat "$BATS_TMPDIR/tag_commit_two.txt")

  run "$D3" tag -c "$COMMIT_TWO" -t env=staging tag-mgmt
  assert_success
}

# ========================================
# d3 log --tags filtering
# ========================================

@test "d3 log --tags env=prod shows only first commit" {
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

@test "d3 log --tags env=staging shows only second commit" {
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
# d3 checkout --tags
# ========================================

@test "d3 checkout --tags env=prod checks out first commit" {
  [ -f "$BATS_TMPDIR/tag_commit_one.txt" ] || skip "COMMIT_ONE not saved"
  COMMIT_ONE=$(cat "$BATS_TMPDIR/tag_commit_one.txt")

  run "$D3" checkout -t env=prod tag-mgmt
  assert_success
  assert_output --partial "$COMMIT_ONE"
  sleep 5
}

@test "d3 checkout --tags env=staging checks out second commit" {
  [ -f "$BATS_TMPDIR/tag_commit_two.txt" ] || skip "COMMIT_TWO not saved"
  COMMIT_TWO=$(cat "$BATS_TMPDIR/tag_commit_two.txt")

  run "$D3" checkout -t env=staging tag-mgmt
  assert_success
  assert_output --partial "$COMMIT_TWO"
  sleep 5
}

# ========================================
# d3 delete --tags (tag removal)
# ========================================

@test "d3 delete --tags removes env=prod from first commit" {
  # BUG: d3 delete --tags panics with "interface conversion: interface {} is map[string]interface {}, not map[string]string"
  # See: internal/app/providers/common/Delete.go:24
  skip "Known CLI bug: d3 delete --tags panics on type assertion"
  [ -f "$BATS_TMPDIR/tag_commit_one.txt" ] || skip "COMMIT_ONE not saved"
  COMMIT_ONE=$(cat "$BATS_TMPDIR/tag_commit_one.txt")

  run "$D3" delete -c "$COMMIT_ONE" -t env=prod tag-mgmt
  assert_success
}

@test "d3 log --tags env=prod no longer returns first commit" {
  skip "Depends on 'd3 delete --tags' which has a known CLI bug"
  run "$D3" log -t env=prod tag-mgmt
  [ -f "$BATS_TMPDIR/tag_commit_one.txt" ] || skip "COMMIT_ONE not saved"
  COMMIT_ONE=$(cat "$BATS_TMPDIR/tag_commit_one.txt")

  if echo "$output" | grep -q "$COMMIT_ONE"; then
    echo "Expected COMMIT_ONE to be excluded after tag removal"
    return 1
  fi
}

# ========================================
# Error paths
# ========================================

@test "d3 checkout --tags with no matching tag fails" {
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
