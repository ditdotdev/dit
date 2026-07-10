# Copyright Dit 2026
# SPDX-License-Identifier: BUSL-1.1

FROM amazon/dynamodb-local:2.6.1

# Add VOLUME directive for dit compatibility
VOLUME ["/home/dynamodblocal"]

# Set working directory (keep same as base image)
WORKDIR /home/dynamodblocal

# Configure DynamoDB Local to use persistent storage
CMD ["-jar", "DynamoDBLocal.jar", "-sharedDb", "-dbPath", "."]
