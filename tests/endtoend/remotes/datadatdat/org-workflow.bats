#!/usr/bin/env bats

# E2E Organization Workflow Tests for Datadatdat Remote Server
# Tests org CRUD, membership management, and CLI org commands.

# Load shared test helpers
load '../../test_helper'
load 'env'

# Admin API key (seeded by Liquibase)
ADMIN_KEY="02b31569a9052bc4b3cf1c3819d4fc048d34c96eca21f2b8e2359b5ecdfec93a"

# Test API keys for org users (raw hex keys)
ORGUSER_A_KEY="aa11111122223333444455556666777788889999aaaabbbbccccddddeeeeffff"
ORGUSER_B_KEY="bb11111122223333444455556666777788889999aaaabbbbccccddddeeeeffff"

# ========================================
# Setup / Teardown
# ========================================

setup_file() {
  run curl -s "$GATEWAY/health"
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || { echo "Gateway not running"; return 1; }

  run curl -s "$AUTH_SERVER/health"
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || { echo "Auth server not running"; return 1; }

  if is_dev; then
    run docker exec datadatdat-postgres pg_isready -U datadatdat
    [[ "$output" == *"accepting connections"* ]] || { echo "Postgres not ready"; return 1; }
  else
    run run_sql_raw "SELECT 1;"
    [[ "$output" == *"1"* ]] || { echo "Postgres not ready"; return 1; }
  fi
}

teardown_file() {
  # Best-effort cleanup — API then DB
  curl -s -X DELETE -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/test-org" 2>/dev/null || true
  curl -s -X DELETE -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/delete-me-org" 2>/dev/null || true

  run_sql_cmd \
    "DELETE FROM org_memberships WHERE org_id IN (SELECT id FROM organizations WHERE name IN ('test-org', 'delete-me-org'));
     DELETE FROM organizations WHERE name IN ('test-org', 'delete-me-org');
     DELETE FROM namespaces WHERE name IN ('test-org', 'delete-me-org');
     DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('orguser-a', 'orguser-b'));
     DELETE FROM namespaces WHERE name IN ('orguser-a', 'orguser-b');
     DELETE FROM users WHERE github_login IN ('orguser-a', 'orguser-b');" 2>/dev/null || true
}

# ========================================
# Health Checks
# ========================================

@test "org: verify auth server is running" {
  run curl -s "$AUTH_SERVER/health"
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

@test "org: verify api-gateway is running" {
  run curl -s "$GATEWAY/health"
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

@test "org: verify postgres is ready" {
  is_dev || skip "Local postgres check only for DEV"
  run docker exec datadatdat-postgres pg_isready -U datadatdat
  assert_success
  assert_output --partial "accepting connections"
}

# ========================================
# Test User Bootstrap (DB — no user creation API)
# ========================================

@test "org: cleanup existing test data from previous runs" {
  run run_sql_cmd \
    "DELETE FROM org_memberships WHERE org_id IN (SELECT id FROM organizations WHERE name IN ('test-org', 'delete-me-org'));
     DELETE FROM organizations WHERE name IN ('test-org', 'delete-me-org');
     DELETE FROM namespaces WHERE name IN ('test-org', 'delete-me-org');
     DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('orguser-a', 'orguser-b'));
     DELETE FROM namespaces WHERE name IN ('orguser-a', 'orguser-b');
     DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('orguser-a', 'orguser-b'));
     DELETE FROM users WHERE github_login IN ('orguser-a', 'orguser-b');"
  assert_success
}

