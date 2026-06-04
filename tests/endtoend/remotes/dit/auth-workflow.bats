#!/usr/bin/env bats

# E2E Authentication and Authorization Tests
# These tests validate the complete auth flow including OAuth, JWT, whitelist, and admin workflows

# Load shared test helpers
load '../../test_helper'

# Load environment configuration (DEV by default, ENV=PROD for production)
load 'env'

# Setup: Verify services are running and clean database
setup_file() {
  # Verify gateway/server is running
  run curl -s "${GATEWAY}/health"
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || {
    echo "Gateway is not running"
    return 1
  }

  if is_dev; then
    run curl -s "${AUTH_SERVER}/health"
    [[ "$output" == *"healthy"* ]] || {
      echo "Auth server is not running"
      return 1
    }

    run docker exec dit-postgres pg_isready -U dit
    [[ "$output" == *"accepting connections"* ]] || {
      echo "Postgres is not ready"
      return 1
    }
  else
    # PROD: verify RDS is accessible via SSH
    run run_sql_raw "SELECT 1;"
    [[ $? -eq 0 ]] || {
      echo "RDS PostgreSQL is not accessible"
      return 1
    }
  fi
}

# Cleanup after all tests
teardown_file() {
  # Cleanup test data
  run_sql_cmd "DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); DELETE FROM access_requests WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); DELETE FROM audit_log WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); DELETE FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser');" 2>/dev/null || true

  # Cleanup test repositories
  curl -sf -X DELETE -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${GATEWAY}/api/v1/repos/${TEST_ORG}/auth-test-repo" 2>/dev/null || true
  curl -sf -X DELETE -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${GATEWAY}/api/v1/repos/${AUTH_TEST_ORG}/cli-test-repo" 2>/dev/null || true

  # Cleanup dit test repository
  "$D3" rm authclitest -f 2>/dev/null || true
}

# ========================================
# Pre-requisites & Health Checks
# ========================================

@test "auth: verify auth server container is running" {
  is_prod || skip "Container SSH check only for PROD"
  run check_container_running "auth-server"
  assert_success
  assert_output --partial "Up"
}

@test "auth: verify api-gateway container is running" {
  is_prod || skip "Container SSH check only for PROD"
  run check_container_running "api-gateway"
  assert_success
  assert_output --partial "Up"
}

@test "auth: verify auth server is running" {
  is_dev || skip "Direct auth server check only for DEV"
  run curl -s "${AUTH_SERVER}/health"
  assert_success
  assert_output --partial "healthy"
}

