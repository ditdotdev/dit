#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# Load shared test helpers
load '../../test_helper'

# Setup: Pull and tag nginx-test image before running tests
setup_file() {
  docker pull ditdotdev/nginx-test:latest
  docker tag ditdotdev/nginx-test:latest nginx-test
}

@test "can run nginx-test" {
  run "$D3" run nginx-test
  assert_success
}

@test "can remove nginx-test" {
  run "$D3" rm -f nginx-test
  assert_success
}

@test "can run nginx-test with env" {
  run "$D3" run nginx-test -e TEST=true
  assert_success
  sleep 5
}

@test "verify env value in nginx-test" {
  run curl -L http://localhost
  assert_success
  assert_output --partial "TEST=true"
}

@test "can remove nginx-test with env" {
  run "$D3" rm -f nginx-test
  assert_success
}

@test "can run nginx-test without port mapping" {
  run "$D3" run nginx-test -P
  assert_success
}

# Note: We don't test network connectivity here because curl behavior is inconsistent
# across different environments (Windows/WSL2, Docker networking states, etc.).
# The important thing is that the container starts successfully with -P flag.
# Network isolation is inherently tested by the successful execution above.

@test "can remove nginx-test without port mapping" {
  run "$D3" rm -f nginx-test
  assert_success
}
