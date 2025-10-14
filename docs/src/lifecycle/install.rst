.. _lifecycle_install:

Installation and Configuration
==============================

Installing Docker
-----------------
Before installing Datadatdat, you must have docker configured on your system and
permission to run privileged Linux containers. For MacOS and Windows, this
means installing `Docker Desktop <https://docs.docker.com/desktop>`_.
For Linux, this means `installing docker <https://docs.docker.com/get-docker>`_
via your distribution-specific mechanism.

If you can run a basic Linux docker container you're ready for the next step::

    $ docker run --rm busybox:latest echo ready
    ready

Downloading Datadatdat
---------------------
To download Datadatdat, head over to the
`Download Page <https://datadatdat.com/download>`_ and download the archive
specific to your platform. Extract the archive and place it in a location that
is part of your ``PATH`` such as ``~/bin`` or ``/usr/local/bin``.

If you can get the current Datadatdat version you're ready for the next step::

    $ d3 --version
    d3 version 0.5.2

Installing Datadatdat
--------------------
While Datadatdat is delivered as a standalone executable, it relies on a
containerized service to do a lot of the heavy lifting. The ``d3 install``
command will download and run these containers. It may take some time
to download the d3 image, but once complete you should be able to see
two containers running named ``d3`` and
``d3``:

.. note::

   **For MacOS users:** By default, MacOS will block unverified binaries (which
   this is). You may receive an error similar to "'d3' cannot be opened
   because the developer cannot be verified."

   To resolve this, click "cancel," then navigate to "System Preferences"->
   "Security and Privacy">"General" where you will see something like:
   "'Datadatdat' was blocked from use because it's not from an identified developer."

   Click "Open Anyway,"
   return to the terminal and re-run ``d3 install``

::

    $ d3 install
    Initializing d3 infrastructure ...
        √ Checking docker installation
        √ Starting d3 server docker containers
    Datadatdat cli successfully installed, happy data versioning :)
    $ docker ps
    CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS              PORTS                    NAMES
    ff80dcdf8d0e        datadatdat:latest        "/datadatdat/run"             9 seconds ago       Up 7 seconds        0.0.0.0:5001->5001/tcp   datadatdat-docker-server
    6b09cccc407a        datadatdat:latest        "/bin/bash /datadatdat/la…"   29 seconds ago      Up 14 seconds                                datadatdat-docker-launch

By deafult, this installs a local docker context, and is equivalent to
``d3 context install -t docker``. If you want to install Datadatdat
for use with Kubernetes, see the :ref:`lifecycle_context` and
:ref:`lifecycle_kubernetes` sections. If you are operating in a corporate
environment without access to the main docker registry, you can manually load
the ```d3``-r
registry`` option to ``d3 install`` to pull from there instead.

When using the local docker context, the ``d3`` container
is responsible for installing ZFS on the Docker or host VM. For more
information on how this works and supported configurations, see the
:ref:`lifecycle_docker` section.

If you can successfully run ``d3 ls``, then you should be all set::

    $ d3 ls
    CONTEXT             REPOSITORY             STATUS
