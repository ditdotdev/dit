#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# E2E Context List Tests
# Tests dit context ls and dit context default (read operations)

# Load shared test helpers
load '../test_helper'

@test "dit context ls shows default docker context" {
  run "$D3" context ls
  assert_success
  assert_output --partial "docker"
}

@test "dit context ls shows column headers" {
  run "$D3" context ls
  assert_success
  assert_output --partial "NAME"
  assert_output --partial "TYPE"
}

@test "dit context default shows current default" {
  run "$D3" context default
  assert_success
  assert_output --partial "docker"
}
