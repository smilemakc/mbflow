.PHONY: help build run test clean docker-build docker-up docker-down install lint migrate

# Default target
help:
	@echo "MBFlow - Workflow Orchestration Engine"
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build the server binary"
	@echo "  run           - Run the server locally"
	@echo "  test          - Run all tests"
	@echo "  test-cover    - Run tests with coverage"
	@echo "  test-race     - Run tests with race detector"
	@echo "  clean         - Remove build artifacts"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-up     - Start Docker Compose services"
	@echo "  docker-down   - Stop Docker Compose services"
	@echo "  install       - Install dependencies"
	@echo "  lint          - Run linters"
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
	@echo "Running tests..."
	go test ./...

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
docker-build:
	@echo "Building Docker image..."
	docker build -t mbflow:latest .

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
	@which golangci-lint > /dev/null || echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
	@which golangci-lint > /dev/null && golangci-lint run || echo "Skipping lint"

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
	@test -n "$(DATABASE_URL)" || (echo "ERROR: DATABASE_URL is not set" && exit 1)
	./bin/migrate -database-url "$(DATABASE_URL)" -command up

migrate-down: migrate-build
	@echo "Rolling back last migration..."
	@test -n "$(DATABASE_URL)" || (echo "ERROR: DATABASE_URL is not set" && exit 1)
	./bin/migrate -database-url "$(DATABASE_URL)" -command down

migrate-status: migrate-build
	@echo "Checking migration status..."
	@test -n "$(DATABASE_URL)" || (echo "ERROR: DATABASE_URL is not set" && exit 1)
	./bin/migrate -database-url "$(DATABASE_URL)" -command status

migrate-reset: migrate-build
	@echo "CAUTION: This will reset all migrations and drop all tables!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		test -n "$(DATABASE_URL)" || (echo "ERROR: DATABASE_URL is not set" && exit 1); \
		./bin/migrate -database-url "$(DATABASE_URL)" -command reset; \
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
