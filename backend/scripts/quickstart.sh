#!/bin/bash

set -e

echo "MBFlow Quick Start Script"
echo "========================="
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if Docker Compose is available
if ! docker compose version &> /dev/null; then
    echo "Error: Docker Compose is not available. Please install Docker Compose."
    exit 1
fi

echo "Starting MBFlow services with Docker Compose..."
docker compose up -d

echo ""
echo "Waiting for services to be healthy..."
sleep 5

# Wait for PostgreSQL
echo -n "Waiting for PostgreSQL..."
for i in {1..30}; do
    if docker compose exec -T postgres pg_isready -U mbflow -d mbflow &> /dev/null; then
        echo " Ready!"
        break
    fi
    echo -n "."
    sleep 1
done

# Wait for Redis
echo -n "Waiting for Redis..."
for i in {1..30}; do
    if docker compose exec -T redis redis-cli ping &> /dev/null; then
        echo " Ready!"
        break
    fi
    echo -n "."
    sleep 1
done

# Wait for API
echo -n "Waiting for MBFlow API..."
for i in {1..30}; do
    if curl -s http://localhost:8181/health &> /dev/null; then
        echo " Ready!"
        break
    fi
    echo -n "."
    sleep 1
done

echo ""
echo "MBFlow is now running!"
echo ""
echo "Services:"
echo "  API:        http://localhost:8181"
echo "  Health:     http://localhost:8181/health"
echo "  Metrics:    http://localhost:8181/metrics"
echo "  PostgreSQL: localhost:5432 (user: mbflow, password: mbflow, db: mbflow)"
echo "  Redis:      localhost:6379"
echo ""
echo "To view logs:"
echo "  docker compose logs -f"
echo ""
echo "To stop services:"
echo "  docker compose down"
echo ""
