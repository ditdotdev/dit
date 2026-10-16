---
title: Committing Data
nav_label: Commit
nav_order: 230
---

Once you have a repository running as outlined in the
[Running Containers](run.md) section, you can start to commit changes. Picking up where
the previous example left off, we can create some data and then commit
that state

```bash
$ docker exec -it mongo mongosh --quiet
test> db.names.insertOne({ firstName: "Katherine", lastName: "Goble" })
{
  acknowledged: true,
  insertedId: ObjectId('6ad2848d607235b843eee88f')
}
test> db.names.insertOne({ firstName: "Mary", lastName: "Jackson" })
{
  acknowledged: true,
  insertedId: ObjectId('6ad2848ec3faa2d55ad9c859')
}
test> db.names.find()
[
  {
    _id: ObjectId('6ad2848c7cf5efb648c869e4'),
    firstName: 'Dorothy',
    lastName: 'Vaughan'
  },
  {
    _id: ObjectId('6ad2848d607235b843eee88f'),
    firstName: 'Katherine',
    lastName: 'Goble'
  },
  {
    _id: ObjectId('6ad2848ec3faa2d55ad9c859'),
    firstName: 'Mary',
    lastName: 'Jackson'
  }
]
test>
$ dit commit -m "hidden figures" mongo
Commit 5c40f81653d24d53a2ce03748bdba3f8
```

The `commit` command took a snapshot of the on-disk state for the mongodb
container `mongo` and stored that in the repository. It should now be
visible in the log

```bash
$ dit log mongo
commit 5c40f81653d24d53a2ce03748bdba3f8
User: Anne Smith
Email: anne.smith@gmail.com
Timestamp: 2026-10-16T20:09:50.852552806Z

hidden figures
```

From here, we can add additional state and create a second commit

```bash
$ docker exec -it mongo mongosh --quiet
test> db.names.insertMany([{ firstName: "Grace", lastName: "Hopper" }, { firstName: "Ada", lastName: "Lovelace" }])
{
  acknowledged: true,
  insertedIds: {
    '0': ObjectId('6ad2848fc0e5ebcfd8b1109c'),
    '1': ObjectId('6ad2848fc0e5ebcfd8b1109d')
  }
}
test> db.names.find()
[
  {
    _id: ObjectId('6ad2848c7cf5efb648c869e4'),
    firstName: 'Dorothy',
    lastName: 'Vaughan'
  },
  {
    _id: ObjectId('6ad2848d607235b843eee88f'),
    firstName: 'Katherine',
    lastName: 'Goble'
  },
  {
    _id: ObjectId('6ad2848ec3faa2d55ad9c859'),
    firstName: 'Mary',
    lastName: 'Jackson'
  },
  {
    _id: ObjectId('6ad2848fc0e5ebcfd8b1109c'),
    firstName: 'Grace',
    lastName: 'Hopper'
  },
  {
    _id: ObjectId('6ad2848fc0e5ebcfd8b1109d'),
    firstName: 'Ada',
    lastName: 'Lovelace'
  }
]
test>
$ dit commit -m "more great scientists" mongo
Commit dc38c922540849d4a2d5865c5dda9460
```

Once we have this committed state, we can easily go back to a previous state
by running `dit checkout`

```bash
$ dit checkout -c 5c40f81653d24d53a2ce03748bdba3f8 mongo
Stopping container mongo
Checkout 5c40f81653d24d53a2ce03748bdba3f8
Starting container mongo
5c40f81653d24d53a2ce03748bdba3f8 checked out
$ docker exec -it mongo mongosh --quiet
test> db.names.find()
[
  {
    _id: ObjectId('6ad2848c7cf5efb648c869e4'),
    firstName: 'Dorothy',
    lastName: 'Vaughan'
  },
  {
    _id: ObjectId('6ad2848d607235b843eee88f'),
    firstName: 'Katherine',
    lastName: 'Goble'
  },
  {
    _id: ObjectId('6ad2848ec3faa2d55ad9c859'),
    firstName: 'Mary',
    lastName: 'Jackson'
  }
]
test>
```

Here you can see that we stopped the container, swapped out the data, and
started it again. And with that, we're back to the original commit we created.

> **Warning:**
> The dit infrastructure has not currently been built for scale, and while it
> should work fine for dozens of commits, creating hundreds or thousands of
> commits or repositories may have adverse effects on the system. This will be
> addressed in a future release.
>

For information on more additional local workflows, see the
local section.