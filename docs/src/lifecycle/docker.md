---
title: Datadatdat with Docker
nav_label: Docker
nav_order: 140
---

Datadatdat for docker is designed to run on any system that supports docker, but
there are some dependencies that limit the set of supported operation systems,
especially on Linux.

To help understand why this is necessary, it helps to understand a bit about the
architecture of Datadatdat. To make d3 possible, there is a container
(`d3`) running in the background that provides data
versioning capabilities on top of [ZFS](http://openzfs.org). This requires
that ZFS be installed on the host operating system, but because of how
out-of-tree kernel modules work, this needs to be done by the d3 software (a
container named `d3` in particular). Datadatdat attempts to
provide pre-built versions for common OSes, as well as a means to build them
on the fly for new versions, but there are limits to this system. If you are
not on a supported operating system, you may find `d3 install` taking a
long time to build binaries, or failing outright.

If we do not have a pre-built version of the ZFS binaries, we will attempt to
build them on the fly. For Linux, we are still limited to the set of supported
distributions, but we can build for slightly different variations or versions
if needed. If you are running a Linux system other than a supported
distribution, you can also compile and install ZFS yourself, provided it's in
the `2.2.x` or `2.3.x` series, and Datadatdat will use that instead of trying to
install its own.

If the installation is taking a while, and you see a `zfs-builder`
container in `docker ps` output, then it's off building a custom version
of ZFS. If you are running a supported operating system, then reach out to the
community to see if new pre-built binaries need to be created.

## Supported environments

d3 ships and tests against three Docker environments. Older paths (Docker
Desktop on macOS with the HyperKit/LinuxKit VM; "Docker for WSL" tech
preview; Docker via boot2docker) are no longer supported.

### macOS — Colima

On macOS, d3 runs against [Colima](https://github.com/abiosoft/colima), which
provisions a lightweight Linux VM with a real kernel that d3 can load ZFS
modules into. Docker Desktop's HyperKit/LinuxKit VM was the previous path
and is no longer supported — out-of-tree kernel modules can't be loaded into
the LinuxKit kernel that Docker Desktop ships.

Install Colima with Homebrew and start it before running `d3 install`:

```bash
brew install colima docker
colima start --cpu 4 --memory 8
```

### Linux — native Docker

On a Linux host, d3 runs against the host's native Docker daemon. The host
kernel needs ZFS modules; d3 will install or build them for the host's
kernel during `d3 install`. The community tries to provide pre-built
binaries for:

* Ubuntu 22.04 (Jammy) and later
* Recent Debian, Fedora, and RHEL derivatives

If you are running one of these flavors and not finding pre-built binaries
available, it's likely just a matter of updating the
[zfs-releases](https://github.com/datadatdat/zfs-releases) repository with
the latest build information. For an unsupported distribution or a new
major version, the
[zfs-builder](https://github.com/datadatdat/zfs-builder) repository describes
how to add a new build target.

### Windows — Docker Desktop with the WSL2 backend

On Windows, d3 runs against Docker Desktop using the WSL2 backend. d3
pre-builds the ZFS kernel modules for the WSL2 kernel and loads them via
`insmod` from a privileged helper container during install — this is what
makes ZFS available inside the WSL2 VM that Docker Desktop runs containers
in.

Make sure the **WSL2 backend** is enabled in Docker Desktop → Settings →
General → "Use the WSL 2 based engine". The legacy Hyper-V backend is not
supported.

> **Note:**
> If a new WSL2 kernel ships and the pre-built ZFS modules for it haven't
> been published yet, `d3 install` will try to build them on the fly inside
> the `zfs-builder` container. If that fails, head over to the
> [zfs-releases](https://github.com/datadatdat/zfs-releases) repository
> and open an issue with your `uname -a` output so a new pre-built can be
> published.

> **Note:**
> If you are using an unsupported Linux distribution, you can always
> [install ZFS](https://github.com/openzfs/zfs) yourself. Datadatdat will
> use any installed ZFS in the `2.2.x` or `2.3.x` series and won't attempt
> to install its own modules.
