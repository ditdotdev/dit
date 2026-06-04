#!/usr/bin/env bats

# E2E Push/Pull Error Handling Tests
# Tests that dit push and dit pull give helpful error messages
# instead of panicking when remote is not configured.

# Load shared test helpers
load '../../test_helper'
load 'env'

REPO_NAME="errortest"

# Cleanup after all tests
teardown_file() {
  "$D3" rm -f "$REPO_NAME" 2>/dev/null || true
}

# ========================================
# Setup: create a repo with a commit but no remote
# ========================================

@test "push-pull-errors: run postgres container" {
  "$D3" rm -f "$REPO_NAME" 2>/dev/null || true
  run "$D3" run -n "$REPO_NAME" -e POSTGRES_PASSWORD=postgres postgres
  assert_success
  assert_output --partial "Running controlled container $REPO_NAME"
}

@test "push-pull-errors: create a commit" {
  run "$D3" commit -m "Error test commit" "$REPO_NAME"
  assert_success
  assert_output --partial "Commit"
}

# ========================================
# Push without remote
# ========================================

@test "push-pull-errors: push without remote gives error not panic" {
  run "$D3" push "$REPO_NAME"
  assert_failure
  # Must not contain a Go panic
  if echo "$output" | grep -q "panic:"; then
    echo "FAIL: push panicked instead of giving a friendly error"
    echo "Output: $output"
    return 1
  fi
  # Should mention remote is not configured
  assert_output --partial "remote"
}

# ========================================
# Pull without remote
# ========================================

@test "push-pull-errors: pull without remote gives error not panic" {
  run "$D3" pull "$REPO_NAME"
  assert_failure
  # Must not contain a Go panic
  if echo "$output" | grep -q "panic:"; then
    echo "FAIL: pull panicked instead of giving a friendly error"
    echo "Output: $output"
    return 1
  fi
  # Should mention remote is not configured
  assert_output --partial "remote"
}

# ========================================
# Push/pull with non-existent remote name
# ========================================

@test "push-pull-errors: push with non-existent remote name gives error" {
  run "$D3" push -r nonexistent "$REPO_NAME"
  assert_failure
  if echo "$output" | grep -q "panic:"; then
    echo "FAIL: push panicked instead of giving a friendly error"
    echo "Output: $output"
    return 1
  fi
  assert_output --partial "remote"
}

@test "push-pull-errors: pull with non-existent remote name gives error" {
  run "$D3" pull -r nonexistent "$REPO_NAME"
  assert_failure
  if echo "$output" | grep -q "panic:"; then
    echo "FAIL: pull panicked instead of giving a friendly error"
    echo "Output: $output"
    return 1
  fi
  assert_output --partial "remote"
}

# ========================================
# Cleanup
# ========================================

@test "push-pull-errors: cleanup" {
  run "$D3" rm -f "$REPO_NAME"
  assert_success
}
