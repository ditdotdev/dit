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
