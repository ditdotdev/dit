# Running a Container in Kubernetes with dit

A minimal walkthrough that uses `dit` to run a Postgres container as a Kubernetes
StatefulSet on a local minikube cluster, then verifies the workload end-to-end.

## What this demo does

- Stands up a single-node Kubernetes cluster on your laptop using **minikube**
  with the Hyper-V driver (genuine VM, upstream kubeadm-installed components).
- Installs a Kubernetes-backed `dit` context. The `dit` server still runs as a
  local Docker container on your host; it talks to the cluster via the standard
  `~/.kube/config`.
- Runs `postgres:latest` via `dit run`. Under the hood `dit` creates a
  PersistentVolumeClaim for the Postgres data directory, a headless Service for
  the exposed ports, and a one-replica StatefulSet. Ports are forwarded back to
  `localhost` via a background `kubectl port-forward`.
- Connects to the forwarded Postgres port to prove the pod is actually serving.
- Tears everything down.

## Prerequisites

- Windows 10/11 Pro, Enterprise, or Education (Hyper-V requires one of these).
- Docker Desktop installed and running (needed by `dit` itself).
- `dit` installed and on `PATH` — see the main [README](README.md).
- Administrator PowerShell for the one-time Hyper-V enable + vSwitch creation.

## Step 1: Install minikube and kubectl

One-time setup. Run in an **admin PowerShell**:

```powershell
# Enable Hyper-V (reboot after this command completes)
Enable-WindowsOptionalFeature -Online -FeatureName Microsoft-Hyper-V -All

# After reboot, install the tooling
choco install minikube kubernetes-cli -y
```

## Step 2: Create a cluster

Minikube's Hyper-V driver needs an **external** virtual switch (the default
internal switch has no internet access, so images can't pull). Create it once:

```powershell
New-VMSwitch -Name "minikube-ext" -NetAdapterName "Ethernet" -AllowManagementOS $true
```

> **Heads-up:** creating the external switch briefly disconnects the NIC it
> binds to. If you're on a VPN, disconnect first.

Then start the cluster:

```powershell
minikube start --driver=hyperv --hyperv-virtual-switch="minikube-ext" --cpus=2 --memory=3072

kubectl config current-context   # should print: minikube
kubectl get nodes                # one Ready node named "minikube"
kubectl get storageclass         # "standard (default)" provisioned by k8s.io/minikube-hostpath
```

> **Note on `minikube status`:** with the `hyperv` driver, `minikube status`
> calls the Hyper-V PowerShell cmdlets which require **admin PowerShell**.
> If you're in a non-admin shell, skip it — `kubectl get nodes` is the
> equivalent health check and works without elevation.

The `standard` StorageClass is what `dit` will use for the Postgres PVC. No
additional CSI setup is required for this demo, because it never calls
`dit commit` — `standard` cannot take VolumeSnapshots. If you want to extend
the demo with commits, see
[Dit with Kubernetes](docs/src/lifecycle/kubernetes.md) for the CSI addon and
storage/snapshot class setup.

### Fallback: the Docker driver

If the Hyper-V vSwitch is painful (VPN conflicts, Wi-Fi that won't re-bind
cleanly, corporate NIC policy), skip Hyper-V and run minikube as a container
inside Docker Desktop instead — less production-like, but zero host-networking
changes:

```powershell
minikube delete                            # only if you already created a hyperv cluster
minikube start --driver=docker --cpus=2 --memory=3072
```

Everything downstream is identical.

## Step 3: Install a Kubernetes context in dit

**Make sure Docker Desktop is running before this step.** The `dit` server
itself runs as a local Docker container on your host (separate from the
minikube VM), so `dit` will shell out to `docker pull` here. If Docker Desktop
isn't up, you'll see an opaque `Error pulling image ditdotdev/dit:vX.Y.Z: exit status 1`.

```bash
dit context install -n k8s-demo -t kubernetes
```

