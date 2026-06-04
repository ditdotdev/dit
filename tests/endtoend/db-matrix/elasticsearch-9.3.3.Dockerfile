# Elasticsearch 9.3.3 with VOLUME directive for dit compatibility
# Base image from Docker Hub
FROM elasticsearch:9.3.3

# Declare volume for Elasticsearch data directory
# This allows dit to detect and manage the data path
VOLUME ["/usr/share/elasticsearch/data"]

# Use the default command from base image
