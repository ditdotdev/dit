---
title: Installing Dit
nav_label: Install
nav_order: 110
---

## Installing Docker

Before installing Dit, you must have docker configured on your system and
permission to run privileged Linux containers. For MacOS and Windows, this
means installing [Docker Desktop](https://docs.docker.com/desktop).
For Linux, this means [installing docker](https://docs.docker.com/get-docker)
via your distribution-specific mechanism.

If you can run a basic Linux docker container you're ready for the next step

```bash
$ docker run --rm busybox:latest echo ready
ready
```

## Downloading Dit

To download Dit, head over to the
[Download Page](https://dit.dev/download) and download the archive
specific to your platform. Extract the archive and place it in a location that
is part of your `PATH` such as `~/bin` or `/usr/local/bin`.

If you can get the current Dit version you're ready for the next step

```bash
$ dit --version
dit version 0.5.2
```

## Installing Dit

While Dit is delivered as a standalone executable, it relies on a
containerized service to do a lot of the heavy lifting. The `dit install`
command will download and run these containers. It may take some time
to download the dit image, but once complete you should be able to see
two containers running named `dit` and
`dit`:

> **Note:**
> **For MacOS users:** By default, MacOS will block unverified binaries (which
> this is). You may receive an error similar to "'dit' cannot be opened
> because the developer cannot be verified."
>
> To resolve this, click "cancel," then navigate to "System Preferences"->
> "Security and Privacy">"General" where you will see something like:
> "'Dit' was blocked from use because it's not from an identified developer."
>
> Click "Open Anyway,"
> return to the terminal and re-run `dit install`
>

```bash
$ dit install
Initializing dit infrastructure ...
    √ Checking docker installation
    √ Starting dit server docker containers
Dit cli successfully installed, happy data versioning :)
$ docker ps
CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS              PORTS                    NAMES
ff80dcdf8d0e        dit:latest        "/ditdotdev/run"             9 seconds ago       Up 7 seconds        0.0.0.0:5001->5001/tcp   dit-docker-server
6b09cccc407a        dit:latest        "/bin/bash /ditdotdev/la…"   29 seconds ago      Up 14 seconds                                dit-docker-launch
```

By default, this installs a local docker context, and is equivalent to
`dit context install -t docker`. If you want to install Dit
for use with Kubernetes, see the [Contexts](context.md) and
[Kubernetes](kubernetes.md) sections. If you are operating in a corporate
environment without access to the main docker registry, you can pass the
`-r <registry>` option to `dit install` to pull images from a private
registry instead.

When using the local docker context, the `dit` container is responsible for
installing ZFS on the Docker or host VM. For more information on how this
works and supported configurations, see [Dit with Docker](docker.md).

If you can successfully run `dit ls`, then you should be all set

```bash
$ dit ls
CONTEXT             REPOSITORY             STATUS
```
