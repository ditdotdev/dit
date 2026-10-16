#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# Image pull policy for `dit run` and `dit clone` (issues #247 and #240).
#
# Exercises the decision table for --pull missing|always|never on both
# commands against a real docker daemon and an SSH remote. "digest" is the
# RepoDigests entry recorded in the commit being cloned (dit run has none):
#
#   policy         | digest   | image:tag present | expected
#   ---------------+----------+-------------------+--------------------------------------------
#   missing        | unknown  | yes               | use local, no pull
#   missing        | unknown  | no                | pull image:tag; exit 1 if pull fails
#   missing/always | recorded | any               | run image:tag@digest; pull the digest only
#                  |          |                   | when absent; never retag image:tag (#240)
#   missing/always | recorded | yes, pull fails   | warn, use local image:tag
#   never          | any      | yes               | use local; warn on digest mismatch
#   never          | recorded | no, digest present| run image:tag@digest
#   never          | any      | no                | exit 1
#   always         | unknown  | yes               | pull image:tag, warn and keep local on failure
#   always         | unknown  | no                | pull; exit 1 if pull fails
#
# Two fixtures drive the table:
#   * ditdotdev/pull-policy-local:test - built here from local-only.Dockerfile,
#     never pushed. Any pull of it fails, which stands in for an unreachable
#     registry without touching the network.
#   * redis:7-alpine / redis:6-alpine - real registry images with a VOLUME.
#     Running redis:7-alpine records its digest in the commit; tagging
#     redis:6-alpine as redis:7-alpine then gives a present image whose
#     RepoDigests do not match the stored digest - the #240 situation of a
#     tag that moved after the data was pushed.

load '../test_helper'

LOCAL_IMAGE="ditdotdev/pull-policy-local:test"
REMOTE_IMAGE="redis:7-alpine"
STALE_IMAGE="redis:6-alpine"
LOCAL_REPO="pullpolicy-local"
REMOTE_REPO="pullpolicy-remote"

build_local_image() {
  # Build from inside the fixture directory so docker sees a native path on
  # every platform (Git Bash path conversion is unreliable for /tmp mounts).
  (cd "$BATS_TEST_DIRNAME" && docker build -q -t "$LOCAL_IMAGE" -f local-only.Dockerfile .)
}

setup_file() {
  docker pull ditdotdev/nginx-test:latest
  docker pull "$REMOTE_IMAGE"
  docker pull "$STALE_IMAGE"
  build_local_image

  # Record the real digest of redis:7-alpine before any retagging.
  docker inspect --format '{{index .RepoDigests 0}}' "$REMOTE_IMAGE" > "$BATS_TMPDIR/remote_digest.txt"

  docker pull ditdotdev/ssh-test-server:latest
  MSYS_NO_PATHCONV=1 docker run -v "${REPO_ROOT}:/workdir" --workdir /workdir --network dit-docker -d --name pullPolicySsh ditdotdev/ssh-test-server:latest
  docker exec pullPolicySsh mkdir -p /local /remote
  SSH_HOST=$(docker inspect -f "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}" pullPolicySsh)
  echo "$SSH_HOST" > "$BATS_TMPDIR/pull_policy_ssh_host.txt"
}

teardown_file() {
  "$D3" rm -f "$LOCAL_REPO" 2>/dev/null || true
  "$D3" rm -f "$REMOTE_REPO" 2>/dev/null || true
  docker rm -f pullPolicySsh 2>/dev/null || true
  docker rmi -f "$LOCAL_IMAGE" 2>/dev/null || true
  # Restore redis:7-alpine to the registry image in case a test left the
  # redis:6-alpine retag in place.
  docker rmi -f "$REMOTE_IMAGE" 2>/dev/null || true
  docker pull "$REMOTE_IMAGE" >/dev/null 2>&1 || true
}

ssh_host() {
  [ -f "$BATS_TMPDIR/pull_policy_ssh_host.txt" ] || skip "SSH host not saved"
  cat "$BATS_TMPDIR/pull_policy_ssh_host.txt"
}

image_id() {
  docker inspect --format '{{.Id}}' "$1"
}

# The image the running clone was started from, e.g. redis@sha256:...
container_image() {
  docker inspect --format '{{.Config.Image}}' "$1"
}

# Remove the committed redis digest from the daemon (it is only ever pulled
# by digest here, so this drops the image) without touching redis:7-alpine.
remove_committed_digest() {
  docker rmi -f "$(cat "$BATS_TMPDIR/remote_digest.txt")" 2>/dev/null || true
}

# ===== dit run: local-only image =====

@test "run: local-only image, default policy uses local image" {
  run "$D3" run -n "$LOCAL_REPO" "$LOCAL_IMAGE"
  assert_success
  assert_output --partial "Running controlled container $LOCAL_REPO"
  run "$D3" rm -f "$LOCAL_REPO"
  assert_success
}

