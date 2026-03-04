#!/usr/bin/env bats

# E2E Error Handling Tests
# Verifies d3 CLI behavior for invalid operations
#
# NOTE: Several d3 commands return exit 0 even when given non-existent repos.
# This file documents the actual behavior. See CLI deficiency notes below.

# Load shared test helpers
load '../test_helper'

# Cleanup any leftover test containers
teardown_file() {
  "$D3" rm -f errortest 2>/dev/null || true
  "$D3" rm -f errortest-dup 2>/dev/null || true
}

# ========================================
# Commands that correctly return non-zero for non-existent repos
# ========================================

@test "d3 rm on non-existent repo fails" {
  run "$D3" rm nonexistent-repo-xyz
  assert_failure
}

@test "d3 checkout on non-existent repo fails" {
  run "$D3" checkout --commit aaaa1111bbbb2222 nonexistent-repo-xyz
  assert_failure
}

# ========================================
# Commands that return exit 0 for non-existent repos (CLI deficiency)
# These document current behavior - ideally these would return non-zero.
# ========================================

@test "d3 log on non-existent repo returns empty output" {
  # DEFICIENCY: returns exit 0 with empty output
  run "$D3" log nonexistent-repo-xyz
  assert_success
}

@test "d3 stop on non-existent repo prints error but exits 0" {
  # DEFICIENCY: returns exit 0 despite printing error
  run "$D3" stop nonexistent-repo-xyz
  assert_success
  assert_output --partial "Error"
}

@test "d3 start on non-existent repo prints error but exits 0" {
  # DEFICIENCY: returns exit 0 despite printing error
  run "$D3" start nonexistent-repo-xyz
  assert_success
  assert_output --partial "Error"
}

@test "d3 status on non-existent repo returns empty data" {
  # DEFICIENCY: returns exit 0 with just column headers
  run "$D3" status nonexistent-repo-xyz
  assert_success
}

@test "d3 commit on non-existent repo exits 0" {
  # DEFICIENCY: returns exit 0 for non-existent repos
  run "$D3" commit -m "should fail" nonexistent-repo-xyz
  assert_success
}

# ========================================
# Push/pull without remote configured
# ========================================

@test "run container for remote error tests" {
  run "$D3" run -n errortest -P mongo
  assert_success
  sleep 5
}

@test "d3 push on repo with no remote fails" {
  run "$D3" push errortest
  assert_failure
}

@test "d3 pull on repo with no remote fails" {
  run "$D3" pull errortest
  assert_failure
}

# ========================================
# Duplicate operations
# ========================================

@test "d3 run with duplicate name fails" {
  run "$D3" run -n errortest -P mongo
  assert_failure
}

# ========================================
# Invalid remote URI
# ========================================

@test "d3 clone from non-existent remote fails" {
  run "$D3" clone -n errortest-dup s3://nonexistent-bucket-xyz-9999/notapath
  assert_failure
}

# ========================================
# Cleanup
# ========================================

@test "cleanup error test containers" {
  run "$D3" rm -f errortest
  assert_success
}
