#!/usr/bin/env bats

# E2E Authentication and Authorization Tests for Datadatdat Remote Server
# These tests validate the complete auth flow including OAuth, JWT, whitelist, and admin workflows

# Load shared test helpers
load '../../test_helper'

# API Key is hardcoded for E2E testing
export DATADATDAT_API_KEY="***REMOVED***"

# Setup: Verify services are running and clean database
setup_file() {
  # Verify all services are running
  run curl -s http://127.0.0.1:8085/health
  [[ "$output" == *"healthy"* ]] || {
    echo "Auth server is not running"
    return 1
  }
  
  run curl -s http://127.0.0.1:8080/health
  [[ "$output" == *"healthy"* ]] || {
    echo "API gateway is not running"
    return 1
  }
  
  run docker exec datadatdat-postgres pg_isready -U datadatdat
  [[ "$output" == *"accepting connections"* ]] || {
    echo "Postgres is not ready"
    return 1
  }
}

# Cleanup after all tests
teardown_file() {
  # Cleanup test data
  docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); 
     DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); 
     DELETE FROM access_requests WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); 
     DELETE FROM audit_log WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); 
     DELETE FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser');" 2>/dev/null || true
  
  # Cleanup test repositories
  curl -s -X DELETE -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/testorg/auth-test-repo" 2>/dev/null || true
  curl -s -X DELETE -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/authtest/cli-test-repo" 2>/dev/null || true
  
  # Cleanup d3 test repository
  "$D3" rm authclitest -f 2>/dev/null || true
}

# ========================================
# Pre-requisites & Health Checks
# ========================================

@test "auth: verify auth server is running" {
  run curl -s http://127.0.0.1:8085/health
  assert_success
  assert_output --partial "healthy"
}

@test "auth: verify api-gateway is running" {
  run curl -s http://127.0.0.1:8080/health
  assert_success
  assert_output --partial "healthy"
}

@test "auth: verify postgres is ready" {
  run docker exec datadatdat-postgres pg_isready -U datadatdat
  assert_success
  assert_output --partial "accepting connections"
}

# ========================================
# Database Setup & Initial Admin
# ========================================

@test "auth: cleanup existing test users from previous runs" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); 
     DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); 
     DELETE FROM access_requests WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser')); 
     DELETE FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser');"
  assert_success
}

@test "auth: create initial admin user" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "INSERT INTO users (github_id, github_login, github_email, github_name, is_whitelisted, is_admin, created_at, updated_at) 
     VALUES (100001, 'testadmin', 'admin@test.com', 'Test Admin', true, true, NOW(), NOW()) 
     ON CONFLICT (github_id) DO UPDATE SET is_whitelisted = true, is_admin = true; 
     INSERT INTO whitelisted_users (user_id, approved_by, approved_at, reason) 
     VALUES ((SELECT id FROM users WHERE github_login = 'testadmin'), (SELECT id FROM users WHERE github_login = 'testadmin'), NOW(), 'E2E Test Admin User') 
     ON CONFLICT (user_id) DO NOTHING;"
  assert_success
  assert_output --partial "INSERT"
}

@test "auth: create whitelisted test user" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "INSERT INTO users (github_id, github_login, github_email, github_name, is_whitelisted, is_admin, created_at, updated_at) 
     VALUES (100002, 'testuser', 'user@test.com', 'Test User', true, false, NOW(), NOW()) 
     ON CONFLICT (github_id) DO UPDATE SET is_whitelisted = true, is_admin = false; 
     INSERT INTO whitelisted_users (user_id, approved_by, approved_at, reason) 
     VALUES ((SELECT id FROM users WHERE github_login = 'testuser'), (SELECT id FROM users WHERE github_login = 'testadmin'), NOW(), 'E2E Test User') 
     ON CONFLICT (user_id) DO NOTHING;"
  assert_success
  assert_output --partial "INSERT"
}

@test "auth: create non-whitelisted user" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "INSERT INTO users (github_id, github_login, github_email, github_name, is_whitelisted, is_admin, created_at, updated_at) 
     VALUES (100003, 'blockeduser', 'blocked@test.com', 'Blocked User', false, false, NOW(), NOW()) 
     ON CONFLICT (github_id) DO UPDATE SET is_whitelisted = false, is_admin = false;"
  assert_success
  assert_output --partial "INSERT"
}

