.. _cli_cmd_stop:

d3 stop
==========

Stops the container associated with the repository. Equivalent to
``docker stop``.

Syntax
------

::

    d3 stop <repository>

Arguments
---------

repository
    *Required*. The name of the target repository.

Example
-------

::

    $ d3 stop hello-world
    hello-world stopped
