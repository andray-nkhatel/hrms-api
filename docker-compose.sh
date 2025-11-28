#!/bin/bash

# Simple Docker Compose operations script for HRMS

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Function to show usage
show_usage() {
    echo -e "${YELLOW}Usage:${NC} $0 [command]"
    echo ""
    echo "Commands:"
    echo "  up       - Start all services"
    echo "  down     - Stop all services"
    echo "  build    - Build images"
    echo "  restart  - Restart all services"
    echo "  logs     - Show logs (follow mode)"
    echo "  status   - Show service status"
    echo "  clean    - Stop and remove containers, volumes"
    echo ""
}

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed or not in PATH${NC}"
    exit 1
fi

# Use docker compose (newer) or docker-compose (older)
if docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
elif command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker-compose"
else
    echo -e "${RED}Error: docker-compose not found${NC}"
    exit 1
fi

# Change to script directory
cd "$SCRIPT_DIR"

# Handle commands
case "${1:-}" in
    up)
        echo -e "${GREEN}Starting services...${NC}"
        $COMPOSE_CMD up -d
        echo -e "${GREEN}✓ Services started${NC}"
        echo -e "${YELLOW}Access the application at: http://localhost:8070${NC}"
        ;;
    down)
        echo -e "${YELLOW}Stopping services...${NC}"
        $COMPOSE_CMD down
        echo -e "${GREEN}✓ Services stopped${NC}"
        ;;
    build)
        echo -e "${YELLOW}Building images...${NC}"
        $COMPOSE_CMD build
        echo -e "${GREEN}✓ Build complete${NC}"
        ;;
    restart)
        echo -e "${YELLOW}Restarting services...${NC}"
        $COMPOSE_CMD restart
        echo -e "${GREEN}✓ Services restarted${NC}"
        ;;
    logs)
        echo -e "${YELLOW}Showing logs (Ctrl+C to exit)...${NC}"
        $COMPOSE_CMD logs -f
        ;;
    status)
        echo -e "${YELLOW}Service status:${NC}"
        $COMPOSE_CMD ps
        ;;
    clean)
        echo -e "${RED}Warning: This will remove containers and volumes${NC}"
        read -p "Are you sure? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo -e "${YELLOW}Stopping and removing containers, volumes...${NC}"
            $COMPOSE_CMD down -v
            echo -e "${GREEN}✓ Cleanup complete${NC}"
        else
            echo "Cancelled"
        fi
        ;;
    "")
        show_usage
        ;;
    *)
        echo -e "${RED}Unknown command: $1${NC}"
        show_usage
        exit 1
        ;;
esac

