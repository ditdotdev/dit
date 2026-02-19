#!/usr/bin/env bats

# Load shared test helpers
load '../../test_helper'

# Setup: Verify datadatdat-remote-server is running
setup_file() {
  # Set API key for authentication
  export DATADATDAT_API_KEY=02b31569a9052bc4b3cf1c3819d4fc048d34c96eca21f2b8e2359b5ecdfec93a
  
  # Verify the server is healthy
  run curl -s http://127.0.0.1:8080/health
  assert_success
  [[ "$output" == *"healthy"* ]] || {
    echo "datadatdat-remote-server is not running or not healthy"
    return 1
  }
}

# Cleanup after all tests
teardown_file() {
  # Best effort cleanup - don't fail if already cleaned
  curl -X DELETE -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/testorg/datadatdat-test" 2>/dev/null || true
  curl -X DELETE -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/webtest/ui-repo" 2>/dev/null || true
  
  # Remove any leftover local repositories
  "$D3" rm -f datadatdattest 2>/dev/null || true
  "$D3" rm -f webuitest 2>/dev/null || true
  "$D3" rm -f webuitestclone 2>/dev/null || true
}

@test "create test repository in datadatdat-remote-server" {
  run curl -X POST -f -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/testorg/datadatdat-test"
  assert_success
  assert_output --partial "datadatdat-test"
}

@test "run empty mongo db" {
  run "$D3" run -n datadatdattest mongo
  assert_success
  assert_output --partial "Running controlled container datadatdattest"
}

@test "create new commit" {
  run "$D3" commit -m "datadatdattest Commit" datadatdattest
  assert_success
  assert_output --partial "Commit"
  
  # Extract commit GUID (second word in output)
  COMMIT_GUID=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$COMMIT_GUID" > "$BATS_TMPDIR/commit_guid.txt"
}

@test "add datadatdat remote succeeds" {
  run "$D3" remote add http://datadatdat-api-gateway:8080/testorg/datadatdat-test datadatdattest
  assert_success
}

@test "repo has datadatdat remote" {
  run "$D3" remote ls datadatdattest
  assert_success
  assert_output --partial "http://datadatdat-api-gateway:8080/testorg/datadatdat-test"
}

@test "list remote commits returns empty list" {
  run "$D3" remote log datadatdattest
  assert_success
  [[ "$output" != *"Commit"* ]]
}

@test "get non-existent remote commit fails" {
  run "$D3" pull datadatdattest
  assert_failure
}

@test "push commit succeeds" {
  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" push datadatdattest
  assert_success
  assert_output --partial "Pushing ${COMMIT_GUID} to 'origin'"
  assert_output --partial "Push completed successfully"
}

@test "list remote commits returns pushed commit" {
  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" remote log datadatdattest
  assert_success
  assert_output --partial "Commit ${COMMIT_GUID}"
  assert_output --partial "datadatdattest Commit"
}

@test "push of same commit fails" {
  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" push datadatdattest
  assert_failure
  assert_output --partial "commit ${COMMIT_GUID} exists in remote 'origin'"
}

@test "delete local commit succeeds" {
  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" delete -c "$COMMIT_GUID" datadatdattest
  assert_success
  assert_output "${COMMIT_GUID} deleted"
}

@test "list local commits is empty" {
  run "$D3" log datadatdattest
  assert_success
  [[ "$output" != *"Commit"* ]]
}

@test "pull original commit succeeds" {
  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" pull datadatdattest
  assert_success
  assert_output --partial "Pulling ${COMMIT_GUID} from 'origin'"
  assert_output --partial "Pull completed successfully"
}

@test "checkout commit succeeds" {
  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" checkout -c "$COMMIT_GUID" datadatdattest
  assert_success
  assert_output --partial "Stopping container datadatdattest"
  assert_output --partial "Checkout ${COMMIT_GUID}"
  assert_output --partial "Starting container datadatdattest"
  assert_output --partial "${COMMIT_GUID} checked out"
}

@test "create second commit" {
  run "$D3" commit -m "Second datadatdattest Commit" datadatdattest
  assert_success
  assert_output --partial "Commit"
  
  # Extract commit GUID
  COMMIT_GUID2=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$COMMIT_GUID2" > "$BATS_TMPDIR/commit_guid2.txt"
}