@test "run: local-only image, --pull never uses local image" {
  run "$D3" run -n "$LOCAL_REPO" --pull never "$LOCAL_IMAGE"
  assert_success
  assert_output --partial "Running controlled container $LOCAL_REPO"
  run "$D3" rm -f "$LOCAL_REPO"
  assert_success
}

@test "run: local-only image, --pull always warns when pull fails and keeps local image" {
  run "$D3" run -n "$LOCAL_REPO" --pull always "$LOCAL_IMAGE"
  assert_success
  assert_output --partial "Warning: unable to pull"
  assert_output --partial "Running controlled container $LOCAL_REPO"
  run "$D3" rm -f "$LOCAL_REPO"
  assert_success
}

@test "run: invalid --pull value exits 1" {
  run "$D3" run -n "$LOCAL_REPO" --pull sometimes "$LOCAL_IMAGE"
  assert_failure
  assert_output --partial "invalid --pull value"
}

@test "run: image absent, --pull never exits 1 without pulling" {
  docker rmi -f "$LOCAL_IMAGE"
  run "$D3" run -n "$LOCAL_REPO" --pull never "$LOCAL_IMAGE"
  assert_failure
  assert_output --partial "not present locally"
  "$D3" rm -f "$LOCAL_REPO" 2>/dev/null || true
}

@test "run: image absent, default policy exits 1 when pull fails" {
  run "$D3" run -n "$LOCAL_REPO" "$LOCAL_IMAGE"
  assert_failure
  assert_output --partial "not found locally"
  "$D3" rm -f "$LOCAL_REPO" 2>/dev/null || true
}

# ===== dit clone: local-only image (issue #247) =====

@test "clone: local-only image, default policy clones without pulling (#247)" {
  SSH_HOST=$(ssh_host)
  build_local_image

  run "$D3" run -n "$LOCAL_REPO" "$LOCAL_IMAGE"
  assert_success
  sleep 5
  run "$D3" commit -m "local-only image" "$LOCAL_REPO"
  assert_success
  run "$D3" remote add -p "skipHostCheck=true" "ssh://root:root@$SSH_HOST/local" "$LOCAL_REPO"
  assert_success
  run "$D3" push "$LOCAL_REPO"
  assert_success
  run "$D3" rm -f "$LOCAL_REPO"
  assert_success

  run "$D3" clone -n "$LOCAL_REPO" -p "skipHostCheck=true" "ssh://root:root@$SSH_HOST/local"
  assert_success
  refute_output --partial "Failed to pull"
  refute_output --partial "Unable to find image"
  run "$D3" rm -f "$LOCAL_REPO"
  assert_success
}

@test "clone: local-only image, --pull never clones with no digest warning" {
  SSH_HOST=$(ssh_host)
  run "$D3" clone -n "$LOCAL_REPO" --pull never -p "skipHostCheck=true" "ssh://root:root@$SSH_HOST/local"
  assert_success
  refute_output --partial "differs from the digest"
  run "$D3" rm -f "$LOCAL_REPO"
  assert_success
}

@test "clone: image absent, --pull never exits 1" {
  SSH_HOST=$(ssh_host)
  docker rmi -f "$LOCAL_IMAGE"
  run "$D3" clone -n "$LOCAL_REPO" --pull never -p "skipHostCheck=true" "ssh://root:root@$SSH_HOST/local"
  assert_failure
  assert_output --partial "not present locally"
  "$D3" rm -f "$LOCAL_REPO" 2>/dev/null || true
}

# ===== dit clone: registry image, digest comparison =====

@test "clone: --pull never with mismatching local digest warns and keeps local image" {
  SSH_HOST=$(ssh_host)

  run "$D3" run -n "$REMOTE_REPO" "$REMOTE_IMAGE"
  assert_success
  sleep 5
  run "$D3" commit -m "registry image" "$REMOTE_REPO"
  assert_success
  run "$D3" remote add -p "skipHostCheck=true" "ssh://root:root@$SSH_HOST/remote" "$REMOTE_REPO"
  assert_success
  run "$D3" push "$REMOTE_REPO"
  assert_success
  run "$D3" rm -f "$REMOTE_REPO"
  assert_success

  # Make the local redis:7-alpine tag point at the redis:6-alpine image so its
  # RepoDigests no longer match the digest stored in the commit.
  docker tag "$STALE_IMAGE" "$REMOTE_IMAGE"
  image_id "$REMOTE_IMAGE" > "$BATS_TMPDIR/stale_image_id.txt"

  run "$D3" clone -n "$REMOTE_REPO" --pull never -p "skipHostCheck=true" "ssh://root:root@$SSH_HOST/remote"
  assert_success
  assert_output --partial "differs from the digest recorded"
  [ "$(image_id "$REMOTE_IMAGE")" = "$(cat "$BATS_TMPDIR/stale_image_id.txt")" ]
  run "$D3" rm -f "$REMOTE_REPO"
  assert_success
}

