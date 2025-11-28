#!/bin/bash

# HRMS Docker Startup Script
# This script helps you start the HRMS system with Docker

set -e

echo "🚀 HRMS Docker Setup"
echo "==================="
echo ""

# Check if .env file exists
if [ ! -f .env ]; then
    echo "⚠️  .env file not found. Creating from .env.example..."
    if [ -f .env.example ]; then
        cp .env.example .env
        echo "✅ Created .env file. Please edit it and set your JWT_SECRET!"
        echo ""
        echo "Generate a secure JWT secret with:"
        echo "  openssl rand -base64 32"
        echo ""
        read -p "Press Enter to continue after editing .env file..."
    else
        echo "❌ .env.example not found. Please create .env manually."
        exit 1
    fi
fi

# Check if JWT_SECRET is set and not default
if grep -q "change-this-secret-key-in-production" .env; then
    echo "⚠️  WARNING: JWT_SECRET is still set to default value!"
    echo "   Please generate a secure secret: openssl rand -base64 32"
    echo ""
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# Check if Docker Compose is available
if ! docker compose version > /dev/null 2>&1 && ! docker-compose version > /dev/null 2>&1; then
    echo "❌ Docker Compose is not installed."
    exit 1
fi

# Determine compose command
if docker compose version > /dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
else
    COMPOSE_CMD="docker-compose"
fi

echo "📦 Starting HRMS services..."
echo ""

# Start services
$COMPOSE_CMD -f docker-compose.prod.yml up -d

echo ""
echo "⏳ Waiting for services to be healthy..."
sleep 5

# Check service status
echo ""
echo "📊 Service Status:"
$COMPOSE_CMD -f docker-compose.prod.yml ps

echo ""
echo "✅ HRMS is starting up!"
echo ""
echo "🌐 Access the application at:"
echo "   - Frontend + API: http://localhost:8070"
echo "   - Swagger Docs:   http://localhost:8070/swagger/index.html"
echo "   - Health Check:   http://localhost:8070/health"
echo ""
echo "📝 View logs with:"
echo "   $COMPOSE_CMD -f docker-compose.prod.yml logs -f"
echo ""
echo "🛑 Stop services with:"
echo "   $COMPOSE_CMD -f docker-compose.prod.yml down"
echo ""

