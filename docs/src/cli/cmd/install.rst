.. _cli_cmd_install:

d3 install
=============

Installs the default Datadatdat context if not currently configured,
equivalent to ``d3 context install -t docker``. Must be run prior to any
other d3 commands. For more information on how to install d3 and what's
required, see the :ref:`lifecycle_install` section. For more information on
managing contexts, see the :ref:`lifecycle_context` section. If you want to
install Datadatdat for kubernetes, see the :ref:`lifecycle_kubernetes` section.
Running this command will do nothing if the "docker" context already exists.

Syntax
------

::

    d3 install

Options
-------

-r, --registry  registry    Docker Registry URL for private repositories.
                            Defaults to ``d3`` from docker hub.
-V, --verbose               Optionally output install details.

Example
-------

::

    $ d3 install -V -r your.registry.address:port
