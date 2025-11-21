# Oracle XE 18-slim-faststart with VOLUME directive for datadatdat compatibility
# Base image from Docker Hub  
FROM gvenzl/oracle-xe:18-slim-faststart

# Declare volume for Oracle data directory
# This allows datadatdat to detect and manage the data path
VOLUME ["/opt/oracle/oradata"]

# Use the default command from base image