@test "org: create orguser-a (admin, whitelisted)" {
  run run_sql_cmd \
    "INSERT INTO users (github_id, github_login, github_email, github_name, github_avatar_url, is_whitelisted, is_admin, created_at, updated_at)
     VALUES (200001, 'orguser-a', 'orguser-a@test.com', 'Org User A', '', true, true, NOW(), NOW())
     ON CONFLICT (github_id) DO UPDATE SET is_whitelisted = true, is_admin = true, github_avatar_url = '';
     INSERT INTO whitelisted_users (user_id, approved_by, approved_at, reason)
     VALUES ((SELECT id FROM users WHERE github_login = 'orguser-a'), (SELECT id FROM users WHERE github_login = 'orguser-a'), NOW(), 'E2E Org Test User A')
     ON CONFLICT (user_id) DO NOTHING;"
  assert_success
  assert_output --partial "INSERT"
}

@test "org: create orguser-b (non-admin, whitelisted)" {
  run run_sql_cmd \
    "INSERT INTO users (github_id, github_login, github_email, github_name, github_avatar_url, is_whitelisted, is_admin, created_at, updated_at)
     VALUES (200002, 'orguser-b', 'orguser-b@test.com', 'Org User B', '', true, false, NOW(), NOW())
     ON CONFLICT (github_id) DO UPDATE SET is_whitelisted = true, is_admin = false, github_avatar_url = '';
     INSERT INTO whitelisted_users (user_id, approved_by, approved_at, reason)
     VALUES ((SELECT id FROM users WHERE github_login = 'orguser-b'), (SELECT id FROM users WHERE github_login = 'orguser-a'), NOW(), 'E2E Org Test User B')
     ON CONFLICT (user_id) DO NOTHING;"
  assert_success
  assert_output --partial "INSERT"
}

@test "org: get orguser-a user ID" {
  run run_sql_raw \
    "SELECT id FROM users WHERE github_login = 'orguser-a' LIMIT 1;"
  assert_success
  ORGUSER_A_ID=$(echo "$output" | tr -d '[:space:]')
  echo "$ORGUSER_A_ID" > "$BATS_TMPDIR/orguser_a_id.txt"
  [[ -n "$ORGUSER_A_ID" ]]
}

@test "org: get orguser-b user ID" {
  run run_sql_raw \
    "SELECT id FROM users WHERE github_login = 'orguser-b' LIMIT 1;"
  assert_success
  ORGUSER_B_ID=$(echo "$output" | tr -d '[:space:]')
  echo "$ORGUSER_B_ID" > "$BATS_TMPDIR/orguser_b_id.txt"
  [[ -n "$ORGUSER_B_ID" ]]
}

@test "org: create API key for orguser-a" {
  ORGUSER_A_ID=$(cat "$BATS_TMPDIR/orguser_a_id.txt")
  KEY_HASH=$(echo -n "$ORGUSER_A_KEY" | sha256sum | cut -d' ' -f1)
  KEY_PREFIX="${ORGUSER_A_KEY:0:8}"

  run run_sql_cmd \
    "INSERT INTO api_keys (user_id, key_hash, key_prefix, name, created_at)
     VALUES ('${ORGUSER_A_ID}', '${KEY_HASH}', '${KEY_PREFIX}', 'Org Test Key A', NOW());"
  assert_success
  assert_output --partial "INSERT"
}

@test "org: create API key for orguser-b" {
  ORGUSER_B_ID=$(cat "$BATS_TMPDIR/orguser_b_id.txt")
  KEY_HASH=$(echo -n "$ORGUSER_B_KEY" | sha256sum | cut -d' ' -f1)
  KEY_PREFIX="${ORGUSER_B_KEY:0:8}"

  run run_sql_cmd \
    "INSERT INTO api_keys (user_id, key_hash, key_prefix, name, created_at)
     VALUES ('${ORGUSER_B_ID}', '${KEY_HASH}', '${KEY_PREFIX}', 'Org Test Key B', NOW());"
  assert_success
  assert_output --partial "INSERT"
}

@test "org: verify orguser-a API key authenticates" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/health"
  assert_success
  assert_output "200"
}

@test "org: verify orguser-b API key authenticates" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $ORGUSER_B_KEY" \
    "$GATEWAY/health"
  assert_success
  assert_output "200"
}

