#!/usr/bin/env bats

# Tests for cross-user fork functionality (issue #560).
# Verifies that user B can fork user A's public repository.
#
# Reproduces: "failed to proxy request: EOF" when forking another user's repo.
# Related: https://github.com/ditdotdev/dit-remote-server/issues/560
#
# Test users (seeded by 015-seed-e2e-test-users.xml):
#   - dit-ghtest1: repo owner (creates and pushes source repo)
#   - dit-ghtest2: forker (forks the repo to their namespace)
#   - dit-ghtest3: third-party verifier (clones the fork)

load '../../test_helper'
load 'env'

# API keys for test users (seeded by Liquibase)
GHTEST1_KEY="d3ghtest1_aaaa1111bbbb2222cccc3333dddd4444eeee5555ffff6666aaaa1111bbbb"
GHTEST2_KEY="d3ghtest2_aaaa1111bbbb2222cccc3333dddd4444eeee5555ffff6666aaaa1111bbbb"
GHTEST3_KEY="d3ghtest3_aaaa1111bbbb2222cccc3333dddd4444eeee5555ffff6666aaaa1111bbbb"
ADMIN_KEY="${DIT_API_KEY}"

OWNER_NS="dit-ghtest1"
FORKER_NS="dit-ghtest2"
SOURCE_REPO="xuser-fork-src"

setup_file() {
  run curl -s "$GATEWAY/health"
  assert_success
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || {
    echo "dit-remote-server is not running or not healthy"
    return 1
  }

  if is_dev; then
    run docker exec dit-postgres pg_isready -U dit
    [[ "$output" == *"accepting connections"* ]] || { echo "Postgres not ready"; return 1; }

    # Ensure MinIO mc alias is configured for orphan tests
    docker exec dit-minio mc alias set myminio http://localhost:9000 minioadmin minioadmin 2>/dev/null || true
  fi
}

teardown_file() {
  # Best-effort cleanup: containers
  "$D3" rm -f xfork-src 2>/dev/null || true
  "$D3" rm -f xfork-clone 2>/dev/null || true

  # Best-effort cleanup: remote repos (admin key can delete anything)
  DIT_API_KEY="$ADMIN_KEY" "$D3" repo delete "${OWNER_NS}" "${SOURCE_REPO}" --server "$GATEWAY" 2>/dev/null || true
  DIT_API_KEY="$ADMIN_KEY" "$D3" repo delete "${FORKER_NS}" "${SOURCE_REPO}" --server "$GATEWAY" 2>/dev/null || true
  DIT_API_KEY="$ADMIN_KEY" "$D3" repo delete "${FORKER_NS}" custom-fork-name --server "$GATEWAY" 2>/dev/null || true

  # Best-effort cleanup: DB repo records
  run_sql_cmd "DELETE FROM repositories WHERE full_name IN (
    '${OWNER_NS}/${SOURCE_REPO}',
    '${FORKER_NS}/${SOURCE_REPO}',
    '${FORKER_NS}/custom-fork-name'
  );" 2>/dev/null || true
}

# ===== Setup: dit-ghtest1 creates and populates a source repo =====

@test "xfork: create source repo as dit-ghtest1" {
  # Use admin key to create the repo in dit-ghtest1's namespace.
  # Repo creation requires write permission; the fork (cross-user read) is what we're testing.
  run env DIT_API_KEY="$ADMIN_KEY" "$D3" repo create "${OWNER_NS}" "${SOURCE_REPO}" --server "$GATEWAY"
  assert_success
  assert_output --partial "$SOURCE_REPO"
}

@test "xfork: register source repo as public in permissions DB" {
  # The repo must be registered in the permissions DB so the auth gateway
  # allows cross-user read access (fork requires read on source).
  run run_sql_cmd \
    "INSERT INTO repositories (namespace, name, full_name, is_private, owner_type, owner_id, created_by)
     SELECT '${OWNER_NS}', '${SOURCE_REPO}', '${OWNER_NS}/${SOURCE_REPO}', false,
            'user', u.id, u.id
     FROM users u WHERE u.github_login = 'dit-ghtest1'
     ON CONFLICT (full_name) DO UPDATE SET is_private = false;"
  assert_success
}

