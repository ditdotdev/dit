## dit clone

Clone a remote repository to local repository

```
dit clone URI [ARGS...] [flags]
```

### Options

```
  -c, --commit string          commit to checkout
  -P, --disable-port-mapping   disable default port mapping from container to localhost
  -h, --help                   help for clone
  -n, --name string            optional new name for repository
  -p, --parameters strings     provider specific parameters. key=value format
      --pull string            image pull policy: missing (use the local image, pull only if absent; clone uses the digest recorded in the commit), always (re-pull the tag; clone uses the recorded digest), never (local images only) (default "missing")
  -t, --tags strings           filter latest commit by tags
```

### Options inherited from parent commands

```
      --context string   Dit Provider Context
```

### SEE ALSO

* [dit](dit)	 - Dit CLI

