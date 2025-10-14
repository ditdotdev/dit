.. _cli_cmd_start:

d3 start
===========

Starts the container associated with the repository. Equivalent to
``docker start``.

Syntax
------

::

    d3 start <repository>

Arguments
---------

repository
    *Required*. The name of the target repository.

Example
-------

::

    $ d3 start hello-world
    hello-world started
