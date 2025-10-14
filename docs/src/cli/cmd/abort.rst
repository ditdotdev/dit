.. _cli_cmd_abort:

d3 abort
===========

Aborts any in-progress push or pull operation for a repository. Most push and
pull operations finish in the context of when they were invoked, but in rare
cases (such as the CLI exiting unexpectedly), they can continue to run in the
background. This command provides a way to abort such operations.

Syntax
------

::

    d3 abort <repository>

Arguments
---------

repository
    *Required*. The name of the target repository.

Example
-------

::

    $ d3 abort hello-world
    aborting operation 13ac9b7a-15f2-41ea-9b61-bc271234d123
