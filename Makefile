.PHONY: run build test clean docker-up docker-down migrate seed swagger

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

# Start PostgreSQL container
docker-up:
	docker-compose up -d

# Stop PostgreSQL container
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

