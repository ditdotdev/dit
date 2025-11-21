FROM amazon/dynamodb-local:latest

# Add VOLUME directive for datadatdat compatibility
VOLUME ["/home/dynamodblocal"]

# Set working directory (keep same as base image)
WORKDIR /home/dynamodblocal

# Configure DynamoDB Local to use persistent storage
CMD ["-jar", "DynamoDBLocal.jar", "-sharedDb", "-dbPath", "."]
