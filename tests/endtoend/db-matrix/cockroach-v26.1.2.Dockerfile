# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

FROM cockroachdb/cockroach:v26.1.2
VOLUME /cockroach/cockroach-data
# The entrypoint is the bare cockroach CLI; without a mode it prints usage and exits.
CMD ["start-single-node", "--insecure"]
