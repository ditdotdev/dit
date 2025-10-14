.. _lifecycle_uninstall:

Uninstalling Datadatdat
==================

The :ref:`cli_cmd_install` command will install supporting Datadatdat infrastructure
automatically, including installing ZFS on the host or Docker VM if necessary.
The :ref:`cli_cmd_uninstall` command will uninstall Datadatdat, destroying any
repositories in the process.

The uninstall process first will uninstall all configured contexts. Once that
is complete, it will remove the underlying d3 container images, as well as
the ``.d3`` directory in your home directory. If you just want to uninstall a
single context while leaving the Datadatdat images and configuration intact, use
the :ref:`cli_cmd_context_uninstall` command.

.. warning::

   Uninstalling d3 will remove all repositories. This operation cannot be
   undone.

If Datadatdat was responsible for installing ZFS on the host VM, it will also
uninstall ZFS. If ZFS was already present on the system when Datadatdat was
installed, then it will leave the ZFS installation as-is.

.. note::

   The process only uninstalls the supporting infrastructure. You will have to
   manually remove the ``d3`` binary yourself.