# ========================================
# Organization Creation
# ========================================

@test "org: create test-org as orguser-a returns 201" {
  run curl -s -w "\n%{http_code}" -X POST \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    -H "Content-Type: application/json" \
    -d '{"name":"test-org","displayName":"Test Organization"}' \
    "$GATEWAY/api/v1/orgs"
  assert_success
  assert_output --partial "test-org"
  assert_output --partial "201"
}

@test "org: get test-org returns org details" {
  run curl -sf -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/test-org"
  assert_success
  assert_output --partial "test-org"
  assert_output --partial "Test Organization"
}

@test "org: test-org members include orguser-a as owner" {
  run curl -sf -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/test-org/members"
  assert_success
  assert_output --partial "owner"
}

@test "org: duplicate org name returns 409" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    -H "Content-Type: application/json" \
    -d '{"name":"test-org"}' \
    "$GATEWAY/api/v1/orgs"
  assert_success
  assert_output "409"
}

@test "org: create org without auth returns 401" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -d '{"name":"unauth-org"}' \
    "$GATEWAY/api/v1/orgs"
  assert_success
  assert_output "401"
}

# ========================================
# Organization Listing
# ========================================

@test "org: list orgs as orguser-a contains test-org" {
  run curl -sf -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs"
  assert_success
  assert_output --partial "test-org"
}

@test "org: list orgs as orguser-b returns empty (not a member)" {
  run curl -sf -H "X-API-Key: $ORGUSER_B_KEY" \
    "$GATEWAY/api/v1/orgs"
  assert_success
  # orguser-b is not a member of any org yet, list should be empty
  assert_output "[]"
}

# ========================================
# Organization Update
# ========================================

@test "org: update test-org as owner returns 200" {
  run curl -s -w "\n%{http_code}" -X PUT \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    -H "Content-Type: application/json" \
    -d '{"displayName":"Updated Test Org","description":"E2E test org"}' \
    "$GATEWAY/api/v1/orgs/test-org"
  assert_success
  assert_output --partial "200"
}

@test "org: verify test-org was updated" {
  run curl -sf -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/test-org"
  assert_success
  assert_output --partial "Updated Test Org"
  assert_output --partial "E2E test org"
}

@test "org: update test-org as non-member returns 403" {
  run curl -s -o /dev/null -w "%{http_code}" -X PUT \
    -H "X-API-Key: $ORGUSER_B_KEY" \
    -H "Content-Type: application/json" \
    -d '{"displayName":"Hijacked"}' \
    "$GATEWAY/api/v1/orgs/test-org"
  assert_success
  assert_output "403"
}

@test "org: get nonexistent org returns 404" {
  run curl -s -o /dev/null -w "%{http_code}" \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/nonexistent-org"
  assert_success
  assert_output "404"
}

# ========================================
# Membership — Add
# ========================================

@test "org: add orguser-b as member of test-org" {
  ORGUSER_B_ID=$(cat "$BATS_TMPDIR/orguser_b_id.txt")

  run curl -s -w "\n%{http_code}" -X POST \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"userId\":\"${ORGUSER_B_ID}\",\"role\":\"member\"}" \
    "$GATEWAY/api/v1/orgs/test-org/members"
  assert_success
  assert_output --partial "200"
}

@test "org: list members returns 2 members" {
  run curl -sf -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/test-org/members"
  assert_success
  # Both orguser-a (owner) and orguser-b (member) should appear
  assert_output --partial "owner"
  assert_output --partial "member"
}

@test "org: list orgs as orguser-b now contains test-org" {
  run curl -sf -H "X-API-Key: $ORGUSER_B_KEY" \
    "$GATEWAY/api/v1/orgs"
  assert_success
  assert_output --partial "test-org"
}

