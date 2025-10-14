.. _cli_cmd_context_install:

d3 context install
=====================

Install a new Datadatdat context. For more information about contexts, see the
:ref:`lifecycle_context` section.

Syntax
------

::

    d3 context install [-t type] [-n name] [-p parameter=value ...] [-v]

Options
-------

-t, --type       type    Optional context type. Must be one of "docker" or
                         "kubernetes". Defaults to "docker".

-n, --name       name    Optional context name. Must be unique. Defaults to
                         the type of the context ("docker" or "kubernetes").

-p, --parameters string  Key=Value pair for provider specific options. See
                         the context-specific documentation for more information.

-v, --verbose            Enable verbose logging. Some contexts can provide
                         additional information about the installation
                         process.

Example
-------

::

    $ d3 context install -t kubernetes -n newcontext
    Initializing d3 infrastructure ...
    Checking docker installation 100% │███████████████████████████████████│ 100/100 (0:00:00 / 0:00:00)
    Starting d3 server docker containers 100% │████████████████████████│ 100/100 (0:00:15 / 0:00:00)
    Datadatdat cli successfully installed, happy data versioning :)
