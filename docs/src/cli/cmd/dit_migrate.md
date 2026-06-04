## dit migrate

Migrate an existing docker database container to dit repository

### Synopsis

Migrate an existing docker database container to dit repository. 
Container becomes the new name of the docker container.

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

