# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# ditdotdev/minio is our source build of the final upstream MinIO release
# (upstream is archived and its Docker Hub repo was deleted). Its default
# command is bare `minio`, which prints help and exits; the matrix test
# cannot pass a container command, so this wrapper starts the server.
# Tagged ditdotdev/minio:RELEASE.2025-10-15T17-29-55Z-server; the db-matrix
# workflow pulls it from Docker Hub like every other wrapper.
FROM ditdotdev/minio:RELEASE.2025-10-15T17-29-55Z
CMD ["server", "/data"]
