.PHONY: help install dev build test clean docker-up docker-down migrate

# Default target
help:
	@echo "Agent Observability Platform - Development Commands"
	@echo ""
	@echo "Available targets:"
	@echo "  install      - Install dependencies"
	@echo "  dev          - Run all services in development mode"
	@echo "  build        - Build all services"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-up    - Start Docker services (PostgreSQL, Redis, ClickHouse)"
	@echo "  docker-down  - Stop Docker services"
	@echo "  migrate      - Run database migrations"
	@echo ""

# Install dependencies
install:
	@echo "Installing Go dependencies..."
	cd backend && go mod download
	@echo "Installing frontend dependencies..."
	cd frontend && npm install
	@echo "Dependencies installed!"

# Run in development mode
dev-auth:
	@echo "Starting auth service..."
	cd backend/cmd/auth-service && go run .

dev-ingest:
	@echo "Starting ingest service..."
	cd backend/cmd/ingest-service && go run .

dev:
	@echo "Starting all services..."
	@echo "Run 'make dev-auth' or 'make dev-ingest' to start individual services"

# Build all services
build:
	@echo "Building backend services..."
	mkdir -p bin
	cd backend/cmd/auth-service && go build -o ../../../bin/auth-service
	cd backend/cmd/ingest-service && go build -o ../../../bin/ingest-service
	@echo "Build complete! Binaries in ./bin/"
	@echo "  - auth-service (port 8000)"
	@echo "  - ingest-service (port 8001)"

# Run tests
test:
	@echo "Running backend tests..."
	cd backend && go test ./... -v
	@echo "Tests complete!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf backend/cmd/*/auth-service
	@echo "Clean complete!"

# Start Docker services
docker-up:
	@echo "Starting Docker services..."
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Docker services started!"
	@echo "PostgreSQL: localhost:5432"
	@echo "Redis: localhost:6379"
	@echo "ClickHouse: localhost:8123"
	@echo "Adminer: http://localhost:8080"

# Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	docker-compose down
	@echo "Docker services stopped!"

# Run database migrations
migrate:
	@echo "Running database migrations..."
	@docker exec -i agent-obs-postgres psql -U postgres -d agent_obs < database/schema.sql
	@echo "Migrations complete!"

# Initialize development environment
init: docker-up
	@echo "Waiting for database to be ready..."
	@sleep 5
	@make migrate
	@echo "Environment initialized!"

# Lint code
lint:
	@echo "Running linters..."
	cd backend && golangci-lint run
	@echo "Linting complete!"
