# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# Elasticsearch 8.19.14 with VOLUME directive for dit compatibility
# Base image from Docker Hub
FROM elasticsearch:8.19.14

# Declare volume for Elasticsearch data directory
# This allows dit to detect and manage the data path
VOLUME ["/usr/share/elasticsearch/data"]

# Use the default command from base image
