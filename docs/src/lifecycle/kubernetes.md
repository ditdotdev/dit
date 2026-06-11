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

Dit requires a Kubernetes cluster with the following configuration options:

* There must be a CSI (Container Storage Interface) driver installed that
  supports the [alpha snapshot](https://kubernetes-csi.github.io/docs/snapshot-restore-feature.html)
  capabilities. Dit does not yet work with the
  [beta snapshot APIs](https://kubernetes.io/blog/2019/12/09/kubernetes-1-17-feature-cis-volume-snapshot-beta/).
* The [VolumeSnapshotDataSource](https://v1-13.docs.kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/)
  feature gate must be enabled.
* The [VolumeSnapshot](https://kubernetes.io/docs/concepts/storage/volume-snapshots/)
  API must be enabled.
* The storage class and snapshot class Dit uses — whether specified at install
  time (see [Installing a Kubernetes Context](#installing-a-kubernetes-context))
  or inherited from the cluster default — must use a CSI driver with snapshot
  capabilities.

Dit currently uses the default Kubernetes config file, cluster and namespace
as defined the `.kube/config` file in your home directory. Future versions will make these
configurable.

The dit server still runs as a container on the local workstation. A local
Docker installation is required, though no special privileges or operating
system support is necessary. This also means that all the metadata is local to the
user, so two users cannot share dit repositories in a shared Kubernetes
cluster. The pods themselves will be accessible to any kubernetes user, but
there is no way to manage them as Dit repositories on a different system.

Each push or pull operation is run as a separate Job, requiring that the
`ditdotdev/dit` image be avaialble to the cluster.

## Installing a Kubernetes Context

Install a Kubernetes context with `dit context install -t kubernetes`. The
storage and snapshot classes Dit uses for its PersistentVolumeClaims (volumes)
and VolumeSnapshots (commits) are set with the `-p storageClass=<name>` and
`-p snapshotClass=<name>` parameters. Both **must** be backed by a CSI driver
with snapshot support.

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
* A service that maps all exposed ports to the ports of the Pods within the
  StatefulSet.

Each commit corresponds to a VolumeSnapshot.

By default, Dit will make all ports available on the local system. This
is accomplished by running `kubectl port-forward` for each known port. This
is a fairly fragile process, since that process can die or the system
restarted at any time. This will be replaced with a more reliable mechanism
in the future.

## Limitations

> **Beta:** Kubernetes support is currently in a _beta_ state. Many elements of
> configurability and reliability have not yet been fully fleshed out, and it
> may not work in all environments.

In addition to the general immaturity of Kubernetes support, there are some
specific known limitations with beta:

* There is no method to specify volume sizes. While the amount of data pushed
  and pulled will remain the logical size of the dataset, volumes must be
  statically sized in Kubernetes. Currently, these are always 1GiB.
* Dit currently always uses the default ~/.kube configuration, and there isn't
  a way to control the namespace and cluster used. If the default configuration
  is changed after the context is installed, it can result in inconsistent
  state.
* The storage class and snapshot class are fixed at install time via
  `-p storageClass=` / `-p snapshotClass=` (see
  [Installing a Kubernetes Context](#installing-a-kubernetes-context)) and
  cannot be changed afterward without re-installing the context. Both must be
  backed by a CSI driver with snapshot support.
* There are various failure modes, such as failing to pull an image, that
  aren't handled well by Dit. These can result in hangs or hard to diagnose
  errors.
* Port forwarding is very simplistic. Dit simply spawns `kubectl port-forward`
  in the background, and tries to kill it when stopping port forwarding. If
  the system is restarted, or that process dies, it will need to be manually
  restarted, either by running the `kubectl` directly, or stopping and
  starting the repository.

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