@test "org: add orguser-b again returns 409" {
  ORGUSER_B_ID=$(cat "$BATS_TMPDIR/orguser_b_id.txt")

  run curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"userId\":\"${ORGUSER_B_ID}\",\"role\":\"member\"}" \
    "$GATEWAY/api/v1/orgs/test-org/members"
  assert_success
  assert_output "409"
}

@test "org: member cannot add new members (403)" {
  ORGUSER_A_ID=$(cat "$BATS_TMPDIR/orguser_a_id.txt")

  # orguser-b is a member, not admin/owner — should be rejected
  run curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "X-API-Key: $ORGUSER_B_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"userId\":\"${ORGUSER_A_ID}\",\"role\":\"member\"}" \
    "$GATEWAY/api/v1/orgs/test-org/members"
  assert_success
  assert_output "403"
}

# ========================================
# Membership — Update Role
# ========================================

@test "org: update orguser-b role to admin" {
  ORGUSER_B_ID=$(cat "$BATS_TMPDIR/orguser_b_id.txt")

  run curl -s -w "\n%{http_code}" -X PUT \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    -H "Content-Type: application/json" \
    -d '{"role":"admin"}' \
    "$GATEWAY/api/v1/orgs/test-org/members/${ORGUSER_B_ID}"
  assert_success
  assert_output --partial "200"
}

@test "org: verify orguser-b role is now admin" {
  run curl -sf -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/test-org/members"
  assert_success
  assert_output --partial "admin"
}

@test "org: update orguser-b role back to member" {
  ORGUSER_B_ID=$(cat "$BATS_TMPDIR/orguser_b_id.txt")

  run curl -s -w "\n%{http_code}" -X PUT \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    -H "Content-Type: application/json" \
    -d '{"role":"member"}' \
    "$GATEWAY/api/v1/orgs/test-org/members/${ORGUSER_B_ID}"
  assert_success
  assert_output --partial "200"
}

# ========================================
# Membership — Remove
# ========================================

@test "org: cannot remove last owner (orguser-a)" {
  ORGUSER_A_ID=$(cat "$BATS_TMPDIR/orguser_a_id.txt")

  run curl -s -o /dev/null -w "%{http_code}" -X DELETE \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/test-org/members/${ORGUSER_A_ID}"
  assert_success
  assert_output "400"
}

@test "org: remove orguser-b from test-org" {
  ORGUSER_B_ID=$(cat "$BATS_TMPDIR/orguser_b_id.txt")

  run curl -s -w "\n%{http_code}" -X DELETE \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/test-org/members/${ORGUSER_B_ID}"
  assert_success
  assert_output --partial "200"
}

@test "org: list members after removal returns 1 member" {
  run curl -sf -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/test-org/members"
  assert_success
  assert_output --partial "owner"
  # orguser-b should no longer appear
  [[ "$output" != *"orguser-b"* ]] || [[ "$(echo "$output" | grep -c '"member"')" -eq 0 ]]
}

@test "org: list orgs as orguser-b returns empty again" {
  run curl -sf -H "X-API-Key: $ORGUSER_B_KEY" \
    "$GATEWAY/api/v1/orgs"
  assert_success
  assert_output "[]"
}

@test "org: re-add orguser-b as member for subsequent tests" {
  ORGUSER_B_ID=$(cat "$BATS_TMPDIR/orguser_b_id.txt")

  run curl -s -w "\n%{http_code}" -X POST \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"userId\":\"${ORGUSER_B_ID}\",\"role\":\"member\"}" \
    "$GATEWAY/api/v1/orgs/test-org/members"
  assert_success
  assert_output --partial "200"
}

# ========================================
# Organization Delete
# ========================================

@test "org: create delete-me-org" {
  run curl -s -w "\n%{http_code}" -X POST \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    -H "Content-Type: application/json" \
    -d '{"name":"delete-me-org"}' \
    "$GATEWAY/api/v1/orgs"
  assert_success
  assert_output --partial "delete-me-org"
  assert_output --partial "201"
}

