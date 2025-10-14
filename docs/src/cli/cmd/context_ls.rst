.. _cli_cmd_context_ls:

d3 context ls
================

Lists configured Datadatdat contexts. This configuration is read from the
``~/.d3`` file. For more information on managing Datadatdat
contexts, see the :ref:`lifecycle_context` section.

For each context, the command will display:

* The context name. If the context is the default context, then an additional
  " (*)" will be appended to the name.
* The context type. One of "docker" or "kubernetes"

Syntax
------

::

    d3 context ls

Example
-------

::

    $ d3 context ls
    NAME                  TYPE
    kubernetes (*)        kubernetes
    docker                docker
