.. _cli_cmd_context_uninstall:

d3 context uninstall
=======================

Uninstall a d3 context. This will permanently remove the context and any
associated repositories (if '-f' is specified). For more information about
contexts, see the :ref:`lifecycle_context` section.

Syntax
------

::

    d3 context uninstall [-f] context

Arguments
---------

context
    *Required*. The name of the target context.

Options
-------

-f, --force              Force the removal of any repositories in the context.
                         By default, the command will fail if any repositories
                         exist.

Example
-------

::

    $ d3 context uninstall newcontext
    Removing Datadatdat Docker volume 100% │███████████████████████████████████│ 100/100 (0:00:00 / 0:00:00)
    Uninstalled d3 infrastructure