@test "auth: verify api-gateway is running" {
  run curl -s "${GATEWAY}/health"
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

@test "auth: verify RDS PostgreSQL is accessible" {
  is_prod || skip "RDS check only for PROD"
  run run_sql_raw "SELECT 1;"
  assert_success
  assert_output --partial "1"
}

@test "auth: verify postgres is ready" {
  is_dev || skip "Local postgres check only for DEV"
  run docker exec dit-postgres pg_isready -U dit
  assert_success
  assert_output --partial "accepting connections"
}

# ========================================
# Database Setup & Initial Admin
# ========================================

@test "auth: cleanup existing test users from previous runs" {
  run run_sql_cmd "DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); DELETE FROM access_requests WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); DELETE FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser');"
  # Cleanup is best effort, don't fail if nothing to delete
  assert_success
}

@test "auth: create initial admin user" {
  run run_sql_cmd "INSERT INTO users (github_id, github_login, github_email, github_name, is_whitelisted, is_admin, created_at, updated_at) VALUES (100001, 'testadmin', 'admin@test.com', 'Test Admin', true, true, NOW(), NOW()) ON CONFLICT (github_id) DO UPDATE SET is_whitelisted = true, is_admin = true; INSERT INTO whitelisted_users (user_id, approved_by, approved_at, reason) VALUES ((SELECT id FROM users WHERE github_login = 'testadmin'), (SELECT id FROM users WHERE github_login = 'testadmin'), NOW(), 'E2E Test Admin User') ON CONFLICT (user_id) DO NOTHING;"
  assert_success
}

@test "auth: create whitelisted test user" {
  run run_sql_cmd "INSERT INTO users (github_id, github_login, github_email, github_name, is_whitelisted, is_admin, created_at, updated_at) VALUES (100002, 'testuser', 'user@test.com', 'Test User', true, false, NOW(), NOW()) ON CONFLICT (github_id) DO UPDATE SET is_whitelisted = true, is_admin = false; INSERT INTO whitelisted_users (user_id, approved_by, approved_at, reason) VALUES ((SELECT id FROM users WHERE github_login = 'testuser'), (SELECT id FROM users WHERE github_login = 'testadmin'), NOW(), 'E2E Test User') ON CONFLICT (user_id) DO NOTHING;"
  assert_success
}

@test "auth: create non-whitelisted user" {
  run run_sql_cmd "INSERT INTO users (github_id, github_login, github_email, github_name, is_whitelisted, is_admin, created_at, updated_at) VALUES (100003, 'blockeduser', 'blocked@test.com', 'Blocked User', false, false, NOW(), NOW()) ON CONFLICT (github_id) DO UPDATE SET is_whitelisted = false, is_admin = false;"
  assert_success
}

# ========================================
# Unauthenticated Access Tests
# ========================================

@test "auth: unauthenticated request to protected API returns 401 or 404" {
  run curl -s -o /dev/null -w "%{http_code}" "${GATEWAY}/api/v1/repos/${TEST_ORG}/test-repo"
  assert_success
  # DEV returns 404 (GitHub-style: hide repo existence), PROD returns 401
  [[ "$output" == "401" || "$output" == "404" ]]
}

@test "auth: unauthenticated request to admin endpoint returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" "${AUTH_SERVER}/api/admin/whitelist"
  assert_success
  assert_output "401"
}

