.. _cli_cmd_remote_rm:

d3 remote rm
===============

Remove a remote from a repository. For more information on managing remotes,
see the :ref:`remote_addremove` section.

Syntax
------

::

    d3 remote rm <repository> <remote>

Arguments
---------

repository
    *Required*. The name of the target repository.

remote
    *Required*. The name of the remote to remove.

Example
-------

::

    $ d3 remote rm hello-world origin
    Removed origin from hello-world
