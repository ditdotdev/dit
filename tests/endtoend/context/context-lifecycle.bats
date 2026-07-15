#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# E2E Context Lifecycle Tests
#
# Exercises every subcommand of `dit context --help`:
#   install, ls, default, uninstall
# Plus the global `--context` flag routing on commands that consume it:
#   dit ls, dit run, dit rm
#
# The existing context-list.bats covers read-only smoke checks. This file
# covers the full lifecycle and the routing behavior that was silently
# broken (see GitHub issue for --context flag ignored by run/rm/ls).

load '../test_helper'

SECOND_CTX="ctxtest-alt"
TEST_REPO="ctxtest-nginx"

setup_file() {
  # Best-effort cleanup in case a prior run left state behind
  "$D3" rm -f "$TEST_REPO" --context "$SECOND_CTX" 2>/dev/null || true
  "$D3" rm -f "$TEST_REPO" --context docker 2>/dev/null || true
  "$D3" context uninstall -f "$SECOND_CTX" 2>/dev/null || true

  # Ensure the pre-existing 'docker' context's server is running. Some
  # prior runs may have wiped it. We need it up before installing the
  # second context so we can assert that installing the second one
  # leaves the first one's server alone.
  local docker_server_up
  docker_server_up=$(docker ps --filter "name=^dit-docker-server\$" --format '{{.Names}}')
  if [ -z "$docker_server_up" ]; then
    "$D3" context uninstall -f docker 2>/dev/null || true
    "$D3" context install -n docker -t docker
  fi

  # nginx-test image needed for `dit run` tests
  docker pull ditdotdev/nginx-test:latest >/dev/null 2>&1 || true
  docker tag ditdotdev/nginx-test:latest nginx-test 2>/dev/null || true

  # Install a second docker-type context alongside the existing 'docker' one
  "$D3" context install -n "$SECOND_CTX" -t docker

  # Normalize: docker is the default for the duration of these tests
  "$D3" context default docker 2>/dev/null || true
}

teardown_file() {
  "$D3" rm -f "$TEST_REPO" --context "$SECOND_CTX" 2>/dev/null || true
  "$D3" rm -f "$TEST_REPO" --context docker 2>/dev/null || true
  "$D3" context uninstall -f "$SECOND_CTX" 2>/dev/null || true
  # Make sure docker is the default context after cleanup
  "$D3" context default docker 2>/dev/null || true
}

# ---------------------------------------------------------------
# dit context install / ls / uninstall lifecycle
# ---------------------------------------------------------------

@test "context install: second context appears in ls" {
  run "$D3" context ls
  assert_success
  assert_output --partial "docker"
  assert_output --partial "$SECOND_CTX"
}

@test "context ls: shows NAME and TYPE headers" {
  run "$D3" context ls
  assert_success
  assert_output --partial "NAME"
  assert_output --partial "TYPE"
}

@test "context ls: marks the default context with (*)" {
  run "$D3" context ls
  assert_success
  # The default context (docker by convention) should be annotated
  assert_output --partial "docker (*)"
}

@test "context install: does not wipe other docker-type contexts' running servers" {
  # The pre-existing 'docker' context's server was alive before setup_file
  # ran (enforced in setup_file). Installing $SECOND_CTX (also docker-type)
  # must not remove the 'docker' context's server — only its own predecessor
  # (if any) or nothing.
  run docker ps --filter "name=^dit-docker-server\$" --format '{{.Names}}'
  assert_success
  assert_output "dit-docker-server"
}

# ---------------------------------------------------------------
# dit context default (get / set)
# ---------------------------------------------------------------

@test "context default (no args): prints current default" {
  run "$D3" context default
  assert_success
  assert_output --partial "docker"
}

@test "context default <name>: switches default, ls reflects it" {
  run "$D3" context default "$SECOND_CTX"
  assert_success

  run "$D3" context default
  assert_success
  assert_output --partial "$SECOND_CTX"

  run "$D3" context ls
  assert_success
  assert_output --partial "$SECOND_CTX (*)"

  # Restore docker as default for remaining tests
  run "$D3" context default docker
  assert_success
}

