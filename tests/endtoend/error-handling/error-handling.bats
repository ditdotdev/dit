#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# E2E Error Handling Tests
# Verifies dit CLI returns non-zero exit codes for invalid operations

# Load shared test helpers
load '../test_helper'

# Cleanup any leftover test containers
teardown_file() {
  "$D3" rm -f errortest 2>/dev/null || true
  "$D3" rm -f errortest-dup 2>/dev/null || true
  "$D3" rm -f errortest-s3-dup 2>/dev/null || true
  "$D3" rm -f errortest-s3web-dup 2>/dev/null || true
  "$D3" rm -f dup-clone-target 2>/dev/null || true
}

# ========================================
# Commands that correctly return non-zero for non-existent repos
# ========================================

@test "dit rm on non-existent repo fails" {
  run "$D3" rm nonexistent-repo-xyz
  assert_failure
}

@test "dit checkout on non-existent repo fails" {
  run "$D3" checkout --commit aaaa1111bbbb2222 nonexistent-repo-xyz
  assert_failure
}

# ========================================
# Commands that correctly return non-zero for non-existent repos
# ========================================

@test "dit log on non-existent repo fails" {
  run "$D3" log nonexistent-repo-xyz
  assert_failure
  assert_output --partial "Error"
}

@test "dit stop on non-existent repo fails" {
  run "$D3" stop nonexistent-repo-xyz
  assert_failure
  assert_output --partial "Error"
}

@test "dit start on non-existent repo fails" {
  run "$D3" start nonexistent-repo-xyz
  assert_failure
  assert_output --partial "Error"
}

@test "dit status on non-existent repo fails" {
  run "$D3" status nonexistent-repo-xyz
  assert_failure
  assert_output --partial "Error"
}

@test "dit commit on non-existent repo fails" {
  run "$D3" commit -m "should fail" nonexistent-repo-xyz
  assert_failure
  assert_output --partial "Error"
}

# ========================================
# Push/pull without remote configured
# ========================================

@test "run container for remote error tests" {
  run "$D3" run -n errortest -P mongo
  assert_success
  sleep 5
}

@test "dit push on repo with no remote fails" {
  run "$D3" push errortest
  assert_failure
}

@test "dit pull on repo with no remote fails" {
  run "$D3" pull errortest
  assert_failure
}

# ========================================
# Duplicate operations
# ========================================

@test "dit run with duplicate name fails" {
  run "$D3" run -n errortest -P mongo
  assert_failure
}

# ========================================
# Invalid remote URI
# ========================================

@test "dit clone from non-existent s3 remote fails loudly" {
  run "$D3" clone -n errortest-s3-dup s3://nonexistent-bucket-xyz-9999/notapath
  assert_failure
  refute_output ""
}

@test "dit clone from non-existent s3web remote fails loudly" {
  run "$D3" clone -n errortest-s3web-dup s3web://nonexistent-bucket-xyz-9999.s3-website-us-west-2.amazonaws.com/notapath
  assert_failure
  refute_output ""
}

@test "dit clone with duplicate repo name fails loudly (regression #103)" {
  # 'errortest' was already created by an earlier test in this file; cloning
  # into the same name forces CreateRepository to fail on the dit server. Pre-fix,
  # dit clone silently exited 0 with no output; post-fix it must exit non-zero
  # AND print a non-empty error message.
  run "$D3" clone -n errortest s3web://demo-dit.s3-website-us-west-2.amazonaws.com/hello-world/postgres
  assert_failure
  refute_output ""
}

# ========================================
# Cleanup
# ========================================

@test "cleanup error test containers" {
  run "$D3" rm -f errortest
  assert_success
}
