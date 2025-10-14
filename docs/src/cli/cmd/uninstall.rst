.. _cli_cmd_uninstall:

d3 uninstall
===============

Uninstall all d3 infrastructure. This will uninstall all contexts,
equivalent to ``d3 context uninstall``. It will then remove any d3 images
pulled locally, as well as the ``~/.d3`` directory. For more information on
what exactly is cleaned up, see the :ref:`lifecycle_uninstall` section. This
command will fail if any active repositories exist, unless the ``-f`` option is
supplied.

.. warning::

   If you specify the ``-f`` force option, *all* d3 repositories will be
   forcibly destroyed. This action is not recoverable. Proceed with caution.

Syntax
------

::

    d3 uninstall [-f]

Options
-------

-f, --force     Forcibly stop and remove any repositories that currently exist.
