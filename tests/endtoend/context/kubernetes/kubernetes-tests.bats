#!/usr/bin/env bats

# E2E Kubernetes context tests
# These run as part of `make e2e` and exercise the full d3 lifecycle on a
# kubernetes context WITHOUT requiring the datadatdat dev server to be up.
# Auto-skip when no kubernetes cluster is reachable (so `make e2e` on a
# host without minikube no-ops these instead of failing).
#
# Regression coverage on this branch (PR #109):
#   494966a  #108: kubeconfig is mounted as a flattened single file
#   3127780  --context flag actually routes to the named context
#   8999994  StatefulSet ports / 698c43c PVC pulled from Volume.Config
#   a76775e  d3 status reports k8s state, not docker fallback "detached"
#   b7d040a  port-forward survives d3 exit and is killed by d3 rm
#
# Pre-existing host requirements (matches docker-tests.bats conventions):
#   - kubectl in PATH and configured (e.g. minikube context)
#   - default StorageClass (minikube provisions "standard" automatically)
#
# Test ordering: setup_file installs the context; tests run in file order
# and share the StatefulSet between them; teardown_file uninstalls.

load '../../test_helper'

CTX="k8stest"
REPO="demo-db"

setup_file() {
  if ! kubectl cluster-info >/dev/null 2>&1; then
    export D3_K8S_SKIP=1
    return 0
  fi

  # Pre-pull the test image so `d3 run` doesn't time out on a slow registry
  docker pull postgres:latest >/dev/null 2>&1 || true

  # Best-effort cleanup of any prior run's state
  "$D3" rm -f "$REPO" --context "$CTX" 2>/dev/null || true
  "$D3" context uninstall -f "$CTX" 2>/dev/null || true
  kubectl delete statefulset,svc -l "datadatdatRepository=$REPO" --ignore-not-found >/dev/null 2>&1 || true
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
  "$D3" rm -f "$REPO" --context "$CTX" 2>/dev/null || true
  "$D3" context uninstall -f "$CTX" 2>/dev/null || true
}

setup() {
  if [ -n "$D3_K8S_SKIP" ]; then
    skip "no reachable kubernetes cluster (kubectl cluster-info failed)"
  fi
}

# ---------------------------------------------------------------
# Server container: regression guards for issue #108
# ---------------------------------------------------------------

@test "k8s context install: server container is running" {
  run docker ps --filter "name=^datadatdat-${CTX}-server\$" --format '{{.Names}}'
  assert_success
  assert_output "datadatdat-${CTX}-server"
}

@test "k8s context install: kubeconfig is bind-mounted as a single flat file (#108)" {
  run docker inspect "datadatdat-${CTX}-server" --format '{{range .Mounts}}{{.Destination}}={{.Source}};{{end}}'
  assert_success
  # The fix mounts ~/.datadatdat/kubeconfig-<ctx> -> /root/.kube/config (file),
  # NOT the whole ~/.kube/ directory. If the regression returned, we'd see
  # /root/.kube=<dir> in the output instead.
  assert_output --partial "/root/.kube/config="
  refute_output --partial "/root/.kube=/"
}

@test "k8s context install: server starts cleanly (no NoSuchFileException, no Kotlin exception)" {
  # Direct regression guard for #108 (NoSuchFileException on the kubeconfig
  # file refs) and the larger "server crashed silently during install"
  # class. We do NOT assert a positive "Running with context kubernetes-csi"
  # marker here because the server's stdout is async vs the Ktor /v1
  # endpoint becoming reachable — that line may not have been flushed yet
  # when this test runs, even though the readiness probe already returned 200.
  run docker logs "datadatdat-${CTX}-server"
  assert_success
  refute_output --partial "NoSuchFileException"
  refute_output --partial "Exception in thread"
}

# ---------------------------------------------------------------
# d3 run: StatefulSet creation regressions
# ---------------------------------------------------------------

@test "d3 run --context: creates the repository and reports forwarding" {
  run "$D3" run postgres:latest -n "$REPO" -e POSTGRES_HOST_AUTH_METHOD=trust --context "$CTX"
  assert_success
  assert_output --partial "Creating repository $REPO"
  # Regression guard: repo name comes from -n, not from the context name
  refute_output --partial "Creating repository $CTX"
  assert_output --partial "Forwarding local ports"
}

@test "k8s pod ${REPO}-0 reaches Ready" {
  run kubectl wait --for=condition=ready "pod/${REPO}-0" --timeout=180s
  assert_success
}

@test "k8s resources: StatefulSet, Service, Pod present with the right labels" {
  run kubectl get statefulset,svc,pod -l "datadatdatRepository=$REPO" -o name
  assert_success
  assert_output --partial "statefulset.apps/$REPO"
  assert_output --partial "service/$REPO"
  assert_output --partial "pod/${REPO}-0"
}

