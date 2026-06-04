#!/usr/bin/env bats

# Load shared test helpers
load '../../test_helper'

# Load environment configuration (DEV by default, ENV=PROD for production)
load 'env'

# Setup: Verify server is running
setup_file() {
  # Verify the server is healthy
  run curl -s "${GATEWAY}/health"
  assert_success
  [[ "$output" == *"${HEALTH_EXPECT}"* ]] || {
    echo "Server is not running or not healthy"
    echo "Health check returned: $output"
    return 1
  }
}

# Cleanup after all tests
teardown_file() {
  # Best effort cleanup - don't fail if already cleaned
  "$D3" repo delete "${TEST_ORG}" dit-test --server "${GATEWAY}" 2>/dev/null || true
  "$D3" repo delete "${WEB_TEST_ORG}" ui-repo --server "${GATEWAY}" 2>/dev/null || true

  # Remove any leftover local repositories
  "$D3" rm -f dittest 2>/dev/null || true
  "$D3" rm -f webuitest 2>/dev/null || true
  "$D3" rm -f webuitestclone 2>/dev/null || true
}

@test "run empty mongo db" {
  run "$D3" run -n dittest mongo
  assert_success
  assert_output --partial "Running controlled container dittest"
}

@test "dit ls shows new repository" {
  run "$D3" ls
  assert_success
  assert_output --partial "dittest"
  assert_output --partial "running"
}

@test "dit status shows running container" {
  run "$D3" status dittest
  assert_success
  assert_output --partial "running"
}

@test "create new commit" {
  run "$D3" commit -m "dittest Commit" dittest
  assert_success
  assert_output --partial "Commit"

  # Extract commit GUID (second word in output)
  COMMIT_GUID=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$COMMIT_GUID" > "$BATS_TMPDIR/commit_guid.txt"
}

@test "add dit remote succeeds" {
  run "$D3" remote add "${REMOTE_URL}/${TEST_ORG}/dit-test" dittest
  assert_success
}

@test "repo has dit remote" {
  run "$D3" remote ls dittest
  assert_success
  assert_output --partial "${REMOTE_URL}/${TEST_ORG}/dit-test"
}

@test "list remote commits returns empty list" {
  run "$D3" remote log dittest
  assert_success
  [[ "$output" != *"Commit"* ]]
}

@test "get non-existent remote commit fails" {
  run "$D3" pull dittest
  assert_failure
}

@test "push commit succeeds" {
  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" push dittest
  assert_success
  assert_output --partial "Pushing ${COMMIT_GUID} to 'origin'"
  assert_output --partial "Push completed successfully"
}

@test "list remote commits returns pushed commit" {
  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" remote log dittest
  assert_success
  assert_output --partial "Commit ${COMMIT_GUID}"
  assert_output --partial "dittest Commit"
}

@test "push of same commit fails" {
  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" push dittest
  assert_failure
  assert_output --partial "commit ${COMMIT_GUID} exists in remote 'origin'"
}

@test "delete local commit succeeds" {
  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" delete -c "$COMMIT_GUID" dittest
  assert_success
  assert_output "${COMMIT_GUID} deleted"
}

@test "list local commits is empty" {
  run "$D3" log dittest
  assert_success
  [[ "$output" != *"Commit"* ]]
}

@test "pull original commit succeeds" {

  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" pull dittest
  assert_success
  assert_output --partial "Pulling ${COMMIT_GUID} from 'origin'"
  assert_output --partial "Pull completed successfully"
}

@test "checkout commit succeeds" {

  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" checkout -c "$COMMIT_GUID" dittest
  assert_success
  assert_output --partial "Stopping container dittest"
  assert_output --partial "Checkout ${COMMIT_GUID}"
  assert_output --partial "Starting container dittest"
  assert_output --partial "${COMMIT_GUID} checked out"
}

@test "dit status shows commit after checkout" {

  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" status dittest
  assert_success
  assert_output --partial "running"
  assert_output --partial "$COMMIT_GUID"
}

@test "dit remote log shows author and message after push" {
  COMMIT_GUID=$(cat "$BATS_TMPDIR/commit_guid.txt")
  run "$D3" remote log dittest
  assert_success
  assert_output --partial "Commit ${COMMIT_GUID}"
  assert_output --partial "Author:"
  assert_output --partial "Message:"
  assert_output --partial "dittest Commit"
}

