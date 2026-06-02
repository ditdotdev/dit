#!/usr/bin/env bats

# Tests for cloning with a specific commit ID (dit clone -c).
# Regression tests for:
#   - DatadadatRemoteServer.getCommit not extracting properties field
#   - OperationOrchestrator.findLocalCommit ClassCastException (tags List vs Map)
#   - MetadataProvider.createCommit ClassCastException (tags List vs Map)
#   - Clone.go missing return after empty remoteCommits

# Load shared test helpers
load '../../test_helper'
load 'env'

setup_file() {
  # Verify the server is healthy
  run curl -s ${GATEWAY}/health
  assert_success
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || {
    echo "dit-remote-server is not running or not healthy"
    return 1
  }
}

teardown_file() {
  # Best effort cleanup
  "$D3" rm -f clonetest 2>/dev/null || true
  "$D3" rm -f clonecommittest 2>/dev/null || true
  "$D3" rm -f clonecommittest2 2>/dev/null || true
  "$D3" rm -f cloneauthtest 2>/dev/null || true
  "$D3" repo delete clonetest clonecommit --server "${GATEWAY}" 2>/dev/null || true
}

# ===== Setup: create repo with 3 commits and push to DRS =====

@test "clone-c: create remote repository" {
  run "$D3" repo create clonetest clonecommit --server "${GATEWAY}"
  assert_success
  assert_output --partial "clonecommit"
}

@test "clone-c: run postgres container" {
  run "$D3" run -n clonetest postgres -e POSTGRES_PASSWORD=postgres
  assert_success
  assert_output --partial "Running controlled container clonetest"
}

@test "clone-c: wait for postgres to be ready" {
  run bash -c "for i in {1..18}; do docker exec clonetest pg_isready && break || sleep 5; done"
  assert_success
  assert_output --partial "accepting connections"
}

@test "clone-c: create first commit (empty database)" {
  run "$D3" commit -m "Initial empty database" clonetest
  assert_success
  assert_output --partial "Commit"

  COMMIT_1=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$COMMIT_1" > "$BATS_TMPDIR/clone_commit_1.txt"
}

@test "clone-c: add remote" {
  run "$D3" remote add ${REMOTE_URL}/clonetest/clonecommit clonetest
  assert_success
}