This boots the `dit` server as a local Docker container named after the
context (`dit-k8s-demo-server`), listening on a randomly chosen `localhost`
port that `dit` records in `~/.dit/config`. First run pulls the
`ditdotdev/dit` image and may take a few minutes.

## Step 4: Run Postgres in the cluster

```bash
dit run postgres:latest -n demo-db \
    -e POSTGRES_HOST_AUTH_METHOD=trust \
    --context k8s-demo
```

`dit`'s own view of the repository:

```bash
dit status demo-db --context k8s-demo     # expect: running
dit ls --context k8s-demo                 # expect demo-db listed
```

Kubernetes' view — proof that `dit` created real cluster resources:

```bash
# StatefulSet, Service, and Pod all carry the ditRepository label
kubectl get statefulset,svc,pod -l ditRepository=demo-db

# PVCs are server-generated with GUID-style names and no matching label;
# list all PVCs to see the one backing demo-db
kubectl get pvc
```

Expected output:

- `statefulset.apps/demo-db` — `READY 1/1`
- `service/demo-db` — headless (`ClusterIP: None`) with port 5432 mapped
- `pod/demo-db-0` — `Running`
- a `persistentvolumeclaim/<guid>-v0` — `Bound`, 1Gi, `standard` StorageClass
  (the GUID is generated by the dit server; the volume backs `/var/lib/postgresql`)

Prove the pod is actually serving — `dit` forwards Postgres's port to
`localhost:5432` automatically:

```bash
# If you have psql installed
psql -h localhost -U postgres -c "select version();"

# Otherwise, exec into the pod
kubectl exec -it demo-db-0 -- psql -U postgres -c "select version();"
```

You should see a Postgres version string. That's the workload running inside
your Kubernetes cluster, exposed back to your laptop, fronted by `dit`.

## Step 5: Cleanup

```bash
# Remove the repository (deletes StatefulSet, Service, and PVC)
dit rm -f demo-db --context k8s-demo

# Remove the dit server container
dit uninstall -f --context k8s-demo

# Stop (or delete) the cluster
minikube stop       # preserves the cluster; `minikube start` resumes it
# or
minikube delete     # destroys the Hyper-V VM entirely
```

## Known limitations

These are current constraints of `dit`'s Kubernetes provider — they don't
affect this demo, but know them before extending it:

- **Namespace is hardcoded to `default`.** There's no flag to change it yet.
- **`~/.kube/config` is hardcoded.** `dit` always uses the standard kubeconfig
  location and whatever context is currently selected, and the server container
  keeps a copy of the kubeconfig taken at `dit context install` time. Don't
  switch kube contexts between `dit` commands; to point at a different cluster,
  re-install the dit context.
- **PVC size is fixed at 1 GiB** per image volume. Not configurable.
- **Port forwarding is fragile.** `dit` spawns `kubectl port-forward` in the
  background. If it dies (host sleep, process crash), run
  `dit stop demo-db && dit start demo-db` to re-establish.
- **`dit commit` needs snapshot-capable CSI storage.** This demo deliberately
  sticks with minikube's default `standard` StorageClass, which cannot take
  VolumeSnapshots — so `dit commit` will fail on this context. Commits do work
  on Kubernetes (dit uses the GA `snapshot.storage.k8s.io/v1` API): enable the
  addons with `minikube addons enable volumesnapshots` and
  `minikube addons enable csi-hostpath-driver`, then install the context with
  `-p storageClass=csi-hostpath-sc -p snapshotClass=csi-hostpath-snapclass`.
  See [Dit with Kubernetes](docs/src/lifecycle/kubernetes.md) for details.

## Troubleshooting

- `dit run` hangs on "Waiting for deployment to be ready":
  run `kubectl describe pod demo-db-0` in another shell. Usually an image pull
  failure or an unschedulable pod (insufficient CPU/memory on the node).
- `psql: could not connect`: the background `kubectl port-forward` may have
  died. `dit stop demo-db && dit start demo-db` restarts it.
- `kubectl` talks to the wrong cluster: check
  `kubectl config current-context` — it should be `minikube`.
