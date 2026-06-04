#!/usr/bin/env bats

# E2E Tests: Repository Permissions Matrix
# Tests every combination of user class x repo visibility x operation (read/write/delete).
#
# User classes tested:
#   - Unauthenticated
#   - Authenticated (no relation to repo)
#   - Org member (member role)
#   - Collaborator (read)
#   - Collaborator (write)
#   - Collaborator (admin)
#   - Org admin (admin role)
#   - Org owner (owner role)
#   - System admin (is_admin=true)
#
# Repo types: public, private
# Operations: read (GET), write (POST), delete (DELETE)
#
# Related: https://github.com/ditdotdev/dit-remote-server/issues/450

load '../../test_helper'
load 'env'

# Test user API keys (seeded by 015-seed-e2e-test-users.xml)
GHTEST1_KEY="d3ghtest1_aaaa1111bbbb2222cccc3333dddd4444eeee5555ffff6666aaaa1111bbbb"
GHTEST2_KEY="d3ghtest2_aaaa1111bbbb2222cccc3333dddd4444eeee5555ffff6666aaaa1111bbbb"
GHTEST3_KEY="d3ghtest3_aaaa1111bbbb2222cccc3333dddd4444eeee5555ffff6666aaaa1111bbbb"
ADMIN_KEY="${DIT_API_KEY}"

PERM_ORG="permtest-org"
PUB_REPO="perm-public"
PRIV_REPO="perm-private"

# Helper: create a repo in the manifest store and register it in the permissions DB
create_and_register_repo() {
  local org="$1" repo="$2" is_private="$3"
  curl -sf -X POST -H "Authorization: Bearer $ADMIN_KEY" \
    "$GATEWAY/api/v1/repos/${org}/${repo}" >/dev/null 2>&1 || true
  run_sql_cmd \
    "INSERT INTO repositories (namespace, name, full_name, is_private, owner_type, owner_id, created_by)
     SELECT '${org}', '${repo}', '${org}/${repo}', ${is_private},
            'organization', o.id, (SELECT id FROM users WHERE github_login = 'd3-ghtest1')
     FROM organizations o WHERE o.name = '${org}'
     ON CONFLICT (full_name) DO UPDATE SET is_private = ${is_private};" >/dev/null 2>&1
}

# Helper: delete a repo from manifest store and permissions DB
delete_repo() {
  local org="$1" repo="$2"
  curl -sf -X DELETE -H "Authorization: Bearer $ADMIN_KEY" \
    "$GATEWAY/api/v1/repos/${org}/${repo}" >/dev/null 2>&1 || true
  run_sql_cmd "DELETE FROM repositories WHERE full_name = '${org}/${repo}';" >/dev/null 2>&1 || true
}

# Helper: set d3-ghtest2's collaborator permission on a repo (or remove it)
set_collab() {
  local repo_full="$1" permission="$2"
  run_sql_cmd "DELETE FROM repo_collaborators WHERE repo_id = (SELECT id FROM repositories WHERE full_name = '${repo_full}') AND user_id = (SELECT id FROM users WHERE github_login = 'd3-ghtest2');" >/dev/null 2>&1
  if [[ -n "$permission" ]]; then
    run_sql_cmd "INSERT INTO repo_collaborators (repo_id, user_id, permission) VALUES ((SELECT id FROM repositories WHERE full_name = '${repo_full}'), (SELECT id FROM users WHERE github_login = 'd3-ghtest2'), '${permission}');" >/dev/null 2>&1
  fi
}

# Helper: set d3-ghtest2's org membership role (or remove it)
set_org_role() {
  local role="$1"
  run_sql_cmd "DELETE FROM org_memberships WHERE org_id = (SELECT id FROM organizations WHERE name = '${PERM_ORG}') AND user_id = (SELECT id FROM users WHERE github_login = 'd3-ghtest2');" >/dev/null 2>&1
  if [[ -n "$role" ]]; then
    run_sql_cmd "INSERT INTO org_memberships (org_id, user_id, role) VALUES ((SELECT id FROM organizations WHERE name = '${PERM_ORG}'), (SELECT id FROM users WHERE github_login = 'd3-ghtest2'), '${role}');" >/dev/null 2>&1
  fi
}

# Helper: ensure repo exists (re-create if a previous test's bug deleted it)
ensure_repo() {
  local org="$1" repo="$2" is_private="$3"
  local count
  count=$(run_sql_raw "SELECT COUNT(*) FROM repositories WHERE full_name = '${org}/${repo}';" | tr -d '[:space:]')
  if [[ "$count" != "1" ]]; then
    create_and_register_repo "$org" "$repo" "$is_private"
  fi
}