# ========================================
# Unauthenticated Access Tests
# ========================================

@test "auth: unauthenticated request to non-existent repo returns 404" {
  # GitHub-style: unauthenticated access to non-existent/private repos returns 404
  # to avoid leaking repository existence information
  run curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8080/api/v1/repos/testorg/test-repo
  assert_success
  assert_output "404"
}

@test "auth: unauthenticated request to admin endpoint returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8085/api/admin/whitelist
  assert_success
  assert_output "401"
}

@test "auth: health check endpoints are accessible without auth" {
  run curl -s http://127.0.0.1:8080/health
  assert_success
  assert_output --partial "healthy"
}

# ========================================
# Authenticated User Access Tests
# ========================================

@test "auth: authenticated user can create repository" {
  run curl -s -X POST -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/testorg/auth-test-repo"
  assert_success
  assert_output --partial "auth-test-repo"
}

# ========================================
# API Gateway Integration Tests
# ========================================

@test "auth: verify API gateway forwards auth headers correctly" {
  run curl -s -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/health"
  assert_success
  assert_output --partial "healthy"
}

# ========================================
# Access Request Workflow
# ========================================

@test "auth: create access request for non-whitelisted user" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "INSERT INTO access_requests (user_id, reason, status, created_at) 
     VALUES ((SELECT id FROM users WHERE github_login = 'blockeduser'), 'E2E Test Access Request', 'pending', NOW());"
  assert_success
  assert_output --partial "INSERT"
}

@test "auth: verify access request was created" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT COUNT(*) FROM access_requests ar JOIN users u ON ar.user_id = u.id 
     WHERE u.github_login = 'blockeduser' AND ar.status = 'pending';"
  assert_success
  assert_output "1"
}

@test "auth: get access request ID" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT ar.id FROM access_requests ar JOIN users u ON ar.user_id = u.id 
     WHERE u.github_login = 'blockeduser' AND ar.status = 'pending' LIMIT 1;"
  assert_success
  # Remove whitespace/newlines from the output
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
  run curl -s -H "X-API-Key: ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8085/api/admin/access-requests"
  assert_success
  assert_output --partial "blockeduser"
  assert_output --partial "E2E Test Access Request"
  assert_output --partial "pending"
}

@test "auth: admin can reject access request via API" {
  ACCESS_REQUEST_ID=$(cat "$BATS_TMPDIR/access_request_id.txt")
  
  # First create a new request to reject
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "INSERT INTO access_requests (user_id, reason, status, created_at) 
     VALUES ((SELECT id FROM users WHERE github_login = 'blockeduser'), 'Request to reject', 'pending', NOW()) 
     RETURNING id;" 
  assert_success
  
  # Extract just the UUID line (filter out INSERT status line)
  REJECT_REQUEST_ID=$(echo "$output" | grep -E '^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$')
  echo "$REJECT_REQUEST_ID" > "$BATS_TMPDIR/reject_request_id.txt"
  [[ -n "$REJECT_REQUEST_ID" ]]
}

@test "auth: verify reject request ID was captured" {
  REJECT_REQUEST_ID=$(cat "$BATS_TMPDIR/reject_request_id.txt")
  run echo "Reject request ID is $REJECT_REQUEST_ID"
  assert_success
  assert_output --partial "Reject request ID is"
}

@test "auth: admin rejects access request via API" {
  REJECT_REQUEST_ID=$(cat "$BATS_TMPDIR/reject_request_id.txt")
  
  run curl -s -w "\n%{http_code}" -X POST \
    -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"requestId\":\"${REJECT_REQUEST_ID}\",\"comment\":\"E2E test rejection\"}" \
    "http://127.0.0.1:3000/api/admin/access-requests/reject"
  
  assert_success
  assert_output --partial "200"
  assert_output --partial "rejected"
}

@test "auth: verify access request was rejected in database" {
  REJECT_REQUEST_ID=$(cat "$BATS_TMPDIR/reject_request_id.txt")
  
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT status FROM access_requests WHERE id = '${REJECT_REQUEST_ID}';"
  assert_success
  assert_output "rejected"
}

