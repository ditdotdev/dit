#!/usr/bin/env bats

# Load shared test helpers
load '../../test_helper'

# Test parameters
REMOTE="s3"
# Unique S3 path per run to avoid collisions between concurrent runs.
# See issue ditdotdev/dit-server#118.
RUN_SUFFIX="${E2E_RUN_SUFFIX:-local}"
URI="s3://demo-ditdotdev/simple-test/${RUN_SUFFIX}"
REPO="hello-world"

# Ensure API is ready (upgrade test may have restarted infrastructure)
setup_file() {
  for i in $(seq 1 36); do
    if "$D3" ls 2>/dev/null | grep -q "CONTEXT"; then
      return 0
    fi
    sleep 5
  done
}

@test "can clone hello-world/postgres" {
  run "$D3" clone -n hello-world s3web://demo-dit.s3-website-us-west-2.amazonaws.com/hello-world/postgres
  assert_success
  assert_output --partial "Running controlled container hello-world"
  assert_output --partial "Starting container hello-world"
}

@test "s3 > remote > add" {
  run "$D3" remote add -r "$REMOTE" "$URI" "$REPO"
  assert_success
}

@test "s3 > remote > ls > has s3" {
  run "$D3" remote ls "$REPO"
  assert_success
  assert_output --partial "$URI"
}

@test "s3 > commit" {
  run "$D3" commit -m "Test $REMOTE Commit" "$REPO"
  assert_success
  assert_output --partial "Commit"
  
  # Extract commit GUID and save to temp file
  COMMIT_GUID=$(echo "$output" | grep -o "Commit [a-f0-9]*" | awk '{print $2}')
  echo "$COMMIT_GUID" > "$BATS_TMPDIR/s3_commit_guid.txt"
}

@test "s3 > remote > log > missing commit" {
  [ -f "$BATS_TMPDIR/s3_commit_guid.txt" ] || skip "COMMIT_GUID not saved"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/s3_commit_guid.txt")
  
  run "$D3" remote log "$REPO"
  assert_success
  
  # Should NOT contain the commit yet
  if echo "$output" | grep -q "Commit $COMMIT_GUID"; then
    echo "Expected commit to be excluded from remote log before push"
    return 1
  fi
}

@test "s3 > push" {
  [ -f "$BATS_TMPDIR/s3_commit_guid.txt" ] || skip "COMMIT_GUID not saved"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/s3_commit_guid.txt")
  
  run "$D3" push -r "$REMOTE" -c "$COMMIT_GUID" "$REPO"
  assert_success
  assert_output --partial "Pushing $COMMIT_GUID to '$REMOTE'"
  assert_output --partial "Push completed successfully"
}

@test "s3 > remote > log > has commit" {
  [ -f "$BATS_TMPDIR/s3_commit_guid.txt" ] || skip "COMMIT_GUID not saved"
  COMMIT_GUID=$(cat "$BATS_TMPDIR/s3_commit_guid.txt")
  
  run "$D3" remote log "$REPO"
  assert_success
  assert_output --partial "Commit $COMMIT_GUID"
}

@test "remove s3 assets" {
  run aws s3 rm "$URI" --recursive
  assert_success
}

@test "remove hello-world" {
  run "$D3" rm -f "$REPO"
  assert_success
}
