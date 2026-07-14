---
title: Push and Pull
nav_label: Push/Pull
nav_order: 320
---

The [dit push](../cli/cmd/dit_push.md) and [dit pull](../cli/cmd/dit_pull.md) commands form the basis of
sharing data via remote repositories. Unlike git, however, they transfer
only a single commit to or from the remote repository. There is no notion
of pulling "all commits" and then checking out one of them.

Exactly how each provider transfers data varies. Some, like S3, only do full
transfers of data as a single archive. Others, like SSH, will use rsync to
hopefully transfer only incremental data.

Each push and pull runs asynchronously in the context of the dit container,
but progress is streamed to the command line while it's being run. In rare
cases, it's possible to exit the CLI while the operation is ongoing. In this
case, you may get a message that an operation is in progress. You can either
wait for it to complete, or abort it with [dit abort](../cli/cmd/dit_abort.md).

While the CLI does not provide full-fledged management of remotes (something
specific to each remote), you can get a list of remote commits using the
[dit remote log](../cli/cmd/dit_remote_log.md) command.

> **Note:**
> Dit doesn't currently retry after network errors or other interruptions.
> This capabilities will be added in a future release.
