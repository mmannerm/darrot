# Mock Discord API Server Makefile

.PHONY: help build run stop clean test logs health deps

# Default target
help: ## Show this help message
	@echo "Mock Discord API Server"
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development targets
deps: ## Download Go dependencies
	go mod download
	go mod tidy

build: ## Build the mock server binary
	go build -o mock-discord .

run: ## Run the mock server locally
	go run .

test: ## Run unit tests
	go test -v ./...

# Docker targets
docker-build: ## Build Docker image
	docker build -t mock-discord:latest .

docker-run: ## Run mock server in Docker
	docker-compose up -d mock-discord

docker-stop: ## Stop Docker containers
	docker-compose down

docker-logs: ## Show Docker container logs
	docker-compose logs -f mock-discord

# Testing targets
test-env-up: ## Start test environment
	docker-compose -f docker-compose.test.yml up -d mock-discord

test-env-down: ## Stop test environment
	docker-compose -f docker-compose.test.yml down

test-acceptance: ## Run acceptance tests
	docker-compose -f docker-compose.test.yml up --build acceptance-tests

# Health and monitoring
health: ## Check server health
	@curl -s http://localhost:8080/health | jq '.' || echo "Server not responding"

status: ## Show container status
	docker-compose ps

# Cleanup targets
clean: ## Clean up build artifacts
	rm -f mock-discord
	go clean

clean-docker: ## Clean up Docker resources
	docker-compose down -v
	docker-compose -f docker-compose.test.yml down -v
	docker image prune -f

# Development workflow
dev: deps build run ## Full development setup

# CI/CD targets
ci-test: docker-build test-env-up ## CI test pipeline
	@echo "Waiting for mock server to be ready..."
	@timeout 30 bash -c 'until curl -s http://localhost:18080/health > /dev/null; do sleep 1; done'
	@echo "Running acceptance tests..."
	docker-compose -f docker-compose.test.yml up --build --exit-code-from acceptance-tests acceptance-tests
	$(MAKE) test-env-down

# Quick commands
up: docker-run ## Alias for docker-run
down: docker-stop ## Alias for docker-stop
restart: down up ## Restart containers