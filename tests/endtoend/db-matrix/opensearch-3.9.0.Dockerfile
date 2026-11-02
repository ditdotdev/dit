# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

FROM opensearchproject/opensearch:3.9.0
# The upstream image ships no keystore, so OpenSearch writes one on every
# start. dit clone stops the container for the checkout right after starting
# it; a stop during that write leaves config/opensearch.keystore.tmp behind,
# in the container's own layer rather than the data volume, and the next start
# refuses to overwrite it (#273). Create it once here instead.
RUN bin/opensearch-keystore create
VOLUME ["/usr/share/opensearch/data"]