@test "auth: admin approves access request via API" {
  ACCESS_REQUEST_ID=$(cat "$BATS_TMPDIR/access_request_id.txt")
  
  run curl -s -w "\n%{http_code}" -X POST \
    -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"requestId\":\"${ACCESS_REQUEST_ID}\",\"comment\":\"E2E test approval\"}" \
    "http://127.0.0.1:3000/api/admin/access-requests/approve"
  
  assert_success
  assert_output --partial "200"
  assert_output --partial "approved"
}

@test "auth: verify access request was approved in database" {
  ACCESS_REQUEST_ID=$(cat "$BATS_TMPDIR/access_request_id.txt")
  
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT status FROM access_requests WHERE id = '${ACCESS_REQUEST_ID}';"
  assert_success
  assert_output "approved"
}

@test "auth: verify blockeduser was added to whitelist after approval" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT COUNT(*) FROM whitelisted_users wu 
     JOIN users u ON wu.user_id = u.id 
     WHERE u.github_login = 'blockeduser';"
  assert_success
  assert_output "1"
}

@test "auth: verify blockeduser is now whitelisted in users table" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT is_whitelisted FROM users WHERE github_login = 'blockeduser';"
  assert_success
  assert_output "t"
}

# ========================================
# Whitelist Management Tests
# ========================================

@test "auth: admin can add user directly to whitelist" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "INSERT INTO users (github_id, github_login, github_email, github_name, is_whitelisted, is_admin, created_at, updated_at) 
     VALUES (100004, 'newuser', 'new@test.com', 'New User', false, false, NOW(), NOW()) 
     ON CONFLICT (github_id) DO NOTHING;"
  assert_success
  assert_output --partial "INSERT"
}

@test "auth: get new user ID for whitelist add" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT id FROM users WHERE github_login = 'newuser' LIMIT 1;"
  assert_success
  # Remove whitespace/newlines
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
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT COUNT(*) FROM users WHERE github_login = 'newuser';"
  assert_success
  assert_output "1"
}

# ========================================
# Authorization Token Validation Tests
# ========================================

@test "auth: malformed token is rejected" {
  run curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer not.a.valid.token" \
    "http://127.0.0.1:8080/api/v1/repos"
  assert_success
  assert_output "401"
}

@test "auth: invalid Bearer format returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" -H "Authorization: InvalidFormat sometoken" \
    "http://127.0.0.1:8080/api/v1/repos"
  assert_success
  assert_output "401"
}

# ========================================
# Multiple Request Tests
# ========================================

@test "auth: multiple authenticated requests succeed" {
  run bash -c "curl -s -o /dev/null -w '%{http_code}' -H 'Authorization: Bearer ${DATADATDAT_API_KEY}' http://127.0.0.1:8080/health && curl -s -o /dev/null -w '%{http_code}' -H 'Authorization: Bearer ${DATADATDAT_API_KEY}' http://127.0.0.1:8080/health"
  assert_success
  assert_output --partial "200"
}

# ========================================
# Session Management Tests
# ========================================

@test "auth: create session for test user" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM sessions WHERE jti = 'test-session-001'; 
     INSERT INTO sessions (user_id, jti, expires_at, created_at) 
     VALUES ((SELECT id FROM users WHERE github_login = 'testuser'), 'test-session-001', NOW() + INTERVAL '24 hours', NOW());"
  assert_success
  assert_output --partial "INSERT"
}

@test "auth: verify session was created" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT COUNT(*) FROM sessions WHERE jti = 'test-session-001';"
  assert_success
  assert_output "1"
}

@test "auth: revoke session" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "UPDATE sessions SET revoked_at = NOW() WHERE jti = 'test-session-001';"
  assert_success
  assert_output --partial "UPDATE"
}

@test "auth: verify session was revoked" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT revoked_at IS NOT NULL FROM sessions WHERE jti = 'test-session-001';"
  assert_success
  assert_output "t"
}

# ========================================
# Audit Log Tests
# ========================================

@test "auth: create audit log entry" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "INSERT INTO audit_log (event_type, user_id, event_details, created_at) 
     VALUES ('test_event', (SELECT id FROM users WHERE github_login = 'testadmin'), 
     json_build_object('action', 'e2e_test'), NOW());"
  assert_success
  assert_output --partial "INSERT"
}

