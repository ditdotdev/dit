## dit migrate

Migrate an existing docker database container to dit repository

### Synopsis

Migrate an existing docker database container to dit repository.
Container becomes the new name of the docker container.

The source container must be stopped. The data under each volume the image
declares is copied into the new repository, whatever backs it (anonymous
volume, named volume, or bind mount). If the copy fails, the repository is
rolled back and removed, the source container is left unchanged, and the
command exits 1.

If the image declares no VOLUME, or its data lives somewhere other than the
declared paths, pass --volume with each absolute container path to copy. The
paths given replace the image's declared volumes and become the repository's
volumes in order (v0, v1, ...).

Examples:
  dit migrate -s oldPostgres ditPostgres
  dit migrate -s legacy-sqlite -v /app/data sqlite-repo

```
dit migrate [REPOSITORY] [flags]
```

### Options

```
  -h, --help[=true|false]    help for migrate
  -s, --source string        source docker database container (required)
  -v, --volume stringArray   absolute container path to copy instead of the image's declared volumes (repeatable)
```

### Options inherited from parent commands

```
      --context string   Dit Provider Context
```

### SEE ALSO

* [dit](dit)	 - Dit CLI

