#!/usr/bin/env bats

# E2E Tests: Whitelist Approval Flow
# Replicates bug where admin-approved users still see "Access Request Pending"
# because CreateWhitelistEntry inserts user_id but NOT github_login,
# while IsGitHubLoginWhitelisted queries WHERE github_login = $1.
#
# Related: https://github.com/datadatdat/datadatdat-remote-server/issues/450

# Load shared test helpers
load '../../test_helper'

# Load environment configuration (DEV by default, ENV=PROD for production)
load 'env'

# Test user constants
APPROVAL_TEST_ADMIN_GH_ID=200001
APPROVAL_TEST_ADMIN_LOGIN="approval-testadmin"
APPROVAL_TEST_USER_GH_ID=200002
APPROVAL_TEST_USER_LOGIN="approval-testuser"

# Setup: Verify services are running
setup_file() {
  run curl -s "${GATEWAY}/health"
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || {
    echo "Gateway is not running"
    return 1
  }

  if is_dev; then
    run docker exec datadatdat-postgres pg_isready -U datadatdat
    [[ "$output" == *"accepting connections"* ]] || {
      echo "Postgres is not ready"
      return 1
    }
  fi
}

# Cleanup after all tests
teardown_file() {
  run_sql_cmd "DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('${APPROVAL_TEST_ADMIN_LOGIN}', '${APPROVAL_TEST_USER_LOGIN}')); DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('${APPROVAL_TEST_ADMIN_LOGIN}', '${APPROVAL_TEST_USER_LOGIN}')); DELETE FROM whitelisted_users WHERE github_login IN ('${APPROVAL_TEST_ADMIN_LOGIN}', '${APPROVAL_TEST_USER_LOGIN}'); DELETE FROM access_requests WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('${APPROVAL_TEST_ADMIN_LOGIN}', '${APPROVAL_TEST_USER_LOGIN}')); DELETE FROM audit_log WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('${APPROVAL_TEST_ADMIN_LOGIN}', '${APPROVAL_TEST_USER_LOGIN}')); DELETE FROM users WHERE github_login IN ('${APPROVAL_TEST_ADMIN_LOGIN}', '${APPROVAL_TEST_USER_LOGIN}');" 2>/dev/null || true
}

# ========================================
# Setup: Create test users
# ========================================

@test "whitelist-approval: cleanup previous test data" {
  run run_sql_cmd "DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('${APPROVAL_TEST_ADMIN_LOGIN}', '${APPROVAL_TEST_USER_LOGIN}')); DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('${APPROVAL_TEST_ADMIN_LOGIN}', '${APPROVAL_TEST_USER_LOGIN}')); DELETE FROM whitelisted_users WHERE github_login IN ('${APPROVAL_TEST_ADMIN_LOGIN}', '${APPROVAL_TEST_USER_LOGIN}'); DELETE FROM access_requests WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('${APPROVAL_TEST_ADMIN_LOGIN}', '${APPROVAL_TEST_USER_LOGIN}')); DELETE FROM audit_log WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('${APPROVAL_TEST_ADMIN_LOGIN}', '${APPROVAL_TEST_USER_LOGIN}')); DELETE FROM users WHERE github_login IN ('${APPROVAL_TEST_ADMIN_LOGIN}', '${APPROVAL_TEST_USER_LOGIN}');"
  assert_success
}

@test "whitelist-approval: create admin user" {
  run run_sql_cmd "INSERT INTO users (github_id, github_login, github_email, github_name, is_whitelisted, is_admin, created_at, updated_at) VALUES (${APPROVAL_TEST_ADMIN_GH_ID}, '${APPROVAL_TEST_ADMIN_LOGIN}', 'approval-admin@test.com', 'Approval Test Admin', true, true, NOW(), NOW()) ON CONFLICT (github_id) DO UPDATE SET is_whitelisted = true, is_admin = true;"
  assert_success
}

@test "whitelist-approval: create non-whitelisted user (simulates first OAuth login)" {
  run run_sql_cmd "INSERT INTO users (github_id, github_login, github_email, github_name, is_whitelisted, is_admin, created_at, updated_at) VALUES (${APPROVAL_TEST_USER_GH_ID}, '${APPROVAL_TEST_USER_LOGIN}', 'approval-user@test.com', 'Approval Test User', false, false, NOW(), NOW()) ON CONFLICT (github_id) DO UPDATE SET is_whitelisted = false, is_admin = false;"
  assert_success
}

# ========================================
# Verify: User is NOT whitelisted initially
# ========================================

@test "whitelist-approval: user is not in whitelisted_users table" {
  run run_sql_raw "SELECT COUNT(*) FROM whitelisted_users WHERE github_login = '${APPROVAL_TEST_USER_LOGIN}';"
  assert_success
  assert_output "0"
}

