.. _cli_cmd_remote_ls:

d3 remote ls
===============

List remotes for the given repository. For more information on managing remotes,
see the :ref:`remote_addremove` section.

Syntax
------

::

    d3 remote ls <repository>

Arguments
---------

repository
    *Required*. The name of the target repository.

Example
-------

::

    $ d3 remote ls hello-world
    REMOTE                PROVIDER
    origin                s3
