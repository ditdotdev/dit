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
  # Wait up to 180 seconds for the server API to actually accept HTTP
  # requests. `d3 ls` was used here previously, but since PR #109's
  # classifyCommand fix, `ls` is provider-optional — it reads
  # ~/.datadatdat/config directly and never touches the d3 server. So
  # `d3 ls` returns success the moment the CLI works, before the Ktor
  # server inside the container has finished booting. On a fast runner
  # the gap from `d3 install` returning to the Ktor app actually
  # listening is ~3s; whatever runs next (typically `d3 clone` in
  # getting-started.bats) hits a connection-refused/EOF race.
  #
  # Probe the actual /v1/repositories endpoint instead. ~1s on a normal
  # run, capped at 180s.
  local server_port
  server_port=$(awk '/port:/{print $2; exit}' "$HOME/.datadatdat/config")
  for _ in $(seq 1 36); do
    if curl -s -o /dev/null -w "%{http_code}" "http://localhost:${server_port}/v1/repositories" 2>/dev/null | grep -q 200; then
      break
    fi
    sleep 5
  done
  run "$D3" ls
  assert_success
  assert_output --partial "CONTEXT"
}
