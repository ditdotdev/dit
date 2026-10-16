## dit run

Create repository and start container

### Synopsis

Create repository and start container.
Containers associated with a repository can be launched using context specific
run arguments and passed verbatim using '--' as the flag.

Docker example: 'dit run --disable-port-mapping postgres -- -p 2345:5432'
Privileged example: 'dit run --privileged icr.io/db2_community/db2:latest'
Local image example: 'dit run --pull never myorg/postgres:16'

```
dit run [IMAGE] [flags]
```

### Options

```
  -n, --name string            optional new name for repository
  -e, --env strings            container specific environment variables
  -P, --disable-port-mapping   disable default port mapping from container to localhost
      --privileged             run container in privileged mode with extended permissions
      --pull string            image pull policy: missing (use the local image, pull only if absent; clone uses the digest recorded in the commit), always (re-pull the tag; clone uses the recorded digest), never (local images only) (default "missing")
  -t, --tags strings           filter latest commit by tags
  -h, --help                   help for run
```

### Options inherited from parent commands

```
      --context string   Dit Provider Context
```

### SEE ALSO

* [dit](dit)	 - Dit CLI

