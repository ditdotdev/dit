FROM amazon/dynamodb-local:3.3.0

# Add VOLUME directive for dit compatibility
VOLUME ["/home/dynamodblocal"]

# Set working directory (keep same as base image)
WORKDIR /home/dynamodblocal

# Configure DynamoDB Local to use persistent storage
CMD ["-jar", "DynamoDBLocal.jar", "-sharedDb", "-dbPath", "."]
