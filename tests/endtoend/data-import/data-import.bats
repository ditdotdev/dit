#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# E2E Data Import Tests
# Tests dit cp (copy) and dit migrate commands

# Load shared test helpers
load '../test_helper'

# Cleanup after all tests
teardown_file() {
  "$D3" rm -f cp-test 2>/dev/null || true
  "$D3" rm -f migrate-dest 2>/dev/null || true
  docker rm -f migrate-src 2>/dev/null || true
  "$D3" rm -f migrate-bad-dest 2>/dev/null || true
  docker rm -f migrate-bad 2>/dev/null || true
}

# ========================================
# dit cp tests
# ========================================

@test "run mongo for cp test" {
  run "$D3" run -n cp-test -P mongo
  assert_success
  assert_output --partial "Running controlled container cp-test"
  sleep 5
}

@test "dit cp copies directory into repository" {
  # dit cp copies a directory's contents into a volume mount.
  # Use repo-relative path since Git Bash /tmp is not visible to Docker Desktop on Windows.
  mkdir -p "${REPO_ROOT}/cptest_data"
  echo "test content for cp" > "${REPO_ROOT}/cptest_data/cptest_file.txt"
  run "$D3" cp -s "${REPO_ROOT}/cptest_data" -d /data/configdb cp-test
  rm -rf "${REPO_ROOT}/cptest_data"
  assert_success
}

@test "docker exec confirms copied file exists in container" {
  run docker exec cp-test ls /data/configdb/cptest_file.txt
  assert_success
}

@test "dit commit after cp succeeds" {
  run "$D3" commit -m "After file copy" cp-test
  assert_success
  assert_output --partial "Commit"
}

@test "dit cp on non-existent repo fails" {
  echo "test" > "$BATS_TMPDIR/cptest_file2.txt"
  run "$D3" cp -s "$BATS_TMPDIR/cptest_file2.txt" -d /tmp/ nonexistent-cp-xyz
  assert_failure
}

@test "cleanup cp-test" {
  run "$D3" rm -f cp-test
  assert_success
}

# ========================================
# dit migrate tests
# ========================================
#
# The source is an unmanaged PostgreSQL container loaded with the Pagila
# sample schema (tests/endtoend/data-import/pagila/, PostgreSQL License).
# The schema depends on the pgvector extension, so the source image is the
# pgvector-maintained build of postgres:18. Migrating must carry the Pagila
# database across; a migrated repo that comes up as a fresh, empty cluster is
# the bug tracked in ditdotdev/dit#256.

PAGILA_SCHEMA="${BATS_TEST_DIRNAME}/pagila/pagila-schema.sql"
MIGRATE_SRC_IMAGE="pgvector/pgvector:pg18"

# Wait until PostgreSQL in the given container accepts connections.
wait_for_postgres() {
  local container="$1"
  for _ in $(seq 1 30); do
    if docker exec "$container" pg_isready -U postgres -q; then
      return 0
    fi
    sleep 2
  done
  return 1
}

@test "run unmanaged docker container for migrate" {
  run docker run -d --name migrate-src -e POSTGRES_PASSWORD=secret "$MIGRATE_SRC_IMAGE"
  assert_success
}

@test "check to see if migrate-src database is up and running" {
  run wait_for_postgres migrate-src
  assert_success
}

@test "load the pagila schema into migrate-src" {
  run docker exec migrate-src psql -U postgres -v ON_ERROR_STOP=1 -c 'CREATE DATABASE pagila;'
  assert_success
  run bash -c "docker exec -i migrate-src psql -U postgres -d pagila -v ON_ERROR_STOP=1 -q < '$PAGILA_SCHEMA'"
  assert_success
}

@test "migrate-src holds the pagila film table before migration" {
  run docker exec migrate-src psql -U postgres -d pagila -tAc "SELECT to_regclass('public.film');"
  assert_success
  assert_output "film"
}

@test "stop unmanaged container before migrate" {
  run docker stop migrate-src
  assert_success
}

@test "dit migrate captures unmanaged container" {
  run "$D3" migrate -s migrate-src migrate-dest
  assert_success
  refute_output --partial "Failed to copy data"
}

@test "dit ls shows migrated repository" {
  run "$D3" ls
  assert_success
  assert_output --partial "migrate-dest"
}

@test "dit status on migrated repo shows running" {
  run "$D3" status migrate-dest
  assert_success
  assert_output --partial "running"
}

@test "migrated repo still holds the pagila database" {
  run wait_for_postgres migrate-dest
  assert_success
  run docker exec migrate-dest psql -U postgres -tAc "SELECT datname FROM pg_database WHERE datname = 'pagila';"
  assert_success
  assert_output "pagila"
}

@test "migrated repo still holds the pagila film table" {
  run docker exec migrate-dest psql -U postgres -d pagila -tAc "SELECT to_regclass('public.film');"
  assert_success
  assert_output "film"
}

@test "dit commit on migrated repo succeeds" {
  run "$D3" commit -m "Migrated repo commit" migrate-dest
  assert_success
  assert_output --partial "Commit"
}

# ----------------------------------------
# Rollback when the data copy fails
# ----------------------------------------
#
# Bind-mount a file, not a directory, at the image's declared volume path.
# docker cp cannot read a directory listing from it, so the copy fails and
# migrate must remove the repository and volumes it had created (#256).

@test "create unmanaged container whose volume path cannot be copied" {
  local license
  license="$(get_docker_path "${BATS_TEST_DIRNAME}/pagila")/LICENSE.txt"
  run docker create --name migrate-bad -v "${license}:/var/lib/postgresql" "$MIGRATE_SRC_IMAGE"
  assert_success
}

@test "dit migrate fails and rolls back when the data copy fails" {
  run "$D3" migrate -s migrate-bad migrate-bad-dest
  assert_failure
  assert_output --partial "Error: failed to copy data to volume v0"
  assert_output --partial "Rolled back repository migrate-bad-dest; migrate-bad is unchanged"
  refute_output --partial "migrated to controlled environment"
}

@test "rolled back repository is not listed" {
  run "$D3" ls
  assert_success
  refute_output --partial "migrate-bad-dest"
}

@test "rolled back repository left no docker volume behind" {
  run docker volume ls --format '{{.Name}}'
  assert_success
  refute_output --partial "migrate-bad-dest_v0"
}

@test "rolled back repository left no container behind" {
  run docker ps -a --format '{{.Names}}'
  assert_success
  refute_output --partial "migrate-bad-dest"
  assert_output --partial "migrate-bad"
}

@test "cleanup migrated repo" {
  "$D3" rm -f migrate-dest 2>/dev/null || true
  docker rm -f migrate-src 2>/dev/null || true
  docker rm -f migrate-bad 2>/dev/null || true
  true
}
