#!/usr/bin/env bats

# Load shared test helpers
load '../test_helper'

@test "can install datadatdat" {
  run "$D3" install
  assert_success
  # Note: stdout check commented out in original YAML
  # assert_output --partial "Datadatdat CLI successfully installed, happy data versioning :)"
  
  # Wait 20 seconds as in original test
  sleep 20
}

@test "d3 server is running" {
  run docker inspect --type container --format='{{.State.Status}}' datadatdat-docker-server
  assert_success
  assert_output --partial "running"
}

@test "d3 launch is running" {
  run docker inspect --type container --format='{{.State.Status}}' datadatdat-docker-launch
  assert_success
  assert_output --partial "running"
}
