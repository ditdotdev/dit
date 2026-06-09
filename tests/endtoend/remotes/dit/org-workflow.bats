#!/usr/bin/env bats

# E2E Organization Workflow Tests for Dit Remote Server
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
    run docker exec dit-postgres pg_isready -U dit
    [[ "$output" == *"accepting connections"* ]] || { echo "Postgres not ready"; return 1; }
  else
    run run_sql_raw "SELECT 1;"
    [[ "$output" == *"1"* ]] || { echo "Postgres not ready"; return 1; }
  fi
}

teardown_file() {
  # Best-effort cleanup — API then DB
  DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org delete test-org --server "$GATEWAY" 2>/dev/null || true
  DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org delete delete-me-org --server "$GATEWAY" 2>/dev/null || true

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
  run docker exec dit-postgres pg_isready -U dit
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

@test "org: create test-org as orguser-a via CLI" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org create test-org \
    --display-name "Test Organization" --server "$GATEWAY"
  assert_success
  assert_output --partial "test-org"
}

@test "org: get test-org returns org details via CLI" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org info test-org --server "$GATEWAY"
  assert_success
  assert_output --partial "test-org"
  assert_output --partial "Test Organization"
}

@test "org: test-org members include orguser-a as owner via CLI" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org members test-org --server "$GATEWAY"
  assert_success
  assert_output --partial "owner"
}

@test "org: duplicate org name fails" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org create test-org --server "$GATEWAY"
  assert_failure
  assert_output --partial "already exists"
}

@test "org: create org without auth fails" {
  # Truly anonymous: unset the env API key AND point HOME at an empty dir so
  # the CLI can't fall back to a stored credential. A prior suite
  # (auth-status "restore auth for subsequent tests") does `dit auth login`,
  # which persists a credential in ~/.dit; without an isolated HOME the CLI
  # would use it and the create would (correctly) succeed, defeating the test.
  run env -u DIT_API_KEY HOME="$(mktemp -d)" "$D3" org create unauth-org --server "$GATEWAY"
  assert_failure
}

# ========================================
# Organization Listing
# ========================================

@test "org: list orgs as orguser-a contains test-org" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org list --server "$GATEWAY"
  assert_success
  assert_output --partial "test-org"
}

@test "org: list orgs as orguser-b returns empty (not a member)" {
  run env DIT_API_KEY="$ORGUSER_B_KEY" "$D3" org list --server "$GATEWAY"
  assert_success
  # orguser-b is not a member of any org yet
  assert_output --partial "No organizations found"
}

# ========================================
# Organization Update
# ========================================

@test "org: update test-org as owner succeeds" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org update test-org \
    --display-name "Updated Test Org" --description "E2E test org" --server "$GATEWAY"
  assert_success
  assert_output --partial "Updated organization test-org"
}

@test "org: verify test-org was updated" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org info test-org --server "$GATEWAY"
  assert_success
  assert_output --partial "Updated Test Org"
  assert_output --partial "E2E test org"
}

@test "org: update test-org as non-member fails" {
  run env DIT_API_KEY="$ORGUSER_B_KEY" "$D3" org update test-org \
    --display-name "Hijacked" --server "$GATEWAY"
  assert_failure
  assert_output --partial "forbidden"
}

@test "org: get nonexistent org fails" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org info nonexistent-org --server "$GATEWAY"
  assert_failure
  assert_output --partial "not found"
}

# ========================================
# Membership — Add
# ========================================

@test "org: add orguser-b as member of test-org" {
  ORGUSER_B_ID=$(cat "$BATS_TMPDIR/orguser_b_id.txt")

  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org member add test-org "$ORGUSER_B_ID" \
    --role member --server "$GATEWAY"
  assert_success
  assert_output --partial "Added"
}

@test "org: list members returns 2 members" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org members test-org --server "$GATEWAY"
  assert_success
  # Both orguser-a (owner) and orguser-b (member) should appear
  assert_output --partial "owner"
  assert_output --partial "member"
}

@test "org: list orgs as orguser-b now contains test-org" {
  run env DIT_API_KEY="$ORGUSER_B_KEY" "$D3" org list --server "$GATEWAY"
  assert_success
  assert_output --partial "test-org"
}

@test "org: add orguser-b again fails (already a member)" {
  ORGUSER_B_ID=$(cat "$BATS_TMPDIR/orguser_b_id.txt")

  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org member add test-org "$ORGUSER_B_ID" \
    --role member --server "$GATEWAY"
  assert_failure
  assert_output --partial "already a member"
}

@test "org: member cannot add new members" {
  ORGUSER_A_ID=$(cat "$BATS_TMPDIR/orguser_a_id.txt")

  # orguser-b is a member, not admin/owner — should be rejected
  run env DIT_API_KEY="$ORGUSER_B_KEY" "$D3" org member add test-org "$ORGUSER_A_ID" \
    --role member --server "$GATEWAY"
  assert_failure
  assert_output --partial "forbidden"
}