@test "create second commit" {
  run "$D3" commit -m "Second dittest Commit" dittest
  assert_success
  assert_output --partial "Commit"

  # Extract commit GUID
  COMMIT_GUID2=$(echo "$output" | grep -oE 'Commit [a-f0-9-]+' | awk '{print $2}')
  echo "$COMMIT_GUID2" > "$BATS_TMPDIR/commit_guid2.txt"
}

@test "push second commit succeeds" {
  run "$D3" push dittest
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "list remote commits shows both commits" {
  run "$D3" remote log dittest
  assert_success
  assert_output --partial "Second dittest Commit"
  assert_output --partial "dittest Commit"
}

@test "remove remote succeeds" {
  run "$D3" remote rm dittest origin
  assert_success
  assert_output "Removed origin from dittest"
}

@test "remove dittest repository succeeds" {
  run "$D3" rm -f dittest
  assert_success
  assert_output --partial "Removing repository dittest"
}

@test "test list repos by org CLI (before cleanup)" {
  run "$D3" repo ls --org "${TEST_ORG}" --server "${GATEWAY}"
  assert_success
  assert_output --partial "${TEST_ORG}/dit-test"
}

@test "cleanup - delete test repository from server" {
  run "$D3" repo delete "${TEST_ORG}" dit-test --server "${GATEWAY}"
  assert_success
}

# ===== Web UI Tests =====

@test "web UI: create test repo for web UI tests" {
  run "$D3" repo create "${WEB_TEST_ORG}" ui-repo --server "${GATEWAY}"
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
  run "$D3" remote add "${REMOTE_URL}/${WEB_TEST_ORG}/ui-repo" webuitest
  assert_success
}

@test "web UI: push initial commit" {
  run "$D3" push webuitest
  assert_success
  assert_output --partial "Push completed successfully"
}

@test "web UI: test commits list API returns initial commit" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  run curl -sf -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${WEB_UI}/api/v1/repos/${WEB_TEST_ORG}/ui-repo/commits"
  assert_success
  assert_output --partial "${WEB_COMMIT_1}"
  assert_output --partial "Initial web UI test commit"
}

@test "web UI: test individual commit API" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  run curl -sf -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${WEB_UI}/api/v1/repos/${WEB_TEST_ORG}/ui-repo/commits/${WEB_COMMIT_1}"
  assert_success
  assert_output --partial "Initial web UI test commit"
}

@test "web UI: commit details API includes non-zero size" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  run curl -sf -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${WEB_UI}/api/v1/repos/${WEB_TEST_ORG}/ui-repo/commits/${WEB_COMMIT_1}"
  assert_success
  # Size should be present in the response
  [[ "$output" == *'"size":'* ]]
  # Verify size is not 0 (volume archives have actual data)
  ! [[ "$output" == *'"size":0'* ]]
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
  run curl -sf -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${WEB_UI}/api/v1/repos/${WEB_TEST_ORG}/ui-repo/commits"
  assert_success
  assert_output --partial "${WEB_COMMIT_1}"
  assert_output --partial "${WEB_COMMIT_2}"
  assert_output --partial "Initial web UI test commit"
  assert_output --partial "Added test table"
}

@test "web UI: verify second commit details via API" {
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  run curl -sf -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${WEB_UI}/api/v1/repos/${WEB_TEST_ORG}/ui-repo/commits/${WEB_COMMIT_2}"
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
  run curl -s -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${GATEWAY}/api/v1/repos/${WEB_TEST_ORG}/ui-repo/manifest"
  assert_success
  assert_output --partial "${WEB_COMMIT_1}"
  assert_output --partial "${WEB_COMMIT_2}"
  assert_output --partial "${WEB_COMMIT_3}"
}

@test "web UI: verify all 3 commits via web UI API" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  WEB_COMMIT_3=$(cat "$BATS_TMPDIR/web_commit_3.txt")
  run curl -sf -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${WEB_UI}/api/v1/repos/${WEB_TEST_ORG}/ui-repo/commits"
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
  run curl -s -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${WEB_UI}/api/v1/repos/fake/nonexistent/commits"
  assert_success
  # Gateway returns {"code":404,"message":"..."} for non-existent repos
  [[ "$output" == *'"error"'* ]] || [[ "$output" == *'"message"'* ]]
}

