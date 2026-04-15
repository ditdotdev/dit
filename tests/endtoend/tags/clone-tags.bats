#!/usr/bin/env bats

# Load shared test helpers
load '../test_helper'

# Unique S3 path per run to avoid collisions between concurrent runs.
# See issue datadatdat/datadatdat-server#118.
RUN_SUFFIX="${E2E_RUN_SUFFIX:-local}"
S3_URI="s3://datadatdat-testdata/e2etest/${RUN_SUFFIX}"

@test "can launch postgres" {
  run "$D3" run postgres
  assert_success
}

@test "can create commit with tag=one" {
  run "$D3" commit -t tag=one postgres
  assert_success
  assert_output --partial "Commit"
  
  # Extract commit ID and save to temp file
  COMMIT_ONE=$(echo "$output" | grep -o "Commit [a-f0-9]*" | awk '{print $2}')
  echo "$COMMIT_ONE" > "$BATS_TMPDIR/commit_one.txt"
}

@test "can create commit with tag=two" {
  run "$D3" commit -t tag=two postgres
  assert_success
  assert_output --partial "Commit"
  
  # Extract commit ID and save to temp file
  COMMIT_TWO=$(echo "$output" | grep -o "Commit [a-f0-9]*" | awk '{print $2}')
  echo "$COMMIT_TWO" > "$BATS_TMPDIR/commit_two.txt"
}

@test "can add s3 remote" {
  run "$D3" remote add "$S3_URI" postgres
  assert_success
}

@test "can push tag=two" {
  run "$D3" push -t tag=two postgres
  assert_success
}

@test "commit two exists in remote" {
  [ -f "$BATS_TMPDIR/commit_one.txt" ] || skip "COMMIT_ONE not saved"
  [ -f "$BATS_TMPDIR/commit_two.txt" ] || skip "COMMIT_TWO not saved"
  
  COMMIT_ONE=$(cat "$BATS_TMPDIR/commit_one.txt")
  COMMIT_TWO=$(cat "$BATS_TMPDIR/commit_two.txt")
  
  run "$D3" remote log postgres
  assert_success
  assert_output --partial "$COMMIT_TWO"
  
  # Should NOT contain COMMIT_ONE yet
  if echo "$output" | grep -q "$COMMIT_ONE"; then
    echo "Expected COMMIT_ONE to be excluded from remote log"
    return 1
  fi
}

@test "can push tag=one" {
  run "$D3" push -t tag=one postgres
  assert_success
}

@test "commit one exists in remote" {
  [ -f "$BATS_TMPDIR/commit_one.txt" ] || skip "COMMIT_ONE not saved"
  
  COMMIT_ONE=$(cat "$BATS_TMPDIR/commit_one.txt")
  
  run "$D3" remote log postgres
  assert_success
  assert_output --partial "$COMMIT_ONE"
}

@test "can remove postgres" {
  run "$D3" rm -f postgres
  assert_success
}

@test "can clone tag=one" {
  run "$D3" clone -n postgres "${S3_URI}?tag=tag=one"
  assert_success
}

@test "commit one exists locally" {
  [ -f "$BATS_TMPDIR/commit_one.txt" ] || skip "COMMIT_ONE not saved"
  
  COMMIT_ONE=$(cat "$BATS_TMPDIR/commit_one.txt")
  
  run "$D3" log postgres
  assert_success
  assert_output --partial "$COMMIT_ONE"
}

@test "can remove cloned postgres" {
  run "$D3" rm -f postgres
  assert_success
}

@test "clone of non-existent tag fails" {
  run "$D3" clone -n postgres2 "${S3_URI}?tag=tag=three"
  assert_failure
}

@test "can cleanup S3 assets" {
  run aws s3 rm "$S3_URI" --recursive
  assert_success
}
