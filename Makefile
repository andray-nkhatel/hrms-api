.PHONY: run build test clean docker-up docker-down migrate seed swagger docker-build docker-up-prod docker-down-prod docker-logs docker-restart

# Run the application
run:
	go run main.go

# Generate Swagger documentation
swagger:
	swag init

# Build the application
build:
	go build -o bin/hrms-api main.go

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Start PostgreSQL container (development)
docker-up:
	docker-compose up -d

# Stop PostgreSQL container (development)
docker-down:
	docker-compose down

# Run database migrations (automatic on startup)
migrate:
	@echo "Migrations run automatically on application startup"

# Seed database (automatic on startup)
seed:
	@echo "Seed data is created automatically on application startup"

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/"; \
	fi

# Docker Production Commands

# Build Docker image
docker-build:
	docker-compose -f docker-compose.prod.yml build

# Start production services
docker-up-prod:
	docker-compose -f docker-compose.prod.yml up -d

# Stop production services
docker-down-prod:
	docker-compose -f docker-compose.prod.yml down

# View production logs
docker-logs:
	docker-compose -f docker-compose.prod.yml logs -f

# Restart production services
docker-restart:
	docker-compose -f docker-compose.prod.yml restart

# Rebuild and restart production services
docker-rebuild:
	docker-compose -f docker-compose.prod.yml up -d --build

# Stop and remove volumes (⚠️ deletes database)
docker-clean:
	docker-compose -f docker-compose.prod.yml down -v

# Show production service status
docker-ps:
	docker-compose -f docker-compose.prod.yml ps

# Quick start (uses startup script)
docker-start:
	./docker-start.sh

# Docker Production Commands

# Build Docker image
docker-build:
	docker-compose -f docker-compose.prod.yml build

# Start production services
docker-up-prod:
	docker-compose -f docker-compose.prod.yml up -d

# Stop production services
docker-down-prod:
	docker-compose -f docker-compose.prod.yml down

# View production logs
docker-logs:
	docker-compose -f docker-compose.prod.yml logs -f

# Restart production services
docker-restart:
	docker-compose -f docker-compose.prod.yml restart

# Rebuild and restart production services
docker-rebuild:
	docker-compose -f docker-compose.prod.yml up -d --build

# Stop and remove volumes (⚠️ deletes database)
docker-clean:
	docker-compose -f docker-compose.prod.yml down -v

# Show production service status
docker-ps:
	docker-compose -f docker-compose.prod.yml ps

# Quick start (uses startup script)
docker-start:
	./docker-start.sh
