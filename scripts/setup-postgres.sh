#!/bin/bash

# MBFlow PostgreSQL Setup Script
# This script helps you quickly set up PostgreSQL for MBFlow

set -e

CONTAINER_NAME="mbflow-postgres"
POSTGRES_PASSWORD="postgres"
POSTGRES_DB="mbflow"
POSTGRES_PORT="5432"

echo "🚀 MBFlow PostgreSQL Setup"
echo "=========================="
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    echo "   Visit: https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if container already exists
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "📦 Container '${CONTAINER_NAME}' already exists."
    
    # Check if it's running
    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        echo "✅ Container is already running."
    else
        echo "▶️  Starting existing container..."
        docker start ${CONTAINER_NAME}
        echo "✅ Container started successfully."
    fi
else
    echo "📥 Creating new PostgreSQL container..."
    docker run --name ${CONTAINER_NAME} \
        -e POSTGRES_PASSWORD=${POSTGRES_PASSWORD} \
        -e POSTGRES_DB=${POSTGRES_DB} \
        -p ${POSTGRES_PORT}:5432 \
        -d postgres:15
    
    echo "⏳ Waiting for PostgreSQL to be ready..."
    sleep 3
    
    echo "✅ PostgreSQL container created and started successfully."
fi

echo ""
echo "📊 Container Status:"
docker ps --filter name=${CONTAINER_NAME} --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

echo ""
echo "🔗 Connection Details:"
echo "   DSN: postgres://postgres:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"
echo ""
echo "💡 Useful Commands:"
echo "   Stop container:    docker stop ${CONTAINER_NAME}"
echo "   Start container:   docker start ${CONTAINER_NAME}"
echo "   Remove container:  docker rm -f ${CONTAINER_NAME}"
echo "   View logs:         docker logs ${CONTAINER_NAME}"
echo "   Connect with psql: docker exec -it ${CONTAINER_NAME} psql -U postgres -d ${POSTGRES_DB}"
echo ""
echo "🎉 PostgreSQL is ready! You can now start the MBFlow server:"
echo "   go run cmd/server/main.go"
echo ""
