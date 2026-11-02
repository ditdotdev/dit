# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# The image runs as its nonroot user (uid 65532) and declares /data and /logs
# as volumes, but ships them owned by root, so RocksDB cannot create its
# directory in a fresh volume (COPY --chown cannot change an existing
# directory's owner, and the image has no shell to chown with). Give the
# server its own volume that does not exist upstream, created here owned by
# nonroot; db_env_args points SURREAL_PATH at it (#262).
FROM busybox:1.37 AS dirs
RUN mkdir -p /out/surrealdb

FROM surrealdb/surrealdb:v3
COPY --from=dirs --chown=65532:65532 /out/surrealdb /var/lib/surrealdb
VOLUME ["/var/lib/surrealdb"]
# The entrypoint is the bare surreal CLI; it needs the start subcommand.
CMD ["start"]
