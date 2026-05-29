## d3 migrate

Migrate an existing docker database container to datadatdat repository

### Synopsis

Migrate an existing docker database container to datadatdat repository. 
Container becomes the new name of the docker container.

Example: 'd3 migrate -s oldPostgres datadatdatPostgres'

```
d3 migrate [REPOSITORY] [flags]
```

### Options

```
  -h, --help            help for migrate
  -s, --source string   source docker database container (required)
```

### Options inherited from parent commands

```
      --context string   Datadatdat Provider Context
```

### SEE ALSO

* [d3](d3.md)	 - Datadatdat CLI