@test "push second commit succeeds" {
  run "$D3" push datadatdattest
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "list remote commits shows both commits" {
  run "$D3" remote log datadatdattest
  assert_success
  assert_output --partial "Second datadatdattest Commit"
  assert_output --partial "datadatdattest Commit"
}

@test "remove remote succeeds" {
  run "$D3" remote rm datadatdattest origin
  assert_success
  assert_output "Removed origin from datadatdattest"
}

@test "remove datadatdattest repository succeeds" {
  run "$D3" rm -f datadatdattest
  assert_success
  assert_output --partial "Removing repository datadatdattest"
}

@test "test list repos by org API endpoint - testorg (before cleanup)" {
  run curl -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:3000/api/v1/repos/testorg"
  assert_success
  assert_output --partial "testorg/datadatdat-test"
}

@test "cleanup - delete test repository from datadatdat-remote-server" {
  run curl -X DELETE -f -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/testorg/datadatdat-test"
  assert_success
}

# ===== Web UI Tests =====

@test "web UI: create test repo for web UI tests" {
  run curl -X POST -f -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/webtest/ui-repo"
  assert_success
  assert_output --partial "ui-repo"
}

@test "web UI: run postgres for web UI tests" {
  run "$D3" run -n webuitest postgres -e POSTGRES_PASSWORD=postgres
  assert_success
  assert_output --partial "Running controlled container webuitest"
}

@test "web UI: check to see if database is up and running" {
  run bash -c "for i in {1..18}; do docker exec webuitest pg_isready && break || sleep 5; done"
  assert_success
  assert_output --partial "accepting connections"
}

@test "web UI: create initial commit" {
  run "$D3" commit -m "Initial web UI test commit" webuitest
  assert_success
  assert_output --partial "Commit"
  
  # Extract commit GUID
  WEB_COMMIT_1=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$WEB_COMMIT_1" > "$BATS_TMPDIR/web_commit_1.txt"
}

@test "web UI: add remote" {
  run "$D3" remote add http://datadatdat-api-gateway:8080/webtest/ui-repo webuitest
  assert_success
}

@test "web UI: push initial commit" {
  run "$D3" push webuitest
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "web UI: test commits list API returns initial commit" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  run curl -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:3000/api/v1/repos/webtest/ui-repo/commits"
  assert_success
  assert_output --partial "${WEB_COMMIT_1}"
  assert_output --partial "Initial web UI test commit"
}

@test "web UI: test individual commit API" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  run curl -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:3000/api/v1/repos/webtest/ui-repo/commits/${WEB_COMMIT_1}"
  assert_success
  assert_output --partial "Initial web UI test commit"
}

@test "web UI: make database changes for second commit" {
  run docker exec webuitest psql -U postgres -c \
    "CREATE TABLE test_table (id SERIAL PRIMARY KEY, name VARCHAR(100));"
  assert_success
  assert_output --partial "CREATE TABLE"
}

@test "web UI: create second commit" {
  run "$D3" commit -m "Added test table" webuitest
  assert_success
  assert_output --partial "Commit"
  
  # Extract commit GUID
  WEB_COMMIT_2=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$WEB_COMMIT_2" > "$BATS_TMPDIR/web_commit_2.txt"
}

@test "web UI: push second commit" {
  run "$D3" push webuitest
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "web UI: test commit data freshness (no caching) - verify 2 commits" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  run curl -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:3000/api/v1/repos/webtest/ui-repo/commits"
  assert_success
  assert_output --partial "${WEB_COMMIT_1}"
  assert_output --partial "${WEB_COMMIT_2}"
  assert_output --partial "Initial web UI test commit"
  assert_output --partial "Added test table"
}

@test "web UI: verify second commit details via API" {
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  run curl -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:3000/api/v1/repos/webtest/ui-repo/commits/${WEB_COMMIT_2}"
  assert_success
  assert_output --partial "${WEB_COMMIT_2}"
  assert_output --partial "Added test table"
}

@test "web UI: make database changes for third commit" {
  run docker exec webuitest psql -U postgres -c \
    "INSERT INTO test_table (name) VALUES ('Alice'), ('Bob');"
  assert_success
  assert_output --partial "INSERT"
}

@test "web UI: create third commit" {
  run "$D3" commit -m "Added test data" webuitest
  assert_success
  assert_output --partial "Commit"
  
  # Extract commit GUID
  WEB_COMMIT_3=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$WEB_COMMIT_3" > "$BATS_TMPDIR/web_commit_3.txt"
}

@test "web UI: push third commit" {
  run "$D3" push webuitest
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "web UI: test manifest updates on push - verify all 3 commits in manifest" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  WEB_COMMIT_3=$(cat "$BATS_TMPDIR/web_commit_3.txt")
  run curl -s -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:8080/api/v1/repos/webtest/ui-repo/manifest"
  assert_success
  assert_output --partial "${WEB_COMMIT_1}"
  assert_output --partial "${WEB_COMMIT_2}"
  assert_output --partial "${WEB_COMMIT_3}"
}