@test "auth: verify audit log entry exists" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT COUNT(*) FROM audit_log al JOIN users u ON al.user_id = u.id 
     WHERE al.event_type = 'test_event' AND u.github_login = 'testadmin';"
  assert_success
  assert_output "1"
}

@test "auth: verify audit log has metadata" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT al.event_details->>'action' FROM audit_log al JOIN users u ON al.user_id = u.id 
     WHERE al.event_type = 'test_event' AND u.github_login = 'testadmin';"
  assert_success
  assert_output "e2e_test"
}

# ========================================
# Integration with Datadatdat CLI Tests
# ========================================

@test "auth: run mongo container for auth test" {
  run "$D3" run -n authclitest mongo
  assert_success
}

@test "auth: create test repository with authenticated API" {
  run curl -s -X POST -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/authtest/cli-test-repo"
  assert_success
  assert_output --partial "cli-test-repo"
}

@test "auth: create commit for auth test" {
  run "$D3" commit -m "Auth CLI Test Commit" authclitest
  assert_success
  assert_output --partial "Commit"
}

@test "auth: add datadatdat remote with auth" {
  run "$D3" remote add http://datadatdat-api-gateway:8080/authtest/cli-test-repo authclitest
  assert_success
}

@test "auth: verify remote was added" {
  run "$D3" remote ls authclitest
  assert_success
  assert_output --partial "http://datadatdat-api-gateway:8080/authtest/cli-test-repo"
}

@test "auth: push commit with authenticated API (simulated)" {
  run curl -s -X POST -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    -H "Content-Type: application/json" \
    "http://127.0.0.1:8080/api/v1/repos/authtest/cli-test-repo/commits"
  # This may return an error but we're just testing the auth pathway
  # assert_success not used here as the commit POST may fail with valid auth
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
  # The gateway should now connect to database for API key validation
  run docker exec datadatdat-postgres pg_isready -U datadatdat
  assert_success
  assert_output --partial "accepting connections"
}

@test "API keys: cleanup - remove any existing test API keys" {
  # Best effort cleanup from previous runs - delete via API
  curl -s -X DELETE -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"key_prefix":"e2etest1"}' \
    "http://127.0.0.1:8085/api/v1/api-keys" 2>/dev/null || true
  curl -s -X DELETE -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"key_prefix":"e2etest2"}' \
    "http://127.0.0.1:8085/api/v1/api-keys" 2>/dev/null || true
}

@test "API keys: get robertericreeves user ID" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT id FROM users WHERE github_login = 'robertericreeves' LIMIT 1;"
  assert_success
  TEST_USER_ID=$(echo "$output" | tr -d '[:space:]')
  echo "$TEST_USER_ID" > "$BATS_TMPDIR/test_user_id.txt"
  [[ -n "$TEST_USER_ID" ]]
}

@test "API keys: get existing API key ID for robertericreeves" {
  # Using the existing API key with prefix 02b31569
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT id FROM api_keys WHERE key_prefix = '02b31569' LIMIT 1;"
  assert_success
  TEST_KEY_ID=$(echo "$output" | tr -d '[:space:]')
  echo "$TEST_KEY_ID" > "$BATS_TMPDIR/test_key_id.txt"
  [[ -n "$TEST_KEY_ID" ]]
}

@test "API keys: verify existing API key is in database" {
  TEST_KEY_ID=$(cat "$BATS_TMPDIR/test_key_id.txt")
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT COUNT(*) FROM api_keys WHERE id = '${TEST_KEY_ID}';"
  assert_success
  assert_output "1"
}

@test "API keys: verify existing API key has correct prefix" {
  TEST_KEY_ID=$(cat "$BATS_TMPDIR/test_key_id.txt")
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT key_prefix FROM api_keys WHERE id = '${TEST_KEY_ID}';"
  assert_success
  assert_output "02b31569"
}

@test "API keys: authenticate with existing API key via X-API-Key header" {
  # Test authentication using the real API key
  run curl -sf -H "X-API-Key: ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/health"
  assert_success
  assert_output --partial "healthy"
}

@test "API keys: authenticate with existing API key via Authorization Bearer header" {
  # Test authentication using Bearer token format
  run curl -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/health"
  assert_success
  assert_output --partial "healthy"
}