@test "clone: --pull always with mismatching local digest pulls by digest and leaves the local tag alone" {
  SSH_HOST=$(ssh_host)
  EXPECTED_DIGEST=$(cat "$BATS_TMPDIR/remote_digest.txt")
  remove_committed_digest

  run "$D3" clone -n "$REMOTE_REPO" --pull always -p "skipHostCheck=true" "ssh://root:root@$SSH_HOST/remote"
  assert_success
  # The committed digest was pulled and the clone runs it ...
  run docker inspect --type image "$EXPECTED_DIGEST"
  assert_success
  [ "$(container_image "$REMOTE_REPO")" = "$EXPECTED_DIGEST" ]
  # ... while redis:7-alpine still points at the redis:6-alpine image.
  [ "$(image_id "$REMOTE_IMAGE")" = "$(cat "$BATS_TMPDIR/stale_image_id.txt")" ]
  run "$D3" rm -f "$REMOTE_REPO"
  assert_success
}

# ===== dit clone: default policy runs the committed digest (issue #240) =====

@test "clone: default policy with mismatching local digest runs the committed digest (#240)" {
  SSH_HOST=$(ssh_host)
  EXPECTED_DIGEST=$(cat "$BATS_TMPDIR/remote_digest.txt")
  remove_committed_digest

  run "$D3" clone -n "$REMOTE_REPO" -p "skipHostCheck=true" "ssh://root:root@$SSH_HOST/remote"
  assert_success
  run docker inspect --type image "$EXPECTED_DIGEST"
  assert_success
  [ "$(container_image "$REMOTE_REPO")" = "$EXPECTED_DIGEST" ]
  [ "$(image_id "$REMOTE_IMAGE")" = "$(cat "$BATS_TMPDIR/stale_image_id.txt")" ]
  run "$D3" rm -f "$REMOTE_REPO"
  assert_success
}

@test "clone: default policy with image absent pulls the committed digest, not the tag (#240)" {
  SSH_HOST=$(ssh_host)
  EXPECTED_DIGEST=$(cat "$BATS_TMPDIR/remote_digest.txt")
  remove_committed_digest
  docker rmi -f "$REMOTE_IMAGE"

  run "$D3" clone -n "$REMOTE_REPO" -p "skipHostCheck=true" "ssh://root:root@$SSH_HOST/remote"
  assert_success
  run docker inspect --type image "$EXPECTED_DIGEST"
  assert_success
  [ "$(container_image "$REMOTE_REPO")" = "$EXPECTED_DIGEST" ]
  # The tag was neither pulled nor created by a retag.
  run docker inspect --type image "$REMOTE_IMAGE"
  assert_failure
  run "$D3" rm -f "$REMOTE_REPO"
  assert_success
}

@test "clone: --pull always with the committed digest present does not pull" {
  SSH_HOST=$(ssh_host)
  EXPECTED_DIGEST=$(cat "$BATS_TMPDIR/remote_digest.txt")
  BEFORE=$(image_id "$EXPECTED_DIGEST")

  run "$D3" clone -n "$REMOTE_REPO" --pull always -p "skipHostCheck=true" "ssh://root:root@$SSH_HOST/remote"
  assert_success
  # Image identity is the proof: a pull would replace the image ID. (The
  # "Pulling <commit>" line is dit's data pull.)
  [ "$(image_id "$EXPECTED_DIGEST")" = "$BEFORE" ]
  [ "$(container_image "$REMOTE_REPO")" = "$EXPECTED_DIGEST" ]
  run docker inspect --type image "$REMOTE_IMAGE"
  assert_failure
  run "$D3" rm -f "$REMOTE_REPO"
  assert_success
}

@test "clone: default policy with the committed digest present does not pull" {
  SSH_HOST=$(ssh_host)
  EXPECTED_DIGEST=$(cat "$BATS_TMPDIR/remote_digest.txt")
  BEFORE=$(image_id "$EXPECTED_DIGEST")

  run "$D3" clone -n "$REMOTE_REPO" -p "skipHostCheck=true" "ssh://root:root@$SSH_HOST/remote"
  assert_success
  [ "$(image_id "$EXPECTED_DIGEST")" = "$BEFORE" ]
  [ "$(container_image "$REMOTE_REPO")" = "$EXPECTED_DIGEST" ]
  run "$D3" rm -f "$REMOTE_REPO"
  assert_success
}

@test "clone: --pull never with only the committed digest present runs it" {
  SSH_HOST=$(ssh_host)
  EXPECTED_DIGEST=$(cat "$BATS_TMPDIR/remote_digest.txt")

  run "$D3" clone -n "$REMOTE_REPO" --pull never -p "skipHostCheck=true" "ssh://root:root@$SSH_HOST/remote"
  assert_success
  [ "$(container_image "$REMOTE_REPO")" = "$EXPECTED_DIGEST" ]
  run "$D3" rm -f "$REMOTE_REPO"
  assert_success
}
