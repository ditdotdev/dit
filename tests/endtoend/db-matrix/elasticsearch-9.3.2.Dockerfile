# Elasticsearch 9.3.2 with VOLUME directive for datadatdat compatibility
# Base image from Docker Hub
FROM elasticsearch:9.3.2

# Declare volume for Elasticsearch data directory
# This allows datadatdat to detect and manage the data path
VOLUME ["/usr/share/elasticsearch/data"]

# Use the default command from base image
