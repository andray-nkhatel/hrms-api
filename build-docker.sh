#!/bin/bash

# Build script for HRMS API and Client Docker image
# This script builds both the Go API and Vue client into a single Docker image

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
IMAGE_NAME="hrms-api"
IMAGE_TAG="${1:-latest}"
PORT=8070

echo -e "${GREEN}=== Building HRMS Docker Image ===${NC}"
echo -e "Image: ${IMAGE_NAME}:${IMAGE_TAG}"
echo -e "Port: ${PORT}"
echo ""

# Get the script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"
PARENT_DIR="$(dirname "$PROJECT_ROOT")"
CLIENT_DIR="$PARENT_DIR/hrmsclient"

# Check if client directory exists
if [ ! -d "$CLIENT_DIR" ]; then
    echo -e "${RED}Error: Client directory not found at $CLIENT_DIR${NC}"
    exit 1
fi

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running. Please start Docker and try again.${NC}"
    exit 1
fi

echo -e "${YELLOW}Step 1: Building Docker image...${NC}"
echo -e "Build context: $PARENT_DIR"
echo -e "Dockerfile: $PROJECT_ROOT/Dockerfile"
echo ""

# Build from parent directory so we can access both hrms-api and hrmsclient
docker build \
    --build-arg BUILDKIT_INLINE_CACHE=1 \
    -t "${IMAGE_NAME}:${IMAGE_TAG}" \
    -f "$PROJECT_ROOT/Dockerfile" \
    "$PARENT_DIR"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Docker image built successfully!${NC}"
    echo ""
    echo -e "${GREEN}=== Build Complete ===${NC}"
    echo -e "Image: ${IMAGE_NAME}:${IMAGE_TAG}"
    echo ""
    echo -e "${YELLOW}To run the container:${NC}"
    echo -e "  docker run -p ${PORT}:${PORT} -e PORT=${PORT} ${IMAGE_NAME}:${IMAGE_TAG}"
    echo ""
    echo -e "${YELLOW}To run with database (using docker-compose):${NC}"
    echo -e "  cd $PROJECT_ROOT"
    echo -e "  docker-compose up -d postgres"
    echo -e "  docker run -p ${PORT}:${PORT} --network hrms-api_default -e DB_HOST=postgres ${IMAGE_NAME}:${IMAGE_TAG}"
else
    echo -e "${RED}✗ Docker build failed!${NC}"
    exit 1
fi

