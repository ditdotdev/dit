#!/usr/bin/env bats

# E2E Kubernetes + remote tests
# These run as part of `make e2e-server` and exercise commit / push / clone
# against the datadatdat dev server (docker compose) AND the public S3 / S3web
# hello-world remotes. Auto-skip when no kubernetes cluster is reachable OR
# the dev datadatdat-server isn't healthy.
#
# Pre-requisites for full coverage:
#   - kubectl + reachable cluster (e.g. minikube)
#   - datadatdat-server dev stack: docker compose up -d in datadatdat-server
#   - AWS_* env vars for s3 push/pull. Public hello-world clone works without
#     credentials because s3web hits the bucket's website endpoint.
#
# Test ordering: setup_file installs the context and runs postgres; tests
# build commits on top of that pod, push, then clone into fresh repos.

load '../../test_helper'
load '../../remotes/datadatdat/env'

CTX="k8sremotetest"
REPO="commit-test"
S3WEB_URL="s3web://demo-datadatdat.s3-website-us-west-2.amazonaws.com/hello-world/postgres"
S3_URL="s3://demo-datadatdat/hello-world/postgres"

# Wait for a Pod's Ready condition. Default 36 iterations × 5s = 180s.
# Mirrors the helper in kubernetes-tests.bats — bounded poll on the
# explicit condition we want, rather than `kubectl wait --timeout=Ns`
# where the wall clock is decoupled from the predicate.
wait_pod_ready() {
  local pod="$1"
  local iters="${2:-36}"
  for _ in $(seq 1 "$iters"); do
    if kubectl get "$pod" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null | grep -q '^True$'; then
      return 0
    fi
    sleep 5
  done
  echo "$pod did not reach Ready within $((iters * 5))s"
  kubectl describe "$pod" 2>/dev/null || true
  return 1
}

# Wait for postgres in the given pod to actually accept psql
# connections. Pod-Ready (per wait_pod_ready) only means the postgres
# process started; this fills the postgres-startup-to-listening gap
# that surfaces as `psql: ... no such file or directory` on the
# socket. Mirrors the helper in kubernetes-tests.bats.
wait_postgres_ready() {
  local pod="$1"
  local iters="${2:-30}"
  for _ in $(seq 1 "$iters"); do
    if kubectl exec "$pod" -- psql -U postgres -c "select 1" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "$pod postgres did not accept psql connections within $iters seconds"
  return 1
}

setup_file() {
  if ! kubectl cluster-info >/dev/null 2>&1; then
    export D3_K8S_SKIP=1
    return 0
  fi
  if ! curl -s "${GATEWAY}/health" 2>/dev/null | grep -q "${HEALTH_EXPECT}"; then
    export D3_K8S_SKIP=1
    return 0
  fi

  docker pull postgres:latest >/dev/null 2>&1 || true

  # Cleanup any prior state for these test names
  for r in "$REPO" hello-clone-datadatdat hello-clone-s3 hello-clone-s3web; do
    "$D3" rm -f "$r" --context "$CTX" 2>/dev/null || true
  done
  "$D3" context uninstall -f "$CTX" 2>/dev/null || true
  rm -f "$HOME/.datadatdat/portforward-${REPO}-"*.pid 2>/dev/null || true

  # The d3 server itself talks to the remote API by hostname (the
  # `datadatdat-api-gateway` compose hostname), so the server has to
  # be on the compose network — that's done by the network connect
  # below. But the server then spawns operation jobs as k8s pods, and
  # those pods run in the cluster's CNI network where docker DNS isn't
  # available, so the in-pod upload would otherwise fail with `Name
  # or service not known`. Plumb a hostAlias into the d3 server's env
  # (server reads DATADATDAT_K8S_POD_HOST_ALIASES and applies it as
  # hostAliases on every job pod) BEFORE installing the context.
  #
  # `=auto` tells the server to discover the right IP at job-creation
  # time (datadatdat-server PR #157). The server probes a busybox pod
  # for `host.minikube.internal` (works on docker/hyperv/hyperkit/kvm2/
  # qemu drivers) and falls back to the node InternalIP for
  # --driver=none on Linux CI (where node = host = where api-gateway
  # is bound). This replaces the prior hard-coded `kubectl get nodes`
  # heuristic, which only worked on --driver=none and broke local Mac
  # / Windows minikube setups.
  if [ -z "$DATADATDAT_K8S_POD_HOST_ALIASES" ]; then
    export DATADATDAT_K8S_POD_HOST_ALIASES="datadatdat-api-gateway=auto"
  fi

  "$D3" context install -n "$CTX" -t kubernetes

  # The kubernetes-context d3 server runs in the default docker bridge
  # network, so it can't resolve `datadatdat-api-gateway` when the
  # bats remote URL is a compose-internal hostname. The local-context
  # d3 server is wired up by the workflow's "Connect Docker networks"
  # step; the kubernetes-context server is created later (here) so we
  # have to wire it up the same way ourselves. Idempotent / no-op
  # outside the CI compose env.
  docker network connect datadatdat-docker "datadatdat-${CTX}-server" 2>/dev/null || true

  # Wait for the embedded Ktor app to start serving /v1/.
  local server_port
  server_port=$(awk -v ctx="$CTX:" '$0 ~ ctx{f=1} f && /port:/{print $2; exit}' "$HOME/.datadatdat/config")
  for _ in $(seq 1 60); do
    if curl -s -o /dev/null -w "%{http_code}" "http://localhost:${server_port}/v1/repositories" 2>/dev/null | grep -q 200; then
      break
    fi
    sleep 2
  done
}

