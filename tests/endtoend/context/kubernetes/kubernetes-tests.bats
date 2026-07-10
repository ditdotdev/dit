#!/usr/bin/env bats
# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# E2E Kubernetes context tests
# These run as part of `make e2e` and exercise the full dit lifecycle on a
# kubernetes context WITHOUT requiring the dit dev server to be up.
# Auto-skip when no kubernetes cluster is reachable (so `make e2e` on a
# host without minikube no-ops these instead of failing).
#
# Regression coverage on this branch (PR #109):
#   494966a  #108: kubeconfig is mounted as a flattened single file
#   3127780  --context flag actually routes to the named context
#   8999994  StatefulSet ports / 698c43c PVC pulled from Volume.Config
#   a76775e  dit status reports k8s state, not docker fallback "detached"
#   b7d040a  port-forward survives dit exit and is killed by dit rm
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

# ---------------------------------------------------------------
# Polling helpers
#
# All waits in this file follow the same pattern: explicitly check
# the condition we want, in a bounded loop. The only arbitrary value
# is the iteration count; sleep is fixed at 5s for cluster-state
# polls, 1s for local TCP probes (see test 12). Avoids the
# `kubectl wait --timeout=Ns` pattern where the wall-clock timeout
# is decoupled from the condition being checked.
# ---------------------------------------------------------------

# Wait for a Pod's Ready condition. Default 36 iterations × 5s = 180s.
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

# Wait for a VolumeSnapshot's status.readyToUse to flip to true.
# Default 60 iterations × 5s = 300s. csi-hostpath on minikube-on-GHA
# can take minutes; on a real CSI it's sub-second.
wait_snapshot_ready() {
  local snap="$1"
  local iters="${2:-60}"
  for _ in $(seq 1 "$iters"); do
    if [ "$(kubectl get "volumesnapshot/$snap" -o jsonpath='{.status.readyToUse}' 2>/dev/null)" = "true" ]; then
      return 0
    fi
    sleep 5
  done
  return 1
}

# Wait for postgres in the given pod to actually accept psql
# connections. Pod-Ready (per wait_pod_ready) only means the postgres
# process started; this fills the postgres-startup-to-listening gap
# that surfaces as `psql: ... no such file or directory` on the
# socket or `Connection refused` on the TCP port. Used after every
# pod recreation that's followed immediately by a psql call.
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

  # Pre-pull the test image so `dit run` doesn't time out on a slow registry
  docker pull postgres:latest >/dev/null 2>&1 || true

  # Best-effort cleanup of any prior run's state
  "$D3" rm -f "$REPO" --context "$CTX" 2>/dev/null || true
  "$D3" context uninstall -f "$CTX" 2>/dev/null || true
  kubectl delete statefulset,svc -l "ditRepository=$REPO" --ignore-not-found >/dev/null 2>&1 || true
  rm -f "$HOME/.dit/portforward-${REPO}-"*.pid 2>/dev/null || true

  # Pin storage + snapshot classes for dit-provisioned PVCs. Relying on the
  # cluster default is fragile: minikube's `default-storageclass` addon
  # re-asserts `standard` (k8s.io/minikube-hostpath, no snapshot support) as
  # the default on every `minikube start`, so any new PVC lands on a class
  # whose driver can't fulfill VolumeSnapshots and `dit commit` snapshots
  # fail with "snapshotting non-CSI volumes is not supported". csi-hostpath-sc
  # + csi-hostpath-snapclass come from minikube's csi-hostpath-driver addon;
  # override via env if your cluster names them differently.
  local sc="${D3_K8S_STORAGE_CLASS:-csi-hostpath-sc}"
  local snapclass="${D3_K8S_SNAPSHOT_CLASS:-csi-hostpath-snapclass}"
  "$D3" context install -n "$CTX" -t kubernetes \
    -p "storageClass=${sc}" \
    -p "snapshotClass=${snapclass}"

  # Wait for the embedded Ktor app to start serving /v1/.
  local server_port
  server_port=$(awk -v ctx="$CTX:" '$0 ~ ctx{f=1} f && /port:/{print $2; exit}' "$HOME/.dit/config")
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
  run docker ps --filter "name=^dit-${CTX}-server\$" --format '{{.Names}}'
  assert_success
  assert_output "dit-${CTX}-server"
}

