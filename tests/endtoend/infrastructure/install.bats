#!/usr/bin/env bats

# Load shared test helpers
load '../test_helper'

@test "can install datadatdat" {
  run "$D3" install
  assert_success
  # Note: stdout check commented out in original YAML
  # assert_output --partial "Datadatdat CLI successfully installed, happy data versioning :)"
}

@test "d3 server is running" {
  # Wait up to 180 seconds (36 x 5) for server to be running
  run bash -c 'for i in {1..36}; do docker inspect --type container --format="{{.State.Status}}" datadatdat-docker-server 2>/dev/null | grep -q "running" && break || sleep 5; done && docker inspect --type container --format="{{.State.Status}}" datadatdat-docker-server'
  assert_success
  assert_output --partial "running"
}

@test "d3 launch is running" {
  # Wait up to 180 seconds (36 x 5) for launch to be running
  run bash -c 'for i in {1..36}; do docker inspect --type container --format="{{.State.Status}}" datadatdat-docker-launch 2>/dev/null | grep -q "running" && break || sleep 5; done && docker inspect --type container --format="{{.State.Status}}" datadatdat-docker-launch'
  assert_success
  assert_output --partial "running"
}

@test "d3 API is ready" {
  # Wait up to 180 seconds (36 x 5) for the server API to accept requests
  run bash -c 'PORT=$(docker port datadatdat-docker-server 2>/dev/null | head -1 | grep -oP "\d+$"); for i in {1..36}; do curl -sf "http://localhost:$PORT/v1/repositories" > /dev/null 2>&1 && break || sleep 5; done && curl -sf "http://localhost:$PORT/v1/repositories" > /dev/null'
  assert_success
}
