.PHONY: build clean test run validate health docker-build docker-run

# Application name
APP_NAME := enhanced-flex-monitor
VERSION := 1.0.0

# Build the application
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(APP_NAME) .

# Clean build artifacts
clean:
	rm -f $(APP_NAME)
	go clean

# Download dependencies
deps:
	go mod tidy
	go mod download

# Run tests
test:
	go test -v ./...

# Run with default config
run: build
	./$(APP_NAME) -config config.yml

# Validate configuration
validate: build
	./$(APP_NAME) -config config.yml -validate

# Run health check
health: build
	./$(APP_NAME) -config config.yml -health

# Test alerts
test-alerts: build
	./$(APP_NAME) -config config.yml -test-alerts

# Show version
version: build
	./$(APP_NAME) -version

# Build Docker image
docker-build:
	docker build -t $(APP_NAME):$(VERSION) .
	docker tag $(APP_NAME):$(VERSION) $(APP_NAME):latest

# Run in Docker
docker-run:
	docker run --rm \
		-e NEW_RELIC_API_KEY="${NEW_RELIC_API_KEY}" \
		-e NEW_RELIC_ACCOUNT_ID="${NEW_RELIC_ACCOUNT_ID}" \
		$(APP_NAME):latest

# Development workflow
dev: deps build validate

# Release workflow
release: clean deps build test validate

# Help
help:
	@echo "Available commands:"
	@echo "  build        - Build the application"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Download dependencies"
	@echo "  test         - Run tests"
	@echo "  run          - Run with default config"
	@echo "  validate     - Validate configuration"
	@echo "  health       - Run health check"
	@echo "  test-alerts  - Test alert channels"
	@echo "  version      - Show version"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run in Docker"
	@echo "  dev          - Development workflow"
	@echo "  release      - Release workflow"