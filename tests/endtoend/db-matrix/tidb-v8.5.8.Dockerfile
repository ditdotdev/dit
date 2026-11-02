# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

FROM pingcap/tidb:v8.5.8
VOLUME ["/tmp/tidb"]
# The base image ships no SQL client; the matrix test writes and reads one
# row over the MySQL protocol from inside the container (#262).
RUN dnf install -y mysql && dnf clean all