@test "k8s resources: Service is headless (ClusterIP None)" {
  run kubectl get svc "$REPO" -o jsonpath='{.spec.clusterIP}'
  assert_success
  assert_output "None"
}

@test "k8s resources: Service exposes 5432" {
  run kubectl get svc "$REPO" -o jsonpath='{.spec.ports[0].port}'
  assert_success
  assert_output "5432"
}

@test "k8s resources: at least one PVC bound" {
  # PVCs created by the d3 server have GUID-style names and aren't labeled.
  # Just verify *some* PVC is Bound in the namespace where d3 runs.
  run bash -c "kubectl get pvc -o jsonpath='{range .items[?(@.status.phase==\"Bound\")]}{.metadata.name}{\"\\n\"}{end}'"
  assert_success
  [ -n "$output" ] || {
    echo "expected at least one Bound PVC in default namespace"
    kubectl get pvc
    return 1
  }
}

# ---------------------------------------------------------------
# d3 status / d3 ls: regression guard for a76775e
# ---------------------------------------------------------------

@test "d3 status --context returns 'running' (regression for a76775e)" {
  # Pre-fix this returned "detached" because common.Status used docker.inspect
  # which can't see kubernetes pods.
  run "$D3" status "$REPO" --context "$CTX"
  assert_success
  assert_output --partial "running"
  refute_output --partial "detached"
}

@test "d3 ls --context shows the repo running" {
  run "$D3" ls --context "$CTX"
  assert_success
  assert_output --partial "$REPO"
  assert_output --partial "running"
}

# ---------------------------------------------------------------
# Port-forward: regression guards for ed09a47 + b7d040a
# ---------------------------------------------------------------

@test "port-forward: localhost:5432 is reachable (regression for b7d040a)" {
  # Try for up to 10s; on slow CI the port-forward can take a beat to come up.
  for _ in $(seq 1 10); do
    if (echo > /dev/tcp/127.0.0.1/5432) 2>/dev/null; then
      return 0
    fi
    sleep 1
  done
  echo "port 5432 never became reachable on 127.0.0.1 after d3 run; port-forward leaked or died"
  netstat -an 2>/dev/null | head -20 || ss -tln 2>/dev/null | head -20
  return 1
}

@test "port-forward: pid file is recorded under ~/.datadatdat" {
  run bash -c "ls $HOME/.datadatdat/portforward-${REPO}-*.pid 2>/dev/null"
  assert_success
  assert_output --partial "portforward-${REPO}-"
}

# ---------------------------------------------------------------
# Postgres connectivity end-to-end
# ---------------------------------------------------------------

@test "postgres responds via kubectl exec" {
  run kubectl exec "${REPO}-0" -- psql -U postgres -c "select version();"
  assert_success
  assert_output --partial "PostgreSQL"
}

@test "postgres responds via the forwarded localhost port" {
  if ! command -v psql >/dev/null 2>&1; then
    skip "psql client not installed; skipping localhost port-forward test"
  fi
  run psql -h localhost -U postgres -c "select 1 as ok;"
  assert_success
  assert_output --partial "1"
}

# ---------------------------------------------------------------
# stop / start lifecycle (StatefulSet replica scaling)
# ---------------------------------------------------------------

@test "d3 stop scales the StatefulSet down" {
  run "$D3" stop "$REPO" --context "$CTX"
  assert_success
  # StatefulSet replicas drops to 0; the pod transitions to Completed and
  # may linger in that state — `kubectl get pod` still returns a row. Check
  # the replica count instead of pod absence.
  for _ in $(seq 1 30); do
    replicas=$(kubectl get statefulset "$REPO" -o jsonpath='{.spec.replicas}' 2>/dev/null)
    if [ "$replicas" = "0" ]; then
      return 0
    fi
    sleep 1
  done
  echo "StatefulSet ${REPO} did not scale to 0 replicas within 30s of d3 stop"
  kubectl get statefulset "$REPO"
  return 1
}

@test "d3 start scales it back up and pod becomes Ready" {
  run "$D3" start "$REPO" --context "$CTX"
  assert_success
  run kubectl wait --for=condition=ready "pod/${REPO}-0" --timeout=180s
  assert_success
}

# ---------------------------------------------------------------
# d3 commit / d3 checkout: snapshot-backed time travel
#
# Each commit becomes a CSI VolumeSnapshot. These tests exercise the
# whole arc — write data, commit, destructive change, checkout, verify
# data is back. Requires the volumesnapshots + csi-hostpath-driver
# minikube addons; tests skip with a clear message when the
# VolumeSnapshot CRD isn't installed (so `make e2e` on a developer's
# laptop without those addons no-ops these instead of failing).
#
# State is shared between the commit test and the checkout test via a
# tempfile because BATS @test bodies run in subshells — env vars don't
# persist across @tests, but writes to /tmp do.
# ---------------------------------------------------------------

COMMIT_STATE="/tmp/d3-k8s-bats-commit-${REPO}"

