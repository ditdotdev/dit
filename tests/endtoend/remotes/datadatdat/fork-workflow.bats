#!/usr/bin/env bats

# Tests for repository fork functionality.
# Verifies:
#   - Fork a public repo via API
#   - Forked repo contains all commits from source
#   - Clone forked repo and verify data integrity
#   - Fork with custom name
#   - Cannot fork to same namespace with duplicate name
#   - Forked repo is independently modifiable

# Load shared test helpers
load '../../test_helper'

setup_file() {
  export DATADATDAT_API_KEY=02b31569a9052bc4b3cf1c3819d4fc048d34c96eca21f2b8e2359b5ecdfec93a

  # Verify the server is healthy
  run curl -s http://127.0.0.1:8080/health
  assert_success
  [[ "$output" == *"healthy"* ]] || {
    echo "datadatdat-remote-server is not running or not healthy"
    return 1
  }
}

teardown_file() {
  # Best effort cleanup
  "$D3" rm -f forksrc 2>/dev/null || true
  "$D3" rm -f forkclone 2>/dev/null || true
  "$D3" rm -f forkclone2 2>/dev/null || true
  curl -X DELETE -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/forktest/source-repo" 2>/dev/null || true
  curl -X DELETE -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/forkdest/source-repo" 2>/dev/null || true
  curl -X DELETE -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/forkdest/custom-name" 2>/dev/null || true
}

# ===== Setup: create and populate a source repo =====

@test "fork: create source repository" {
  run curl -X POST -f -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/forktest/source-repo"
  assert_success
  assert_output --partial "source-repo"
}

@test "fork: run postgres container for source" {
  run "$D3" run -n forksrc -P postgres -e POSTGRES_PASSWORD=postgres
  assert_success
  assert_output --partial "Running controlled container forksrc"
}

@test "fork: wait for source postgres to be ready" {
  run bash -c "for i in {1..18}; do docker exec forksrc pg_isready && break || sleep 5; done"
  assert_success
  assert_output --partial "accepting connections"
}

@test "fork: create first commit (empty database)" {
  run "$D3" commit -m "Initial empty database" forksrc
  assert_success
  assert_output --partial "Commit"

  COMMIT_1=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$COMMIT_1" > "$BATS_TMPDIR/fork_commit_1.txt"
}

@test "fork: add remote to source" {
  run "$D3" remote add http://datadatdat-api-gateway:8080/forktest/source-repo forksrc
  assert_success
}

