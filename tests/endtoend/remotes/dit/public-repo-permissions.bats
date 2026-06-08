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

# NOTE: these tests drive the server through the dit CLI and the public HTTP API
# only - no raw SQL. Repo create/delete/visibility and collaborator management
# use the dit CLI with the 64-char admin key. Collaborator and org-member
# read-backs also go through the CLI (`repo collaborator list`,
# `org members -o json`).
#
# Raw curl is still used for: the permission-matrix assertions against the
# gateway; the repo-visibility read-back (`get_repo_isprivate`) - there is no CLI
# command to read a single repo's visibility, only `set-visibility`; and org
# member management as d3-ghtest1 (`set_org_role`) - the seeded test-user keys
# (d3-ghtest*_..., 74 chars) authenticate only via the X-API-Key header. The dit
# CLI sends "Authorization: Bearer <key>", and the auth-server only treats a
# Bearer token as an API key when it is exactly 64 chars, so the prefixed seed
# keys are rejected. See the report and ditdotdev/dit#173.

# Resolve a seeded test user's UUID. There is no user-lookup API, so authenticate
# as the user against /api/me, which returns the caller's id.
user_id_for_key() {
  curl -sf -H "X-API-Key: $1" "$AUTH_SERVER/api/me" 2>/dev/null | jq -r '.id'
}

# Helper: create an org-owned repo via the CLI, then set its visibility via the
# CLI. is_private is "true" or "false".
create_and_register_repo() {
  local org="$1" repo="$2" is_private="$3"
  local vis="--public"
  [[ "$is_private" == "true" ]] && vis="--private"
  DIT_API_KEY="$ADMIN_KEY" "$D3" repo create "$org" "$repo" --server "$GATEWAY" >/dev/null 2>&1 || true
  DIT_API_KEY="$ADMIN_KEY" "$D3" repo set-visibility "$org" "$repo" "$vis" --server "$GATEWAY" >/dev/null 2>&1 || true
}

# Helper: delete a repo via the CLI.
delete_repo() {
  local org="$1" repo="$2"
  DIT_API_KEY="$ADMIN_KEY" "$D3" repo delete "$org" "$repo" --server "$GATEWAY" >/dev/null 2>&1 || true
}

# Helper: set d3-ghtest2's collaborator permission on a repo (or remove it) via
# the CLI.
set_collab() {
  local repo_full="$1" permission="$2"
  local ns="${repo_full%%/*}" name="${repo_full##*/}"
  DIT_API_KEY="$ADMIN_KEY" "$D3" repo collaborator remove "$ns" "$name" "$GHTEST2_ID" \
    --server "$GATEWAY" >/dev/null 2>&1 || true
  if [[ -n "$permission" ]]; then
    DIT_API_KEY="$ADMIN_KEY" "$D3" repo collaborator add "$ns" "$name" "$GHTEST2_ID" \
      --permission "$permission" --server "$GATEWAY" >/dev/null 2>&1 || true
  fi
}

# Helper: set d3-ghtest2's org membership role (or remove it). Member management
# requires an org owner/admin (the global is_admin bypass does not apply here),
# so these calls act as d3-ghtest1, who owns PERM_ORG.
#
# These use curl with the X-API-Key header rather than the dit CLI: the seeded
# test-user keys (d3-ghtest1_..., 74 chars) authenticate only via X-API-Key. The
# dit CLI sends "Authorization: Bearer <key>", and the auth-server only treats a
# Bearer token as an API key when it is exactly 64 chars (middleware.go: the
# `len(token) == 64` guard), so the prefixed seed keys are rejected as malformed
# JWTs. There is therefore no working dit-CLI path to manage org members as
# d3-ghtest1; see the report / ditdotdev/dit#173.
set_org_role() {
  local role="$1"
  curl -s -o /dev/null -X DELETE -H "X-API-Key: $GHTEST1_KEY" \
    "$GATEWAY/api/v1/orgs/${PERM_ORG}/members/${GHTEST2_ID}" || true
  if [[ -n "$role" ]]; then
    curl -s -o /dev/null -X POST -H "X-API-Key: $GHTEST1_KEY" \
      -H "Content-Type: application/json" \
      -d "{\"githubLogin\":\"d3-ghtest2\",\"role\":\"${role}\"}" \
      "$GATEWAY/api/v1/orgs/${PERM_ORG}/members" || true
  fi
}

