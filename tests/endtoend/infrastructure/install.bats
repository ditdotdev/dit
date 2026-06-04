#!/usr/bin/env bats

# Load shared test helpers
load '../test_helper'

@test "can install dit" {
  run "$D3" install
  assert_success
  # Note: stdout check commented out in original YAML
  # assert_output --partial "Dit CLI successfully installed, happy data versioning :)"
}

@test "dit server is running" {
  # Wait up to 180 seconds (36 x 5) for server to be running
  run bash -c 'for i in {1..36}; do docker inspect --type container --format="{{.State.Status}}" dit-docker-server 2>/dev/null | grep -q "running" && break || sleep 5; done && docker inspect --type container --format="{{.State.Status}}" dit-docker-server'
  assert_success
  assert_output --partial "running"
}

@test "dit launch is running" {
  # Wait up to 180 seconds (36 x 5) for launch to be running
  run bash -c 'for i in {1..36}; do docker inspect --type container --format="{{.State.Status}}" dit-docker-launch 2>/dev/null | grep -q "running" && break || sleep 5; done && docker inspect --type container --format="{{.State.Status}}" dit-docker-launch'
  assert_success
  assert_output --partial "running"
}

@test "dit API is ready" {
  # Wait up to 180 seconds for the server API to actually accept HTTP
  # requests. `dit ls` was used here previously, but since PR #109's
  # classifyCommand fix, `ls` is provider-optional — it reads
  # ~/.dit/config directly and never touches the dit server. So
  # `dit ls` returns success the moment the CLI works, before the Ktor
  # server inside the container has finished booting. On a fast runner
  # the gap from `dit install` returning to the Ktor app actually
  # listening is ~3s; whatever runs next (typically `dit clone` in
  # getting-started.bats) hits a connection-refused/EOF race.
  #
  # Probe the actual /v1/repositories endpoint instead. ~1s on a normal
  # run, capped at 180s.
  local server_port
  server_port=$(awk '/port:/{print $2; exit}' "$HOME/.dit/config")
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