teardown_file() {
  if [ -n "$D3_K8S_SKIP" ]; then return 0; fi
  # Capture k8stest d3 server logs BEFORE uninstall removes the
  # container. Two routing details:
  #
  #  - The workflow's later "Show compose and k8s logs" step globs
  #    `docker ps -a` which excludes removed containers, so by the
  #    time it runs the k8stest server is gone and `docker logs`
  #    returns "no such container". This block has to run inside
  #    BATS where the container still exists.
  #
  #  - BATS captures regular stdout from teardown_file and only shows
  #    it on certain failure paths (we observed it being entirely
  #    swallowed in PR #639 run 25077761780 even with failed tests).
  #    File descriptor 3 is BATS's "always show" channel — output
  #    written there bypasses BATS's filtering and surfaces in the
  #    test log unconditionally. Same FD 3 convention used by the
  #    rest of the BATS ecosystem (`bats-assert`, etc.).
  #
  # Failures in `d3 push` / `d3 commit` from the tests above return
  # a 500 from inside the server (api-gateway never sees the request),
  # so without these logs there's no way to diagnose Bug 2.
  {
    echo "=== docker logs datadatdat-${CTX}-server (pre-teardown) ==="
    # Dump the full log. Earlier `tail -300` was too small: tests 9/10
    # are chatty with /v1/.../status polls that pushed the test-7 push
    # 500 (the thing we actually need to diagnose) out of the window.
    docker logs "datadatdat-${CTX}-server" 2>&1 || true
    echo "=== end of datadatdat-${CTX}-server logs ==="
  } >&3
  for r in "$REPO" hello-clone-datadatdat hello-clone-s3 hello-clone-s3web; do
    "$D3" rm -f "$r" --context "$CTX" 2>/dev/null || true
  done
  "$D3" context uninstall -f "$CTX" 2>/dev/null || true
}

setup() {
  if [ -n "$D3_K8S_SKIP" ]; then
    skip "no reachable kubernetes cluster or datadatdat dev server not healthy"
  fi
}

# ---------------------------------------------------------------
# Run postgres on the k8s context
# ---------------------------------------------------------------

@test "k8s + remote: postgres comes up" {
  run "$D3" run postgres:latest -n "$REPO" -e POSTGRES_HOST_AUTH_METHOD=trust --context "$CTX"
  assert_success
  wait_pod_ready "pod/${REPO}-0"
  # Pod-Ready means postgres process started; the next test does
  # `kubectl exec ... psql` immediately, so wait until postgres is
  # actually accepting connections to avoid the unix-socket race.
  wait_postgres_ready "${REPO}-0"
}

# ---------------------------------------------------------------
# Three commits, each adding a different table so we can verify the
# clone path round-trips real data.
# ---------------------------------------------------------------

@test "commit 1: write table t1 and d3 commit" {
  run kubectl exec "${REPO}-0" -- psql -U postgres -c "create table t1(id int); insert into t1 values(1)"
  assert_success
  run "$D3" commit -m "add t1" --context "$CTX" "$REPO"
  assert_success
}

@test "commit 2: write table t2 and d3 commit" {
  run kubectl exec "${REPO}-0" -- psql -U postgres -c "create table t2(id int); insert into t2 values(2)"
  assert_success
  run "$D3" commit -m "add t2" --context "$CTX" "$REPO"
  assert_success
}

@test "commit 3: write table t3 and d3 commit" {
  run kubectl exec "${REPO}-0" -- psql -U postgres -c "create table t3(id int); insert into t3 values(3)"
  assert_success
  run "$D3" commit -m "add t3" --context "$CTX" "$REPO"
  assert_success
}

@test "d3 log shows all 3 commits" {
  run "$D3" log "$REPO" --context "$CTX"
  assert_success
  assert_output --partial "add t1"
  assert_output --partial "add t2"
  assert_output --partial "add t3"
}

# ---------------------------------------------------------------
# Push to the dev datadatdat remote, then clone back into a fresh repo
# ---------------------------------------------------------------

@test "remote add: datadatdat dev" {
  run "$D3" remote add "${REMOTE_URL}/${TEST_ORG}/k8stest-repo" "$REPO" --context "$CTX"
  assert_success
}

@test "push: all commits go to the dev datadatdat remote" {
  run "$D3" push "$REPO" --context "$CTX"
  assert_success
}

@test "clone (datadatdat): pull the pushed repo back, pod comes up, t3 exists" {
  run "$D3" clone -n hello-clone-datadatdat --context "$CTX" "${REMOTE_URL}/${TEST_ORG}/k8stest-repo"
  assert_success
  wait_pod_ready pod/hello-clone-datadatdat-0
  run kubectl exec hello-clone-datadatdat-0 -- psql -U postgres -c "select count(*) from t3"
  assert_success
  assert_output --partial "1"
}

# ---------------------------------------------------------------
# Public hello-world remotes
# ---------------------------------------------------------------

@test "clone (s3web): hello-world/postgres from public website endpoint" {
  run "$D3" clone -n hello-clone-s3web --context "$CTX" "$S3WEB_URL"
  assert_success
  # 60 iters × 5s = 300s. Pulling postgres + cloning the snapshot back
  # takes longer than a fresh `d3 run` because the dataset has actual
  # data in it.
  wait_pod_ready pod/hello-clone-s3web-0 60
}

@test "clone (s3): hello-world/postgres from S3 bucket" {
  if [ -z "$AWS_ACCESS_KEY_ID" ]; then
    skip "AWS_ACCESS_KEY_ID not set; skipping authenticated S3 clone"
  fi
  run "$D3" clone -n hello-clone-s3 --context "$CTX" "$S3_URL"
  assert_success
  wait_pod_ready pod/hello-clone-s3-0 60
}
