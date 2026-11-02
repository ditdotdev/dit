# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# The entrypoint is tini with no command; yugabyted must be told to start in
# the foreground. The image runs as uid 10001, and its declared volumes
# (/mnt/disk0, /mnt/disk1) are root-owned, so yugabyted cannot create its
# base_dir inside a fresh dit volume; a chown here would not stick because
# those paths are already VOLUMEs upstream. Give it a volume that does not
# exist upstream, created here owned by the yugabyte user, and point base_dir
# at it so that is what dit versions (#262).
FROM busybox:1.37 AS dirs
RUN mkdir -p /out/var

FROM yugabytedb/yugabyte:2025.1.3.2-b1
COPY --from=dirs --chown=10001:100 /out/var /home/yugabyte/var
VOLUME ["/home/yugabyte/var"]
CMD ["bin/yugabyted", "start", "--background=false", "--base_dir=/home/yugabyte/var"]