@test "clone-c: push first commit" {
  run "$D3" push clonetest
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "clone-c: add table and create second commit" {
  run docker exec clonetest psql -U postgres -c \
    "CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(100), email VARCHAR(100));"
  assert_success
  assert_output --partial "CREATE TABLE"

  run "$D3" commit -m "Added users table" clonetest
  assert_success
  assert_output --partial "Commit"

  COMMIT_2=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$COMMIT_2" > "$BATS_TMPDIR/clone_commit_2.txt"
}

@test "clone-c: push second commit" {
  run "$D3" push clonetest
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "clone-c: insert data and create third commit" {
  run docker exec clonetest psql -U postgres -c \
    "INSERT INTO users (name, email) VALUES ('Alice', 'alice@example.com'), ('Bob', 'bob@example.com');"
  assert_success
  assert_output --partial "INSERT"

  run "$D3" commit -m "Added sample users" clonetest
  assert_success
  assert_output --partial "Commit"

  COMMIT_3=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$COMMIT_3" > "$BATS_TMPDIR/clone_commit_3.txt"
}

@test "clone-c: push third commit" {
  run "$D3" push clonetest
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "clone-c: verify 3 commits in remote" {
  COMMIT_1=$(cat "$BATS_TMPDIR/clone_commit_1.txt")
  COMMIT_2=$(cat "$BATS_TMPDIR/clone_commit_2.txt")
  COMMIT_3=$(cat "$BATS_TMPDIR/clone_commit_3.txt")

  run "$D3" remote log clonetest
  assert_success
  assert_output --partial "$COMMIT_1"
  assert_output --partial "$COMMIT_2"
  assert_output --partial "$COMMIT_3"
}

@test "clone-c: remove local repo" {
  run "$D3" rm -f clonetest
  assert_success
  assert_output --partial "clonetest removed"
}

# ===== Test: clone with specific commit ID =====

@test "clone-c: clone latest commit by ID" {
  COMMIT_3=$(cat "$BATS_TMPDIR/clone_commit_3.txt")
  local attempt
  for attempt in 1 2 3; do
    run "$D3" clone ${REMOTE_URL}/clonetest/clonecommit -c "$COMMIT_3" -n clonecommittest
    if [[ $status -eq 0 ]]; then break; fi
    "$D3" rm -f clonecommittest 2>/dev/null || true
    sleep 3
  done
  assert_success
  assert_output --partial "checked out"
}

@test "clone-c: wait for cloned postgres (latest) to be ready" {
  run bash -c "for i in {1..18}; do docker exec clonecommittest pg_isready && break || sleep 5; done"
  assert_success
  assert_output --partial "accepting connections"
}

@test "clone-c: verify cloned data has users table with data" {
  run docker exec clonecommittest psql -U postgres -c "SELECT name FROM users ORDER BY name;"
  assert_success
  assert_output --partial "Alice"
  assert_output --partial "Bob"
}

@test "clone-c: cleanup clone of latest commit" {
  run "$D3" rm -f clonecommittest
  assert_success
  assert_output --partial "clonecommittest removed"
}

@test "clone-c: clone middle commit by ID (users table, no data)" {
  COMMIT_2=$(cat "$BATS_TMPDIR/clone_commit_2.txt")
  local attempt
  for attempt in 1 2 3; do
    run "$D3" clone ${REMOTE_URL}/clonetest/clonecommit -c "$COMMIT_2" -n clonecommittest2
    if [[ $status -eq 0 ]]; then break; fi
    "$D3" rm -f clonecommittest2 2>/dev/null || true
    sleep 3
  done
  assert_success
  assert_output --partial "checked out"
}

@test "clone-c: wait for cloned postgres (middle) to be ready" {
  run bash -c "for i in {1..18}; do docker exec clonecommittest2 pg_isready && break || sleep 5; done"
  assert_success
  assert_output --partial "accepting connections"
}

@test "clone-c: verify middle commit has empty users table" {
  run docker exec clonecommittest2 psql -U postgres -c "SELECT count(*) FROM users;"
  assert_success
  assert_output --partial "0"
}

@test "clone-c: cleanup clone of middle commit" {
  run "$D3" rm -f clonecommittest2
  assert_success
  assert_output --partial "clonecommittest2 removed"
}

# ===== Test: clone without -c still works (latest commit) =====

@test "clone-c: clone without -c gets latest commit" {
  run "$D3" clone -n clonecommittest ${REMOTE_URL}/clonetest/clonecommit
  assert_success
  assert_output --partial "checked out"
}

@test "clone-c: wait for cloned postgres to be ready" {
  run bash -c "for i in {1..18}; do docker exec clonecommittest pg_isready && break || sleep 5; done"
  assert_success
  assert_output --partial "accepting connections"
}

@test "clone-c: clone without -c has users with data" {
  run docker exec clonecommittest psql -U postgres -c "SELECT name FROM users ORDER BY name;"
  assert_success
  assert_output --partial "Alice"
  assert_output --partial "Bob"
}

# ===== Test: clone without auth shows helpful error =====

@test "clone-c: logout to clear stored credentials" {
  "$D3" auth logout --server ${REMOTE_URL} 2>/dev/null || true
}

@test "clone-c: clone without auth fails" {
  run env -u DIT_API_KEY "$D3" clone ${REMOTE_URL}/clonetest/clonecommit -n cloneauthtest
  assert_failure
}

@test "clone-c: cleanup unauthenticated clone attempt" {
  "$D3" rm -f cloneauthtest 2>/dev/null || true
}

@test "clone-c: re-login for final cleanup" {
  run "$D3" auth login --server ${REMOTE_URL} --api-key "$DIT_API_KEY"
  assert_success
}

@test "clone-c: final cleanup" {
  run "$D3" rm -f clonecommittest
  assert_success

  run "$D3" repo delete clonetest clonecommit --server "${GATEWAY}"
  assert_success
}