@test "web UI: test API error handling - invalid commit ID returns error" {
  run curl -s -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${WEB_UI}/api/v1/repos/${WEB_TEST_ORG}/ui-repo/commits/invalid-commit-id"
  assert_success
  [[ "$output" == *'"error"'* ]] || [[ "$output" == *'"message"'* ]]
}

@test "web UI: test list all repos via CLI" {
  run "$D3" repo ls --server "${GATEWAY}"
  assert_success
  assert_output --partial "${WEB_TEST_ORG}/ui-repo"
}

@test "web UI: repos list API includes totalSize" {

  # Query the specific repo manifest to check totalSize (avoids stale data from other repos)
  run curl -sLf -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${GATEWAY}/api/v1/repos/${WEB_TEST_ORG}/ui-repo/manifest"
  assert_success
  # totalSize should be present in the response
  [[ "$output" == *'"totalSize":'* ]]
  # Verify totalSize is not 0 (repo has actual volume data)
  ! [[ "$output" == *'"totalSize":0'* ]]
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

# ===== Clone Tests =====
# Stop webuitest container first to free Docker resources for clone

@test "web UI: stop webuitest container before clone" {
  run "$D3" stop webuitest
  assert_success
}

@test "web UI: cleanup - remove previous clone if exists" {
  # Best effort - don't fail if doesn't exist
  "$D3" rm webuitestclone -f 2>/dev/null || true
}

@test "web UI: test clone with manifest - clone from remote" {

  run "$D3" clone -n webuitestclone "${REMOTE_URL}/${WEB_TEST_ORG}/ui-repo"
  assert_success
  assert_output --partial "checked out"
}

@test "web UI: dit ls shows cloned repo" {

  run "$D3" ls
  assert_success
  assert_output --partial "webuitestclone"
}

@test "web UI: dit status on cloned repo shows running" {

  run "$D3" status webuitestclone
  assert_success
  assert_output --partial "running"
  assert_output --partial "/var/lib/postgresql"
}

@test "web UI: dit remote log on cloned repo shows all commits" {

  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  WEB_COMMIT_3=$(cat "$BATS_TMPDIR/web_commit_3.txt")
  run "$D3" remote log webuitestclone
  assert_success
  assert_output --partial "${WEB_COMMIT_1}"
  assert_output --partial "${WEB_COMMIT_2}"
  assert_output --partial "${WEB_COMMIT_3}"
  assert_output --partial "Initial web UI test commit"
  assert_output --partial "Added test table"
  assert_output --partial "Added test data"
}

@test "web UI: cleanup - remove cloned repo" {

  run "$D3" rm webuitestclone -f
  assert_success
  assert_output --partial "webuitestclone removed"
}

@test "web UI: restart webuitest after clone tests" {
  run "$D3" start webuitest
  assert_success
}

# ===== Delete Commit & Repo Tests =====
# Full user experience validation:
#   1. Verify all 3 commits exist via dit CLI
#   2. Delete one commit via browser (web UI)
#   3. Verify deleted commit is gone via dit remote log
#   4. Delete entire repo via browser
#   5. Verify repo no longer exists via dit remote log
#   6. Verify storage and postgres database are cleaned up

@test "web UI: dit remote log shows all 3 commits before delete" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  WEB_COMMIT_3=$(cat "$BATS_TMPDIR/web_commit_3.txt")
  run "$D3" remote log webuitest
  assert_success
  assert_output --partial "${WEB_COMMIT_1}"
  assert_output --partial "${WEB_COMMIT_2}"
  assert_output --partial "${WEB_COMMIT_3}"
}

@test "web UI: delete second commit via browser" {
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  run curl -X DELETE -sf -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${WEB_UI}/api/v1/repos/${WEB_TEST_ORG}/ui-repo/commits/${WEB_COMMIT_2}"
  assert_success
  assert_output --partial '"deleted"'
}

@test "web UI: dit remote log confirms deleted commit is gone" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  WEB_COMMIT_3=$(cat "$BATS_TMPDIR/web_commit_3.txt")
  run "$D3" remote log webuitest
  assert_success
  # Commit 2 should be gone
  [[ "$output" != *"${WEB_COMMIT_2}"* ]]
  # Commits 1 and 3 should still be present
  assert_output --partial "${WEB_COMMIT_1}"
  assert_output --partial "${WEB_COMMIT_3}"
}

