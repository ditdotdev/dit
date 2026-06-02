---
title: Upgrading Dit
nav_label: Upgrade
nav_order: 160
---

Dit can be automatically upgraded with the cli_cmd_upgrade command.
This command works by:

1. Downloading a newest version of the dit binary
2. Finding the location of the `dit` binary in your `PATH`.
3. Copying over the new dit binary
4. Running post-installation phase of the new binary, which may stop
   all repositories and upgrade the dit docker container in the process.

This will require access to download binaries from GitHub.
If any parts of this fail, it should leave the original dit installation
intact.

> **Warning:**
> As of version `0.4.0`, upgrade is not currently working. This will be
> addressed in a future release.
>

> **Note:**
> If your dit binary is not found in the PATH, you can specify the
> `--path` option to point to where dit can be found.
>

## Manual Upgrade

Dit does not currently support upgrading to a dit binary that has been
manually downloaded, such as when corporate firewalls prevent dit from
automatically downloading from GitHub. Until this is supported, you will have
to uninstall and re-install dit, destroying any active repositories in the
process.