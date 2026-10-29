# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# Oracle XE 11 slim-faststart with VOLUME directive for dit compatibility
# Base image from Docker Hub
FROM gvenzl/oracle-xe:11-slim-faststart

# Declare volume for Oracle data directory
# This allows dit to detect and manage the data path
# XE 11 installs under ORACLE_BASE=/u01/app/oracle, not /opt/oracle like
# 18, 21 and Free; a volume at /opt/oracle/oradata versions nothing (#262).
VOLUME ["/u01/app/oracle/oradata"]

# Use the default command from base image
