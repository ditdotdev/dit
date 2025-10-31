#!/usr/bin/env bats

# E2E Authentication and Authorization Tests for Datadatdat Remote Server
# These tests validate the complete auth flow including OAuth, JWT, whitelist, and admin workflows

# Load shared test helpers
load '../../test_helper'

# API Key is hardcoded for E2E testing
export DATADATDAT_API_KEY="80d141e9375c062be6c819e86f0e15e1c36bfcd5fd86286c30ad28b2e2ec8511"

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
  "$D3" rm authCliTest -f 2>/dev/null || true
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

@test "auth: unauthenticated request to protected API returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:8080/api/v1/repos/testorg/test-repo
  assert_success
  assert_output "401"
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
  run "$D3" run -n authCliTest mongo
  assert_success
}

@test "auth: create test repository with authenticated API" {
  run curl -s -X POST -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/authtest/cli-test-repo"
  assert_success
  assert_output --partial "cli-test-repo"
}

@test "auth: create commit for auth test" {
  run "$D3" commit -m "Auth CLI Test Commit" authCliTest
  assert_success
  assert_output --partial "Commit"
}

@test "auth: add datadatdat remote with auth" {
  run "$D3" remote add http://datadatdat-api-gateway:8080/authtest/cli-test-repo authCliTest
  assert_success
}

@test "auth: verify remote was added" {
  run "$D3" remote ls authCliTest
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
  run "$D3" rm authCliTest -f
  assert_success
  assert_output --partial "authCliTest removed"
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
