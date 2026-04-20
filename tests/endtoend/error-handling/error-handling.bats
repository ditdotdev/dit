#!/usr/bin/env bats

# E2E Error Handling Tests
# Verifies d3 CLI returns non-zero exit codes for invalid operations

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

@test "d3 rm on non-existent repo fails" {
  run "$D3" rm nonexistent-repo-xyz
  assert_failure
}

@test "d3 checkout on non-existent repo fails" {
  run "$D3" checkout --commit aaaa1111bbbb2222 nonexistent-repo-xyz
  assert_failure
}

# ========================================
# Commands that correctly return non-zero for non-existent repos
# ========================================

@test "d3 log on non-existent repo fails" {
  run "$D3" log nonexistent-repo-xyz
  assert_failure
  assert_output --partial "Error"
}

@test "d3 stop on non-existent repo fails" {
  run "$D3" stop nonexistent-repo-xyz
  assert_failure
  assert_output --partial "Error"
}

@test "d3 start on non-existent repo fails" {
  run "$D3" start nonexistent-repo-xyz
  assert_failure
  assert_output --partial "Error"
}

@test "d3 status on non-existent repo fails" {
  run "$D3" status nonexistent-repo-xyz
  assert_failure
  assert_output --partial "Error"
}

@test "d3 commit on non-existent repo fails" {
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

@test "d3 clone from non-existent s3 remote fails loudly" {
  run "$D3" clone -n errortest-s3-dup s3://nonexistent-bucket-xyz-9999/notapath
  assert_failure
  refute_output ""
}

@test "d3 clone from non-existent s3web remote fails loudly" {
  run "$D3" clone -n errortest-s3web-dup s3web://nonexistent-bucket-xyz-9999.s3-website-us-west-2.amazonaws.com/notapath
  assert_failure
  refute_output ""
}

@test "d3 clone with duplicate repo name fails loudly (regression #103)" {
  # 'errortest' was already created by an earlier test in this file; cloning
  # into the same name forces CreateRepository to fail on the d3 server. Pre-fix,
  # d3 clone silently exited 0 with no output; post-fix it must exit non-zero
  # AND print a non-empty error message.
  run "$D3" clone -n errortest s3web://demo-datadatdat.s3-website-us-west-2.amazonaws.com/hello-world/postgres
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
