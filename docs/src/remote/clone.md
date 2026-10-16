---
title: Cloning from a Remote
nav_label: Clone
nav_order: 330
---

The [dit clone](../cli/cmd/dit_clone.md) command will create a new repository using the
configuration from a remote. It is equivalent to creating a new repository with
an identical configuration, adding the remote, and pulling down the latest
commit

```bash
$ dit clone -n hello-world s3web://demos.dit.dev/hello-world/postgres
```

The docker environment is persisted with each commit, but runtime parameters are
not and can be specified with the `--` argument flag. See [dit clone](../cli/cmd/dit_clone.md)
for more details.

> **Note:**
> Clone starts the repository from the exact image the commit was made
> with. Each commit records the image digest, and clone runs
> `image:tag@sha256:...` rather than whatever `image:tag` resolves to today,
> so a tag that moved to a newer database version after the data was pushed
> cannot start that newer version over older data files. The digest is
> pulled only when it is not already on your docker daemon, and the local
> `image:tag` is never changed. If the digest cannot be pulled, dit warns and
> falls back to the local `image:tag`. Pass `--pull never` to refuse to pull
> at all: the local `image:tag` is used (with a warning if it differs from
> the committed digest), or the digest if only that is present. Commits
> pushed by older versions of dit may carry no digest; those clone by tag,
> and `--pull always` re-pulls the tag from the registry.

> **Note:**
> The clone command uses the latest commit by default. To clone a specific
> commit, add the commit GUID to the URI with a `#` tag. Example::
>
> $ dit clone -n hello-world s3web://demos.dit.dev/hello-world/postgres#0f53a6a4-90ff-4f8c-843a-a6cce36f4f4f
>

> **Note:**
> The clone command supports filtering the latest commit by tag, which can be done
> via the command line or as part of the URL. To specify tags in the URL, provide
> them as one or more "tag" query parameter. Note that due to a current limitation,
> this must be provided after the "--" delimiteer.
>
> $ dit clone -- s3://my-bucket/hello-world?tag=label=nightly
