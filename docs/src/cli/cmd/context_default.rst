.. _cli_cmd_context_default:

d3 context default
=====================

Get or set the default Datadatdat context. For more information on managing contexts,
see the :ref:`lifecycle_context` section,

Syntax
------

::

    d3 context default [context]

Arguments
---------

context
    Optional. If specified the named context is made the default context.
    Otherwise, the current default context is displayed.

Example
-------

::

    $ d3 context default
    kubernetes
    $ d3 context default mycontext
    $ d3 context default
    mycontext
