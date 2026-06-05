---
title: Remote Repositories
nav_label: Remote
nav_order: 300
---

While managing data locally on your laptop is all well and good, part of the
power of source code management is the ability to share that data with
others. Much like git, Dit has the notion of `remote repositories` that
act as an endpoint for push and pull.

There are a few important general things to be aware of:

* Dit commits do not have a strict dependency on the previous commit from
  which it was created. Because they are much larger, we allow them to be
  pushed and pulled independently. For this reason, cli_cmd_clone and
  cli_cmd_pull will not pull down `all commits`, only the one specified
  by the user.
* Dit does not support the notion of merging. While concepts like tagging
  and branching will be added over time, generically merging data at the
  on-disk level is not possible.
* Different remote providers have different performance characteristics,
  including whether they support incremental transfers. Some will
  always to a full data transfer, while others have a means to identify
  only changed blocks. Dit is designed to work with small
  datasets (<10GB), using it for anything remotely large may have adverse
  effects on the system.

> **Warning:**
> Dit currently ships with two very basic providers, the remote_provider_s3
> and the remote_provider_ssh. These are only introductory providers, designed
> to have zero dependencies on external software. But as such, they
> will face challenges across security, performance, and robustness when
> operated at scale in an enterprise setting. As Dit matures, we will be
> working with the community and partners to help develop remote providers
> with more robust capabilities.
>