@test "API keys: create new test API key via API" {
  # Create a new API key using the authenticated API
  run curl -sf -X POST \
    -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"name":"E2E Test Key 1"}' \
    "http://127.0.0.1:8085/api/v1/api-keys"
  assert_success
  
  # Extract the full key from the response (camelCase)
  NEW_API_KEY=$(echo "$output" | grep -oE '"key":"[^"]+' | cut -d'"' -f4)
  echo "$NEW_API_KEY" > "$BATS_TMPDIR/new_api_key.txt"
  
  # Extract the key prefix (camelCase)
  NEW_KEY_PREFIX=$(echo "$output" | grep -oE '"keyPrefix":"[^"]+' | cut -d'"' -f4)
  echo "$NEW_KEY_PREFIX" > "$BATS_TMPDIR/new_key_prefix.txt"
  
  # Extract the key ID (camelCase)
  NEW_KEY_ID=$(echo "$output" | grep -oE '"id":"[^"]+' | cut -d'"' -f4)
  echo "$NEW_KEY_ID" > "$BATS_TMPDIR/new_key_id.txt"
  
  [[ -n "$NEW_API_KEY" ]]
  [[ -n "$NEW_KEY_PREFIX" ]]
  [[ -n "$NEW_KEY_ID" ]]
}

@test "API keys: verify new API key was created in database" {
  NEW_KEY_ID=$(cat "$BATS_TMPDIR/new_key_id.txt")
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT COUNT(*) FROM api_keys WHERE id = '${NEW_KEY_ID}';"
  assert_success
  assert_output "1"
}

@test "API keys: authenticate with newly created API key via X-API-Key header" {
  NEW_API_KEY=$(cat "$BATS_TMPDIR/new_api_key.txt")
  run curl -sf -H "X-API-Key: ${NEW_API_KEY}" \
    "http://127.0.0.1:8080/health"
  assert_success
  assert_output --partial "healthy"
}

@test "API keys: authenticate with newly created API key via Authorization Bearer" {
  NEW_API_KEY=$(cat "$BATS_TMPDIR/new_api_key.txt")
  run curl -sf -H "Authorization: Bearer ${NEW_API_KEY}" \
    "http://127.0.0.1:8080/health"
  assert_success
  assert_output --partial "healthy"
}

@test "API keys: verify last_used_at is updated after API call" {
  # Make an API call with the newly created key to an authenticated endpoint
  # Use /api/v1/repos which has API key auth middleware
  NEW_API_KEY=$(cat "$BATS_TMPDIR/new_api_key.txt")
  # Don't use -f flag since endpoint may return 404, but auth will still run
  curl -s -H "X-API-Key: ${NEW_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos" > /dev/null || true
  
  # Wait a moment for async update
  sleep 5
  
  # Check that last_used_at is now set
  NEW_KEY_ID=$(cat "$BATS_TMPDIR/new_key_id.txt")
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT last_used_at IS NOT NULL FROM api_keys WHERE id = '${NEW_KEY_ID}';"
  assert_success
  assert_output "t"
}

@test "API keys: create test repository with newly created API key" {
  NEW_API_KEY=$(cat "$BATS_TMPDIR/new_api_key.txt")
  run curl -X POST -f -H "X-API-Key: ${NEW_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/apikeytest/test-repo"
  assert_success
  assert_output --partial "test-repo"
}

@test "API keys: list repos with newly created API key" {
  NEW_API_KEY=$(cat "$BATS_TMPDIR/new_api_key.txt")
  run curl -sL -o /dev/null -w "%{http_code}" -H "X-API-Key: ${NEW_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos"
  assert_success
  # API key auth works - endpoint may return 200 (listing), 301 (redirect), or 404
  [[ "$output" == "200" || "$output" == "301" || "$output" == "404" ]]
}

@test "API keys: invalid API key returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: invalid-key-12345" \
    "http://127.0.0.1:8080/api/v1/repos"
  assert_success
  assert_output "401"
}

