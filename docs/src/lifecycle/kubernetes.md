---
title: Dit with Kubernetes
nav_label: Kubernetes
nav_order: 150
---

Dit provides a way to run repositories in different container environments,
known as "contexts" (see [Contexts](context.md) for more information). A
Kubernetes context represents a set of repositories running in a cluster,
accessed via the Kubernetes API. This cluster could be local to the machine,
hosted centrally, or delivered as a cloud service. Through Dit, not only can
these repositories be run in a simple fashion with powerful data controls, data
can be shared between them (such as pushing a dataset from a CI/CD Kubernetes
cluster and later cloning for local debugging).

## Kubernetes Requirements

Dit requires a Kubernetes cluster with the following configuration:

* The [VolumeSnapshot](https://kubernetes.io/docs/concepts/storage/volume-snapshots/)
  API (`snapshot.storage.k8s.io/v1`, GA since Kubernetes 1.20) must be
  available — that is, the VolumeSnapshot CRDs and snapshot controller must be
  installed. Most managed clusters (EKS, GKE, AKS) ship these out of the box;
  on minikube they come from the `volumesnapshots` addon.
* There must be a CSI (Container Storage Interface) driver installed that
  supports [volume snapshots](https://kubernetes-csi.github.io/docs/snapshot-restore-feature.html).
* The storage class and snapshot class Dit uses — whether specified at install
  time (see [Installing a Kubernetes Context](#installing-a-kubernetes-context))
  or inherited from the cluster default — must use a CSI driver with snapshot
  capabilities.

Dit uses the default Kubernetes config file (`~/.kube/config`) with whatever
context is currently selected, and always operates in the `default` namespace.
At install time this config is flattened and copied into the Dit server
container (stored as `~/.dit/kubeconfig-<context>`), so changes made to
`~/.kube/config` after `dit context install` — switching clusters, rotating
credentials — do not reach the server until the context is re-installed.
Future versions will make the cluster and namespace configurable.

The dit server still runs as a container on the local workstation. A local
Docker installation is required, though no special privileges or operating
system support is necessary. This also means that all the metadata is local to the
user, so two users cannot share dit repositories in a shared Kubernetes
cluster. The pods themselves will be accessible to any kubernetes user, but
there is no way to manage them as Dit repositories on a different system.

Each push or pull operation is run as a separate Kubernetes Job, which
requires the `ditdotdev/dit` image to be pullable from within the cluster. On
clusters that cannot reach Docker Hub (air-gapped or private-registry
environments), point Dit at a mirrored copy with
`-p ditImage=<registry>/dit:<tag>` at install time.

## Installing a Kubernetes Context

Install a Kubernetes context with `dit context install -n <name> -t kubernetes`.
If `-n` is omitted, the context is named after its type (`kubernetes`
here). The storage and snapshot classes Dit
uses for its PersistentVolumeClaims (volumes) and VolumeSnapshots (commits) are
set with the `-p storageClass=<name>` and `-p snapshotClass=<name>` parameters.
Both **must** be backed by a CSI driver with snapshot support.

Pin these explicitly rather than relying on the cluster default. Many clusters
default to a non-CSI storage class that cannot snapshot — for example minikube's
`standard` (`k8s.io/minikube-hostpath`) — in which case `dit commit` fails with
`snapshotting non-CSI volumes is not supported` (or, on older paths, silently
captures an empty volume).

The exact class names vary by cluster. List what is available with
`kubectl get storageclass` and `kubectl get volumesnapshotclass`, and pick a
storage class and a snapshot class that share the same CSI driver.

**minikube** — enable the `volumesnapshots` and `csi-hostpath-driver` addons,
which provide the `csi-hostpath-sc` storage class and `csi-hostpath-snapclass`
snapshot class:

```bash
minikube addons enable volumesnapshots
minikube addons enable csi-hostpath-driver

dit context install -n minikube -t kubernetes \
  -p storageClass=csi-hostpath-sc \
  -p snapshotClass=csi-hostpath-snapclass
```

> **Local prerequisite (running the test suite).** minikube's
> `default-storageclass` addon re-asserts the non-CSI `standard` class as the
> cluster default on every `minikube start`, so a dit-provisioned volume can
> land on a class that cannot snapshot. Before running `make test-kubernetes`,
> run `make k8s-csi-default` to promote `csi-hostpath-sc` to the default (and
> demote `standard`) so dit-provisioned volumes use the CSI driver — otherwise
> VolumeSnapshots fail with `snapshotting non-CSI volumes is not supported`.
> The target is idempotent; re-run it after each `minikube start`. (CI already
> does this in `release.yml` / `pull-request.yml`.)

**A managed/cloud cluster** — use the storage class and snapshot class backed by
your provider's CSI driver. The names below are examples; substitute the ones
your cluster actually exposes (AWS EBS, GCE PD, Azure Disk, etc.):

```bash
dit context install -n prod -t kubernetes \
  -p storageClass=ebs-sc \
  -p snapshotClass=ebs-vsc
```

If you omit these parameters, Dit falls back to the cluster's default storage
class and snapshot class, which only works when that default is itself a CSI
snapshot-capable class.

## Kubernetes Architecture

A Kubernetes repository consists of:

* A PersistentVolumeClaim for each volume identified in the image metadata.
  These are currently always hardcoded to be 1GiB, and use the storage class
  given at install time (`-p storageClass=`), or the cluster default if none was
  specified. Each is given a unique GUID and name.
* A StatefulSet with the same name as the repository.
* Within that StatefulSet, all PersistentVolumeClaims mapped to the directories
  identified in the image metadata. The pod name is the same as the repository
  name.
* A headless Service (same name as the repository) that exposes all ports
  declared by the image to the Pod within the StatefulSet.

Each commit corresponds to a VolumeSnapshot.

By default, Dit makes all exposed ports available on the local system by
spawning a background `kubectl port-forward` process for each known port.
Disable this with `-P` / `--disable-port-mapping` on `dit run` or `dit clone`.
Each forwarder's process id is recorded under `~/.dit`
(`portforward-<repository>-<port>.pid`) so that `dit stop` and `dit rm` can
terminate it, but nothing supervises the process while it runs — if it dies
(host restart, sleep), re-establish it with `dit stop <repo>` followed by
`dit start <repo>`. This will be replaced with a more reliable mechanism in
the future.

## Limitations

> **Beta:** Kubernetes support is currently in a _beta_ state. Many elements of
> configurability and reliability have not yet been fully fleshed out, and it
> may not work in all environments.

In addition to the general immaturity of Kubernetes support, there are some
specific known limitations with beta:

* There is no method to specify volume sizes. While the amount of data pushed
  and pulled will remain the logical size of the dataset, volumes must be
  statically sized in Kubernetes. Currently, these are always 1GiB.
* Dit always uses the default `~/.kube/config` and the `default` namespace;
  there is no way to select a different namespace or cluster. The server keeps
  the copy of the kubeconfig taken at install time while the CLI reads the
  live file, so changing the kubeconfig (or its selected context) after
  installation can leave the two pointing at different clusters. To switch
  clusters, re-install the context.
* The storage class and snapshot class are fixed at install time via
  `-p storageClass=` / `-p snapshotClass=` (see
  [Installing a Kubernetes Context](#installing-a-kubernetes-context)) and
  cannot be changed afterward without re-installing the context. Both must be
  backed by a CSI driver with snapshot support.
* There are various failure modes, such as failing to pull an image, that
  aren't handled well by Dit. These can result in hangs or hard to diagnose
  errors.
* Port forwarding is simplistic. Dit spawns `kubectl port-forward` in the
  background and kills it on `dit stop` / `dit rm`, but nothing restarts it
  automatically. If the system is restarted, or that process dies, re-establish
  it by stopping and starting the repository
  (`dit stop <repo> && dit start <repo>`).

## Troubleshooting

### Commits succeed but clones or checkouts come up empty

If `dit commit` reports success but a later `dit clone` or `dit checkout` produces
an empty database (for example, `ERROR: relation "<table>" does not exist`), the
cluster's **default** storage class is almost certainly not snapshot-capable.

This happens when the storage class backing the volume is not CSI
snapshot-capable. If you did not pin a class at install time, Dit inherits the
cluster default — and a default that lacks a working CSI snapshot driver fails:
on current clusters `dit commit` errors with `snapshotting non-CSI volumes is
not supported`, and on older paths it fails silently, recording a commit whose
VolumeSnapshot never captures any data so clones and checkouts restore an empty
volume (a telltale sign is a commit whose reported size is only a few bytes).

Diagnose it:

```bash
# A commit's snapshot should become READY=true with a non-empty restore size.
# READY=<none> / SIZE=<none> means the source volume's StorageClass cannot snapshot.
kubectl get volumesnapshot

# Confirm which StorageClass is the default and that a CSI snapshot class exists.
kubectl get storageclass
kubectl get volumesnapshotclass
```

The preferred fix is to pin a CSI snapshot-capable class when installing the
context, rather than relying on the default (see
[Installing a Kubernetes Context](#installing-a-kubernetes-context)):

```bash
dit context uninstall -f <context>
dit context install -n <context> -t kubernetes \
  -p storageClass=<csi-storage-class> \
  -p snapshotClass=<csi-snapshot-class>
```

Alternatively, promote a CSI snapshot-capable class to be the cluster default and
demote the old one:

```bash
kubectl patch storageclass <old-default> -p \
  '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"false"}}}'
kubectl patch storageclass <csi-snapshot-class> -p \
  '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

Existing PersistentVolumeClaims keep the storage class they were created with, so
re-create the repository (or re-install the context) after changing the class.