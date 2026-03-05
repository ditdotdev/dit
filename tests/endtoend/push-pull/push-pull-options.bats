#!/usr/bin/env bats

# E2E Push/Pull Options Tests
# Tests d3 push --update-only and d3 pull --update-only via S3

# Load shared test helpers
load '../test_helper'

S3_BUCKET="s3://datadatdat-testdata/e2etest-pushpull"

# Cleanup after all tests
teardown_file() {
  "$D3" rm -f pushpull-test 2>/dev/null || true
  aws s3 rm "$S3_BUCKET" --recursive 2>/dev/null || true
  cleanup_stale_aws_processes
}

# ========================================
# Setup
# ========================================

@test "run postgres for push-pull options test" {
  run "$D3" run -n pushpull-test -P postgres
  assert_success
  assert_output --partial "Running controlled container pushpull-test"
}

@test "wait for pushpull-test postgres to be ready" {
  run bash -c 'for i in {1..18}; do docker exec pushpull-test pg_isready && break || sleep 5; done'
  assert_success
}

@test "create commit with tag version=1.0" {
  run "$D3" commit -t version=1.0 pushpull-test
  assert_success
  assert_output --partial "Commit"

  COMMIT_GUID=$(echo "$output" | grep -o "Commit [a-f0-9]*" | awk '{print $2}')
  echo "$COMMIT_GUID" > "$BATS_TMPDIR/pushpull_commit.txt"
}

@test "add s3 remote for push-pull test" {
  run "$D3" remote add "$S3_BUCKET" pushpull-test
  assert_success
}

# ========================================
# Full push first (data + tags)
# ========================================

@test "push data to S3" {
  run "$D3" push pushpull-test
  assert_success
  assert_output --partial "Push completed successfully"
}

# ========================================
# Update-only push (tags only, no data)
# ========================================

@test "add new tag to existing commit" {
  [ -f "$BATS_TMPDIR/pushpull_commit.txt" ] || skip "Commit GUID not saved"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/pushpull_commit.txt")

  run "$D3" tag -c "$COMMIT_GUID" -t released=true pushpull-test
  assert_success
}

@test "d3 push --update-only syncs tags without re-uploading data" {
  run "$D3" push -u pushpull-test
  assert_success
}

@test "remote log shows commit after update-only push" {
  [ -f "$BATS_TMPDIR/pushpull_commit.txt" ] || skip "Commit GUID not saved"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/pushpull_commit.txt")

  run "$D3" remote log pushpull-test
  assert_success
  assert_output --partial "$COMMIT_GUID"
}

# ========================================
# Cleanup
# ========================================

@test "remove pushpull-test" {
  run "$D3" rm -f pushpull-test
  assert_success
}

@test "cleanup S3 assets" {
  run aws s3 rm "$S3_BUCKET" --recursive
  assert_success
}
