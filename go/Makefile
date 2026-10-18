.PHONY: help build build-linux run run-dev test test-integration test-cover test-race test-verbose \
	clean docker-build docker-build-api docker-build-ui docker-up docker-down docker-logs install fmt lint \
	run-example-basic run-example-custom run-example-embedded \
	migrate-build migrate-up migrate-down migrate-status migrate-reset db-setup db-reset

# migrate-reset uses bash-only syntax ([[ ]], read -n)
SHELL := /bin/bash

# Database URL for migrate-* targets: MBFLOW_DATABASE_URL takes precedence, DATABASE_URL is the fallback.
DB_URL ?= $(or $(MBFLOW_DATABASE_URL),$(DATABASE_URL))

# Docker image names/tags
API_IMAGE ?= mbflow-api:local
UI_IMAGE ?= mbflow-ui:local

# Default target
help:
	@echo "MBFlow - Workflow Orchestration Engine"
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build the server binary"
	@echo "  run           - Run the server locally"
	@echo "  test          - Run tests (embedded postgres, no Docker)"
	@echo "  test-integration - Run integration tests (requires Docker)"
	@echo "  test-cover    - Run tests with coverage"
	@echo "  test-race     - Run tests with race detector"
	@echo "  clean         - Remove build artifacts"
	@echo "  docker-build  - Build API and UI Docker images"
	@echo "  docker-up     - Start Docker Compose services"
	@echo "  docker-down   - Stop Docker Compose services"
	@echo "  install       - Install dependencies"
	@echo "  lint          - Run golangci-lint (fails on issues)"
	@echo "  fmt           - Format code"
	@echo "  migrate-build - Build migration tool"
	@echo "  migrate-up    - Run database migrations"
	@echo "  migrate-down  - Rollback last migration"
	@echo "  migrate-status- Check migration status"
	@echo "  migrate-reset - Reset all migrations (CAUTION)"

# Build targets
build:
	@echo "Building MBFlow server..."
	go build -o mbflow-server ./cmd/server

build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -o mbflow-server-linux ./cmd/server

# Run targets
run: build
	@echo "Starting MBFlow server..."
	./mbflow-server

run-dev:
	@echo "Starting MBFlow server in development mode..."
	LOG_LEVEL=debug go run ./cmd/server

# Test targets
test:
	@echo "Running tests (embedded postgres, no Docker)..."
	go test ./...

test-integration:
	@echo "Running integration tests (requires Docker)..."
	go test -tags=integration ./...

test-cover:
	@echo "Running tests with coverage..."
	go test -cover ./...
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-race:
	@echo "Running tests with race detector..."
	go test -race ./...

test-verbose:
	@echo "Running tests with verbose output..."
	go test -v ./...

# Docker targets
docker-build: docker-build-api docker-build-ui

docker-build-api:
	@echo "Building API image $(API_IMAGE)..."
	docker build -t $(API_IMAGE) -f Dockerfile .

docker-build-ui:
	@echo "Building UI image $(UI_IMAGE)..."
	docker build -t $(UI_IMAGE) -f ../ui/Dockerfile ../ui

docker-up:
	@echo "Starting Docker Compose services..."
	docker compose up -d

docker-down:
	@echo "Stopping Docker Compose services..."
	docker compose down

docker-logs:
	docker compose logs -f

# Development targets
install:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Running linters..."
	@command -v golangci-lint > /dev/null || (echo "golangci-lint not installed. Install v2 with: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

# Clean targets
clean:
	@echo "Cleaning build artifacts..."
	rm -f mbflow-server mbflow-server-linux
	rm -f coverage.out coverage.html
	go clean -cache

# Example targets
run-example-basic:
	@echo "Running basic usage example..."
	go run examples/basic_usage/main.go

run-example-custom:
	@echo "Running custom executor example..."
	go run examples/custom_executor/main.go

run-example-embedded:
	@echo "Running embedded server example..."
	go run examples/embedded_server/main.go

# Migration targets
migrate-build:
	@echo "Building migration tool..."
	go build -o bin/migrate ./cmd/migrate

migrate-up: migrate-build
	@echo "Running database migrations..."
	@test -n "$(DB_URL)" || (echo "ERROR: MBFLOW_DATABASE_URL (or DATABASE_URL) is not set" && exit 1)
	./bin/migrate -database-url "$(DB_URL)" -command up

migrate-down: migrate-build
	@echo "Rolling back last migration..."
	@test -n "$(DB_URL)" || (echo "ERROR: MBFLOW_DATABASE_URL (or DATABASE_URL) is not set" && exit 1)
	./bin/migrate -database-url "$(DB_URL)" -command down

migrate-status: migrate-build
	@echo "Checking migration status..."
	@test -n "$(DB_URL)" || (echo "ERROR: MBFLOW_DATABASE_URL (or DATABASE_URL) is not set" && exit 1)
	./bin/migrate -database-url "$(DB_URL)" -command status

migrate-reset: migrate-build
	@echo "CAUTION: This will reset all migrations and drop all tables!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		test -n "$(DB_URL)" || (echo "ERROR: MBFLOW_DATABASE_URL (or DATABASE_URL) is not set" && exit 1); \
		./bin/migrate -database-url "$(DB_URL)" -command reset; \
	fi

# Database targets
db-setup:
	@echo "Setting up PostgreSQL..."
	@which psql > /dev/null || echo "PostgreSQL client not installed"
	createdb mbflow || true
	psql mbflow -c "CREATE USER mbflow WITH PASSWORD 'mbflow';" || true

db-reset:
	@echo "Resetting database..."
	dropdb mbflow || true
	createdb mbflow