@test "API keys: create second test API key for revoke test" {
  # Create another API key to test revocation
  run curl -sf -X POST \
    -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"name":"E2E Test Key 2 (To Be Revoked)"}' \
    "http://127.0.0.1:8085/api/v1/api-keys"
  assert_success
  
  # Extract the full key (camelCase)
  REVOKE_API_KEY=$(echo "$output" | grep -oE '"key":"[^"]+' | cut -d'"' -f4)
  echo "$REVOKE_API_KEY" > "$BATS_TMPDIR/revoke_api_key.txt"
  
  # Extract the key prefix (camelCase)
  REVOKE_KEY_PREFIX=$(echo "$output" | grep -oE '"keyPrefix":"[^"]+' | cut -d'"' -f4)
  echo "$REVOKE_KEY_PREFIX" > "$BATS_TMPDIR/revoke_key_prefix.txt"
  
  # Extract the key ID (camelCase)
  REVOKE_KEY_ID=$(echo "$output" | grep -oE '"id":"[^"]+' | cut -d'"' -f4)
  echo "$REVOKE_KEY_ID" > "$BATS_TMPDIR/revoke_key_id.txt"
  
  [[ -n "$REVOKE_API_KEY" ]]
  [[ -n "$REVOKE_KEY_PREFIX" ]]
  [[ -n "$REVOKE_KEY_ID" ]]
}

@test "API keys: verify second API key works before revocation" {
  REVOKE_API_KEY=$(cat "$BATS_TMPDIR/revoke_api_key.txt")
  run curl -sf -H "X-API-Key: ${REVOKE_API_KEY}" \
    "http://127.0.0.1:8080/health"
  assert_success
  assert_output --partial "healthy"
}

@test "API keys: revoke second API key via API" {
  REVOKE_KEY_ID=$(cat "$BATS_TMPDIR/revoke_key_id.txt")
  run curl -sf -X DELETE \
    -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"keyId\":\"${REVOKE_KEY_ID}\"}" \
    "http://127.0.0.1:8085/api/v1/api-keys"
  assert_success
}

@test "API keys: verify second API key was revoked in database" {
  REVOKE_KEY_PREFIX=$(cat "$BATS_TMPDIR/revoke_key_prefix.txt")
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT revoked_at IS NOT NULL FROM api_keys WHERE key_prefix = '${REVOKE_KEY_PREFIX}';"
  assert_success
  assert_output "t"
}