@test "org: delete as non-owner returns 403" {
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE \
    -H "X-API-Key: $ORGUSER_B_KEY" \
    "$GATEWAY/api/v1/orgs/delete-me-org"
  assert_success
  assert_output "403"
}

@test "org: delete as owner returns 200" {
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/delete-me-org"
  assert_success
  assert_output "200"
}

@test "org: verify delete-me-org is gone (404)" {
  run curl -s -o /dev/null -w "%{http_code}" \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/delete-me-org"
  assert_success
  assert_output "404"
}

@test "org: delete nonexistent org returns 404" {
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/nonexistent-org"
  assert_success
  assert_output "404"
}

# ========================================
# d3 CLI — org list
# ========================================

@test "org: d3 org list as orguser-a shows test-org" {
  run env DATADATDAT_API_KEY="$ORGUSER_A_KEY" "$D3" org list --server "$GATEWAY"
  assert_success
  assert_output --partial "test-org"
}

@test "org: d3 org list as orguser-b shows test-org (member)" {
  run env DATADATDAT_API_KEY="$ORGUSER_B_KEY" "$D3" org list --server "$GATEWAY"
  assert_success
  assert_output --partial "test-org"
}

@test "org: d3 org ls alias works" {
  run env DATADATDAT_API_KEY="$ORGUSER_A_KEY" "$D3" org ls --server "$GATEWAY"
  assert_success
  assert_output --partial "test-org"
}

@test "org: d3 auth logout clears stored credentials" {
  # First, store a credential so we can verify logout removes it
  run "$D3" auth login --server "$GATEWAY" --api-key "$ORGUSER_A_KEY"
  assert_success
  run "$D3" auth logout --server "$GATEWAY"
  assert_success
}

@test "org: d3 org list without auth returns error" {
  run env -u DATADATDAT_API_KEY "$D3" org list --server "$GATEWAY"
  assert_failure
}

# ========================================
# Cleanup
# ========================================

@test "org: cleanup - delete test-org via API" {
  run curl -s -X DELETE -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/test-org"
  assert_success
}

@test "org: cleanup - verify test-org deleted" {
  run curl -s -o /dev/null -w "%{http_code}" \
    -H "X-API-Key: $ORGUSER_A_KEY" \
    "$GATEWAY/api/v1/orgs/test-org"
  assert_success
  assert_output "404"
}

@test "org: cleanup - delete API keys for test users" {
  run run_sql_cmd \
    "DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('orguser-a', 'orguser-b'));"
  assert_success
  assert_output --partial "DELETE"
}

@test "org: cleanup - delete whitelisted_users entries" {
  run run_sql_cmd \
    "DELETE FROM whitelisted_users WHERE user_id IN (SELECT id FROM users WHERE github_login IN ('orguser-a', 'orguser-b'));"
  assert_success
  assert_output --partial "DELETE"
}

@test "org: cleanup - delete namespaces for test users" {
  run run_sql_cmd \
    "DELETE FROM namespaces WHERE name IN ('orguser-a', 'orguser-b', 'test-org', 'delete-me-org');"
  assert_success
  assert_output --partial "DELETE"
}

@test "org: cleanup - delete test users" {
  run run_sql_cmd \
    "DELETE FROM users WHERE github_login IN ('orguser-a', 'orguser-b');"
  assert_success
  assert_output --partial "DELETE"
}

@test "org: cleanup - verify all test users removed" {
  run run_sql_raw \
    "SELECT COUNT(*) FROM users WHERE github_login IN ('orguser-a', 'orguser-b');"
  assert_success
  assert_output "0"
}

@test "org: cleanup - verify no test orgs remain" {
  run run_sql_raw \
    "SELECT COUNT(*) FROM organizations WHERE name IN ('test-org', 'delete-me-org');"
  assert_success
  assert_output "0"
}
