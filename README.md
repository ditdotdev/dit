# Dit
## Your Code. Your Environment. Your Data.

![](https://github.com/ditdotdev/ditdotdev/workflows/Publish/badge.svg)
![](https://github.com/ditdotdev/ditdotdev/workflows/End%20to%20End%20Test/badge.svg)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/ditdotdev/dit)
![GitHub All Releases](https://img.shields.io/github/downloads/ditdotdev/ditdotdev/total)

**Has Claude destroyed your database?** Would you like to have it back, the way `git checkout` brings your code back?

**Have you spent hours — or days — creating a dev database environment?**

**Have you ever wanted to just issue a `clone` and get a copy of another developer's database for development?**

Dit is git for your data. It versions Docker-based databases with the commands you already know — `commit`, `checkout`, `log`, `push`, `pull`, and `clone` — backed by ZFS snapshots, so every commit and checkout is instant no matter how big the database is.

## Quick Start

### 1. Requirements

Install and start [Docker Desktop](https://www.docker.com/products/docker-desktop) for your operating system.

> **macOS users with Colima:** if you use Colima instead of Docker Desktop, run `docker context use colima` first.

### 2. Download

Grab the package for your OS and architecture from the [releases](https://github.com/ditdotdev/ditdotdev/releases) page, unzip it, and move the `dit` binary to a directory on your `PATH`.

### 3. Install

Dit ships as a binary with an accompanying Docker image. One command sets everything up:

```bash
dit install
```

### 4. Version your first database

```bash
# Start a versioned PostgreSQL container
dit run postgres -n mydb -e POSTGRES_PASSWORD=postgres

# Take a commit — a point-in-time snapshot of the database
dit commit mydb -m "Initial commit"

# Make some changes
docker exec mydb psql -U postgres -c "CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(100));"
dit commit mydb -m "Added users table"

# Something (or someone, or some AI) destroys your data?
# List your commits and roll back — just like git.
dit log mydb
dit checkout mydb -c <commit-id>
```

### 5. Or clone a ready-made database

```bash
dit clone -n hello-world s3web://demos.dit.dev/hello-world/postgres
```

That's a fully loaded PostgreSQL database, running locally, in one command. Your teammates can `push` their databases to a shared remote and you can `clone` or `pull` them the same way.

## Download. Try. Get Support.

1. **Download** the latest release from the [releases](https://github.com/ditdotdev/ditdotdev/releases) page.
2. **Try** the [Quick Start](#quick-start) above — it takes about five minutes.
3. **Get support** in the [Dit community Slack](https://join.dit.dev). Found a bug? [Open an issue](https://github.com/ditdotdev/dit/issues).

## Documentation

The full documentation lives in this repository:

* [Getting Started](docs/src/start.md) — a guided tour of the concepts and workflow
* [Installing, configuring, and upgrading](docs/src/lifecycle/) — including [Dit with Kubernetes](docs/src/lifecycle/kubernetes.md)
* [Working with local repositories](docs/src/local/) — running containers, committing, tagging, migrating data
* [Remotes](docs/src/remote/) — push, pull, clone, and the S3/SSH/S3Web remote providers
* [CLI reference](docs/src/cli/) — every command and flag
* [Demo datasets](https://github.com/ditdotdev/dit-demos) — the sample databases used in the quick start

## <a id="development"></a> Development and Testing

### Host ZFS Pool Setup

On a fresh native-Linux or WSL2 dev box, provision the loop-backed ZFS pools that dit expects:

```bash
bash scripts/setup-zfs-pools.sh
```

Pass `--clean` to destroy and recreate the pools. See [`DEVELOPING.md`](DEVELOPING.md) for full development setup.

## <a id="contribute"></a>Contributing

This project follows the Dit community best practices:

  * [Contributing](https://github.com/ditdotdev/.github/blob/master/CONTRIBUTING.md)
  * [Code of Conduct](https://github.com/ditdotdev/.github/blob/master/CODE_OF_CONDUCT.md)
  * [Community Support](https://github.com/ditdotdev/.github/blob/master/SUPPORT.md)

It is maintained by the [Dit community maintainers](https://github.com/ditdotdev/.github/blob/master/MAINTAINERS.md)

For more information on how it works, and how to build and release new versions,
see the [Development Guidelines](DEVELOPING.md).

## License

This project is licensed under the Business Source License 1.1 (BUSL-1.1).
On the Change Date (four years from the publication of each version), the
license for that version converts to the Mozilla Public License 2.0
(MPL-2.0). See [LICENSE](LICENSE) for the full terms.
