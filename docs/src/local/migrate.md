---
title: Migrating Data
nav_label: Migrate
nav_order: 250
---

Dit provides lots of freedom for how you might populate data into a new
repository. You might run some scripts to generate data, load data from
a database dump, or replicate data into the container. Dit does not try to
encode these database-specific mechanisms into the tool, but does provide some
platform-agnostic capabilities for loading data. In this section, we show
how you can migrate data from a container you already have running on your
laptop.

## Source Container

The source container must be from an image available in a docker registry.
Dit copies the data under every volume the image declares (see the
[Copying Repositories](copy.md) section for how to find them). It does not
matter how that data is backed: an anonymous volume created by `docker run`,
a named volume, or a directory bind-mounted from your host all work. For
example, any of these can be migrated:

```
docker run -d --name pg postgres:18
docker run -d --name pg -v pgdata:/var/lib/postgresql postgres:18
docker run -d --name pg -v /home/henrietta/postgres:/var/lib/postgresql postgres:18
```

The container must be stopped. This is both to ensure that the underlying
data is not changing, and because the new repository will use the same
configuration, potentially creating conflicts with ports and other global
resources.

## Migrating a Container

Assuming the container meets the above criteria, you can create a repository
from it by running

```bash
dit migrate -s somecontainer myrepo
```

This will get the configuration of the container and use it verbatim for the
new repository, then copy the contents of each declared volume out of the
stopped container and into the repository, one volume at a time. The copy
runs inside the Dit server, so it behaves the same on Linux and on Docker
Desktop for Windows and macOS.

If any copy fails, `dit migrate` removes the partially created repository
and its volumes, leaves the source container untouched, and exits with a
non-zero status. Nothing is left behind for you to clean up.

> **Note:**
> There is no way to alter the configuration of the container at the time
> it is migrated, you must use the identical configuration as the source
> container.

## Images Without a Declared Volume

Some images keep their data in the container's root filesystem without a
`VOLUME` directive, and some keep it somewhere other than the path they
declare. With no declared volume, `dit migrate` has nothing to tell it where
the data lives and stops:

```
No volumes found for image <image>; pass --volume <container-path> to name the data to copy
```

Name the data paths yourself with `--volume` (short `-v`). Each path must be
absolute and is read from the stopped container exactly as a declared volume
would be, so it does not matter whether Docker keeps it in the writable layer,
a named volume, or a bind mount. Repeat the flag for more than one path; the
paths become the repository's volumes in the order given.

```bash
dit migrate -s legacy-sqlite -v /app/data sqlite-repo
```

When the image does declare volumes and you also pass `--volume`, the paths you
give replace the declared ones and Dit prints a note listing the paths in use.
A path that does not exist in the source container fails the copy, and the
repository is rolled back as described above.