# ========================================
# Setup / Teardown
# ========================================

setup_file() {
  run curl -s "$GATEWAY/health"
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || { echo "Gateway not running"; return 1; }

  if is_dev; then
    run docker exec dit-postgres pg_isready -U dit
    [[ "$output" == *"accepting connections"* ]] || { echo "Postgres not ready"; return 1; }
  fi
}

teardown_file() {
  curl -sf -X DELETE -H "Authorization: Bearer $ADMIN_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}" 2>/dev/null || true
  curl -sf -X DELETE -H "Authorization: Bearer $ADMIN_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}" 2>/dev/null || true

  run_sql_cmd \
    "DELETE FROM repo_collaborators WHERE repo_id IN (SELECT id FROM repositories WHERE namespace = '${PERM_ORG}');
     DELETE FROM repositories WHERE namespace = '${PERM_ORG}';
     DELETE FROM org_memberships WHERE org_id IN (SELECT id FROM organizations WHERE name = '${PERM_ORG}');
     DELETE FROM organizations WHERE name = '${PERM_ORG}';
     DELETE FROM namespaces WHERE name = '${PERM_ORG}';" 2>/dev/null || true
}

# ========================================
# Infrastructure
# ========================================

@test "perms: verify gateway is running" {
  run curl -s "$GATEWAY/health"
  assert_success
  assert_output --partial "${HEALTH_EXPECT}"
}

@test "perms: verify all test user API keys work" {
  for key in "$GHTEST1_KEY" "$GHTEST2_KEY" "$GHTEST3_KEY"; do
    run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $key" "$GATEWAY/api/v1/repos/"
    assert_output "200"
  done
}

# ========================================
# Fixture: Create org and repos
# ========================================

@test "perms: cleanup previous test data" {
  run run_sql_cmd \
    "DELETE FROM repo_collaborators WHERE repo_id IN (SELECT id FROM repositories WHERE namespace = '${PERM_ORG}');
     DELETE FROM repositories WHERE namespace = '${PERM_ORG}';
     DELETE FROM org_memberships WHERE org_id IN (SELECT id FROM organizations WHERE name = '${PERM_ORG}');
     DELETE FROM organizations WHERE name = '${PERM_ORG}';
     DELETE FROM namespaces WHERE name = '${PERM_ORG}';"
  assert_success
}

@test "perms: create org owned by d3-ghtest1" {
  run curl -s -w "\n%{http_code}" -X POST \
    -H "X-API-Key: $GHTEST1_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"${PERM_ORG}\",\"displayName\":\"Permissions Test Org\"}" \
    "$AUTH_SERVER/api/v1/orgs"
  assert_success
  assert_output --partial "201"
}

@test "perms: create public repo" {
  create_and_register_repo "$PERM_ORG" "$PUB_REPO" "false"
  run run_sql_raw "SELECT is_private FROM repositories WHERE full_name = '${PERM_ORG}/${PUB_REPO}';"
  assert_success
  assert_output "f"
}

@test "perms: create private repo" {
  create_and_register_repo "$PERM_ORG" "$PRIV_REPO" "true"
  run run_sql_raw "SELECT is_private FROM repositories WHERE full_name = '${PERM_ORG}/${PRIV_REPO}';"
  assert_success
  assert_output "t"
}

# ========================================
# 1. UNAUTHENTICATED USER
# ========================================

@test "perms: unauth - CAN read public repo" {
  run curl -s -o /dev/null -w "%{http_code}" "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: unauth - CANNOT read private repo" {
  run curl -s -o /dev/null -w "%{http_code}" "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  [[ "$output" == "401" || "$output" == "404" ]]
}

@test "perms: unauth - CANNOT write public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}/commits"
  assert_success
  [[ "$output" == "401" || "$output" == "404" ]]
}

@test "perms: unauth - CANNOT delete public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  [[ "$output" == "401" || "$output" == "404" ]]
}

# ========================================
# 2. AUTHENTICATED (no relation to repo) - d3-ghtest3
# ========================================

@test "perms: authed-stranger - CAN read public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST3_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: authed-stranger - CANNOT read private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST3_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "404"
}

@test "perms: authed-stranger - CANNOT write public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST3_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}/commits"
  assert_success
  assert_output "404"
}

@test "perms: authed-stranger - CANNOT delete public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST3_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "404"
}

@test "perms: authed-stranger - CANNOT write private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST3_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}/commits"
  assert_success
  assert_output "404"
}

