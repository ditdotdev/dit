#!/usr/bin/env bats

# Load shared test helpers
load '../test_helper'

@test "can uninstall dit" {
  run "$D3" uninstall
  assert_success
  assert_output --partial "Uninstalled dit infrastructure"
}

@test "dit server is not running" {
  run docker inspect --type container --format='{{.State.Status}}' dit-docker-server
  assert_failure
  assert_output --partial "Error response from daemon: No such container: dit-docker-server"
}

@test "dit launch is not running" {
  run docker inspect --type container --format='{{.State.Status}}' dit-docker-launch
  assert_failure
  assert_output --partial "Error response from daemon: No such container: dit-docker-launch"
}

@test "dit docker images not removed" {
  run docker inspect --type image --format='{{.RepoTags}}' dit:latest
  assert_success
  assert_output --partial "dit:latest"
}

@test "ZFS pool destroyed after uninstall" {
  # TODO: https://github.com/ditdotdev/ditdotdev/issues/90
  # Skipped: uninstall does not destroy ZFS pools that have datasets or
  # were not created by the current install. Needs investigation into
  # whether uninstall should force-destroy or leave pools intact.
  skip "ZFS pool destruction behavior under investigation"
  run sudo zpool list dit-docker
  assert_failure
}

@test "uninstall output has no teardown warning" {
  # Re-install so we can test uninstall output for warnings
  run "$D3" install
  assert_success
  run "$D3" uninstall
  assert_success
  # Should NOT contain the teardown failure warning
  refute_output --partial "Failed to teardown dit servers"
}

@test "re-install dit" {
  run "$D3" install
  assert_success
}

@test "can uninstall dit and remove docker images" {
  run "$D3" uninstall --remove-images
  assert_success
  assert_output --partial "Removing Dit Docker image"
}

# This test is commented out in the original YAML as it sometimes fails
# @test "dit docker images removed" {
#   run docker inspect --type image --format='{{.RepoTags}}' dit:latest
#   assert_failure
#   assert_output --partial "Error: No such image: dit:latest"
# }