@test "web UI: verify all 3 commits via web UI API" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  WEB_COMMIT_3=$(cat "$BATS_TMPDIR/web_commit_3.txt")
  run curl -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:3000/api/v1/repos/webtest/ui-repo/commits"
  assert_success
  assert_output --partial "${WEB_COMMIT_1}"
  assert_output --partial "${WEB_COMMIT_2}"
  assert_output --partial "${WEB_COMMIT_3}"
  assert_output --partial "Initial web UI test commit"
  assert_output --partial "Added test table"
  assert_output --partial "Added test data"
}

@test "web UI: checkout first commit" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  run "$D3" checkout -c "$WEB_COMMIT_1" webuitest
  assert_success
  assert_output --partial "checked out"
}

@test "web UI: test API error handling - non-existent repo returns error" {
  run curl -s -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:3000/api/v1/repos/fake/nonexistent/commits"
  assert_success
  assert_output --partial '"error":'
}

@test "web UI: test API error handling - invalid commit ID returns error" {
  run curl -s -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:3000/api/v1/repos/webtest/ui-repo/commits/invalid-commit-id"
  assert_success
  assert_output --partial "error"
}

@test "web UI: test list all repos API endpoint" {
  run curl -sf -H "Authorization: Bearer ${DATADATDAT_API_KEY}" \
    "http://127.0.0.1:3000/api/v1/repos"
  assert_success
  assert_output --partial "webtest/ui-repo"
}

@test "web UI: verify empty database at first commit" {
  # Wait for database
  run bash -c "for i in {1..18}; do docker exec webuitest pg_isready && break || sleep 5; done"
  assert_success
  
  run bash -c "docker exec webuitest psql -U postgres -c '\\dt' 2>&1"
  assert_success
  assert_output --partial "Did not find any"
}

@test "web UI: checkout second commit" {
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  run "$D3" checkout -c "$WEB_COMMIT_2" webuitest
  assert_success
  assert_output --partial "checked out"
}

@test "web UI: verify table exists at second commit" {
  # Wait for database
  run bash -c "for i in {1..18}; do docker exec webuitest pg_isready && break || sleep 5; done"
  assert_success
  
  run docker exec webuitest psql -U postgres -c "\\dt"
  assert_success
  assert_output --partial "test_table"
}

@test "web UI: verify table is empty at second commit" {
  run docker exec webuitest psql -U postgres -c "SELECT COUNT(*) FROM test_table;"
  assert_success
  assert_output --partial " 0"
}

@test "web UI: checkout third commit" {
  WEB_COMMIT_3=$(cat "$BATS_TMPDIR/web_commit_3.txt")
  run "$D3" checkout -c "$WEB_COMMIT_3" webuitest
  assert_success
  assert_output --partial "checked out"
}

@test "web UI: verify data exists at third commit" {
  # Wait for database
  run bash -c "for i in {1..18}; do docker exec webuitest pg_isready && break || sleep 5; done"
  assert_success
  
  run docker exec webuitest psql -U postgres -c "SELECT COUNT(*) FROM test_table;"
  assert_success
  assert_output --partial " 2"
}

@test "web UI: verify correct data at third commit" {
  run docker exec webuitest psql -U postgres -c "SELECT name FROM test_table ORDER BY name;"
  assert_success
  assert_output --partial "Alice"
  assert_output --partial "Bob"
}

@test "web UI: cleanup - remove test repo" {
  run "$D3" rm webuitest -f
  assert_success
  assert_output --partial "webuitest removed"
}

@test "web UI: cleanup - remove previous clone if exists" {
  # Best effort - don't fail if doesn't exist
  "$D3" rm webuitestclone -f 2>/dev/null || true
}

@test "web UI: test clone with manifest - clone from remote" {
  run "$D3" clone -n webuitestclone http://datadatdat-api-gateway:8080/webtest/ui-repo
  assert_success
  assert_output --partial "checked out"
}

@test "web UI: test clone with manifest - verify all commits present" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  WEB_COMMIT_3=$(cat "$BATS_TMPDIR/web_commit_3.txt")
  run "$D3" remote log webuitestclone
  assert_success
  assert_output --partial "${WEB_COMMIT_1}"
  assert_output --partial "${WEB_COMMIT_2}"
  assert_output --partial "${WEB_COMMIT_3}"
}

@test "web UI: cleanup - remove cloned repo" {
  run "$D3" rm webuitestclone -f
  assert_success
  assert_output --partial "webuitestclone removed"
}