@test "perms: authed-stranger - CANNOT delete private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST3_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "404"
}

# ========================================
# 3. ORG MEMBER (member role) - d3-ghtest2
# ========================================

@test "perms: org-member - setup role" {
  set_collab "${PERM_ORG}/${PUB_REPO}" ""
  set_collab "${PERM_ORG}/${PRIV_REPO}" ""
  set_org_role "member"

  run run_sql_raw "SELECT role FROM org_memberships WHERE org_id = (SELECT id FROM organizations WHERE name = '${PERM_ORG}') AND user_id = (SELECT id FROM users WHERE github_login = 'd3-ghtest2');"
  assert_success
  assert_output "member"
}

@test "perms: org-member - CAN read public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: org-member - CAN read private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: org-member - CANNOT write public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST2_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}/commits"
  assert_success
  assert_output "404"
}

@test "perms: org-member - CANNOT write private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST2_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}/commits"
  assert_success
  assert_output "404"
}

@test "perms: org-member - CANNOT delete public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "404"
}

@test "perms: org-member - CANNOT delete private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "404"
}

# ========================================
# 4. COLLABORATOR (read) - d3-ghtest2
# ========================================

@test "perms: collab-read - setup role" {
  set_org_role ""
  set_collab "${PERM_ORG}/${PUB_REPO}" "read"
  set_collab "${PERM_ORG}/${PRIV_REPO}" "read"

  run run_sql_raw "SELECT permission FROM repo_collaborators WHERE repo_id = (SELECT id FROM repositories WHERE full_name = '${PERM_ORG}/${PUB_REPO}') AND user_id = (SELECT id FROM users WHERE github_login = 'd3-ghtest2');"
  assert_success
  assert_output "read"
}

@test "perms: collab-read - CAN read public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: collab-read - CAN read private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: collab-read - CANNOT write public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST2_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}/commits"
  assert_success
  assert_output "404"
}

@test "perms: collab-read - CANNOT write private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST2_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}/commits"
  assert_success
  assert_output "404"
}

@test "perms: collab-read - CANNOT delete public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "404"
}

@test "perms: collab-read - CANNOT delete private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "404"
}

# ========================================
# 5. COLLABORATOR (write) - d3-ghtest2
# ========================================

@test "perms: collab-write - setup role" {
  set_collab "${PERM_ORG}/${PUB_REPO}" "write"
  set_collab "${PERM_ORG}/${PRIV_REPO}" "write"

  run run_sql_raw "SELECT permission FROM repo_collaborators WHERE repo_id = (SELECT id FROM repositories WHERE full_name = '${PERM_ORG}/${PUB_REPO}') AND user_id = (SELECT id FROM users WHERE github_login = 'd3-ghtest2');"
  assert_success
  assert_output "write"
}

@test "perms: collab-write - CAN read public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: collab-write - CAN read private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: collab-write - CAN write public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST2_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}/commits"
  assert_success
  # 400 = permission granted, but empty commit body rejected by backend
  [[ "$output" != "404" ]]
}

@test "perms: collab-write - CAN write private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST2_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}/commits"
  assert_success
  [[ "$output" != "404" ]]
}

@test "perms: collab-write - BUG CANNOT delete public repo" {
  # BUG (issue #450): gateway uses CanWriteRepo for DELETE instead of CanManageRepo
  ensure_repo "$PERM_ORG" "$PUB_REPO" "false"
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "404"
}

@test "perms: collab-write - BUG CANNOT delete private repo" {
  # BUG (issue #450): same as above for private repos
  ensure_repo "$PERM_ORG" "$PRIV_REPO" "true"
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "404"
}

# ========================================
# 6. COLLABORATOR (admin) - d3-ghtest2
# ========================================

@test "perms: collab-admin - setup role" {
  ensure_repo "$PERM_ORG" "$PUB_REPO" "false"
  ensure_repo "$PERM_ORG" "$PRIV_REPO" "true"
  set_collab "${PERM_ORG}/${PUB_REPO}" "admin"
  set_collab "${PERM_ORG}/${PRIV_REPO}" "admin"

  run run_sql_raw "SELECT permission FROM repo_collaborators WHERE repo_id = (SELECT id FROM repositories WHERE full_name = '${PERM_ORG}/${PUB_REPO}') AND user_id = (SELECT id FROM users WHERE github_login = 'd3-ghtest2');"
  assert_success
  assert_output "admin"
}

@test "perms: collab-admin - CAN read public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: collab-admin - CAN read private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: collab-admin - CAN write public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST2_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}/commits"
  assert_success
  # 400 = permission granted, but empty commit body rejected by backend
  [[ "$output" != "404" ]]
}

