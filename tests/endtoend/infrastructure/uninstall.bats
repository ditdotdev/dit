#!/usr/bin/env bats

# Load shared test helpers
load '../test_helper'

@test "can uninstall datadatdat" {
  run "$D3" uninstall
  assert_success
  assert_output --partial "Uninstalled datadatdat infrastructure"
}

@test "d3 server is not running" {
  run docker inspect --type container --format='{{.State.Status}}' datadatdat-docker-server
  assert_failure
  assert_output --partial "Error response from daemon: No such container: datadatdat-docker-server"
}

@test "d3 launch is not running" {
  run docker inspect --type container --format='{{.State.Status}}' datadatdat-docker-launch
  assert_failure
  assert_output --partial "Error response from daemon: No such container: datadatdat-docker-launch"
}

@test "d3 docker images not removed" {
  run docker inspect --type image --format='{{.RepoTags}}' datadatdat:latest
  assert_success
  assert_output --partial "datadatdat:latest"
}

@test "ZFS pool destroyed after uninstall" {
  # The datadatdat-docker pool should not exist after uninstall.
  run sudo zpool list datadatdat-docker
  assert_failure
}

@test "uninstall output has no teardown warning" {
  # Re-install so we can test uninstall output for warnings
  run "$D3" install
  assert_success
  run "$D3" uninstall
  assert_success
  # Should NOT contain the teardown failure warning
  refute_output --partial "Failed to teardown datadatdat servers"
}

@test "re-install datadatdat" {
  run "$D3" install
  assert_success
}

@test "can uninstall d3 and remove docker images" {
  run "$D3" uninstall --remove-images
  assert_success
  assert_output --partial "Removing Datadatdat Docker image"
}

# This test is commented out in the original YAML as it sometimes fails
# @test "d3 docker images removed" {
#   run docker inspect --type image --format='{{.RepoTags}}' datadatdat:latest
#   assert_failure
#   assert_output --partial "Error: No such image: datadatdat:latest"
# }
