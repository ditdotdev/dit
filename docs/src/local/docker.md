---
title: Managing Docker
nav_label: Docker
nav_order: 220
---

Dit relies heavily on docker to provide database capabilities. Dit will
inspect docker images to determine what data needs to be persisted, and
uses docker to run the resulting containers. There are several important
implications of this approach, and current limitations that users should
be aware of.

## Docker Images

Docker images form the basis of the repository configuration. They provide
the software used to serve up the data, along with the configuration around
what data needs to be persisted. This latter piece is critical - every image
must have at least one declared volume. For example, the `postgres:11`
image has the following in its Dockerfile

```
VOLUME /var/lib/postgresql/data
```

Which is then visible in the output of `docker inspect postgres:11`

```
"Volumes": {
    "/var/lib/postgresql/data": {}
},
```

If you're unsure if a docker image can be used under dit, look for one or
more volume declarations.

> **Note:**
> Dit doesn't currently support running images that have been built but
> not pushed to a registry. For now, if you are building your own images you
> will need to push them to a registry somewhere, either on dockerhub or
> a private registry, before using them in dit. This will be fixed in a
> future release.
>

> **Note:**
> Dit requires that all volumes be declared within the image metadata.
> Manual overrides to create new points for persistent data are not supported.
> If you have a use case for this, head over to the
> `community <https://dit.dev/community>` and file a feature request
> for us to better understand the need.
>

## Docker Containers

Every dit repository corresponds to exactly one docker container. The
configuration of this container is defined when the cli_cmd_run
command is executed. The name of the repository must match the name of
the container.

> **Note:**
> All Dit repositories must be run with the `-d` (daemonize) flag
> so that the container runs in the background. Synchronous execution of
> containers is not supported. If you have a use case for this, head
> over the [community](https://dit.dev/community) to submit a
> feature request.
>

The docker container must adhere to some general container principles to
be used by dit:

* The container must not maintain state outside of the persistent volumes
  that needs to be preserved. Running a new container on the same data should
  behave exactly as if an existing container was started.
* The container must handle recovering from abnormal shutdown (think pulling
  out the metaphorical power cord) without human intervention. If you need to
  quiesce the container prior to snapshotting data state, you will need to
  automate these extra steps yourself.

> **Warning:**
> If you are running `docker exec` to change configuration after the container
> has started, this configuration likely won't be preserved, and you should look
> into building your own custom image instead.
>

Docker containers can be managed by ``dit, such as
starting and stopping the container. If you `docker rm` the container,
it will show up as ``dit, though it can be restarted by
running `dit start` even in the detached state.

## Docker Configuration

Each time a commit is created, we store both the docker configuration and
the image digest of the image used at the time the commit was created. So
even if you're using a tag like `postgres:latest`, dit will record the
exact image used and make sure to use that image when that commit is later
pulled down. This has the benefit that other people don't need to know
exactly how to run the container, and what image was used to create the
data.

The downside is that this configuration is set in stone at the time the
repository is created. If you need a slightly different configuration,
such as mapping to a different host port or running on a different network,
there is no way to change that. Nor is there a way to run a newer image
on top of the same data. We understand that this limitation is particularly
restrictive, and will be adding support to override or change the docker
configuration as part of future work.

> **Note:**
> Dit does not currently support changing the docker configuration of a
> repository, so all consumers of a particular commit must use the exact
> same configuration. This will be relaxed in a future release.
>

## Other Container Technology

Dit began its life focused on the tool most commonly used by individual
developers on their laptops: Docker. But containers can be used in many
other different places and configurations, such as running multiple containers
through docker compose, running in a Kubernetes cluster, or on a hosted
container platform such as AWS Fargate. We believe that Dit can and will
grow to be able to control data in these types of environments, though the
approach may look very different than the mechanism today.

Keep up to date with our [future roadmap](https://dit.dev/future), and
join the [community](https://dit.dev/community) to help us evolve
in the future.