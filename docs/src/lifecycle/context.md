---
title: Contexts
nav_label: Contexts
nav_order: 130
---

All dit repositories are associated with a single context. Each context
has a type, either Docker or Kubernetes, that defines how it manages
repositories within it.

* Docker contexts manage containers locally on the current workstation.
* Kubernetes contexts manage containers within a Kubernetes cluster.

In both cases, there is a local container that stores the metadata associated
with the repositories and orchestrates their lifecycle. This container is
named `dit-<context>-server` — for example, `dit-docker-server` for the
default local Docker context.

## Context Configuration

Contexts are installed through the
[dit context install](../cli/cmd/dit_context_install.md) command.
Each context has:

* A type ("docker" or "kubernetes"), set with `-t`
* A name, set with `-n` (defaults to the context type if not specified)
* Optional context-specific parameters, set with `-p key=value`

The [dit install](../cli/cmd/dit_install.md) command is an alias for
`dit context install -t docker`, and will create a default docker context
for managing local containers.

The context configuration is stored in the `~/.dit/config` file in your
home directory. This is a YAML file that contains a `contexts` object that
is a map of context configurations, with the key being the name of the context
and the fields the following:

* `host` - Host to connect to. Currently always "localhost"
* `type` - Type of the context. One of "docker" or "kubernetes"
* `port` - Port that the context container is listening on. Selected at
  random when the context is installed.

While this file can be edited by hand, it is recommended to use the Dit
context commands. To list available contexts, use the
[dit context ls](../cli/cmd/dit_context_ls.md) command. To uninstall a
context, use the [dit context uninstall](../cli/cmd/dit_context_uninstall.md)
command.

## Selecting Contexts

In most situations, a single Dit context is sufficient. When a single
context is in place, repositories can simply be referenced by their name,
and any new repository is created within that context.

Repositories can also be referenced by their fully qualified name,
`<context>/<repository>`. This can be used to uniquely identify any
repository, even when multiple contexts are configured. This can
also be used to select which context to use when creating a new repository,
such as `dit run mongo -n contextone/mongo`.

If the context is not specified, but there is more than one context configured,
Dit will attempt to determine the appropriate context in one of two ways:

* If referencing an existing repository, as opposed to creating a new
  repository, then Dit will try to find a repository with the matching
  name, but will generate an error if repositories with multiple names
  exist.
* If creating a new repository, then the default context (as noted in the
  context configuration file) is used.

The default context is identified in `dit context ls` output via a
" (*)" indicator. You can also get the default context with the
[dit context default](../cli/cmd/dit_context_default.md) command. To set the
default context, run `dit context default <name>`.