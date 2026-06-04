#!/usr/bin/env bats

# E2E Upgrade Tests
# Tests dit upgrade (idempotent - upgrading to same version)

# Load shared test helpers
load '../test_helper'

@test "dit upgrade runs without error" {
  run "$D3" upgrade
  assert_success
}

@test "dit server is still running after upgrade" {
  run bash -c 'for i in {1..36}; do docker inspect --type container --format="{{.State.Status}}" dit-docker-server 2>/dev/null | grep -q "running" && break || sleep 5; done && docker inspect --type container --format="{{.State.Status}}" dit-docker-server'
  assert_success
  assert_output --partial "running"
}

@test "dit launch is still running after upgrade" {
  run bash -c 'for i in {1..36}; do docker inspect --type container --format="{{.State.Status}}" dit-docker-launch 2>/dev/null | grep -q "running" && break || sleep 5; done && docker inspect --type container --format="{{.State.Status}}" dit-docker-launch'
  assert_success
  assert_output --partial "running"
}

@test "dit API is still responsive after upgrade" {
  for i in $(seq 1 36); do
    run "$D3" ls
    if [ "$status" -eq 0 ] && echo "$output" | grep -q "CONTEXT"; then
      break
    fi
    sleep 5
  done
  run "$D3" ls
  assert_success
  assert_output --partial "CONTEXT"
}
