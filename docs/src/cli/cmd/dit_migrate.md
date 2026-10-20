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

Example: 'dit migrate -s oldPostgres ditPostgres'

```
dit migrate [REPOSITORY] [flags]
```

### Options

```
  -h, --help            help for migrate
  -s, --source string   source docker database container (required)
```

### Options inherited from parent commands

```
      --context string   Dit Provider Context
```

### SEE ALSO

* [dit](dit)	 - Dit CLI

