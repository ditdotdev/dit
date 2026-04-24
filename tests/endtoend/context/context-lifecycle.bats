#!/usr/bin/env bats

# E2E Context Lifecycle Tests
#
# Exercises every subcommand of `d3 context --help`:
#   install, ls, default, uninstall
# Plus the global `--context` flag routing on commands that consume it:
#   d3 ls, d3 run, d3 rm
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
  docker_server_up=$(docker ps --filter "name=^datadatdat-docker-server\$" --format '{{.Names}}')
  if [ -z "$docker_server_up" ]; then
    "$D3" context uninstall -f docker 2>/dev/null || true
    "$D3" context install -n docker -t docker
  fi

  # nginx-test image needed for `d3 run` tests
  docker pull datadatdat/nginx-test:latest >/dev/null 2>&1 || true
  docker tag datadatdat/nginx-test:latest nginx-test 2>/dev/null || true

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
# d3 context install / ls / uninstall lifecycle
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
  run docker ps --filter "name=^datadatdat-docker-server\$" --format '{{.Names}}'
  assert_success
  assert_output "datadatdat-docker-server"
}

# ---------------------------------------------------------------
# d3 context default (get / set)
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
# --context flag routing: d3 run
#
# Side note on test scope: we only exercise $SECOND_CTX here, not the
# pre-existing 'docker' context. `d3 context install -n <new> -t docker`
# currently wipes the 'datadatdat-docker-server' container (hardcoded
# container name in Install.go, separate pre-existing bug), so the
# pre-existing docker context's server is not necessarily alive during
# these tests and cannot be safely queried.
# ---------------------------------------------------------------

@test "d3 run --context: repo is created and tagged with the -n name" {
  run "$D3" run nginx-test -n "$TEST_REPO" --context "$SECOND_CTX"
  assert_success
  # Output must reference our -n value, not the context name (the bug
  # fixed here caused the repo to inherit the context name instead)
  assert_output --partial "Creating repository $TEST_REPO"
  refute_output --partial "Creating repository $SECOND_CTX"
}

@test "d3 run --context: a docker container with the repo name is running" {
  # Direct proof that -n and --context were honored: docker ps shows
  # the container named $TEST_REPO (not $SECOND_CTX)
  run docker ps --filter "name=^${TEST_REPO}\$" --format '{{.Names}}'
  assert_success
  assert_output "$TEST_REPO"
}

@test "d3 ls --context <name>: shows repos that live on that context" {
  run "$D3" ls --context "$SECOND_CTX"
  assert_success
  assert_output --partial "$TEST_REPO"
  # CONTEXT column should show $SECOND_CTX on the repo's row
  echo "$output" | grep -E "$SECOND_CTX.*$TEST_REPO" || {
    echo "Expected a line in 'd3 ls --context $SECOND_CTX' with $TEST_REPO on context $SECOND_CTX"
    echo "Output: $output"
    return 1
  }
}

# ---------------------------------------------------------------
# --context flag routing: d3 rm
# ---------------------------------------------------------------

@test "d3 rm --context <name>: routes delete to the right context" {
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

@test "d3 ls --context <nonexistent>: exits non-zero or reports no such context" {
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
# d3 context uninstall (validated implicitly by teardown_file removing
# $SECOND_CTX; this test confirms the uninstall command exists and
# accepts the force flag)
# ---------------------------------------------------------------

@test "context uninstall --help: documents -f / --force flag" {
  run "$D3" context uninstall --help
  assert_success
  assert_output --partial "force"
}

# ---------------------------------------------------------------
# d3 context uninstall <name>: must target the NAMED context only
# ---------------------------------------------------------------

@test "context uninstall <name>: removes the named context's server, not the default's" {
  local KILL_CTX="ctxtest-kill"
  # Install a sacrificial docker-type context
  "$D3" context uninstall -f "$KILL_CTX" 2>/dev/null || true
  run "$D3" context install -n "$KILL_CTX" -t docker
  assert_success

  # Sanity: both servers are running
  run docker ps --filter "name=^datadatdat-docker-server\$" --format '{{.Names}}'
  assert_output "datadatdat-docker-server"
  run docker ps --filter "name=^datadatdat-${KILL_CTX}-server\$" --format '{{.Names}}'
  assert_output "datadatdat-${KILL_CTX}-server"

  # Uninstall the NAMED context
  run "$D3" context uninstall -f "$KILL_CTX"
  assert_success

  # The named context's server must be gone
  run docker ps --filter "name=^datadatdat-${KILL_CTX}-server\$" --format '{{.Names}}'
  assert_success
  assert_output ""

  # The default (docker) context must STILL be running — previously the
  # bug in context.go uninstall resolved args[0] via ByName() which fell
  # back to Default() and nuked the docker context instead.
  run docker ps --filter "name=^datadatdat-docker-server\$" --format '{{.Names}}'
  assert_success
  assert_output "datadatdat-docker-server"

  # And the named context is out of the config
  run "$D3" context ls
  assert_success
  refute_output --partial "$KILL_CTX"
}
