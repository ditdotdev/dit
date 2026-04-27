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

  # Wait for /v1/ to respond, then warm up the first POST route. See
  # datadatdat-server#139 — the very first POST after server startup
  # returns EOF on the client. Removing this block once that's fixed.
  local server_port
  server_port=$(awk -v ctx="$CTX:" '$0 ~ ctx{f=1} f && /port:/{print $2; exit}' "$HOME/.datadatdat/config")
  for _ in $(seq 1 60); do
    if curl -s -o /dev/null -w "%{http_code}" "http://localhost:${server_port}/v1/repositories" 2>/dev/null | grep -q 200; then
      break
    fi
    sleep 2
  done
  curl -s -o /dev/null -X POST -H "Content-Type: application/json" \
    -d '{"name":"warmup-noop","properties":{}}' \
    "http://localhost:${server_port}/v1/repositories" || true
  curl -s -o /dev/null -X DELETE \
    "http://localhost:${server_port}/v1/repositories/warmup-noop" || true
}

teardown_file() {
  if [ -n "$D3_K8S_SKIP" ]; then return 0; fi
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
  run kubectl wait --for=condition=ready "pod/${REPO}-0" --timeout=180s
  assert_success
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
  run kubectl wait --for=condition=ready pod/hello-clone-datadatdat-0 --timeout=180s
  assert_success
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
  run kubectl wait --for=condition=ready pod/hello-clone-s3web-0 --timeout=300s
  assert_success
}

@test "clone (s3): hello-world/postgres from S3 bucket" {
  if [ -z "$AWS_ACCESS_KEY_ID" ]; then
    skip "AWS_ACCESS_KEY_ID not set; skipping authenticated S3 clone"
  fi
  run "$D3" clone -n hello-clone-s3 --context "$CTX" "$S3_URL"
  assert_success
  run kubectl wait --for=condition=ready pod/hello-clone-s3-0 --timeout=300s
  assert_success
}