@test "web UI: deleted commit data is gone from minio" {
  has_minio || skip "Minio not available in PROD"
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  run docker exec dit-minio mc ls --recursive myminio/dit-dev/${WEB_TEST_ORG}/ui-repo/commits/${WEB_COMMIT_2}/ 2>&1
  [[ -z "$output" ]] || [[ "$output" == *"Object does not exist"* ]] || assert_failure
}

@test "web UI: delete third commit via browser" {
  WEB_COMMIT_3=$(cat "$BATS_TMPDIR/web_commit_3.txt")
  run curl -X DELETE -sf -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${WEB_UI}/api/v1/repos/${WEB_TEST_ORG}/ui-repo/commits/${WEB_COMMIT_3}"
  assert_success
  assert_output --partial '"deleted"'
}

@test "web UI: dit remote log confirms only first commit remains" {
  WEB_COMMIT_1=$(cat "$BATS_TMPDIR/web_commit_1.txt")
  WEB_COMMIT_2=$(cat "$BATS_TMPDIR/web_commit_2.txt")
  WEB_COMMIT_3=$(cat "$BATS_TMPDIR/web_commit_3.txt")
  run "$D3" remote log webuitest
  assert_success
  assert_output --partial "${WEB_COMMIT_1}"
  [[ "$output" != *"${WEB_COMMIT_2}"* ]]
  [[ "$output" != *"${WEB_COMMIT_3}"* ]]
}

@test "web UI: delete repo via browser" {
  run curl -X DELETE -sf -H "Authorization: Bearer ${DIT_API_KEY}" \
    "${WEB_UI}/api/v1/repos/${WEB_TEST_ORG}/ui-repo"
  assert_success
  assert_output --partial '"deleted"'
}

@test "web UI: dit pull fails after repo deleted" {
  run "$D3" pull webuitest
  assert_failure
}

@test "web UI: dit remote log shows no commits after repo deleted" {
  run "$D3" remote log webuitest
  [[ "$output" != *"Initial web UI test commit"* ]]
}

@test "web UI: deleted repo commit data is gone from minio" {
  has_minio || skip "Minio not available in PROD"
  run docker exec dit-minio mc ls --recursive myminio/dit-dev/${WEB_TEST_ORG}/ui-repo/commits/ 2>&1
  [[ -z "$output" ]] || [[ "$output" == *"Object does not exist"* ]] || assert_failure
}

@test "web UI: deleted repo journal data is gone from minio" {
  has_minio || skip "Minio not available in PROD"
  run docker exec dit-minio mc ls --recursive myminio/dit-dev/journals/${WEB_TEST_ORG}/ui-repo/ 2>&1
  [[ -z "$output" ]] || [[ "$output" == *"Object does not exist"* ]] || assert_failure
}

@test "web UI: deleted repo cache is gone from minio" {
  has_minio || skip "Minio not available in PROD"
  run docker exec dit-minio mc ls myminio/dit-dev/cache/manifests/${WEB_TEST_ORG}/ui-repo.json 2>&1
  [[ -z "$output" ]] || [[ "$output" == *"Object does not exist"* ]] || assert_failure
}

@test "web UI: deleted repo is gone from postgres" {
  run run_sql_raw "SELECT count(*) FROM repositories WHERE namespace='${WEB_TEST_ORG}' AND name='ui-repo';"
  assert_success
  count=$(echo "$output" | tr -d '[:space:]')
  [[ "$count" == "0" ]]
}

