# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

FROM yugabytedb/yugabyte:2025.2.2.2-b11
# The entrypoint is tini with no command; yugabyted must be told to start in
# the foreground. Its default base_dir is /root/var, outside the image's
# declared volumes (/mnt/disk0, /mnt/disk1), so point it at one of them or
# dit versions nothing (#262).
CMD ["bin/yugabyted", "start", "--background=false", "--base_dir=/mnt/disk0/yugabyted"]
