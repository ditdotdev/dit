# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# Elasticsearch 9.2.8 with VOLUME directive for dit compatibility
# Base image from Docker Hub
FROM elasticsearch:9.2.8

# Declare volume for Elasticsearch data directory
# This allows dit to detect and manage the data path
VOLUME ["/usr/share/elasticsearch/data"]

# Use the default command from base image