# ========================================
# Membership — Update Role
# ========================================

@test "org: update orguser-b role to admin" {
  ORGUSER_B_ID=$(cat "$BATS_TMPDIR/orguser_b_id.txt")

  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org member set-role test-org "$ORGUSER_B_ID" \
    --role admin --server "$GATEWAY"
  assert_success
  assert_output --partial "Updated"
}

@test "org: verify orguser-b role is now admin" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org members test-org --server "$GATEWAY"
  assert_success
  assert_output --partial "admin"
}

@test "org: update orguser-b role back to member" {
  ORGUSER_B_ID=$(cat "$BATS_TMPDIR/orguser_b_id.txt")

  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org member set-role test-org "$ORGUSER_B_ID" \
    --role member --server "$GATEWAY"
  assert_success
  assert_output --partial "Updated"
}

# ========================================
# Membership — Remove
# ========================================

@test "org: cannot remove last owner (orguser-a)" {
  ORGUSER_A_ID=$(cat "$BATS_TMPDIR/orguser_a_id.txt")

  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org member remove test-org "$ORGUSER_A_ID" \
    --server "$GATEWAY"
  assert_failure
  assert_output --partial "cannot remove"
}

@test "org: remove orguser-b from test-org" {
  ORGUSER_B_ID=$(cat "$BATS_TMPDIR/orguser_b_id.txt")

  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org member remove test-org "$ORGUSER_B_ID" \
    --server "$GATEWAY"
  assert_success
  assert_output --partial "Removed"
}

@test "org: list members after removal returns 1 member" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org members test-org --server "$GATEWAY"
  assert_success
  assert_output --partial "owner"
  # orguser-b should no longer appear as a member
  refute_output --partial "member"
}

@test "org: list orgs as orguser-b returns empty again" {
  run env DIT_API_KEY="$ORGUSER_B_KEY" "$D3" org list --server "$GATEWAY"
  assert_success
  assert_output --partial "No organizations found"
}

@test "org: re-add orguser-b as member for subsequent tests" {
  ORGUSER_B_ID=$(cat "$BATS_TMPDIR/orguser_b_id.txt")

  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org member add test-org "$ORGUSER_B_ID" \
    --role member --server "$GATEWAY"
  assert_success
  assert_output --partial "Added"
}

# ========================================
# Organization Delete
# ========================================

@test "org: create delete-me-org" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org create delete-me-org --server "$GATEWAY"
  assert_success
  assert_output --partial "delete-me-org"
}

@test "org: delete as non-owner fails" {
  run env DIT_API_KEY="$ORGUSER_B_KEY" "$D3" org delete delete-me-org --server "$GATEWAY"
  assert_failure
  assert_output --partial "forbidden"
}

@test "org: delete as owner succeeds" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org delete delete-me-org --server "$GATEWAY"
  assert_success
  assert_output --partial "Deleted"
}

@test "org: verify delete-me-org is gone" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org info delete-me-org --server "$GATEWAY"
  assert_failure
  assert_output --partial "not found"
}

@test "org: delete nonexistent org fails" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org delete nonexistent-org --server "$GATEWAY"
  assert_failure
  assert_output --partial "not found"
}

# ========================================
# dit CLI — org list
# ========================================

@test "org: dit org list as orguser-a shows test-org" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org list --server "$GATEWAY"
  assert_success
  assert_output --partial "test-org"
}

@test "org: dit org list as orguser-b shows test-org (member)" {
  run env DIT_API_KEY="$ORGUSER_B_KEY" "$D3" org list --server "$GATEWAY"
  assert_success
  assert_output --partial "test-org"
}

@test "org: dit org ls alias works" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org ls --server "$GATEWAY"
  assert_success
  assert_output --partial "test-org"
}

@test "org: dit auth logout clears stored credentials" {
  # First, store a credential so we can verify logout removes it
  run "$D3" auth login --server "$GATEWAY" --api-key "$ORGUSER_A_KEY"
  assert_success
  run "$D3" auth logout --server "$GATEWAY"
  assert_success
}

@test "org: dit org list without auth returns error" {
  run env -u DIT_API_KEY "$D3" org list --server "$GATEWAY"
  assert_failure
}

# ========================================
# Cleanup
# ========================================

@test "org: cleanup - delete test-org via CLI" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org delete test-org --server "$GATEWAY"
  assert_success
}

@test "org: cleanup - verify test-org deleted" {
  run env DIT_API_KEY="$ORGUSER_A_KEY" "$D3" org info test-org --server "$GATEWAY"
  assert_failure
  assert_output --partial "not found"
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
