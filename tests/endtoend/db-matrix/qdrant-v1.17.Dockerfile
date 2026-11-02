# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

FROM qdrant/qdrant:v1.17
VOLUME ["/qdrant/storage"]
# The base image ships no HTTP client; the matrix test writes and reads one
# point through the REST API from inside the container (#262).
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates && rm -rf /var/lib/apt/lists/*
