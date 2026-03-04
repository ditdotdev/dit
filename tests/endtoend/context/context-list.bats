#!/usr/bin/env bats

# E2E Context List Tests
# Tests d3 context ls and d3 context default (read operations)

# Load shared test helpers
load '../test_helper'

@test "d3 context ls shows default docker context" {
  run "$D3" context ls
  assert_success
  assert_output --partial "docker"
}

@test "d3 context ls shows column headers" {
  run "$D3" context ls
  assert_success
  assert_output --partial "NAME"
  assert_output --partial "TYPE"
}

@test "d3 context default shows current default" {
  run "$D3" context default
  assert_success
  assert_output --partial "docker"
}