@test "web UI: verify no repositories exist after cleanup" {
  run "$D3" ls
  assert_success
  # Should just show header, no repositories
  assert_output --partial "CONTEXT"
  assert_output --partial "REPOSITORY"
  assert_output --partial "STATUS"
  # Count lines - should be just header (3 lines or less)
  lines_count=$(echo "$output" | wc -l)
  [[ $lines_count -le 3 ]]
}

# ===== Download API Tests =====

@test "download API: list versions endpoint returns valid JSON" {
  run curl -sf -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" "http://127.0.0.1:3000/api/downloads/versions"
  assert_success
  # Should return JSON with versions array
  assert_output --partial '"versions"'
}

@test "download API: list versions returns v1.6.1" {
  run curl -sf -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" "http://127.0.0.1:3000/api/downloads/versions"
  assert_success
  assert_output --partial '"version":"v1.6.1"'
}

@test "download API: version metadata has required fields" {
  run curl -sf -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" "http://127.0.0.1:3000/api/downloads/versions"
  assert_success
  assert_output --partial '"release_date"'
  assert_output --partial '"platforms"'
  assert_output --partial '"changelog_url"'
}

@test "download API: version details endpoint returns v1.6.1" {
  run curl -sf -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" "http://127.0.0.1:3000/api/downloads/v1.6.1"
  assert_success
  assert_output --partial '"version":"v1.6.1"'
}

@test "download API: v1.6.1 has linux-amd64 platform" {
  run curl -sf -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" "http://127.0.0.1:3000/api/downloads/v1.6.1"
  assert_success
  assert_output --partial '"platform":"linux-amd64"'
  assert_output --partial '"os":"Linux"'
  assert_output --partial '"arch":"x86_64"'
}

@test "download API: v1.6.1 has darwin-arm64 platform" {
  run curl -sf -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" "http://127.0.0.1:3000/api/downloads/v1.6.1"
  assert_success
  assert_output --partial '"platform":"darwin-arm64"'
  assert_output --partial '"os":"macOS"'
  assert_output --partial '"arch":"Apple Silicon"'
}

@test "download API: v1.6.1 has windows platform" {
  run curl -sf -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" "http://127.0.0.1:3000/api/downloads/v1.6.1"
  assert_success
  assert_output --partial '"platform":"windows"'
  assert_output --partial '"os":"Windows"'
}

@test "download API: platform metadata includes filename and size" {
  run curl -sf -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" "http://127.0.0.1:3000/api/downloads/v1.6.1"
  assert_success
  assert_output --partial '"filename"'
  assert_output --partial '"size"'
  assert_output --partial '"sha256"'
}

@test "download API: binary download returns file for linux-amd64" {
  # Download first 1KB to verify it's a binary file (not an error)
  run bash -c "curl -sf -H 'Cookie: datadatdat_token=${DATADATDAT_API_KEY}' 'http://127.0.0.1:3000/api/downloads/v1.6.1/linux-amd64' | head -c 1024 | wc -c"
  assert_success
  # Should be exactly 1024 bytes (1KB downloaded)
  assert_output "1024"
}

@test "download API: binary download has correct content-type header" {
  run curl -sI -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" "http://127.0.0.1:3000/api/downloads/v1.6.1/linux-amd64"
  assert_success
  # Case-insensitive match for content-type header
  assert_output --partial "application/octet-stream"
}

@test "download API: binary download has content-disposition header" {
  run curl -sI -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" "http://127.0.0.1:3000/api/downloads/v1.6.1/linux-amd64"
  assert_success
  # Case-insensitive match for content-disposition header
  assert_output --partial "attachment"
  assert_output --partial "filename="
}

@test "download API: invalid version returns 404" {
  run curl -s -o /dev/null -w "%{http_code}" -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" "http://127.0.0.1:3000/api/downloads/v99.99.99"
  assert_success
  assert_output "404"
}

@test "download API: invalid platform returns 400 or 404" {
  # Invalid platform may return 400 (bad request) or 404 (not found)
  run curl -s -o /dev/null -w "%{http_code}" -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" "http://127.0.0.1:3000/api/downloads/v1.5.0/invalid-platform"
  assert_success
  # Accept either 400 or 404 as valid error responses
  [[ "$output" == "400" || "$output" == "404" ]]
}

@test "download API: health check - storage is accessible" {
  # This test verifies the storage backend (S3/MinIO) is working
  run curl -sf -H "Cookie: datadatdat_token=${DATADATDAT_API_KEY}" "http://127.0.0.1:3000/api/downloads/versions"
  assert_success
  # Should not return error about storage
  [[ "$output" != *"Failed to list versions"* ]]
}