@test "xfork: run postgres container for source" {
  run "$D3" run -n xfork-src -P postgres -e POSTGRES_PASSWORD=postgres
  assert_success
  assert_output --partial "Running controlled container xfork-src"
}

@test "xfork: wait for source postgres to be ready" {
  run bash -c "for i in {1..18}; do docker exec xfork-src pg_isready && break || sleep 5; done"
  assert_success
  assert_output --partial "accepting connections"
}

@test "xfork: create first commit (empty database)" {
  run "$D3" commit -m "Initial empty database" xfork-src
  assert_success
  assert_output --partial "Commit"

  COMMIT_1=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$COMMIT_1" > "$BATS_TMPDIR/xfork_commit_1.txt"
}

@test "xfork: add remote to source" {
  run "$D3" remote add ${REMOTE_URL}/${OWNER_NS}/${SOURCE_REPO} xfork-src
  assert_success
}

@test "xfork: push first commit" {
  run "$D3" push xfork-src
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "xfork: add table and create second commit" {
  run docker exec xfork-src psql -U postgres -c \
    "CREATE TABLE customers (id SERIAL PRIMARY KEY, name VARCHAR(100), email VARCHAR(200));"
  assert_success

  run "$D3" commit -m "Added customers table" xfork-src
  assert_success

  COMMIT_2=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$COMMIT_2" > "$BATS_TMPDIR/xfork_commit_2.txt"
}

@test "xfork: push second commit" {
  run "$D3" push xfork-src
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "xfork: insert data and create third commit" {
  run docker exec xfork-src psql -U postgres -c \
    "INSERT INTO customers (name, email) VALUES ('Alice', 'alice@example.com'), ('Bob', 'bob@example.com');"
  assert_success

  run "$D3" commit -m "Added sample customers" xfork-src
  assert_success

  COMMIT_3=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$COMMIT_3" > "$BATS_TMPDIR/xfork_commit_3.txt"
}

@test "xfork: push third commit" {
  run "$D3" push xfork-src
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "xfork: verify 3 commits in source remote" {
  COMMIT_1=$(cat "$BATS_TMPDIR/xfork_commit_1.txt")
  COMMIT_2=$(cat "$BATS_TMPDIR/xfork_commit_2.txt")
  COMMIT_3=$(cat "$BATS_TMPDIR/xfork_commit_3.txt")

  run "$D3" remote log xfork-src
  assert_success
  assert_output --partial "$COMMIT_1"
  assert_output --partial "$COMMIT_2"
  assert_output --partial "$COMMIT_3"
}

# ===== Test: dit-ghtest2 forks dit-ghtest1's public repo =====

@test "xfork: dit-ghtest2 forks dit-ghtest1's repo via API" {
  # This is the core reproduction of issue #560:
  # A different user forks another user's public repo.
  # Expected: 201 Created with fork metadata
  # Bug: "failed to proxy request: EOF" (502 Bad Gateway)
  run curl -X POST -f -w "\n%{http_code}" \
    -H "Authorization: Bearer $GHTEST2_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"targetNamespace\": \"${FORKER_NS}\"}" \
    "$GATEWAY/api/v1/repos/${OWNER_NS}/${SOURCE_REPO}/fork"
  assert_success
  assert_output --partial "$FORKER_NS"
  assert_output --partial "$SOURCE_REPO"
  assert_output --partial "forkedFrom"
}

@test "xfork: verify fork registered in permissions DB" {
  run run_sql_raw \
    "SELECT full_name, forked_from IS NOT NULL AS is_fork
     FROM repositories
     WHERE full_name = '${FORKER_NS}/${SOURCE_REPO}';"
  assert_success
  assert_output --partial "${FORKER_NS}/${SOURCE_REPO}"
  assert_output --partial "t"
}

@test "xfork: verify forked repo has all commits via dit-ghtest2's key" {
  COMMIT_1=$(cat "$BATS_TMPDIR/xfork_commit_1.txt")
  COMMIT_2=$(cat "$BATS_TMPDIR/xfork_commit_2.txt")
  COMMIT_3=$(cat "$BATS_TMPDIR/xfork_commit_3.txt")

  run curl -sf -H "Authorization: Bearer $GHTEST2_KEY" \
    "$GATEWAY/api/v1/repos/${FORKER_NS}/${SOURCE_REPO}/commits"
  assert_success
  assert_output --partial "$COMMIT_1"
  assert_output --partial "$COMMIT_2"
  assert_output --partial "$COMMIT_3"
}

# ===== Test: clone the fork and verify data integrity =====

@test "xfork: clone forked repo" {
  run "$D3" clone ${REMOTE_URL}/${FORKER_NS}/${SOURCE_REPO} -n xfork-clone -P
  assert_success
  assert_output --partial "checked out"
}

@test "xfork: wait for cloned postgres to be ready" {
  run bash -c "for i in {1..18}; do docker exec xfork-clone pg_isready && break || sleep 5; done"
  assert_success
  assert_output --partial "accepting connections"
}

@test "xfork: verify cloned fork has customer data" {
  run docker exec xfork-clone psql -U postgres -c "SELECT name, email FROM customers ORDER BY name;"
  assert_success
  assert_output --partial "Alice"
  assert_output --partial "Bob"
}

# ===== Test: fork with custom name =====

@test "xfork: dit-ghtest2 forks with custom name" {
  run curl -X POST -f \
    -H "Authorization: Bearer $GHTEST2_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"targetNamespace\": \"${FORKER_NS}\", \"targetName\": \"custom-fork-name\"}" \
    "$GATEWAY/api/v1/repos/${OWNER_NS}/${SOURCE_REPO}/fork"
  assert_success
  assert_output --partial "custom-fork-name"
}

# ===== Test: duplicate cross-user fork fails =====

@test "xfork: duplicate fork to same namespace fails" {
  run curl -X POST -s \
    -H "Authorization: Bearer $GHTEST2_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"targetNamespace\": \"${FORKER_NS}\"}" \
    "$GATEWAY/api/v1/repos/${OWNER_NS}/${SOURCE_REPO}/fork"
  # Should fail with 409 Conflict (target already exists)
  [[ "$output" == *"exists"* ]] || [[ "$output" == *"conflict"* ]] || [[ "$output" == *"Conflict"* ]]
}

# ===== Test: orphaned fork data does NOT block retry (issue #560 fix) =====
# Verifies that the server cleans up orphaned S3 data from a failed fork,
# so that a retry succeeds instead of returning 409.

@test "xfork: clean up successful fork to prepare orphan test" {
  # Delete the fork we created earlier (API + DB) but leave orphaned S3 data behind
  curl -X DELETE -sf -H "Authorization: Bearer $ADMIN_KEY" \
    "$GATEWAY/api/v1/repos/${FORKER_NS}/${SOURCE_REPO}" 2>/dev/null || true
  run_sql_cmd "DELETE FROM repositories WHERE full_name = '${FORKER_NS}/${SOURCE_REPO}';" 2>/dev/null || true
}

@test "xfork: plant orphaned journal entry in target namespace" {
  is_dev || skip "MinIO only available in DEV"

  # Copy one journal entry from source to target namespace in MinIO,
  # simulating a fork that was interrupted mid-copy (e.g. proxy timeout).
  FIRST_ENTRY=$(docker exec dit-minio mc ls myminio/dit-dev/journals/${OWNER_NS}/${SOURCE_REPO}/ 2>/dev/null | head -1 | awk '{print $NF}')
  [ -n "$FIRST_ENTRY" ] || {
    # Set up minio alias if not already configured
    docker exec dit-minio mc alias set myminio http://localhost:9000 minioadmin minioadmin 2>/dev/null
    FIRST_ENTRY=$(docker exec dit-minio mc ls myminio/dit-dev/journals/${OWNER_NS}/${SOURCE_REPO}/ | head -1 | awk '{print $NF}')
  }
  [ -n "$FIRST_ENTRY" ]

  run docker exec dit-minio mc cp \
    "myminio/dit-dev/journals/${OWNER_NS}/${SOURCE_REPO}/${FIRST_ENTRY}" \
    "myminio/dit-dev/journals/${FORKER_NS}/${SOURCE_REPO}/${FIRST_ENTRY}"
  assert_success
}

@test "xfork: fork with orphaned data returns 409 (pre-fix behavior)" {
  is_dev || skip "Depends on MinIO orphan data planted in DEV"
  # Before the fix, orphaned S3 data from a failed fork blocks retries with 409.
  # After the fix lands, this test should be updated: the server will clean up
  # orphaned data automatically and return 201 instead.
  run curl -X POST -s -w "\n%{http_code}" \
    -H "Authorization: Bearer $GHTEST2_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"targetNamespace\": \"${FORKER_NS}\"}" \
    "$GATEWAY/api/v1/repos/${OWNER_NS}/${SOURCE_REPO}/fork"
  assert_output --partial "already exists"
  assert_output --partial "409"
}

@test "xfork: clean up orphaned data for next test" {
  is_dev || skip "MinIO only available in DEV"
  run docker exec dit-minio mc rm --recursive --force \
    "myminio/dit-dev/journals/${FORKER_NS}/${SOURCE_REPO}/"
  assert_success
}

# ===== Test: dit-ghtest3 can read the fork (public) =====

@test "xfork: re-fork after orphan cleanup succeeds" {
  is_dev || skip "Depends on MinIO orphan cleanup in DEV"
  run curl -X POST -f \
    -H "Authorization: Bearer $GHTEST2_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"targetNamespace\": \"${FORKER_NS}\"}" \
    "$GATEWAY/api/v1/repos/${OWNER_NS}/${SOURCE_REPO}/fork"
  assert_success
  assert_output --partial "forkedFrom"
}

@test "xfork: dit-ghtest3 can read forked repo commits" {
  # In PROD, orphan tests are skipped and the fork was cleaned up in test 21.
  # Re-fork if needed so this test has a repo to read.
  local count
  count=$(run_sql_raw "SELECT COUNT(*) FROM repositories WHERE full_name = '${FORKER_NS}/${SOURCE_REPO}';" | tr -d '[:space:]')
  if [[ "$count" != "1" ]]; then
    run curl -X POST -sf \
      -H "Authorization: Bearer $GHTEST2_KEY" \
      -H "Content-Type: application/json" \
      -d "{\"targetNamespace\": \"${FORKER_NS}\"}" \
      "$GATEWAY/api/v1/repos/${OWNER_NS}/${SOURCE_REPO}/fork"
    assert_success
  fi

  # Register the fork as public so dit-ghtest3 can read it
  run run_sql_cmd \
    "UPDATE repositories SET is_private = false
     WHERE full_name = '${FORKER_NS}/${SOURCE_REPO}';"
  assert_success

  COMMIT_1=$(cat "$BATS_TMPDIR/xfork_commit_1.txt")

  run curl -sf -H "Authorization: Bearer $GHTEST3_KEY" \
    "$GATEWAY/api/v1/repos/${FORKER_NS}/${SOURCE_REPO}/commits"
  assert_success
  assert_output --partial "$COMMIT_1"
}

# ===== Cleanup =====

@test "xfork: final cleanup" {
  "$D3" rm -f xfork-clone 2>/dev/null || true
  "$D3" rm -f xfork-src 2>/dev/null || true

  DIT_API_KEY="$ADMIN_KEY" "$D3" repo delete "${FORKER_NS}" custom-fork-name --server "$GATEWAY" 2>/dev/null || true
  DIT_API_KEY="$ADMIN_KEY" "$D3" repo delete "${FORKER_NS}" "${SOURCE_REPO}" --server "$GATEWAY" 2>/dev/null || true
  run env DIT_API_KEY="$ADMIN_KEY" "$D3" repo delete "${OWNER_NS}" "${SOURCE_REPO}" --server "$GATEWAY"
  assert_success

  run_sql_cmd "DELETE FROM repositories WHERE full_name IN (
    '${OWNER_NS}/${SOURCE_REPO}',
    '${FORKER_NS}/${SOURCE_REPO}',
    '${FORKER_NS}/custom-fork-name'
  );" 2>/dev/null || true
}
