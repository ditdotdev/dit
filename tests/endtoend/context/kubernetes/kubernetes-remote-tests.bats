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

  "$D3" context install -n "$CTX" -t kubernetes

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
  # container. The workflow's later "Show compose and k8s logs" step
  # globs over `docker ps -a` which excludes removed containers — so by
  # then the k8stest server is gone and `docker logs` returns "no such
  # container". Failures in `d3 push` / `d3 commit` from the tests
  # below return a 500 from inside this server (api-gateway never sees
  # the request), so without these logs there's no way to diagnose Bug
  # 2 from CI output. Always-on; ~200 lines of postgres init noise on
  # green runs is an acceptable tradeoff for the diagnostic on red.
  echo "=== docker logs datadatdat-${CTX}-server (pre-teardown) ==="
  docker logs "datadatdat-${CTX}-server" 2>&1 | tail -300 || true
  echo "=== end of datadatdat-${CTX}-server logs ==="
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
