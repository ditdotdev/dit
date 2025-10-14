.. _cli_cmd_rm:

d3 rm
========

Removes a repository. This will stop the container and destroy any local data.
This operation cannot be undone.

Syntax
------

::

    d3 rm [-f] <repository>

Arguments
---------

repository
    *Required*. The name of the target repository.

Example
-------

::

    $ d3 rm -f hello-world
    Removing container hello-world
    Deleting volume hello-world/v0
    hello-world removed
