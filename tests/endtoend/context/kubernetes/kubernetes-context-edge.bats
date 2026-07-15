#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# E2E Kubernetes context install/reinstall edge cases (ditdotdev/dit#214)
#
# Companion to kubernetes-tests.bats (which runs the full repo lifecycle on
# one context). This file exercises the install-path edges that #214 fixed:
#   - `-t kubernetes` without -n names the context after the TYPE
#     (pre-fix the cobra default named it "docker", colliding with the
#     default docker context)
#   - installing over a stale same-named server container removes THAT
#     container (pre-fix the cleanup removed the hardcoded dit-kubernetes-*
#     names, so custom-named contexts left stale containers behind and the
#     relaunch failed on the name conflict)
#   - invalid -p parameters are rejected before anything is created
#
# Install-only: no PVCs are provisioned, so no storage/snapshot classes are
# needed. Auto-skips when no kubernetes cluster is reachable, matching
# kubernetes-tests.bats.

load '../../test_helper'

EDGE_CTX="k8sedge"

setup_file() {
  if ! kubectl cluster-info >/dev/null 2>&1; then
    export D3_K8S_SKIP=1
    return 0
  fi

  # alpine is the decoy container image for the stale-server test
  docker pull alpine:latest >/dev/null 2>&1 || true

  # Best-effort cleanup of any prior run's state
  "$D3" context uninstall -f kubernetes 2>/dev/null || true
  "$D3" context uninstall -f "$EDGE_CTX" 2>/dev/null || true
  docker rm -f "dit-${EDGE_CTX}-server" 2>/dev/null || true
}

teardown_file() {
  if [ -n "$D3_K8S_SKIP" ]; then return 0; fi
  "$D3" context uninstall -f kubernetes 2>/dev/null || true
  "$D3" context uninstall -f "$EDGE_CTX" 2>/dev/null || true
  docker rm -f "dit-${EDGE_CTX}-server" 2>/dev/null || true
}

setup() {
  if [ -n "$D3_K8S_SKIP" ]; then
    skip "no reachable kubernetes cluster (kubectl cluster-info failed)"
  fi
}

# ---------------------------------------------------------------
# -n defaults to the context type (#214, first half)
# ---------------------------------------------------------------

@test "context install -t kubernetes without -n: context is named after the type" {
  run "$D3" context install -t kubernetes
  assert_success

  # ls shows a context NAMED kubernetes of TYPE kubernetes (name may carry
  # the default marker). Pre-#214 this row was 'docker  kubernetes' or the
  # install failed on the docker-name collision.
  run "$D3" context ls
  assert_success
  assert_output --regexp '^kubernetes( \(\*\))? +kubernetes'
}

@test "type-named context: server container uses the context-derived name" {
  run docker ps --filter "name=^dit-kubernetes-server\$" --format '{{.Names}}'
  assert_success
  assert_output "dit-kubernetes-server"
}

@test "type-named context: uninstall removes it" {
  run "$D3" context uninstall -f kubernetes
  assert_success

  run "$D3" context ls
  assert_success
  refute_output --regexp '^kubernetes '

  run docker ps --filter "name=^dit-kubernetes-server\$" --format '{{.Names}}'
  assert_output ""
}

# ---------------------------------------------------------------
# Reinstall cleanup targets context-derived names (#214, second half)
# ---------------------------------------------------------------

@test "install over a stale same-named server: stale container is replaced, no name conflict" {
  # Plant a running decoy with the context-derived server name, simulating
  # a stale server left behind by a lost/wiped config. DitServerIsAvailable
  # checks dit-<context>-server, so install takes the cleanup path.
  docker run -d --name "dit-${EDGE_CTX}-server" --label ctxtest=stale alpine:latest sleep 3600

  run "$D3" context install -n "$EDGE_CTX" -t kubernetes
  assert_success

  # Exactly one container holds the name, and it is NOT the decoy.
  # Pre-fix: the cleanup removed "dit-kubernetes-server" (nonexistent),
  # the decoy survived, and the relaunch failed on the name conflict.
  run docker ps --filter "name=^dit-${EDGE_CTX}-server\$" --filter "label=ctxtest=stale" --format '{{.Names}}'
  assert_output ""
  run docker ps --filter "name=^dit-${EDGE_CTX}-server\$" --format '{{.Names}}'
  assert_output "dit-${EDGE_CTX}-server"
}

# ---------------------------------------------------------------
# Parameter validation happens before anything is created
# ---------------------------------------------------------------

@test "context install: invalid -p parameter is rejected (key=value required)" {
  run "$D3" context install -n ctxtest-badparam -t kubernetes -p bogus
  assert_failure
  assert_output --partial "invalid context parameter"

  # nothing half-created
  run "$D3" context ls
  refute_output --partial "ctxtest-badparam"
}
