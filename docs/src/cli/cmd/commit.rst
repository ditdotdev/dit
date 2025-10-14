.. _cli_cmd_commit:

d3 commit
============

Commits the current data state of the container. When creating commits, datadatdat
will use your git configuration to determine the name and email address to use
by running ``git config user.name`` and ``git config user.email``. If
you have not configured git before, you can run
``git config --global user.name <name>`` and
``git config --global user.email <email>`` to set these values.

.. note::

   If you are not a git user and don't want to have to install it to use Datadatdat,
   join the `Community <https://datadatdat.com/community>`_ to help
   design and implement an alternative.

.. warning::

   Datadatdat assumes that it is safe to snapshot the current state of the data
   while the container is running, and that starting the container with
   data in such a state will automatically recover. This would necessarily be
   true for any data store that can survive an unexpected outage. If you
   are working with a container that first must be manually quiesced, you
   should `d3 stop` the container prior to committing state, and
   `d3 start` it afterwards.

Syntax
------

::

    d3 commit [-m message] [-t key[=value] ...] <repository>

Arguments
---------

repository
    *Required*. The name of the target repository.

Options
-------

-m, --message message  Specify a human-readable message associated with the
                       commit. This message, along with author information,
                       will be visible in ``d3 log`` output and propagate
                       with the commit when pushed to, or pulled from, remote
                       repositories. If not provided, then an empty string is
                       used.

-t, --tag tag          Specify a tag to set for the commit. This option can
                       appear multiple times. If the value is omitted, then the
                       empty string is assumed. For more information on tags,
                       see the :ref:`local_tags` section.

Example
-------

::

    $ d3 commit -m "my first commit" -t nightly -t source=qa myrepo
    Commit 470ceb06-ebd3-486a-a4de-7f755df84309
