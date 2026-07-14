#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# Load shared test helpers
load '../test_helper'

@test "can clone hello-world/postgres" {
  run "$D3" clone -n hello-world s3web://demos.dit.dev/hello-world/postgres
  assert_success
  assert_output --partial "Running controlled container hello-world"
  assert_output --partial "Starting container hello-world"
}

@test "can list hello-world/postgres" {
  run "$D3" ls
  assert_success
  assert_output --partial "CONTEXT"
  assert_output --partial "REPOSITORY"
  assert_output --partial "STATUS"
  assert_output --partial "docker"
  assert_output --partial "hello-world"
  assert_output --partial "running"
}

@test "check to see if hello-world database is up and running" {
  run bash -c 'for i in {1..18}; do docker exec hello-world pg_isready && break || sleep 5; done'
  assert_success
}

@test "can get contents of hello-world/postgres" {
  run docker exec hello-world psql postgres://postgres:postgres@localhost/postgres -t -c "SELECT * FROM messages;"
  assert_success
  assert_output --partial "1 | Hello, World!"
}

@test "can remove hello-world/postgres" {
  run "$D3" rm -f hello-world
  assert_success
  assert_output --partial "Removing repository hello-world"
  assert_output --partial "Deleting volume v0"
  assert_output --partial "hello-world removed"
}

@test "can run mongo-test" {
  run "$D3" run -n mongo-test mongo:4
  assert_success
  assert_output --partial "Creating repository mongo-test"
  assert_output --partial "Creating docker volume mongo-test_v0 with path /data/configdb"
  assert_output --partial "Creating docker volume mongo-test_v1 with path /data/db"
  assert_output --partial "Running controlled container mongo-test"
  sleep 5
}

@test "can insert mongo-test Ada Lovelace" {
  run docker exec mongo-test mongo --quiet --eval "db.employees.insert({'firstName':'Ada','lastName':'Lovelace'})"
  assert_success
  assert_output --partial "nInserted"
}

@test "can commit mongo-test" {
  run "$D3" commit -m "First Employee" mongo-test
  assert_success
  assert_output --partial "Commit"
  
  # Extract commit GUID (remove "Commit " prefix) and save to temp file
  COMMIT_GUID=$(echo "$output" | grep -o "Commit [a-f0-9]*" | sed 's/Commit //')
  echo "$COMMIT_GUID" > "$BATS_TMPDIR/commit_guid.txt"
}

@test "can insert mongo-test Grace Hopper" {
  run docker exec mongo-test mongo --quiet --eval "db.employees.insert({'firstName':'Grace','lastName':'Hopper'})"
  assert_success
  assert_output --partial "nInserted"
}

@test "can select employees from mongo-test" {
  run docker exec mongo-test mongo --quiet --eval 'db.employees.find().forEach(printjson)'
  assert_success
  assert_output --partial "Ada"
  assert_output --partial "Grace"
  sleep 2
}

@test "can checkout commit mongo-test" {
  # Read the COMMIT_GUID from the temp file
  [ -f "$BATS_TMPDIR/commit_guid.txt" ] || skip "COMMIT_GUID not saved from previous test"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  
  run "$D3" checkout --commit "$COMMIT_GUID" mongo-test
  assert_success
  assert_output --partial "Stopping container mongo-test"
  assert_output --partial "Checkout $COMMIT_GUID"
  assert_output --partial "Starting container mongo-test"
  assert_output --partial "$COMMIT_GUID checked out"
  sleep 10
}

@test "mongo-test checkout was successful" {
  run docker exec mongo-test mongo --quiet --eval 'db.employees.find().forEach(printjson)'
  assert_success
  # Should NOT contain Grace (we checked out earlier commit)
  if echo "$output" | grep -q "Grace"; then
    echo "Expected Grace to be excluded from output after checkout"
    return 1
  fi
}

@test "can remove mongo-test" {
  run "$D3" rm -f mongo-test
  assert_success
  assert_output --partial "Removing repository mongo-test"
  assert_output --partial "Deleting volume v0"
  assert_output --partial "Deleting volume v1"
  assert_output --partial "mongo-test removed"
}