@test "fork: push first commit" {
  run "$D3" push forksrc
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "fork: add table and create second commit" {
  run docker exec forksrc psql -U postgres -c \
    "CREATE TABLE products (id SERIAL PRIMARY KEY, name VARCHAR(100), price DECIMAL(10,2));"
  assert_success

  run "$D3" commit -m "Added products table" forksrc
  assert_success

  COMMIT_2=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$COMMIT_2" > "$BATS_TMPDIR/fork_commit_2.txt"
}

@test "fork: push second commit" {
  run "$D3" push forksrc
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "fork: insert data and create third commit" {
  run docker exec forksrc psql -U postgres -c \
    "INSERT INTO products (name, price) VALUES ('Widget', 9.99), ('Gadget', 19.99);"
  assert_success

  run "$D3" commit -m "Added sample products" forksrc
  assert_success

  COMMIT_3=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$COMMIT_3" > "$BATS_TMPDIR/fork_commit_3.txt"
}

@test "fork: push third commit" {
  run "$D3" push forksrc
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "fork: verify 3 commits in source remote" {
  COMMIT_1=$(cat "$BATS_TMPDIR/fork_commit_1.txt")
  COMMIT_2=$(cat "$BATS_TMPDIR/fork_commit_2.txt")
  COMMIT_3=$(cat "$BATS_TMPDIR/fork_commit_3.txt")

  run "$D3" remote log forksrc
  assert_success
  assert_output --partial "$COMMIT_1"
  assert_output --partial "$COMMIT_2"
  assert_output --partial "$COMMIT_3"
}

# ===== Test: fork via API =====

@test "fork: fork repo to different namespace via API" {
  run curl -X POST -f \
    -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"targetNamespace": "forkdest"}' \
    "http://127.0.0.1:8080/api/v1/repos/forktest/source-repo/fork"
  assert_success
  assert_output --partial "forkdest"
  assert_output --partial "source-repo"
  assert_output --partial "forkedFrom"
}

@test "fork: verify forked repo has all commits" {
  run curl -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/forkdest/source-repo/commits"
  assert_success

  COMMIT_1=$(cat "$BATS_TMPDIR/fork_commit_1.txt")
  COMMIT_2=$(cat "$BATS_TMPDIR/fork_commit_2.txt")
  COMMIT_3=$(cat "$BATS_TMPDIR/fork_commit_3.txt")

  assert_output --partial "$COMMIT_1"
  assert_output --partial "$COMMIT_2"
  assert_output --partial "$COMMIT_3"
}

@test "fork: clone forked repo and verify data" {
  run "$D3" clone http://datadatdat-api-gateway:8080/forkdest/source-repo -n forkclone -P
  assert_success
  assert_output --partial "checked out"
}

@test "fork: wait for cloned postgres to be ready" {
  run bash -c "for i in {1..18}; do docker exec forkclone pg_isready && break || sleep 5; done"
  assert_success
  assert_output --partial "accepting connections"
}

@test "fork: verify cloned fork has products data" {
  run docker exec forkclone psql -U postgres -c "SELECT name, price FROM products ORDER BY name;"
  assert_success
  assert_output --partial "Gadget"
  assert_output --partial "Widget"
  assert_output --partial "19.99"
}

@test "fork: cleanup clone of fork" {
  run "$D3" rm -f forkclone
  assert_success
}

# ===== Test: fork with custom name =====

@test "fork: fork with custom target name" {
  run curl -X POST -f \
    -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"targetNamespace": "forkdest", "targetName": "custom-name"}' \
    "http://127.0.0.1:8080/api/v1/repos/forktest/source-repo/fork"
  assert_success
  assert_output --partial "custom-name"
}

# ===== Test: duplicate fork fails =====

@test "fork: cannot fork to same name in same namespace" {
  run curl -X POST -s \
    -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"targetNamespace": "forkdest"}' \
    "http://127.0.0.1:8080/api/v1/repos/forktest/source-repo/fork"
  # Should fail with conflict (target already exists)
  [[ "$output" == *"exists"* ]] || [[ "$output" == *"conflict"* ]] || [[ "$output" == *"Conflict"* ]]
}

# ===== Test: fork is independent =====

@test "fork: clone fork for independence test" {
  run "$D3" clone http://datadatdat-api-gateway:8080/forkdest/source-repo -n forkclone2 -P
  assert_success
}

@test "fork: wait for independence clone to be ready" {
  run bash -c "for i in {1..18}; do docker exec forkclone2 pg_isready && break || sleep 5; done"
  assert_success
}

@test "fork: add data to fork (independent from source)" {
  run docker exec forkclone2 psql -U postgres -c \
    "INSERT INTO products (name, price) VALUES ('Fork-Only-Item', 99.99);"
  assert_success

  run "$D3" commit -m "Fork-only commit" forkclone2
  assert_success
}

@test "fork: verify source repo does not have fork-only data" {
  # Source repo's latest commit should NOT have Fork-Only-Item
  COMMIT_3=$(cat "$BATS_TMPDIR/fork_commit_3.txt")
  run "$D3" clone http://datadatdat-api-gateway:8080/forktest/source-repo -c "$COMMIT_3" -n forkclone -P
  assert_success

  run bash -c "for i in {1..18}; do docker exec forkclone pg_isready && break || sleep 5; done"
  assert_success

  run docker exec forkclone psql -U postgres -c "SELECT name FROM products ORDER BY name;"
  assert_success
  refute_output --partial "Fork-Only-Item"
}

# ===== Test: unauthenticated fork fails =====

@test "fork: unauthenticated fork fails" {
  run curl -X POST -s \
    -H "Content-Type: application/json" \
    -d '{"targetNamespace": "forkdest"}' \
    "http://127.0.0.1:8080/api/v1/repos/forktest/source-repo/fork"
  # Should fail - no auth
  [[ "$output" != *"created"* ]]
}

# ===== Cleanup =====

@test "fork: final cleanup" {
  "$D3" rm -f forkclone 2>/dev/null || true
  "$D3" rm -f forkclone2 2>/dev/null || true
  "$D3" rm -f forksrc 2>/dev/null || true

  run curl -X DELETE -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/forktest/source-repo"
  assert_success

  curl -X DELETE -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/forkdest/source-repo" 2>/dev/null || true

  curl -X DELETE -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/forkdest/custom-name" 2>/dev/null || true
}
