# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

FROM yugabytedb/yugabyte:2025.2.2.2-b11
# The entrypoint is tini with no command; yugabyted must be told to start in the foreground.
CMD ["bin/yugabyted", "start", "--background=false"]
