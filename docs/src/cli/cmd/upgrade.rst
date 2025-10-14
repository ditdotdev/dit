.. _cli_cmd_upgrade:

d3 upgrade
=============

Upgrade the d3 software on the host system. This will automatically fetch
the latest version of Datadatdat, replace the current d3 binary, and then
update the d3 supporting infrastructure. For more information on upgrade,
see the :ref:`lifecycle_upgrade` section.

.. warning::

   Upgrade requires all containers to be stopped, or the '-f' option to
   forcefully stop all containers.

Syntax
------

::

    d3 upgrade [-f] [-p path]

Options
-------

-f, --force     Stop all containers.

-p, --path      Specify path to d3 binary. By default, will attempt to find
                the binary in the path. If you are executing d3 as an
                alias, or in a wrapper script, you will need to specify the
                path to the actual d3 binary.