@test "d3 commit: produces a VolumeSnapshot that becomes ReadyToUse" {
  if ! kubectl get crd volumesnapshots.snapshot.storage.k8s.io >/dev/null 2>&1; then
    skip "VolumeSnapshot CRD not installed (enable minikube addons: volumesnapshots, csi-hostpath-driver)"
  fi

  # Plant a known table so the checkout test has something to verify.
  run kubectl exec "${REPO}-0" -- psql -U postgres -c \
    "CREATE TABLE bats_baseline (id INT PRIMARY KEY); INSERT INTO bats_baseline VALUES (42);"
  assert_success

  run "$D3" commit "$REPO" -m "bats baseline" --context "$CTX"
  assert_success
  assert_output --partial "Commit "

  COMMIT_ID=$(echo "$output" | awk '/^Commit / {print $2; exit}')
  [ -n "$COMMIT_ID" ] || {
    echo "could not parse commit id from d3 commit output"
    return 1
  }
  echo "$COMMIT_ID" > "$COMMIT_STATE"

  # The server names snapshots <volumeSet>-<volume>-<commitId> and labels
  # them with datadatdatCommit; pick by label so we don't have to know
  # the volumeSet UUID.
  for _ in $(seq 1 60); do
    snap=$(kubectl get volumesnapshot -l "datadatdatCommit=$COMMIT_ID" \
      -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$snap" ]; then break; fi
    sleep 1
  done
  [ -n "$snap" ] || {
    echo "no VolumeSnapshot with datadatdatCommit=$COMMIT_ID after 60s"
    kubectl get volumesnapshot
    return 1
  }

  run kubectl wait --for=jsonpath='{.status.readyToUse}'=true \
    "volumesnapshot/$snap" --timeout=120s
  assert_success
}

@test "d3 checkout: restores prior database state from snapshot" {
  if ! kubectl get crd volumesnapshots.snapshot.storage.k8s.io >/dev/null 2>&1; then
    skip "VolumeSnapshot CRD not installed (enable minikube addons: volumesnapshots, csi-hostpath-driver)"
  fi
  [ -f "$COMMIT_STATE" ] || skip "no commit captured by previous test"
  COMMIT_ID=$(cat "$COMMIT_STATE")

  # Destructive change so checkout has something to undo.
  run kubectl exec "${REPO}-0" -- psql -U postgres -c "DROP TABLE bats_baseline;"
  assert_success
  # Confirm the table really is gone before the checkout.
  run kubectl exec "${REPO}-0" -- psql -U postgres -c "SELECT 1 FROM bats_baseline LIMIT 1;"
  assert_failure

  run "$D3" checkout "$REPO" -c "$COMMIT_ID" --context "$CTX"
  assert_success

  # checkout tears the pod down and recreates it from the snapshot's
  # PVC clone — wait for the new pod to come back ready.
  run kubectl wait --for=condition=ready "pod/${REPO}-0" --timeout=180s
  assert_success

  run kubectl exec "${REPO}-0" -- psql -U postgres -c \
    "SELECT id FROM bats_baseline;"
  assert_success
  assert_output --partial "42"

  rm -f "$COMMIT_STATE"
}

# ---------------------------------------------------------------
# d3 rm: cleans up cluster resources AND the port-forward
# ---------------------------------------------------------------

@test "d3 rm -f: removes StatefulSet and Service" {
  run "$D3" rm -f "$REPO" --context "$CTX"
  assert_success
  for _ in $(seq 1 30); do
    out=$(kubectl get statefulset,svc -l "datadatdatRepository=$REPO" -o name 2>/dev/null || true)
    if [ -z "$out" ]; then
      return 0
    fi
    sleep 1
  done
  echo "k8s resources for $REPO not cleaned up 30s after d3 rm"
  kubectl get statefulset,svc -l "datadatdatRepository=$REPO"
  return 1
}

@test "d3 rm -f: releases localhost:5432 (regression for b7d040a)" {
  for _ in $(seq 1 10); do
    if ! (echo > /dev/tcp/127.0.0.1/5432) 2>/dev/null; then
      return 0
    fi
    sleep 1
  done
  echo "port 5432 still bound 10s after d3 rm; port-forward orphaned"
  return 1
}

@test "d3 rm -f: removes the port-forward pid file" {
  run bash -c "ls $HOME/.datadatdat/portforward-${REPO}-*.pid 2>/dev/null || true"
  refute_output --partial "portforward-${REPO}-"
}

# ---------------------------------------------------------------
# Final teardown: explicit `d3 context uninstall` before the file's
# teardown_file runs (covers the "uninstall reports nothing if double
# called" path implicitly).
# ---------------------------------------------------------------

@test "d3 context uninstall: succeeds and removes the context" {
  run "$D3" context uninstall -f "$CTX"
  assert_success
  run "$D3" context ls
  refute_output --partial "$CTX"
}