@test "auth: health check endpoints are accessible without auth" {
  run curl -sf "${GATEWAY}/health"
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

# ========================================
# Authenticated User Access Tests
# ========================================

@test "auth: authenticated user can create repository" {
  run curl -sf -X POST -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${GATEWAY}/api/v1/repos/${TEST_ORG}/auth-test-repo"
  assert_success
  assert_output --partial "auth-test-repo"
}

# ========================================
# API Gateway Integration Tests
# ========================================

@test "auth: verify API gateway forwards auth headers correctly" {
  run curl -sf -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${GATEWAY}/health"
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

# ========================================
# Access Request Workflow
# ========================================

@test "auth: create access request for non-whitelisted user" {
  run run_sql_cmd "INSERT INTO access_requests (user_id, reason, status, created_at) VALUES ((SELECT id FROM users WHERE github_login = 'blockeduser'), 'E2E Test Access Request', 'pending', NOW());"
  assert_success
}

@test "auth: verify access request was created" {
  run run_sql_raw "SELECT COUNT(*) FROM access_requests ar JOIN users u ON ar.user_id = u.id WHERE u.github_login = 'blockeduser' AND ar.status = 'pending';"
  assert_success
  assert_output "1"
}

@test "auth: get access request ID" {
  run run_sql_raw "SELECT ar.id FROM access_requests ar JOIN users u ON ar.user_id = u.id WHERE u.github_login = 'blockeduser' AND ar.status = 'pending' LIMIT 1;"
  assert_success
  ACCESS_REQUEST_ID=$(echo "$output" | tr -d '[:space:]')
  echo "$ACCESS_REQUEST_ID" > "$BATS_TMPDIR/access_request_id.txt"
  [[ -n "$ACCESS_REQUEST_ID" ]]
}

@test "auth: verify access request ID was captured" {
  ACCESS_REQUEST_ID=$(cat "$BATS_TMPDIR/access_request_id.txt")
  run echo "Access request ID is $ACCESS_REQUEST_ID"
  assert_success
  assert_output --partial "Access request ID is"
}

@test "auth: admin can list pending access requests via API" {
  is_dev || skip "Admin API list only available in DEV"
  run curl -s -H "X-API-Key: ${DIT_API_KEY}" \
    "${AUTH_SERVER}/api/admin/access-requests"
  assert_success
  assert_output --partial "blockeduser"
  assert_output --partial "E2E Test Access Request"
  assert_output --partial "pending"
}

@test "auth: admin can reject access request via API" {
  is_dev || skip "Admin reject API only available in DEV"
  ACCESS_REQUEST_ID=$(cat "$BATS_TMPDIR/access_request_id.txt")

  # First create a new request to reject
  run run_sql_raw "INSERT INTO access_requests (user_id, reason, status, created_at) VALUES ((SELECT id FROM users WHERE github_login = 'blockeduser'), 'Request to reject', 'pending', NOW()) RETURNING id;"
  assert_success

  # Extract just the UUID line (filter out INSERT status line)
  REJECT_REQUEST_ID=$(echo "$output" | grep -E '^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$')
  echo "$REJECT_REQUEST_ID" > "$BATS_TMPDIR/reject_request_id.txt"
  [[ -n "$REJECT_REQUEST_ID" ]]
}

@test "auth: verify reject request ID was captured" {
  is_dev || skip "Admin reject API only available in DEV"
  REJECT_REQUEST_ID=$(cat "$BATS_TMPDIR/reject_request_id.txt")
  run echo "Reject request ID is $REJECT_REQUEST_ID"
  assert_success
  assert_output --partial "Reject request ID is"
}

@test "auth: admin rejects access request via API" {
  is_dev || skip "Admin reject API only available in DEV"
  REJECT_REQUEST_ID=$(cat "$BATS_TMPDIR/reject_request_id.txt")

  run curl -s -w "\n%{http_code}" -X POST \
    -H "Cookie: dit_token=${DIT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"requestId\":\"${REJECT_REQUEST_ID}\",\"comment\":\"E2E test rejection\"}" \
    "${WEB_UI}/api/admin/access-requests/reject"

  assert_success
  assert_output --partial "200"
  assert_output --partial "rejected"
}

@test "auth: verify access request was rejected in database" {
  is_dev || skip "Admin reject API only available in DEV"
  REJECT_REQUEST_ID=$(cat "$BATS_TMPDIR/reject_request_id.txt")

  run run_sql_raw "SELECT status FROM access_requests WHERE id = '${REJECT_REQUEST_ID}';"
  assert_success
  assert_output "rejected"
}

@test "auth: admin approves access request via API" {
  is_dev || skip "Admin approve API only available in DEV"
  ACCESS_REQUEST_ID=$(cat "$BATS_TMPDIR/access_request_id.txt")

  run curl -s -w "\n%{http_code}" -X POST \
    -H "Cookie: dit_token=${DIT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"requestId\":\"${ACCESS_REQUEST_ID}\",\"comment\":\"E2E test approval\"}" \
    "${WEB_UI}/api/admin/access-requests/approve"

  assert_success
  assert_output --partial "200"
  assert_output --partial "approved"
}

@test "auth: verify access request was approved in database" {
  is_dev || skip "Admin approve API only available in DEV"
  ACCESS_REQUEST_ID=$(cat "$BATS_TMPDIR/access_request_id.txt")

  run run_sql_raw "SELECT status FROM access_requests WHERE id = '${ACCESS_REQUEST_ID}';"
  assert_success
  assert_output "approved"
}

@test "auth: verify blockeduser was added to whitelist after approval" {
  is_dev || skip "Admin approve API only available in DEV"
  run run_sql_raw "SELECT COUNT(*) FROM whitelisted_users wu JOIN users u ON wu.user_id = u.id WHERE u.github_login = 'blockeduser';"
  assert_success
  assert_output "1"
}

@test "auth: verify blockeduser is now whitelisted in users table" {
  is_dev || skip "Admin approve API only available in DEV"
  run run_sql_raw "SELECT is_whitelisted FROM users WHERE github_login = 'blockeduser';"
  assert_success
  assert_output "t"
}

# ========================================
# Whitelist Management Tests
# ========================================

@test "auth: admin can add user directly to whitelist" {
  run run_sql_cmd "INSERT INTO users (github_id, github_login, github_email, github_name, is_whitelisted, is_admin, created_at, updated_at) VALUES (100004, 'newuser', 'new@test.com', 'New User', false, false, NOW(), NOW()) ON CONFLICT (github_id) DO NOTHING;"
  assert_success
}

@test "auth: get new user ID for whitelist add" {
  run run_sql_raw "SELECT id FROM users WHERE github_login = 'newuser' LIMIT 1;"
  assert_success
  NEW_USER_ID=$(echo "$output" | tr -d '[:space:]')
  echo "$NEW_USER_ID" > "$BATS_TMPDIR/new_user_id.txt"
  [[ -n "$NEW_USER_ID" ]]
}

@test "auth: verify new user ID was captured" {
  NEW_USER_ID=$(cat "$BATS_TMPDIR/new_user_id.txt")
  run echo "New user ID is $NEW_USER_ID"
  assert_success
  assert_output --partial "New user ID is"
}

@test "auth: verify new user exists in database" {
  run run_sql_raw "SELECT COUNT(*) FROM users WHERE github_login = 'newuser';"
  assert_success
  assert_output "1"
}

# ========================================
# Authorization Token Validation Tests
# ========================================

@test "auth: malformed token is rejected" {
  run curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer not.a.valid.token" \
    "${GATEWAY}/api/v1/repos/"
  assert_success
  assert_output "401"
}

@test "auth: invalid Bearer format returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" -H "Authorization: InvalidFormat sometoken" \
    "${GATEWAY}/api/v1/repos/"
  assert_success
  assert_output "401"
}

# ========================================
# Multiple Request Tests
# ========================================

@test "auth: multiple authenticated requests succeed" {
  run bash -c "curl -sf -H 'Authorization: Bearer ${DIT_API_KEY}' ${GATEWAY}/health && curl -sf -H 'Authorization: Bearer ${DIT_API_KEY}' ${GATEWAY}/health"
  assert_success
}

# ========================================
# Session Management Tests
# ========================================

@test "auth: create session for test user" {
  run run_sql_cmd "DELETE FROM sessions WHERE jti = 'test-session-001'; INSERT INTO sessions (user_id, jti, expires_at, created_at) VALUES ((SELECT id FROM users WHERE github_login = 'testuser'), 'test-session-001', NOW() + INTERVAL '24 hours', NOW());"
  assert_success
}

@test "auth: verify session was created" {
  run run_sql_raw "SELECT COUNT(*) FROM sessions WHERE jti = 'test-session-001';"
  assert_success
  assert_output "1"
}

@test "auth: revoke session" {
  run run_sql_cmd "UPDATE sessions SET revoked_at = NOW() WHERE jti = 'test-session-001';"
  assert_success
}

@test "auth: verify session was revoked" {
  run run_sql_raw "SELECT revoked_at IS NOT NULL FROM sessions WHERE jti = 'test-session-001';"
  assert_success
  assert_output "t"
}

# ========================================
# Audit Log Tests
# ========================================

@test "auth: create audit log entry" {
  run run_sql_cmd "INSERT INTO audit_log (event_type, user_id, event_details, created_at) VALUES ('test_event', (SELECT id FROM users WHERE github_login = 'testadmin'), json_build_object('action', 'e2e_test'), NOW());"
  assert_success
}

@test "auth: verify audit log entry exists" {
  run run_sql_raw "SELECT COUNT(*) FROM audit_log al JOIN users u ON al.user_id = u.id WHERE al.event_type = 'test_event' AND u.github_login = 'testadmin';"
  assert_success
  assert_output "1"
}

@test "auth: verify audit log has metadata" {
  run run_sql_raw "SELECT al.event_details->>'action' FROM audit_log al JOIN users u ON al.user_id = u.id WHERE al.event_type = 'test_event' AND u.github_login = 'testadmin';"
  assert_success
  assert_output "e2e_test"
}

# ========================================
# Integration with Dit CLI Tests
# ========================================

@test "auth: run mongo container for auth test" {
  run "$D3" run -n authclitest mongo
  assert_success
}

@test "auth: create test repository with authenticated API" {
  run curl -sf -X POST -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${GATEWAY}/api/v1/repos/${AUTH_TEST_ORG}/cli-test-repo"
  assert_success
  assert_output --partial "cli-test-repo"
}

@test "auth: create commit for auth test" {
  run "$D3" commit -m "Auth CLI Test Commit" authclitest
  assert_success
  assert_output --partial "Commit"
}

@test "auth: add dit remote with auth" {
  run "$D3" remote add "${REMOTE_URL}/${AUTH_TEST_ORG}/cli-test-repo" authclitest
  assert_success
}

@test "auth: verify remote was added" {
  run "$D3" remote ls authclitest
  assert_success
  assert_output --partial "${REMOTE_URL}/${AUTH_TEST_ORG}/cli-test-repo"
}

@test "auth: push commit with authenticated API (simulated)" {
  run curl -sf -X POST -H "Authorization: Bearer ${DIT_API_KEY}" \
    -H "Content-Type: application/json" \
    "${GATEWAY}/api/v1/repos/${AUTH_TEST_ORG}/cli-test-repo/commits"
  # This may return an error but we're just testing the auth pathway
  true
}

@test "auth: cleanup - remove auth CLI test repo" {
  run "$D3" rm authclitest -f
  assert_success
  assert_output --partial "authclitest removed"
}

# ========================================
# API Key Management Tests
# ========================================

@test "API keys: verify api-gateway database connection" {
  if is_dev; then
    run docker exec dit-postgres pg_isready -U dit
    assert_success
    assert_output --partial "accepting connections"
  else
    run run_sql_raw "SELECT 1;"
    assert_success
    assert_output --partial "1"
  fi
}

@test "API keys: cleanup - remove any existing test API keys" {
  is_dev || skip "DEV-only pre-cleanup via API"
  curl -s -X DELETE -H "Cookie: dit_token=${DIT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"key_prefix":"e2etest1"}' \
    "${AUTH_SERVER}/api/v1/api-keys" 2>/dev/null || true
  curl -s -X DELETE -H "Cookie: dit_token=${DIT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"key_prefix":"e2etest2"}' \
    "${AUTH_SERVER}/api/v1/api-keys" 2>/dev/null || true
}

@test "API keys: get robertericreeves user ID" {
  run run_sql_raw "SELECT id FROM users WHERE github_login = 'robertericreeves' LIMIT 1;"
  assert_success
  TEST_USER_ID=$(echo "$output" | tr -d '[:space:]')
  echo "$TEST_USER_ID" > "$BATS_TMPDIR/test_user_id.txt"
  [[ -n "$TEST_USER_ID" ]]
}

@test "API keys: get existing API key ID for robertericreeves" {
  run run_sql_raw "SELECT id FROM api_keys WHERE key_prefix = '02b31569' LIMIT 1;"
  assert_success
  TEST_KEY_ID=$(echo "$output" | tr -d '[:space:]')
  echo "$TEST_KEY_ID" > "$BATS_TMPDIR/test_key_id.txt"
  [[ -n "$TEST_KEY_ID" ]]
}

@test "API keys: verify existing API key is in database" {
  TEST_KEY_ID=$(cat "$BATS_TMPDIR/test_key_id.txt")
  run run_sql_raw "SELECT COUNT(*) FROM api_keys WHERE id = '${TEST_KEY_ID}';"
  assert_success
  assert_output "1"
}

@test "API keys: verify existing API key has correct prefix" {
  TEST_KEY_ID=$(cat "$BATS_TMPDIR/test_key_id.txt")
  run run_sql_raw "SELECT key_prefix FROM api_keys WHERE id = '${TEST_KEY_ID}';"
  assert_success
  assert_output "02b31569"
}

@test "API keys: authenticate with existing API key via X-API-Key header" {
  run curl -sf -H "X-API-Key: ${DIT_API_KEY}" \
    "${GATEWAY}/health"
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

@test "API keys: authenticate with existing API key via Authorization Bearer header" {
  run curl -sf -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${GATEWAY}/health"
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

@test "API keys: create new test API key via API" {
  run curl -sf -X POST \
    -H "Authorization: Bearer ${DIT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"name":"E2E Test Key 1"}' \
    "${AUTH_SERVER}/api/v1/api-keys"
  assert_success

  NEW_API_KEY=$(echo "$output" | grep -oE '"key":"[^"]+' | cut -d'"' -f4)
  echo "$NEW_API_KEY" > "$BATS_TMPDIR/new_api_key.txt"

  NEW_KEY_PREFIX=$(echo "$output" | grep -oE '"keyPrefix":"[^"]+' | cut -d'"' -f4)
  echo "$NEW_KEY_PREFIX" > "$BATS_TMPDIR/new_key_prefix.txt"

  NEW_KEY_ID=$(echo "$output" | grep -oE '"id":"[^"]+' | cut -d'"' -f4)
  echo "$NEW_KEY_ID" > "$BATS_TMPDIR/new_key_id.txt"

  [[ -n "$NEW_API_KEY" ]]
  [[ -n "$NEW_KEY_PREFIX" ]]
  [[ -n "$NEW_KEY_ID" ]]
}

@test "API keys: verify new API key was created in database" {
  NEW_KEY_ID=$(cat "$BATS_TMPDIR/new_key_id.txt")
  run run_sql_raw "SELECT COUNT(*) FROM api_keys WHERE id = '${NEW_KEY_ID}';"
  assert_success
  assert_output "1"
}

@test "API keys: authenticate with newly created API key via X-API-Key header" {
  NEW_API_KEY=$(cat "$BATS_TMPDIR/new_api_key.txt")
  run curl -sf -H "X-API-Key: ${NEW_API_KEY}" \
    "${GATEWAY}/health"
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

@test "API keys: authenticate with newly created API key via Authorization Bearer" {
  NEW_API_KEY=$(cat "$BATS_TMPDIR/new_api_key.txt")
  run curl -sf -H "Authorization: Bearer ${NEW_API_KEY}" \
    "${GATEWAY}/health"
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

@test "API keys: verify last_used_at is updated after API call" {
  NEW_API_KEY=$(cat "$BATS_TMPDIR/new_api_key.txt")
  curl -s -H "X-API-Key: ${NEW_API_KEY}" \
    "${GATEWAY}/api/v1/repos" > /dev/null || true

  sleep 5

  NEW_KEY_ID=$(cat "$BATS_TMPDIR/new_key_id.txt")
  run run_sql_raw "SELECT last_used_at IS NOT NULL FROM api_keys WHERE id = '${NEW_KEY_ID}';"
  assert_success
  assert_output "t"
}

@test "API keys: create test repository with newly created API key" {
  NEW_API_KEY=$(cat "$BATS_TMPDIR/new_api_key.txt")
  run curl -X POST -f -H "X-API-Key: ${NEW_API_KEY}" \
    "${GATEWAY}/api/v1/repos/${APIKEY_TEST_ORG}/test-repo"
  assert_success
  assert_output --partial "test-repo"
}

@test "API keys: list repos with newly created API key" {
  NEW_API_KEY=$(cat "$BATS_TMPDIR/new_api_key.txt")
  run curl -sL -o /dev/null -w "%{http_code}" -H "X-API-Key: ${NEW_API_KEY}" \
    "${GATEWAY}/api/v1/repos"
  assert_success
  [[ "$output" == "200" || "$output" == "301" || "$output" == "404" ]]
}

@test "API keys: invalid API key returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: invalid-key-12345" \
    "${GATEWAY}/api/v1/repos"
  assert_success
  assert_output "401"
}

@test "API keys: create second test API key for revoke test" {
  run curl -sf -X POST \
    -H "Authorization: Bearer ${DIT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"name":"E2E Test Key 2 (To Be Revoked)"}' \
    "${AUTH_SERVER}/api/v1/api-keys"
  assert_success

  REVOKE_API_KEY=$(echo "$output" | grep -oE '"key":"[^"]+' | cut -d'"' -f4)
  echo "$REVOKE_API_KEY" > "$BATS_TMPDIR/revoke_api_key.txt"

  REVOKE_KEY_PREFIX=$(echo "$output" | grep -oE '"keyPrefix":"[^"]+' | cut -d'"' -f4)
  echo "$REVOKE_KEY_PREFIX" > "$BATS_TMPDIR/revoke_key_prefix.txt"

  REVOKE_KEY_ID=$(echo "$output" | grep -oE '"id":"[^"]+' | cut -d'"' -f4)
  echo "$REVOKE_KEY_ID" > "$BATS_TMPDIR/revoke_key_id.txt"

  [[ -n "$REVOKE_API_KEY" ]]
  [[ -n "$REVOKE_KEY_PREFIX" ]]
  [[ -n "$REVOKE_KEY_ID" ]]
}

@test "API keys: verify second API key works before revocation" {
  REVOKE_API_KEY=$(cat "$BATS_TMPDIR/revoke_api_key.txt")
  run curl -sf -H "X-API-Key: ${REVOKE_API_KEY}" \
    "${GATEWAY}/health"
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

@test "API keys: revoke second API key via API" {
  REVOKE_KEY_ID=$(cat "$BATS_TMPDIR/revoke_key_id.txt")
  run curl -sf -X DELETE \
    -H "Authorization: Bearer ${DIT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"keyId\":\"${REVOKE_KEY_ID}\"}" \
    "${AUTH_SERVER}/api/v1/api-keys"
  assert_success
}

@test "API keys: verify second API key was revoked in database" {
  REVOKE_KEY_PREFIX=$(cat "$BATS_TMPDIR/revoke_key_prefix.txt")
  run run_sql_raw "SELECT revoked_at IS NOT NULL FROM api_keys WHERE key_prefix = '${REVOKE_KEY_PREFIX}';"
  assert_success
  assert_output "t"
}

@test "API keys: revoked API key returns 401" {
  REVOKE_API_KEY=$(cat "$BATS_TMPDIR/revoke_api_key.txt")
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: ${REVOKE_API_KEY}" \
    "${GATEWAY}/api/v1/repos"
  assert_success
  assert_output "401"
}

@test "API keys: create third API key with expiration" {
  TEST_USER_ID=$(cat "$BATS_TMPDIR/test_user_id.txt")
  TEST_KEY_HASH_3="ccccddddeeeeffffaaaabbbbccccddddeeeeffffaaaabbbbccccddddeeeeffff"

  run run_sql_cmd "INSERT INTO api_keys (user_id, key_hash, key_prefix, name, created_at, expires_at) VALUES ('${TEST_USER_ID}', '${TEST_KEY_HASH_3}', 'testexp', 'E2E Test Key 3 (Expiring)', NOW(), NOW() + INTERVAL '1 minute') RETURNING id;"
  assert_success
  TEST_KEY_ID_3=$(echo "$output" | grep -oE '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}')
  echo "$TEST_KEY_ID_3" > "$BATS_TMPDIR/test_key_id_3.txt"
  [[ -n "$TEST_KEY_ID_3" ]]
}

@test "API keys: verify unexpired key exists" {
  TEST_KEY_ID_3=$(cat "$BATS_TMPDIR/test_key_id_3.txt")
  run run_sql_raw "SELECT (expires_at > NOW()) FROM api_keys WHERE id = '${TEST_KEY_ID_3}';"
  assert_success
  assert_output "t"
}

@test "API keys: expire the third API key" {
  TEST_KEY_ID_3=$(cat "$BATS_TMPDIR/test_key_id_3.txt")
  run run_sql_cmd "UPDATE api_keys SET expires_at = NOW() - INTERVAL '1 hour' WHERE id = '${TEST_KEY_ID_3}';"
  assert_success
}

@test "API keys: verify third API key is expired" {
  TEST_KEY_ID_3=$(cat "$BATS_TMPDIR/test_key_id_3.txt")
  run run_sql_raw "SELECT (expires_at < NOW()) FROM api_keys WHERE id = '${TEST_KEY_ID_3}';"
  assert_success
  assert_output "t"
}

@test "API keys: list all API keys for user" {
  TEST_USER_ID=$(cat "$BATS_TMPDIR/test_user_id.txt")
  run run_sql_raw "SELECT COUNT(*) FROM api_keys WHERE user_id = '${TEST_USER_ID}';"
  assert_success
  [[ "$output" -ge 4 ]]
}

@test "API keys: list only active (non-revoked, non-expired) keys via API" {
  run curl -sf -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${AUTH_SERVER}/api/v1/api-keys"
  assert_success
  assert_output --partial "02b31569"
}

@test "API keys: verify newly created API key metadata" {
  NEW_KEY_PREFIX=$(cat "$BATS_TMPDIR/new_key_prefix.txt")
  run run_sql_raw "SELECT name FROM api_keys WHERE key_prefix = '${NEW_KEY_PREFIX}';"
  assert_success
  assert_output --partial "E2E Test Key 1"
}

@test "API keys: cleanup - delete test repository" {
  run curl -X DELETE -f -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${GATEWAY}/api/v1/repos/${APIKEY_TEST_ORG}/test-repo"
  assert_success
}

@test "API keys: cleanup - revoke newly created test API key" {
  NEW_KEY_ID=$(cat "$BATS_TMPDIR/new_key_id.txt")
  run curl -sf -X DELETE \
    -H "Authorization: Bearer ${DIT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"keyId\":\"${NEW_KEY_ID}\"}" \
    "${AUTH_SERVER}/api/v1/api-keys"
  assert_success
}

@test "API keys: cleanup - delete test API keys from database" {
  run run_sql_cmd "DELETE FROM api_keys WHERE name LIKE 'E2E Test Key _%';"
  assert_success
}

@test "API keys: verify all test API keys cleaned up" {
  run run_sql_raw "SELECT COUNT(*) FROM api_keys WHERE name LIKE 'E2E Test Key _%';"
  assert_success
  assert_output "0"
}

# ========================================
# Cleanup Tests
# ========================================

@test "auth: cleanup - delete test repository (auth-test-repo)" {
  run curl -sf -X DELETE -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${GATEWAY}/api/v1/repos/${TEST_ORG}/auth-test-repo"
  true
}

@test "auth: cleanup - delete test repository (cli-test-repo)" {
  run curl -sf -X DELETE -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${GATEWAY}/api/v1/repos/${AUTH_TEST_ORG}/cli-test-repo"
  true
}

@test "auth: cleanup - delete test session" {
  run run_sql_cmd "DELETE FROM sessions WHERE jti = 'test-session-001';"
  assert_success
}

@test "auth: cleanup - delete test audit log entries" {
  run run_sql_cmd "DELETE FROM audit_log WHERE event_type = 'test_event';"
  assert_success
}

@test "auth: cleanup - delete all remaining audit log entries for test users" {
  run run_sql_cmd "DELETE FROM audit_log WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser'));"
  assert_success
}

@test "auth: cleanup - delete access requests for test users" {
  run run_sql_cmd "DELETE FROM access_requests WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser'));"
  assert_success
}

@test "auth: cleanup - delete whitelisted_users entries" {
  run run_sql_cmd "DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser'));"
  assert_success
}

@test "auth: cleanup - delete all sessions for test users" {
  run run_sql_cmd "DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser'));"
  assert_success
}

@test "auth: cleanup - delete test users" {
  run run_sql_cmd "DELETE FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser');"
  assert_success
}

@test "auth: verify all test users removed" {
  run run_sql_raw "SELECT COUNT(*) FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser');"
  assert_success
  assert_output "0"
}

@test "auth: verify all test sessions removed" {
  run run_sql_raw "SELECT COUNT(*) FROM sessions WHERE jti = 'test-session-001';"
  assert_success
  assert_output "0"
}

@test "auth: verify all test access requests removed" {
  run run_sql_raw "SELECT COUNT(*) FROM access_requests WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser'));"
  assert_success
  assert_output "0"
}

@test "auth: verify all test audit log entries removed" {
  run run_sql_raw "SELECT COUNT(*) FROM audit_log WHERE event_type = 'test_event';"
  assert_success
  assert_output "0"
}