# ---------------------------------------------------------------
# --context flag routing: dit run
#
# Side note on test scope: we only exercise $SECOND_CTX here, not the
# pre-existing 'docker' context. `dit context install -n <new> -t docker`
# currently wipes the 'dit-docker-server' container (hardcoded
# container name in Install.go, separate pre-existing bug), so the
# pre-existing docker context's server is not necessarily alive during
# these tests and cannot be safely queried.
# ---------------------------------------------------------------

@test "dit run --context: repo is created and tagged with the -n name" {
  run "$D3" run nginx-test -n "$TEST_REPO" --context "$SECOND_CTX"
  assert_success
  # Output must reference our -n value, not the context name (the bug
  # fixed here caused the repo to inherit the context name instead)
  assert_output --partial "Creating repository $TEST_REPO"
  refute_output --partial "Creating repository $SECOND_CTX"
}

@test "dit run --context: a docker container with the repo name is running" {
  # Direct proof that -n and --context were honored: docker ps shows
  # the container named $TEST_REPO (not $SECOND_CTX)
  run docker ps --filter "name=^${TEST_REPO}\$" --format '{{.Names}}'
  assert_success
  assert_output "$TEST_REPO"
}

@test "dit ls --context <name>: shows repos that live on that context" {
  run "$D3" ls --context "$SECOND_CTX"
  assert_success
  assert_output --partial "$TEST_REPO"
  # CONTEXT column should show $SECOND_CTX on the repo's row
  echo "$output" | grep -E "$SECOND_CTX.*$TEST_REPO" || {
    echo "Expected a line in 'dit ls --context $SECOND_CTX' with $TEST_REPO on context $SECOND_CTX"
    echo "Output: $output"
    return 1
  }
}

@test "dit ls (no --context): shows repos from every configured context" {
  # $TEST_REPO is on $SECOND_CTX (created by the earlier run --context test).
  # With no --context flag, `dit ls` must show repos across all contexts,
  # not silently filter to the default one. Regression guard for a bug
  # where initProvider's isInstall fallback set context="docker" when
  # args contained "ls", causing the listCmd's --context filter branch
  # to fire with "docker" and hide repos on other contexts.
  run "$D3" ls
  assert_success
  assert_output --partial "$TEST_REPO"
  echo "$output" | grep -E "$SECOND_CTX.*$TEST_REPO" || {
    echo "Expected $TEST_REPO to show under context $SECOND_CTX in 'dit ls'"
    echo "Output: $output"
    return 1
  }
}

# ---------------------------------------------------------------
# --context flag routing: dit rm
# ---------------------------------------------------------------

@test "dit rm --context <name>: routes delete to the right context" {
  # Confirm present before
  run "$D3" ls --context "$SECOND_CTX"
  assert_output --partial "$TEST_REPO"

  run "$D3" rm -f "$TEST_REPO" --context "$SECOND_CTX"
  assert_success

  # Must be gone from $SECOND_CTX
  run "$D3" ls --context "$SECOND_CTX"
  assert_success
  refute_output --partial "$TEST_REPO"

  # And the docker container is gone
  run docker ps --filter "name=^${TEST_REPO}\$" --format '{{.Names}}'
  assert_success
  assert_output ""
}

# ---------------------------------------------------------------
# Error cases
# ---------------------------------------------------------------

@test "dit ls --context <nonexistent>: exits non-zero or reports no such context" {
  run "$D3" ls --context definitely-not-a-real-context
  # Either non-zero exit OR the output mentions 'no such context'.
  # What's NOT acceptable: exit 0 with repos from some other context,
  # which is the silently-wrong behavior we are fixing.
  if [ "$status" -eq 0 ]; then
    echo "$output" | grep -qi "no such context" || {
      echo "Expected failure or 'no such context' message, got clean success"
      echo "Output: $output"
      return 1
    }
  fi
}

# ---------------------------------------------------------------
# dit context uninstall (validated implicitly by teardown_file removing
# $SECOND_CTX; this test confirms the uninstall command exists and
# accepts the force flag)
# ---------------------------------------------------------------

@test "context uninstall --help: documents -f / --force flag" {
  run "$D3" context uninstall --help
  assert_success
  assert_output --partial "force"
}

# ---------------------------------------------------------------
# dit context uninstall <name>: must target the NAMED context only
# ---------------------------------------------------------------

