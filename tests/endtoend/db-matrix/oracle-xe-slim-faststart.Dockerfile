# Oracle XE slim-faststart with VOLUME directive for dit compatibility
# Base image from Docker Hub
FROM gvenzl/oracle-xe:slim-faststart

# Declare volume for Oracle data directory
# This allows dit to detect and manage the data path
VOLUME ["/opt/oracle/oradata"]

# Use the default command from base image
