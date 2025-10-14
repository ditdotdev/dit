.. _cli:

Command Line Reference
======================

The ``d3`` command line is the primary tool for managing repositories and
commits. While there are a number of detailed subcommands, there are some
global options as well.

.. note::

  The ``--help`` option can be used to provide more detail about subcommands
  and their options, such as ``d3 run --help`` or ``d3 remote --help``.

Syntax
------

::

   d3 --help
   d3 --version
   d3 subcommand ...

Options
-------

--version       Display the d3 version and exit.
--help, -h      Display available subcommands.

Subcommands
-----------

.. toctree::
   :maxdepth: 1

   cmd/abort
   cmd/checkout
   cmd/clone
   cmd/commit
   cmd/context_default
   cmd/context_install
   cmd/context_ls
   cmd/context_uninstall
   cmd/delete
   cmd/cp
   cmd/install
   cmd/log
   cmd/ls
   cmd/migrate
   cmd/pull
   cmd/push
   cmd/remote_add
   cmd/remote_log
   cmd/remote_ls
   cmd/remote_rm
   cmd/rm
   cmd/run
   cmd/start
   cmd/status
   cmd/stop
   cmd/tag
   cmd/uninstall
   cmd/upgrade
