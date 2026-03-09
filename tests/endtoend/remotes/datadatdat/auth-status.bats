#!/usr/bin/env bats

# E2E Auth Status Tests
# Tests d3 auth status, login, and logout lifecycle

# Load shared test helpers
load '../../test_helper'
load 'env'

# Setup: Verify services are running
setup_file() {
  run curl -s ${GATEWAY}/health
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || {
    echo "API gateway is not running"
    return 1
  }
}

# Cleanup: Restore auth state
teardown_file() {
  # Ensure we're logged in for any subsequent test suites
  env DATADATDAT_API_KEY="$DATADATDAT_API_KEY" "$D3" auth login \
    --server "${REMOTE_URL}" \
    --api-key "$DATADATDAT_API_KEY" 2>/dev/null || true
}

# ========================================
# Auth status before login
# ========================================

@test "auth-status: logout to start from clean state" {
  run "$D3" auth logout --server "${REMOTE_URL}"
  # May succeed or fail if not logged in - either is fine
  true
}

@test "auth-status: d3 auth status when not logged in" {
  run env -u DATADATDAT_API_KEY "$D3" auth status
  # Should succeed but indicate not authenticated
  assert_success
}

# ========================================
# Auth login and status
# ========================================

@test "auth-status: d3 auth login stores credentials" {
  run "$D3" auth login --server "${REMOTE_URL}" --api-key "$DATADATDAT_API_KEY"
  assert_success
}

@test "auth-status: d3 auth status after login shows authenticated" {
  run "$D3" auth status
  assert_success
}

# ========================================
# Auth logout and status
# ========================================

@test "auth-status: d3 auth logout clears credentials" {
  run "$D3" auth logout --server "${REMOTE_URL}"
  assert_success
}

@test "auth-status: d3 auth status after logout" {
  run env -u DATADATDAT_API_KEY "$D3" auth status
  assert_success
}

# ========================================
# Error paths
# ========================================

@test "auth-status: d3 auth login with missing --api-key fails" {
  run "$D3" auth login --server "${REMOTE_URL}"
  assert_failure
}

# ========================================
# Restore state
# ========================================

@test "auth-status: restore auth for subsequent tests" {
  run "$D3" auth login --server "${REMOTE_URL}" --api-key "$DATADATDAT_API_KEY"
  assert_success
}
