# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

FROM chromadb/chroma:1.5.7
# Chroma 1.x persists to /data (the "Saving data to" line at startup), not
# /chroma/chroma as 0.x did; a volume at the old path versions nothing (#262).
VOLUME ["/data"]
# The base image ships no HTTP client; the matrix test writes and reads one
# embedding through the REST API from inside the container (#262).
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates && rm -rf /var/lib/apt/lists/*
