---
title: Adding and Removing Remotes
nav_label: Add/Remove
nav_order: 310
---

Each repository can have zero or more remotes configured. To add a remote,
use [dit remote add](../cli/cmd/dit_remote_add.md)

```bash
$ dit remote add s3://bucket/path myrepo
```

Remotes are specified as URIs, with the first portion defining the provider
(s3 in the above case), and the rest being specific to that provider. By
default, the remote is named `origin`, but you can also assign remotes
names (required when you have more than one remote). Individual parameters
for each provider can be supplied with the `-p` option, i.e. the ssh provider
can optionally use an `sshKey` parameter instead of the password in the URI.

To get a list of remotes, use [dit remote ls](../cli/cmd/dit_remote_ls.md)

```bash
$ dit remote ls hello-world
REMOTE                PROVIDER
origin                s3
```

Remotes can be removed with the [dit remote rm](../cli/cmd/dit_remote_rm.md) command.