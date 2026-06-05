## dit org member add

Add a member to an organization

### Synopsis

Add a user to an organization with a role (member, admin, or owner).
By default <user> is treated as a user ID; pass --github-login to resolve a GitHub login instead.

```
dit org member add <org> <user> [flags]
```

### Options

```
      --github-login    Treat <user> as a GitHub login instead of a user ID
  -h, --help            help for add
      --role string     Role: member, admin, or owner (default "member")
      --server string   Server URL
```

### Options inherited from parent commands

```
      --context string   Dit Provider Context
```

### SEE ALSO

* [dit org member](dit_org_member)	 - Manage organization members