@test "context uninstall: output names the target context" {
  local UCTX="ctxtest-visible"
  "$D3" context uninstall -f "$UCTX" 2>/dev/null || true
  run "$D3" context install -n "$UCTX" -t docker
  assert_success

  run "$D3" context uninstall -f "$UCTX"
  assert_success
  # The context name must appear in the output so the user knows what was uninstalled
  assert_output --partial "$UCTX"
}

@test "context uninstall: on orphaned state, reports nothing to uninstall (no false success)" {
  local UCTX="ctxtest-empty-uninstall"
  "$D3" context uninstall -f "$UCTX" 2>/dev/null || true

  # Install, then externally nuke the containers + volume to simulate the
  # "already cleaned up" state that previously produced a misleading
  # "Dit Docker volume removed / Uninstalled dit
  # infrastructure" output despite nothing actually being removed.
  run "$D3" context install -n "$UCTX" -t docker
  assert_success
  docker rm -f "dit-${UCTX}-server" "dit-${UCTX}-launch" 2>/dev/null || true
  docker volume rm -f "dit-${UCTX}-data" 2>/dev/null || true

  run "$D3" context uninstall -f "$UCTX"
  assert_success
  assert_output --partial "nothing to uninstall"
}

# ---------------------------------------------------------------
# Install edge cases: name defaulting, collisions, unknown types
# (regressions for ditdotdev/dit#214 and the nil-provider panic)
# ---------------------------------------------------------------

@test "context install without -n: name defaults to the type and collides with 'docker'" {
  # Post-#214 an omitted -n resolves to the resolved -t value. This suite
  # guarantees a context named 'docker' exists, so a second docker-type
  # install without -n must fail on the 'docker' name - NOT succeed under
  # some other name.
  run "$D3" context install -t docker
  assert_failure
  assert_output --partial "context 'docker' already exists"
}

@test "context install: duplicate explicit -n fails with a clear error" {
  run "$D3" context install -n "$SECOND_CTX" -t docker
  assert_failure
  assert_output --partial "context '$SECOND_CTX' already exists"
}

@test "context install: unknown -t type fails cleanly, no panic" {
  run "$D3" context install -n ctxtest-badtype -t bogus
  assert_failure
  assert_output --partial "unknown context type 'bogus'"
  refute_output --partial "panic"

  # nothing half-created
  run "$D3" context ls
  refute_output --partial "ctxtest-badtype"
}

@test "context uninstall: nonexistent context reports no such context" {
  run "$D3" context uninstall -f definitely-not-installed
  assert_failure
  assert_output --partial "no such context"
}

@test "context default <nonexistent>: fails and the existing default survives" {
  run "$D3" context default definitely-not-installed
  assert_failure
  assert_output --partial "no such context"

  # Regression: SetDefault used to un-default everything on an unknown
  # name, leaving NO default context at all.
  run "$D3" context default
  assert_success
  assert_output --partial "docker"
}

@test "context uninstall <name>: removes the named context's server, not the default's" {
  local KILL_CTX="ctxtest-kill"
  # Install a sacrificial docker-type context
  "$D3" context uninstall -f "$KILL_CTX" 2>/dev/null || true
  run "$D3" context install -n "$KILL_CTX" -t docker
  assert_success

  # Sanity: both servers are running
  run docker ps --filter "name=^dit-docker-server\$" --format '{{.Names}}'
  assert_output "dit-docker-server"
  run docker ps --filter "name=^dit-${KILL_CTX}-server\$" --format '{{.Names}}'
  assert_output "dit-${KILL_CTX}-server"

  # Uninstall the NAMED context
  run "$D3" context uninstall -f "$KILL_CTX"
  assert_success

  # The named context's server must be gone
  run docker ps --filter "name=^dit-${KILL_CTX}-server\$" --format '{{.Names}}'
  assert_success
  assert_output ""

  # The default (docker) context must STILL be running — previously the
  # bug in context.go uninstall resolved args[0] via ByName() which fell
  # back to Default() and nuked the docker context instead.
  run docker ps --filter "name=^dit-docker-server\$" --format '{{.Names}}'
  assert_success
  assert_output "dit-docker-server"

  # And the named context is out of the config
  run "$D3" context ls
  assert_success
  refute_output --partial "$KILL_CTX"
}