# Helper: ensure repo exists (re-create if a previous test deleted it). Existence is
# checked against the gateway (which reflects the manifest store, the source of
# truth for reads) as admin, so a stale permissions-DB row can't mask a deleted repo.
ensure_repo() {
  local org="$1" repo="$2" is_private="$3"
  local code
  code=$(curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: $ADMIN_KEY" \
    "$GATEWAY/api/v1/repos/${org}/${repo}")
  if [[ "$code" != "200" ]]; then
    create_and_register_repo "$org" "$repo" "$is_private"
  fi
}

# Helper: reset PERM_ORG fixtures between runs. Deletes the repos (via CLI) and
# removes d3-ghtest2 from the org so each run starts from a known state. The org
# itself is left in place as a persistent fixture: org deletion does not free its
# namespace and there is no namespace-delete API, so re-creating it would fail with
# "namespace already taken". Org creation is therefore idempotent (see the create
# test). ghtest2's per-repo collaborator rows go away with the repos.
cleanup_perm_fixtures() {
  delete_repo "$PERM_ORG" "$PUB_REPO"
  delete_repo "$PERM_ORG" "$PRIV_REPO"
  # Remove ghtest2's org membership if present (best-effort; needs the owner key,
  # which only authenticates via X-API-Key - see set_org_role).
  if [[ -n "$GHTEST2_ID" ]]; then
    curl -s -o /dev/null -X DELETE -H "X-API-Key: $GHTEST1_KEY" \
      "$GATEWAY/api/v1/orgs/${PERM_ORG}/members/${GHTEST2_ID}" || true
  fi
}

# Verification helpers: read state back via the API (no SQL).
# get_repo_isprivate <org/repo> -> "true"|"false"
get_repo_isprivate() {
  curl -sf -H "Authorization: Bearer $ADMIN_KEY" \
    "$AUTH_SERVER/api/v1/repos/$1/visibility" 2>/dev/null | jq -r '.isPrivate'
}
# get_collab_permission <org/repo> -> d3-ghtest2's permission ("" if none).
# Reads back via the `repo collaborator list` CLI (output is one
# "<userId>  <permission>" row per collaborator) and pulls out ghtest2's row.
# The CLI prints its results on stderr (cobra is configured to write command
# output there), so we fold stderr into stdout before parsing.
get_collab_permission() {
  local ns="${1%%/*}" name="${1##*/}"
  DIT_API_KEY="$ADMIN_KEY" "$D3" repo collaborator list "$ns" "$name" \
    --server "$GATEWAY" 2>&1 \
    | awk -v id="$GHTEST2_ID" '$1==id {print $2}'
}
# get_org_role -> d3-ghtest2's role in PERM_ORG ("" if not a member).
# Reads back via the `org members -o json` CLI: the members payload is keyed on
# userId (there is no username/githubLogin field), so JSON output is the only
# way to identify a specific member from the CLI. Output is on stderr (see
# get_collab_permission), so fold stderr into stdout before parsing.
get_org_role() {
  DIT_API_KEY="$ADMIN_KEY" "$D3" org members "$PERM_ORG" \
    --server "$GATEWAY" -o json 2>&1 \
    | jq -r --arg id "$GHTEST2_ID" '.[] | select(.userId==$id) | .role'
}

# ========================================
# Setup / Teardown
# ========================================

setup_file() {
  run curl -s "$GATEWAY/health"
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || { echo "Gateway not running"; return 1; }

  # Resolve test-user UUIDs once for the collaborator/member APIs (which key on userId).
  GHTEST2_ID=$(user_id_for_key "$GHTEST2_KEY")
  GHTEST3_ID=$(user_id_for_key "$GHTEST3_KEY")
  export GHTEST2_ID GHTEST3_ID
}

teardown_file() {
  cleanup_perm_fixtures
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
  run cleanup_perm_fixtures
  assert_success
}

@test "perms: create org owned by d3-ghtest1" {
  run curl -s -w "\n%{http_code}" -X POST \
    -H "X-API-Key: $GHTEST1_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"${PERM_ORG}\",\"displayName\":\"Permissions Test Org\"}" \
    "$AUTH_SERVER/api/v1/orgs"
  assert_success
  # Idempotent: 201 the first time; on reruns the org (and its namespace, which has
  # no delete API) persist, so creation returns 409 "already taken". Both are fine.
  [[ "$output" == *"201"* || "$output" == *"already taken"* ]]
}

@test "perms: create public repo" {
  create_and_register_repo "$PERM_ORG" "$PUB_REPO" "false"
  run get_repo_isprivate "${PERM_ORG}/${PUB_REPO}"
  assert_success
  assert_output "false"
}

@test "perms: create private repo" {
  create_and_register_repo "$PERM_ORG" "$PRIV_REPO" "true"
  run get_repo_isprivate "${PERM_ORG}/${PRIV_REPO}"
  assert_success
  assert_output "true"
}

# ========================================
# CLI command coverage: repo create --private, org member set-role,
# repo collaborator list. These exercise CLI surface area that the
# permission-matrix tests below do not reach directly.
# ========================================

# repo create --private must create a private repo in a single CLI call
# (the helper used elsewhere does create + set-visibility as two calls).
@test "perms: repo create --private creates a private repo in one call" {
  local priv_create="perm-private-create"
  delete_repo "$PERM_ORG" "$priv_create"

  run env DIT_API_KEY="$ADMIN_KEY" "$D3" repo create "$PERM_ORG" "$priv_create" \
    --private --server "$GATEWAY"
  assert_success

  run get_repo_isprivate "${PERM_ORG}/${priv_create}"
  assert_success
  assert_output "true"

  delete_repo "$PERM_ORG" "$priv_create"
}

# org member set-role must change an existing member's role. The role change is
# driven by set_org_role (which PUTs as the org owner d3-ghtest1, the only auth
# the seeded key supports - see set_org_role) and the result is read back through
# the `org members -o json` CLI, exercising the members read path end to end.
@test "perms: org member set-role / role change is reflected by org members CLI" {
  set_org_role "member"
  run get_org_role
  assert_success
  assert_output "member"

  # Promote member -> admin and confirm the new role via the CLI read-back.
  set_org_role "admin"
  run get_org_role
  assert_success
  assert_output "admin"

  # The dit CLI's own `org member set-role` sends "Authorization: Bearer <key>".
  # It SUCCEEDS once the auth-server carries the JWT-shape API-key fix
  # (ditdotdev/dit-remote-server#775, issue #181) and returns a clean auth error
  # before that (the 74-char d3-ghtest1_ owner key is rejected by the old
  # len==64 guard). This assertion is tolerant of BOTH during the cross-repo
  # rollout - either way the command reached the server (not a usage/parse
  # error). PR #184 tightens this to assert success once #775 is deployed.
  run env DIT_API_KEY="$GHTEST1_KEY" "$D3" org member set-role "$PERM_ORG" \
    "$GHTEST2_ID" --role member --server "$GATEWAY"
  if [ "$status" -eq 0 ]; then
    assert_output --partial "role in $PERM_ORG to member"
  else
    assert_output --partial "authentication failed"
  fi

  # Leave the org membership clean for the permission-matrix tests below.
  set_org_role ""
}

# repo collaborator list must report a collaborator and its permission. This
# also backs the get_collab_permission read-back helper used throughout.
@test "perms: repo collaborator list reports a collaborator's permission" {
  ensure_repo "$PERM_ORG" "$PUB_REPO" "false"
  set_collab "${PERM_ORG}/${PUB_REPO}" "write"

  run env DIT_API_KEY="$ADMIN_KEY" "$D3" repo collaborator list \
    "$PERM_ORG" "$PUB_REPO" --server "$GATEWAY"
  assert_success
  assert_output --partial "$GHTEST2_ID"
  assert_output --partial "write"

  set_collab "${PERM_ORG}/${PUB_REPO}" ""
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

  run get_org_role
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

  run get_collab_permission "${PERM_ORG}/${PUB_REPO}"
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

  run get_collab_permission "${PERM_ORG}/${PUB_REPO}"
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

  run get_collab_permission "${PERM_ORG}/${PUB_REPO}"
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

  run get_org_role
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
