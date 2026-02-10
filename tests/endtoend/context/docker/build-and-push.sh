#!/bin/bash
# Build and push multi-architecture nginx-test image
# Usage: ./build-and-push.sh [version]
# Example: ./build-and-push.sh latest

set -e

VERSION=${1:-latest}
IMAGE_NAME="datadatdat/nginx-test"

echo "Building and pushing ${IMAGE_NAME}:${VERSION} for linux/amd64,linux/arm64"

# Create and use a new builder instance that supports multi-platform builds
docker buildx create --name multiarch --use || docker buildx use multiarch

# Build and push multi-architecture image
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag "${IMAGE_NAME}:${VERSION}" \
  --push \
  .

echo "Successfully built and pushed ${IMAGE_NAME}:${VERSION}"
echo "Image supports: linux/amd64, linux/arm64"
