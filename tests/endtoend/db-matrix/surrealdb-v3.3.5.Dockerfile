# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

FROM surrealdb/surrealdb:v3.3.5
# The entrypoint is the bare surreal CLI; it needs the start subcommand.
CMD ["start"]
