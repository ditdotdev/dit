# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

FROM influxdb:3.9.1-core
VOLUME ["/var/lib/influxdb3"]
# InfluxDB 3 requires a bearer token for every request and only prints the
# admin token once at creation, so the matrix test runs the server without
# authorization and talks to it with the bundled influxdb3 CLI (#262).
CMD ["influxdb3", "serve", "--without-auth"]
