---
title: Uninstalling Dit
nav_label: Uninstall
nav_order: 180
---

The cli_cmd_install command will install supporting Dit infrastructure
automatically, including installing ZFS on the host or Docker VM if necessary.
The cli_cmd_uninstall command will uninstall Dit, destroying any
repositories in the process.

The uninstall process first will uninstall all configured contexts. Once that
is complete, it will remove the underlying dit container images, as well as
the `.dit` directory in your home directory. If you just want to uninstall a
single context while leaving the Dit images and configuration intact, use
the cli_cmd_context_uninstall command.

> **Warning:**
> Uninstalling dit will remove all repositories. This operation cannot be
> undone.
>

If Dit was responsible for installing ZFS on the host VM, it will also
uninstall ZFS. If ZFS was already present on the system when Dit was
installed, then it will leave the ZFS installation as-is.

> **Note:**
> The process only uninstalls the supporting infrastructure. You will have to
> manually remove the `dit` binary yourself.
