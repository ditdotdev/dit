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
* The default storage class and snapshot class must use a CSI driver with
  snapshot capabilities.

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

## Kubernetes Architecture

A Kubernetes repository consists of:

* A PersistentVolumeClaim for each volume identified in the image metadata.
  These are currently always hardcoded to be 1GiB, and always use the default
  StorageClass. Each is given a unique GUID and name.
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
* Dit will always use the default storage class and snapshot class. These
  are not currently configurable.
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

Because Dit always uses the default storage class and snapshot class (see
[Kubernetes Requirements](#kubernetes-requirements)), a default class that lacks
a working CSI snapshot driver fails silently: the commit is recorded but its
VolumeSnapshot never captures any data, so clones and checkouts restore an empty
volume. A telltale sign is a commit whose reported size is only a few bytes.

Diagnose it:

```bash
# A commit's snapshot should become READY=true with a non-empty restore size.
# READY=<none> / SIZE=<none> means the source volume's StorageClass cannot snapshot.
kubectl get volumesnapshot

# Confirm which StorageClass is the default and that a CSI snapshot class exists.
kubectl get storageclass
kubectl get volumesnapshotclass
```

If the default storage class is not snapshot-capable, promote a CSI
snapshot-capable class to be the default and demote the old one:

```bash
kubectl patch storageclass <old-default> -p \
  '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"false"}}}'
kubectl patch storageclass <csi-snapshot-class> -p \
  '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

Existing PersistentVolumeClaims keep the storage class they were created with, so
re-create the repository after changing the default.