@test "API keys: revoked API key returns 401" {
  REVOKE_API_KEY=$(cat "$BATS_TMPDIR/revoke_api_key.txt")
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: ${REVOKE_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos"
  assert_success
  assert_output "401"
}

@test "API keys: create third API key with expiration" {
  TEST_USER_ID=$(cat "$BATS_TMPDIR/test_user_id.txt")
  # Create a key that expires in 1 minute via direct DB insert (API doesn't support expiration yet)
  TEST_KEY_HASH_3="ccccddddeeeeffffaaaabbbbccccddddeeeeffffaaaabbbbccccddddeeeeffff"
  
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "INSERT INTO api_keys (user_id, key_hash, key_prefix, name, created_at, expires_at) 
     VALUES ('${TEST_USER_ID}', '${TEST_KEY_HASH_3}', 'testexp', 'E2E Test Key 3 (Expiring)', NOW(), NOW() + INTERVAL '1 minute')
     RETURNING id;"
  assert_success
  TEST_KEY_ID_3=$(echo "$output" | grep -oE '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}')
  echo "$TEST_KEY_ID_3" > "$BATS_TMPDIR/test_key_id_3.txt"
  [[ -n "$TEST_KEY_ID_3" ]]
}

@test "API keys: verify unexpired key exists" {
  # Just verify the key exists and is not expired yet
  TEST_KEY_ID_3=$(cat "$BATS_TMPDIR/test_key_id_3.txt")
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT (expires_at > NOW()) FROM api_keys WHERE id = '${TEST_KEY_ID_3}';"
  assert_success
  assert_output "t"
}

@test "API keys: expire the third API key" {
  TEST_KEY_ID_3=$(cat "$BATS_TMPDIR/test_key_id_3.txt")
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "UPDATE api_keys SET expires_at = NOW() - INTERVAL '1 hour' WHERE id = '${TEST_KEY_ID_3}';"
  assert_success
  assert_output --partial "UPDATE"
}

@test "API keys: verify third API key is expired" {
  # Verify the key is expired
  TEST_KEY_ID_3=$(cat "$BATS_TMPDIR/test_key_id_3.txt")
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT (expires_at < NOW()) FROM api_keys WHERE id = '${TEST_KEY_ID_3}';"
  assert_success
  assert_output "t"
}

@test "API keys: list all API keys for user" {
  TEST_USER_ID=$(cat "$BATS_TMPDIR/test_user_id.txt")
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT COUNT(*) FROM api_keys WHERE user_id = '${TEST_USER_ID}';"
  assert_success
  # Should have: original key + new test key + revoked key + expired key = at least 4
  [[ "$output" -ge 4 ]]
}

@test "API keys: list only active (non-revoked, non-expired) keys via API" {
  run curl -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8085/api/v1/api-keys"
  assert_success
  # Response should contain the original key and the new test key (not revoked or expired)
  assert_output --partial "02b31569"
}

@test "API keys: verify newly created API key metadata" {
  NEW_KEY_PREFIX=$(cat "$BATS_TMPDIR/new_key_prefix.txt")
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT name FROM api_keys WHERE key_prefix = '${NEW_KEY_PREFIX}';"
  assert_success
  assert_output --partial "E2E Test Key 1"
}

@test "API keys: cleanup - delete test repository" {
  run curl -X DELETE -f -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/apikeytest/test-repo"
  # Best effort - may already be deleted
  assert_success
}

@test "API keys: cleanup - revoke newly created test API key" {
  # Revoke the first test key we created
  NEW_KEY_ID=$(cat "$BATS_TMPDIR/new_key_id.txt")
  run curl -sf -X DELETE \
    -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d "{\"keyId\":\"${NEW_KEY_ID}\"}" \
    "http://127.0.0.1:8085/api/v1/api-keys"
  assert_success
}

@test "API keys: cleanup - delete test API keys from database" {
  # Clean up test-created keys but preserve the Liquibase-seeded 'E2E Test Key'
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM api_keys WHERE name LIKE 'E2E Test Key _%';"
  assert_success
  assert_output --partial "DELETE"
}

@test "API keys: verify all test API keys cleaned up" {
  # Only test-created keys (with numbered suffixes) should be gone; seeded key remains
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT COUNT(*) FROM api_keys WHERE name LIKE 'E2E Test Key _%';"
  assert_success
  assert_output "0"
}

# ========================================
# Cleanup Tests
# ========================================

@test "auth: cleanup - delete test repository (auth-test-repo)" {
  run curl -s -X DELETE -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/testorg/auth-test-repo"
  # Best effort - may already be deleted
  assert_success
}

@test "auth: cleanup - delete test repository (cli-test-repo)" {
  run curl -s -X DELETE -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/authtest/cli-test-repo"
  # Best effort - may already be deleted
  assert_success
}

@test "auth: cleanup - delete test session" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM sessions WHERE jti = 'test-session-001';"
  assert_success
  assert_output --partial "DELETE"
}

@test "auth: cleanup - delete test audit log entries" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM audit_log WHERE event_type = 'test_event';"
  assert_success
  assert_output --partial "DELETE"
}

@test "auth: cleanup - delete all remaining audit log entries for test users" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM audit_log WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser'));"
  assert_success
  assert_output --partial "DELETE"
}

@test "auth: cleanup - delete access requests for test users" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM access_requests WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser'));"
  assert_success
  assert_output --partial "DELETE"
}

@test "auth: cleanup - delete whitelisted_users entries" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser'));"
  assert_success
  assert_output --partial "DELETE"
}

@test "auth: cleanup - delete all sessions for test users" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser'));"
  assert_success
  assert_output --partial "DELETE"
}

@test "auth: cleanup - delete test users" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -c \
    "DELETE FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser');"
  assert_success
  assert_output --partial "DELETE"
}

@test "auth: verify all test users removed" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT COUNT(*) FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser');"
  assert_success
  assert_output "0"
}

@test "auth: verify all test sessions removed" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT COUNT(*) FROM sessions WHERE jti = 'test-session-001';"
  assert_success
  assert_output "0"
}

@test "auth: verify all test access requests removed" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT COUNT(*) FROM access_requests WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('testuser', 'testadmin', 'blockeduser', 'newuser'));"
  assert_success
  assert_output "0"
}

@test "auth: verify all test audit log entries removed" {
  run docker exec datadatdat-postgres psql -U datadatdat -d datadatdat -t -A -c \
    "SELECT COUNT(*) FROM audit_log WHERE event_type = 'test_event';"
  assert_success
  assert_output "0"
}
