#!/usr/bin/env bats

# Load shared test helpers
load '../test_helper'

# Cleanup after all tests
teardown_file() {
  # Best effort cleanup - remove any leftover repositories
  "$D3" rm -f one 2>/dev/null || true
  "$D3" rm -f two 2>/dev/null || true
  
  # Remove any leftover contexts
  "$D3" uninstall -f --context one 2>/dev/null || true
  "$D3" uninstall -f --context two 2>/dev/null || true
}

@test "can install context one" {
  run "$D3" context install -n one -t docker
  assert_success
  
  # Wait for context to be fully initialized
  sleep 20
}

@test "d3 server (context one) is running" {
  run docker inspect --type container --format='{{.State.Status}}' datadatdat-one-server
  assert_success
  assert_output --partial "running"
}

@test "d3 launch (context one) is running" {
  run docker inspect --type container --format='{{.State.Status}}' datadatdat-one-launch
  assert_success
  assert_output --partial "running"
}

@test "can install context two" {
  run "$D3" context install -n two -t docker
  assert_success
  
  # Wait for context to be fully initialized
  sleep 20
}

@test "d3 server (context two) is running" {
  run docker inspect --type container --format='{{.State.Status}}' datadatdat-two-server
  assert_success
  assert_output --partial "running"
}

@test "d3 launch (context two) is running" {
  run docker inspect --type container --format='{{.State.Status}}' datadatdat-two-launch
  assert_success
  assert_output --partial "running"
}

@test "can run one/mongo-one in context one" {
  run "$D3" run -P -n one/mongo-one mongo --context one
  assert_success
  assert_output --partial "Running controlled container one"
  
  # Wait for container to be ready
  sleep 5
}

@test "can get status of one/mongo-one" {
  run "$D3" status one
  assert_success
}

@test "can run two/mongo-two in context two" {
  run "$D3" run -P -n two/mongo-two mongo --context two
  assert_success
  assert_output --partial "Running controlled container two"
  
  # Wait for container to be ready
  sleep 5
}

@test "can change default context to two" {
  run "$D3" context default two
  assert_success
}

@test "default context is two" {
  run "$D3" context default
  assert_success
  assert_output "two"
}

@test "can get status of one/mongo-one from default context" {
  run "$D3" status one
  assert_success
}

@test "can remove one/mongo-one" {
  run "$D3" rm -f one
  assert_success
  assert_output --partial "Removing repository one"
}

@test "can remove two/mongo-two" {
  run "$D3" rm -f two
  assert_success
  assert_output --partial "Removing repository two"
}

@test "can uninstall context one" {
  run "$D3" uninstall -f --context one
  assert_success
  assert_output --partial "Uninstalled datadatdat infrastructure"
}

@test "can uninstall context two" {
  run "$D3" uninstall -f --context two
  assert_success
  assert_output --partial "Uninstalled datadatdat infrastructure"
}
