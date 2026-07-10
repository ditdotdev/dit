# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

# SQL Server 2017 with VOLUME directive for dit compatibility
# Base image from Microsoft Container Registry
FROM mcr.microsoft.com/mssql/server:2017-latest

# Declare volume for SQL Server data directory
# This allows dit to detect and manage the data path
VOLUME ["/var/opt/mssql"]

# Expose SQL Server port
EXPOSE 1433

# Run SQL Server process
CMD ["/opt/mssql/bin/sqlservr"]