@test "whitelist-approval: user is_whitelisted is false" {
  run run_sql_raw "SELECT is_whitelisted FROM users WHERE github_login = '${APPROVAL_TEST_USER_LOGIN}';"
  assert_success
  assert_output "f"
}

# ========================================
# Simulate: User creates access request (happens automatically on OAuth)
# ========================================

@test "whitelist-approval: create pending access request for user" {
  run run_sql_cmd "INSERT INTO access_requests (user_id, reason, status, created_at) VALUES ((SELECT id FROM users WHERE github_login = '${APPROVAL_TEST_USER_LOGIN}'), 'Auto-generated access request', 'pending', NOW());"
  assert_success
}

@test "whitelist-approval: verify pending access request exists" {
  run run_sql_raw "SELECT COUNT(*) FROM access_requests ar JOIN users u ON ar.user_id = u.id WHERE u.github_login = '${APPROVAL_TEST_USER_LOGIN}' AND ar.status = 'pending';"
  assert_success
  assert_output "1"
}

@test "whitelist-approval: capture access request ID" {
  run run_sql_raw "SELECT ar.id FROM access_requests ar JOIN users u ON ar.user_id = u.id WHERE u.github_login = '${APPROVAL_TEST_USER_LOGIN}' AND ar.status = 'pending' LIMIT 1;"
  assert_success
  ACCESS_REQUEST_ID=$(echo "$output" | tr -d '[:space:]')
  echo "$ACCESS_REQUEST_ID" > "$BATS_TMPDIR/whitelist_approval_request_id.txt"
  [[ -n "$ACCESS_REQUEST_ID" ]]
}

# ========================================
# Admin approves the access request via API
# ========================================

@test "whitelist-approval: admin approves access request" {
  is_dev || skip "Admin approve API only available in DEV"
  ACCESS_REQUEST_ID=$(cat "$BATS_TMPDIR/whitelist_approval_request_id.txt")

  run curl -s -w "\n%{http_code}" -X POST \
    -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"requestId\":\"${ACCESS_REQUEST_ID}\",\"comment\":\"Approving for E2E whitelist test\"}" \
    "${WEB_UI}/api/admin/access-requests/approve"

  assert_success
  assert_output --partial "200"
  assert_output --partial "approved"
}

@test "whitelist-approval: verify access request status is approved" {
  ACCESS_REQUEST_ID=$(cat "$BATS_TMPDIR/whitelist_approval_request_id.txt")
  run run_sql_raw "SELECT status FROM access_requests WHERE id = '${ACCESS_REQUEST_ID}';"
  assert_success
  assert_output "approved"
}

# ========================================
# THE BUG: After approval, whitelisted_users row is missing github_login
# ========================================

@test "whitelist-approval: verify whitelisted_users entry was created" {
  run run_sql_raw "SELECT COUNT(*) FROM whitelisted_users wu JOIN users u ON wu.user_id = u.id WHERE u.github_login = '${APPROVAL_TEST_USER_LOGIN}';"
  assert_success
  assert_output "1"
}

@test "whitelist-approval: verify users.is_whitelisted was set to true" {
  run run_sql_raw "SELECT is_whitelisted FROM users WHERE github_login = '${APPROVAL_TEST_USER_LOGIN}';"
  assert_success
  assert_output "t"
}

@test "whitelist-approval: BUG - whitelisted_users.github_login should be set but is NULL" {
  # This is the core bug: CreateWhitelistEntry inserts (user_id, approved_by, reason)
  # but does NOT set github_login. The OAuth callback checks
  # IsGitHubLoginWhitelisted which queries WHERE github_login = $1.
  # Since github_login is NULL, the lookup fails and user sees "Access Pending".
  run run_sql_raw "SELECT COALESCE(github_login, 'NULL') FROM whitelisted_users WHERE user_id = (SELECT id FROM users WHERE github_login = '${APPROVAL_TEST_USER_LOGIN}');"
  assert_success

  # This test EXPOSES the bug: github_login is NULL when it should be the user's login.
  # When the bug is fixed, this assertion should pass.
  # Currently it will output 'NULL' proving the bug exists.
  assert_output "${APPROVAL_TEST_USER_LOGIN}"
}

@test "whitelist-approval: BUG - IsGitHubLoginWhitelisted returns false for approved user" {
  # This simulates exactly what HandleCallback does on re-login:
  # SELECT EXISTS(SELECT 1 FROM whitelisted_users WHERE github_login = $1)
  # Because github_login is NULL in the row, this returns false.
  run run_sql_raw "SELECT EXISTS(SELECT 1 FROM whitelisted_users WHERE github_login = '${APPROVAL_TEST_USER_LOGIN}');"
  assert_success

  # This should be 't' (true) after approval, but the bug causes it to be 'f' (false).
  # When the bug is fixed, this assertion should pass.
  assert_output "t"
}
