---
title: Tags
nav_label: Tags
nav_order: 260
---

Commits can be tagged with arbitrary user-controlled. These tags can then
be used to filter logs, control push and pull operations, and augment
other operations.

Tags are structured as a `key=value` map, where each key must be unique.
For example, creating a commit with `build=nightly` and later updating it
to be `build=archive` will overwrite the previous value. As a convenience,
the value can be omitted, in which case the empty string will be used. This
lets tags be used as labels, such as `d3 commit -t nightly ...`.

## Local Tags

A commit can be created with any number of tags set through
cli_cmd_commit. Tags can be added or modified by cli_cmd_tag,
and removed by cli_cmd_delete.

Tags are displayed as part of cli_cmd_log

```bash
$ d3 log postgres
commit 428f81caf63d4314b8f41a31aad2e8b1
User: Eric Schrock
Email: Eric.Schrock@delphix.com
Date: 2019-10-23T20:23:57Z
Tags: foo=bar baz

Commit message
```

## Remote Tags

Tags are automatically propagated as part of push and pull operations. In
addition, these operations can take a `-u` flag, which indicates that
only metadata should be updated. This, combined with the ability to create,
update, and delete tags on local commits, enables the ability to push those
changes to remote repositories (as well as pulling down the latest tags for
a given commit).

Pushing tags only requires that the commit must already exist in the remote.
Otherwise a normal push should be use, which will include both data and tags.

## Filtering by Tag

A number of commands can be augmented by specifying one or more `-t` options
to filter commits by tag. These commands include:

 * cli_cmd_log
 * cli_cmd_remote_log
 * cli_cmd_pull
 * cli_cmd_push
 * cli_cmd_checkout

When tag options are specified, only matching commits (or the latest matching
commit for pull, push, and checkout) are included. The commits must match
all specified tags. If both a key and value are specified, then the tag
must match both exactly. If only a key is specified, then any value will be
considered a match, as simply the existence of the tag is sufficient. Like
tag creation, this allows the value to be optional, treating the tags more
like labels with no explicit value

```bash
$ d3 log -t baz -t foo=bar postgres
commit 428f81caf63d4314b8f41a31aad2e8b1
User: Eric Schrock
Email: Eric.Schrock@delphix.com
Date: 2019-10-23T20:23:57Z
Tags: foo=bar baz

Commit message
```

> **Note:**
> All tags must match for a commit to be included. There is no way to specify
> that one or more tags match (logical OR).
