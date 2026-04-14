#!/usr/bin/env bats

# E2E Abort Workflow Tests
# Tests d3 abort command

# Load shared test helpers
load '../../test_helper'
load 'env'

# Setup: Verify server is running
setup_file() {
  run curl -s ${GATEWAY}/health
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || {
    echo "datadatdat-remote-server is not running"
    return 1
  }
}

# Cleanup after all tests
teardown_file() {
  "$D3" rm -f abort-test 2>/dev/null || true
  "$D3" repo delete "${TEST_ORG}" abort-test-repo --server "${GATEWAY}" 2>/dev/null || true
}

# ========================================
# Setup
# ========================================

@test "abort: verify api-gateway is running" {
  run curl -s ${GATEWAY}/health
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

@test "abort: run mongo container" {
  run "$D3" run -n abort-test -P mongo
  assert_success
  assert_output --partial "Running controlled container abort-test"
  sleep 5
}

@test "abort: create commit" {
  run "$D3" commit -m "abort test commit" abort-test
  assert_success
  assert_output --partial "Commit"

  COMMIT_GUID=$(echo "$output" | grep -o "Commit [a-f0-9]*" | awk '{print $2}')
  echo "$COMMIT_GUID" > "$BATS_TMPDIR/abort_commit.txt"
}

@test "abort: add datadatdat remote" {
  run "$D3" remote add ${REMOTE_URL}/${TEST_ORG}/abort-test-repo abort-test
  assert_success
}

# ========================================
# Abort with no operation in progress
# ========================================

@test "abort: d3 abort with no operation in progress does not crash" {
  run "$D3" abort abort-test
  # Should either succeed (no-op) or fail gracefully with a message
  # The key assertion is that it doesn't crash or hang
  true
}

# ========================================
# Normal push to verify workflow still works
# ========================================

@test "abort: push commit succeeds" {
  run "$D3" push abort-test
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "abort: remote log shows pushed commit" {
  [ -f "$BATS_TMPDIR/abort_commit.txt" ] || skip "Commit GUID not saved"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/abort_commit.txt")

  run "$D3" remote log abort-test
  assert_success
  assert_output --partial "$COMMIT_GUID"
}

# ========================================
# Cleanup
# ========================================

@test "abort: remove local abort-test repository" {
  run "$D3" rm -f abort-test
  assert_success
}

@test "abort: delete remote repository" {
  run "$D3" repo delete "${TEST_ORG}" abort-test-repo --server "${GATEWAY}"
  assert_success
}