@test "web UI: cleanup - remove test repo" {
  run "$D3" rm webuitest -f
  assert_success
  assert_output --partial "webuitest removed"
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

@test "download API: unauthenticated access to /download page returns page" {
  is_prod || skip "Download page test only for PROD"
  run curl -s -o /dev/null -w "%{http_code}" "${GATEWAY}/download"
  assert_success
  assert_output "200"
}

@test "download API: list versions endpoint returns valid JSON" {
  run curl -sf -H "Cookie: dit_token=${DIT_API_KEY}" "${WEB_UI}/api/downloads/versions"
  assert_success
  assert_output --partial '"versions"'
}

@test "download API: list versions returns test version" {
  run curl -sf -H "Cookie: dit_token=${DIT_API_KEY}" "${WEB_UI}/api/downloads/versions"
  assert_success
  assert_output --partial "\"version\":\"${DOWNLOAD_TEST_VERSION}\""
}

@test "download API: version metadata has required fields" {
  run curl -sf -H "Cookie: dit_token=${DIT_API_KEY}" "${WEB_UI}/api/downloads/versions"
  assert_success
  assert_output --partial '"release_date"'
  assert_output --partial '"platforms"'
  assert_output --partial '"changelog_url"'
}

@test "download API: version details endpoint returns test version" {
  run curl -sf -H "Cookie: dit_token=${DIT_API_KEY}" "${WEB_UI}/api/downloads/${DOWNLOAD_TEST_VERSION}"
  assert_success
  assert_output --partial "\"version\":\"${DOWNLOAD_TEST_VERSION}\""
}

@test "download API: test version has linux-amd64 platform" {
  run curl -sf -H "Cookie: dit_token=${DIT_API_KEY}" "${WEB_UI}/api/downloads/${DOWNLOAD_TEST_VERSION}"
  assert_success
  assert_output --partial '"platform":"linux-amd64"'
  assert_output --partial '"os":"Linux"'
  assert_output --partial '"arch":"x86_64"'
}

@test "download API: test version has darwin-arm64 platform" {
  run curl -sf -H "Cookie: dit_token=${DIT_API_KEY}" "${WEB_UI}/api/downloads/${DOWNLOAD_TEST_VERSION}"
  assert_success
  assert_output --partial '"platform":"darwin-arm64"'
  assert_output --partial '"os":"macOS"'
  assert_output --partial '"arch":"Apple Silicon"'
}

@test "download API: test version has windows platform" {
  run curl -sf -H "Cookie: dit_token=${DIT_API_KEY}" "${WEB_UI}/api/downloads/${DOWNLOAD_TEST_VERSION}"
  assert_success
  assert_output --partial '"platform":"windows"'
  assert_output --partial '"os":"Windows"'
}

@test "download API: platform metadata includes filename and size" {
  run curl -sf -H "Cookie: dit_token=${DIT_API_KEY}" "${WEB_UI}/api/downloads/${DOWNLOAD_TEST_VERSION}"
  assert_success
  assert_output --partial '"filename"'
  assert_output --partial '"size"'
  assert_output --partial '"sha256"'
}

@test "download API: binary download returns file for linux-amd64" {
  run bash -c "curl -sf -H 'Cookie: dit_token=${DIT_API_KEY}' '${WEB_UI}/api/downloads/${DOWNLOAD_TEST_VERSION}/linux-amd64' | head -c 1024 | wc -c"
  assert_success
  assert_output "1024"
}

@test "download API: binary download has correct content-type header" {
  run curl -sI -H "Cookie: dit_token=${DIT_API_KEY}" "${WEB_UI}/api/downloads/${DOWNLOAD_TEST_VERSION}/linux-amd64"
  assert_success
  assert_output --partial "application/octet-stream"
}

@test "download API: binary download has content-disposition header" {
  run curl -sI -H "Cookie: dit_token=${DIT_API_KEY}" "${WEB_UI}/api/downloads/${DOWNLOAD_TEST_VERSION}/linux-amd64"
  assert_success
  assert_output --partial "attachment"
  assert_output --partial "filename="
}

@test "download API: invalid version returns 404" {
  run curl -s -o /dev/null -w "%{http_code}" -H "Cookie: dit_token=${DIT_API_KEY}" "${WEB_UI}/api/downloads/v99.99.99"
  assert_success
  assert_output "404"
}

@test "download API: invalid platform returns 400 or 404" {
  run curl -s -o /dev/null -w "%{http_code}" -H "Cookie: dit_token=${DIT_API_KEY}" "${WEB_UI}/api/downloads/${DOWNLOAD_TEST_VERSION}/invalid-platform"
  assert_success
  [[ "$output" == "400" || "$output" == "404" ]]
}

@test "download API: health check - storage is accessible" {
  run curl -sf -H "Cookie: dit_token=${DIT_API_KEY}" "${WEB_UI}/api/downloads/versions"
  assert_success
  [[ "$output" != *"Failed to list versions"* ]]
}
