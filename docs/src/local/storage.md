---
title: Storage
nav_label: Storage
nav_order: 270
---

All of the local dit storage, including the data stored on repositories,
is kept in a single docker volume `dit`. This volume will persist
even across restarts of the dit infrastructure, upgrades of docker, and
other changes on the host.

By default, the `dit` volume is created as a vanilla docker volume,
which uses storage locally on the host system. If you want to use different
storage for your dit work, you can manually create the `dit`
volume yourself prior to running [dit install](../cli/cmd/dit_install.md).

> **Warning:**
> Do not manually change the contents of the `dit` volume, and do
> not change the volume on a running system; use `dit uninstall` first.
> Changing the contents of this volume can have unpredictable effects on
> Dit.
>

> **Warning:**
> If you do create your own `dit` volume, be aware that it will
> automatically destroyed when `dit uninstall` is run. There is not
> currently a way to uninstall dit while preserving the underlying
> volume.
>

## Managing Storage Usage

To view the amount of space consumed by a repository, run the
[dit status](../cli/cmd/dit_status.md) command. This will display output similar to

```bash
$ dit status hello-world
            Status:  running
 Uncompressed Size:  526.5 KiB
   Compressed Size:  254 KiB
       Last Commit:  12c6da4d57004d3497afca4fb914ed58

Volume                          Uncompressed  Compressed
/var/lib/postgresql/data        31.7 MiB      6.9 MiB
```

The compressed size shows the amount of space currently consumed by the
repository, and the amount of space that would be freed if it were to be
destroyed. The volume size represents the amount of data actively being
used. While it can be reduced by freeing up data within the directory,
it may or may not reduce overall data consumption as that data may be
referenced by previous commits.

There is not currently a way to view the amount of storage used by individual
commits.