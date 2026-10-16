---
title: Running Containers
nav_label: Run
nav_order: 210
---

A dit repository combines a docker container and its configuration with data
that backs the volumes within that container. So every repository should
have a running docker container associated with it. For more information on
how the docker configuration is used, see the [Managing Docker](docker.md) section.

To run a repository, you simply take the arguments you'd normally supply to
`docker run` and pass that to `dit run` instead

```bash
$ dit run mongo
Creating repository mongo
Creating docker volume mongo_v0 with path /data/configdb
Creating docker volume mongo_v1 with path /data/db
Running controlled container mongo
$ docker ps --filter name=mongo --format '{{.Names}}  {{.Status}}  {{.Ports}}'
mongo  Up 5 seconds  0.0.0.0:27017->27017/tcp, [::]:27017->27017/tcp
$ docker exec -it mongo mongosh --quiet
test> db.names.insertOne({ firstName: "Dorothy", lastName: "Vaughan" })
{
  acknowledged: true,
  insertedId: ObjectId('6ad2848c7cf5efb648c869e4')
}
test> db.names.find()
[
  {
    _id: ObjectId('6ad2848c7cf5efb648c869e4'),
    firstName: 'Dorothy',
    lastName: 'Vaughan'
  }
]
```

There are a couple things to note here:

* Dit will use any docker image published to a register. In this case,
  we pull down and run `mongo:latest`. The only requirement is that the
  containers must have one or more `VOLUME` declarations that indicate
  where persistent data is stored.
* If the image is already present on your docker daemon, dit uses it as-is
  and does not contact the registry. That makes it safe to test an image you
  built locally before pushing it. Use `--pull always` to refresh the image
  from the registry, or `--pull never` to fail instead of pulling when the
  image is not present.
* In this example, dit mapped the default exposed port from the docker
  image to localhost. This is the same thing as manually adding `-p 27017:27017`
  to map the mongo port to the expected port on your laptop.