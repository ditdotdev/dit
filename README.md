# Dit
## Your Code. Your Environment. Your Data. 

![](https://github.com/ditdotdev/ditdotdev/workflows/Publish/badge.svg)
![](https://github.com/ditdotdev/ditdotdev/workflows/End%20to%20End%20Test/badge.svg)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/ditdotdev/dit)
![GitHub All Releases](https://img.shields.io/github/downloads/ditdotdev/ditdotdev/total)

## CI/CD Pipeline

This repository includes a comprehensive Pull Request 2 workflow with:
- Cross-platform testing (Ubuntu, Windows, macOS)
- Multi-version Go support (1.21, 1.22, 1.23)
- Security scanning and code quality checks
- Coverage reporting and performance benchmarks

## <a id="getting-started"></a> Getting Started

### <a id="requirements"></a> Requirements
Before downloading Dit, be sure that you have the appropriate Docker Desktop Client installed and running for your operating system.
*   [Docker Desktop Client](https://www.docker.com/products/docker-desktop)

### <a id="installation"></a> Installation
The available downloads are listed on the [releases](https://github.com/ditdotdev/ditdotdev/releases) tab. Please download the proper package for your operating system and architecture. 

Dit is distributed as a binary with accompanying docker image. Install Dit by unzipping the downloaded release and moving the binary to a directory included in your system's PATH and running the following command from your CLI:
```bash
dit install
```

**macOS users with Colima:** If you're using Colima instead of Docker Desktop, configure Docker to use Colima first:
```bash
docker context use colima
```

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
