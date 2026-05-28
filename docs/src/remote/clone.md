---
title: Cloning from a Remote
nav_label: Clone
nav_order: 330
---

The cli_cmd_clone command will create a new repository using the
configuration from a remote. It is equivalent to creating a new repository with
an identical configuration, adding the remote, and pulling down the latest
commit

```bash
$ d3 clone -n hello-world s3web://demo.datadatdat.com/hello-world/postgres
```

The docker environment is persisted with each commit, but runtime parameters are
not and can be specified with the `--`` argument flag. See cli_cmd_clone
for more details.

> **Note:**
> The clone command uses the latest commit by default. To clone a specific
> commit, add the commit GUID to the URI with a `#` tag. Example::
>
> $ d3 clone -n hello-world s3web://demo.datadatdat.com/hello-world/postgres#0f53a6a4-90ff-4f8c-843a-a6cce36f4f4f
>

> **Note:**
> The clone command supports filtering the latest commit by tag, which can be done
> via the command line or as part of the URL. To specify tags in the URL, provide
> them as one or more "tag" query parameter. Note that due to a current limitation,
> this must be provided after the "--" delimiteer.
>
> $ d3 clone -- s3://my-bucket/hello-world?tag=label=nightly