@test "k8s context install: kubeconfig is bind-mounted as a single flat file (#108)" {
  run docker inspect "dit-${CTX}-server" --format '{{range .Mounts}}{{.Destination}}={{.Source}};{{end}}'
  assert_success
  # The fix mounts ~/.dit/kubeconfig-<ctx> -> /root/.kube/config (file),
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
  run docker logs "dit-${CTX}-server"
  assert_success
  refute_output --partial "NoSuchFileException"
  refute_output --partial "Exception in thread"
}

# ---------------------------------------------------------------
# dit run: StatefulSet creation regressions
# ---------------------------------------------------------------

@test "dit run --context: creates the repository and reports forwarding" {
  run "$D3" run postgres:latest -n "$REPO" -e POSTGRES_HOST_AUTH_METHOD=trust --context "$CTX"
  assert_success
  assert_output --partial "Creating repository $REPO"
  # Regression guard: repo name comes from -n, not from the context name
  refute_output --partial "Creating repository $CTX"
  assert_output --partial "Forwarding local ports"
}

@test "k8s pod ${REPO}-0 reaches Ready" {
  wait_pod_ready "pod/${REPO}-0"
}

@test "k8s resources: StatefulSet, Service, Pod present with the right labels" {
  run kubectl get statefulset,svc,pod -l "ditRepository=$REPO" -o name
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
  # PVCs created by the dit server have GUID-style names and aren't labeled.
  # Just verify *some* PVC is Bound in the namespace where dit runs.
  run bash -c "kubectl get pvc -o jsonpath='{range .items[?(@.status.phase==\"Bound\")]}{.metadata.name}{\"\\n\"}{end}'"
  assert_success
  [ -n "$output" ] || {
    echo "expected at least one Bound PVC in default namespace"
    kubectl get pvc
    return 1
  }
}

# ---------------------------------------------------------------
# dit status / dit ls: regression guard for a76775e
# ---------------------------------------------------------------

@test "dit status --context returns 'running' (regression for a76775e)" {
  # Pre-fix this returned "detached" because common.Status used docker.inspect
  # which can't see kubernetes pods.
  run "$D3" status "$REPO" --context "$CTX"
  assert_success
  assert_output --partial "running"
  refute_output --partial "detached"
}

@test "dit ls --context shows the repo running" {
  run "$D3" ls --context "$CTX"
  assert_success
  assert_output --partial "$REPO"
  assert_output --partial "running"
}

# ---------------------------------------------------------------
# Port-forward: regression guards for ed09a47 + b7d040a
# ---------------------------------------------------------------

@test "port-forward: localhost:5432 is reachable (regression for b7d040a)" {
  # Try for up to 30s. dit spawns `kubectl port-forward` in the background;
  # on a busy CI runner it can race with the StatefulSet pod actually
  # opening its TCP listener. Observed test 13 (pid file recorded) pass
  # while this one fails on PR #113 run 25070315289 — the pid was
  # written but the connection wasn't established yet. 10s was too tight.
  for _ in $(seq 1 30); do
    if (echo > /dev/tcp/127.0.0.1/5432) 2>/dev/null; then
      return 0
    fi
    sleep 1
  done
  echo "port 5432 never became reachable on 127.0.0.1 after dit run; port-forward leaked or died"
  netstat -an 2>/dev/null | head -20 || ss -tln 2>/dev/null | head -20
  return 1
}

@test "port-forward: pid file is recorded under ~/.dit" {
  run bash -c "ls $HOME/.dit/portforward-${REPO}-*.pid 2>/dev/null"
  assert_success
  assert_output --partial "portforward-${REPO}-"
}

# ---------------------------------------------------------------
# Postgres connectivity end-to-end
#
# `wait_pod_ready` (test 5) returns when k8s says the pod's Ready
# condition is True, but the dit StatefulSet template has no readiness
# probe configured for postgres — so "Ready" only means "the postgres
# process started," not "postgres is accepting connections." On a fast
# runner this race shows up as `psql: ... no such file or directory` on
# the unix socket (postgres hasn't created it yet) or `connection
# refused` on the TCP port. Wait for postgres to actually accept
# connections before each connectivity assertion. ~2 sec on a normal
# run, up to 30 sec on a slow runner before failing.
# ---------------------------------------------------------------

