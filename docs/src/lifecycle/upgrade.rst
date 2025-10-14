.. _lifecycle_upgrade:

Upgrading Datadatdat
===============

Datadatdat can be automatically upgraded with the :ref:`cli_cmd_upgrade` command.
This command works by:

1. Downloading a newest version of the d3 binary
2. Finding the location of the ``d3`` binary in your ``PATH``.
3. Copying over the new d3 binary
4. Running post-installation phase of the new binary, which may stop
   all repositories and upgrade the d3 docker container in the process.

This will require access to download binaries from GitHub.
If any parts of this fail, it should leave the original d3 installation
intact.

.. warning::

   As of version ``0.4.0``, upgrade is not currently working. This will be
   addressed in a future release.

.. note::

   If your d3 binary is not found in the PATH, you can specify the
   ``--path`` option to point to where d3 can be found.

Manual Upgrade
--------------
Datadatdat does not currently support upgrading to a d3 binary that has been
manually downloaded, such as when corporate firewalls prevent d3 from
automatically downloading from GitHub. Until this is supported, you will have
to uninstall and re-install datadatdat, destroying any active repositories in the
process.