@test "perms: collab-admin - CAN write private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST2_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}/commits"
  assert_success
  [[ "$output" != "404" ]]
}

@test "perms: collab-admin - CAN delete public repo" {
  ensure_repo "$PERM_ORG" "$PUB_REPO" "false"
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: collab-admin - CAN delete private repo" {
  ensure_repo "$PERM_ORG" "$PRIV_REPO" "true"
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "200"
}

# ========================================
# 7. ORG ADMIN (admin role) - d3-ghtest2
# ========================================

@test "perms: org-admin - setup role" {
  ensure_repo "$PERM_ORG" "$PUB_REPO" "false"
  ensure_repo "$PERM_ORG" "$PRIV_REPO" "true"
  set_collab "${PERM_ORG}/${PUB_REPO}" ""
  set_collab "${PERM_ORG}/${PRIV_REPO}" ""
  set_org_role "admin"

  run run_sql_raw "SELECT role FROM org_memberships WHERE org_id = (SELECT id FROM organizations WHERE name = '${PERM_ORG}') AND user_id = (SELECT id FROM users WHERE github_login = 'd3-ghtest2');"
  assert_success
  assert_output "admin"
}

@test "perms: org-admin - CAN read public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: org-admin - CAN read private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: org-admin - CAN write public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST2_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}/commits"
  assert_success
  # 400 = permission granted, but empty commit body rejected by backend
  [[ "$output" != "404" ]]
}

@test "perms: org-admin - CAN write private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST2_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}/commits"
  assert_success
  [[ "$output" != "404" ]]
}

@test "perms: org-admin - CAN delete public repo" {
  ensure_repo "$PERM_ORG" "$PUB_REPO" "false"
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: org-admin - CAN delete private repo" {
  ensure_repo "$PERM_ORG" "$PRIV_REPO" "true"
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "200"
}

# ========================================
# 8. ORG OWNER - d3-ghtest1
# ========================================

@test "perms: org-owner - setup repos" {
  ensure_repo "$PERM_ORG" "$PUB_REPO" "false"
  ensure_repo "$PERM_ORG" "$PRIV_REPO" "true"
  # d3-ghtest2 cleanup
  set_collab "${PERM_ORG}/${PUB_REPO}" ""
  set_collab "${PERM_ORG}/${PRIV_REPO}" ""
  set_org_role ""
}

@test "perms: org-owner - CAN read public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST1_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: org-owner - CAN read private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $GHTEST1_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: org-owner - CAN write public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST1_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}/commits"
  assert_success
  # 400 = permission granted, but empty commit body rejected by backend
  [[ "$output" != "404" ]]
}

@test "perms: org-owner - CAN write private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $GHTEST1_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}/commits"
  assert_success
  [[ "$output" != "404" ]]
}

@test "perms: org-owner - CAN delete public repo" {
  ensure_repo "$PERM_ORG" "$PUB_REPO" "false"
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST1_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: org-owner - CAN delete private repo" {
  ensure_repo "$PERM_ORG" "$PRIV_REPO" "true"
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $GHTEST1_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "200"
}

# ========================================
# 9. SYSTEM ADMIN (is_admin=true) - robertericreeves via admin API key
# ========================================

@test "perms: sysadmin - setup repos" {
  ensure_repo "$PERM_ORG" "$PUB_REPO" "false"
  ensure_repo "$PERM_ORG" "$PRIV_REPO" "true"
}

@test "perms: sysadmin - CAN read public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $ADMIN_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: sysadmin - CAN read private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $ADMIN_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: sysadmin - CAN write public repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $ADMIN_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}/commits"
  assert_success
  # 400 = permission granted, but empty commit body rejected by backend
  [[ "$output" != "404" ]]
}

@test "perms: sysadmin - CAN write private repo" {
  run curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: $ADMIN_KEY" -H "Content-Type: application/json" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}/commits"
  assert_success
  [[ "$output" != "404" ]]
}

@test "perms: sysadmin - CAN delete public repo" {
  ensure_repo "$PERM_ORG" "$PUB_REPO" "false"
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $ADMIN_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "200"
}

@test "perms: sysadmin - CAN delete private repo" {
  ensure_repo "$PERM_ORG" "$PRIV_REPO" "true"
  run curl -s -o /dev/null -w "%{http_code}" -X DELETE -H "X-API-Key: $ADMIN_KEY" \
    "$GATEWAY/api/v1/repos/${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "200"
}