@test "postgres responds via kubectl exec" {
  wait_postgres_ready "${REPO}-0"
  run kubectl exec "${REPO}-0" -- psql -U postgres -c "select version();"
  assert_success
  assert_output --partial "PostgreSQL"
}

@test "postgres responds via the forwarded localhost port" {
  if ! command -v psql >/dev/null 2>&1; then
    skip "psql client not installed; skipping localhost port-forward test"
  fi
  # Pin to 127.0.0.1, NOT localhost. psql resolves `localhost` to IPv6
  # `::1` first on GHA runners, but `kubectl port-forward` only binds
  # IPv4 by default — so psql gets `Connection refused (::1)` and
  # never falls through to v4. Surfaced on PR #113 run 25072827847
  # where test 12 (`echo > /dev/tcp/127.0.0.1/5432`) passed but this
  # one failed in the same window.
  for _ in $(seq 1 30); do
    if psql -h 127.0.0.1 -U postgres -c "select 1" >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done
  run psql -h 127.0.0.1 -U postgres -c "select 1 as ok;"
  assert_success
  assert_output --partial "1"
}

# ---------------------------------------------------------------
# stop / start lifecycle (StatefulSet replica scaling)
# ---------------------------------------------------------------

@test "dit stop scales the StatefulSet down" {
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
  echo "StatefulSet ${REPO} did not scale to 0 replicas within 30s of dit stop"
  kubectl get statefulset "$REPO"
  return 1
}

@test "dit start scales it back up and pod becomes Ready" {
  run "$D3" start "$REPO" --context "$CTX"
  assert_success
  wait_pod_ready "pod/${REPO}-0"
}

# ---------------------------------------------------------------
# dit commit / dit checkout: snapshot-backed time travel
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

COMMIT_STATE="/tmp/dit-k8s-bats-commit-${REPO}"

@test "dit commit: produces a VolumeSnapshot that becomes ReadyToUse" {
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
    echo "could not parse commit id from dit commit output"
    return 1
  }
  echo "$COMMIT_ID" > "$COMMIT_STATE"

  # The server names snapshots <volumeSet>-<volume>-<commitId> and labels
  # them with ditCommit; pick by label so we don't have to know
  # the volumeSet UUID.
  for _ in $(seq 1 60); do
    snap=$(kubectl get volumesnapshot -l "ditCommit=$COMMIT_ID" \
      -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$snap" ]; then break; fi
    sleep 1
  done
  [ -n "$snap" ] || {
    echo "no VolumeSnapshot with ditCommit=$COMMIT_ID after 60s"
    kubectl get volumesnapshot
    return 1
  }

  # csi-hostpath on minikube-on-GHA-runners can take several minutes
  # to flush a snapshot to ReadyToUse; on a real CSI backend (GKE pd,
  # EBS, etc.) it's sub-second. Default helper iters = 60 × 5s = 300s.
  if ! wait_snapshot_ready "$snap"; then
    echo "VolumeSnapshot $snap did not become ReadyToUse"
    echo "--- kubectl describe volumesnapshot/$snap ---"
    kubectl describe "volumesnapshot/$snap" || true
    echo "--- kubectl describe volumesnapshotcontent (linked from snap) ---"
    vsc=$(kubectl get "volumesnapshot/$snap" -o jsonpath='{.status.boundVolumeSnapshotContentName}' 2>/dev/null || true)
    [ -n "$vsc" ] && kubectl describe "volumesnapshotcontent/$vsc" || echo "no boundVolumeSnapshotContentName"
    echo "--- csi-hostpathplugin logs (tail 100, all containers) ---"
    kubectl -n kube-system logs ds/csi-hostpathplugin --all-containers --tail=100 2>&1 || true
    echo "--- snapshot-controller logs (tail 100) ---"
    kubectl -n kube-system logs -l app=snapshot-controller --tail=100 2>&1 || \
      kubectl -n kube-system logs -l app.kubernetes.io/name=snapshot-controller --tail=100 2>&1 || \
      echo "no snapshot-controller pod found"
    return 1
  fi
}

@test "dit checkout: restores prior database state from snapshot" {
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

  # Capture the pod's UID so we can assert checkout actually recreated
  # it. dit checkout has to swap the StatefulSet's PVC reference AND
  # force pod recreation — patching volumes alone doesn't roll a
  # StatefulSet. If the pod stays the same (same UID), the new PVC is
  # never mounted and postgres serves the stale state.
  old_uid=$(kubectl get "pod/${REPO}-0" -o jsonpath='{.metadata.uid}' 2>/dev/null || true)
  [ -n "$old_uid" ] || { echo "could not capture pre-checkout pod UID"; return 1; }

  run "$D3" checkout "$REPO" -c "$COMMIT_ID" --context "$CTX"
  assert_success
  # Print captured output so we can see what dit checkout actually did
  # (assert_success swallows stdout on success otherwise).
  echo "--- dit checkout output ---"
  echo "$output"
  echo "--- end dit checkout output ---"

  # Wait for the StatefulSet controller to recreate the pod. Same name,
  # different UID. If UID never changes, dit checkout didn't trigger
  # pod recreation and the new PVC is unused — fail loudly with the
  # StatefulSet description so we can see what state dit left it in.
  for _ in $(seq 1 36); do
    new_uid=$(kubectl get "pod/${REPO}-0" -o jsonpath='{.metadata.uid}' 2>/dev/null || true)
    if [ -n "$new_uid" ] && [ "$new_uid" != "$old_uid" ]; then
      break
    fi
    sleep 5
  done
  if [ -z "$new_uid" ] || [ "$new_uid" = "$old_uid" ]; then
    echo "pod UID did not change after dit checkout"
    echo "  old: $old_uid"
    echo "  new: ${new_uid:-(no pod)}"
    echo "--- kubectl describe statefulset/${REPO} ---"
    kubectl describe "statefulset/${REPO}" || true
    echo "--- kubectl get pvc ---"
    kubectl get pvc || true
    return 1
  fi

  # New pod is up — wait for postgres inside it to accept connections.
  wait_pod_ready "pod/${REPO}-0"
  wait_postgres_ready "${REPO}-0"

  run kubectl exec "${REPO}-0" -- psql -U postgres -c \
    "SELECT id FROM bats_baseline;"
  assert_success
  assert_output --partial "42"

  rm -f "$COMMIT_STATE"
}

# ---------------------------------------------------------------
# dit rm: cleans up cluster resources AND the port-forward
# ---------------------------------------------------------------

@test "dit rm -f: removes StatefulSet and Service" {
  run "$D3" rm -f "$REPO" --context "$CTX"
  assert_success
  for _ in $(seq 1 30); do
    out=$(kubectl get statefulset,svc -l "ditRepository=$REPO" -o name 2>/dev/null || true)
    if [ -z "$out" ]; then
      return 0
    fi
    sleep 1
  done
  echo "k8s resources for $REPO not cleaned up 30s after dit rm"
  kubectl get statefulset,svc -l "ditRepository=$REPO"
  return 1
}

@test "dit rm -f: releases localhost:5432 (regression for b7d040a)" {
  for _ in $(seq 1 10); do
    if ! (echo > /dev/tcp/127.0.0.1/5432) 2>/dev/null; then
      return 0
    fi
    sleep 1
  done
  echo "port 5432 still bound 10s after dit rm; port-forward orphaned"
  return 1
}

@test "dit rm -f: removes the port-forward pid file" {
  run bash -c "ls $HOME/.dit/portforward-${REPO}-*.pid 2>/dev/null || true"
  refute_output --partial "portforward-${REPO}-"
}

# ---------------------------------------------------------------
# Final teardown: explicit `dit context uninstall` before the file's
# teardown_file runs (covers the "uninstall reports nothing if double
# called" path implicitly).
# ---------------------------------------------------------------

@test "dit context uninstall: succeeds and removes the context" {
  run "$D3" context uninstall -f "$CTX"
  assert_success
  run "$D3" context ls
  refute_output --partial "$CTX"
}
