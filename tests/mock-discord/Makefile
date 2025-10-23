# Mock Discord API Server Makefile

.PHONY: help build run stop clean test logs health deps

# Container runtime detection (prefer podman for local dev, docker for CI)
CONTAINER_RUNTIME := $(shell command -v podman 2> /dev/null)
ifndef CONTAINER_RUNTIME
    CONTAINER_RUNTIME := $(shell command -v docker 2> /dev/null)
endif
ifndef CONTAINER_RUNTIME
    $(error Neither podman nor docker found. Please install one of them.)
endif

# Compose command detection
COMPOSE_CMD := $(shell command -v podman-compose 2> /dev/null)
ifndef COMPOSE_CMD
    COMPOSE_CMD := $(shell command -v docker-compose 2> /dev/null)
endif
ifndef COMPOSE_CMD
    COMPOSE_CMD := $(CONTAINER_RUNTIME) compose
endif

# Default target
help: ## Show this help message
	@echo "Mock Discord API Server"
	@echo "Container Runtime: $(CONTAINER_RUNTIME)"
	@echo "Compose Command: $(COMPOSE_CMD)"
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

# Container targets
container-build: ## Build container image
	$(CONTAINER_RUNTIME) build -t mock-discord:latest .

container-run: ## Run mock server in container
	$(COMPOSE_CMD) up -d mock-discord

container-stop: ## Stop containers
	$(COMPOSE_CMD) down

container-logs: ## Show container logs
	$(COMPOSE_CMD) logs -f mock-discord

# Docker aliases (for compatibility and CI)
docker-build: container-build ## Alias for container-build
docker-run: container-run ## Alias for container-run  
docker-stop: container-stop ## Alias for container-stop
docker-logs: container-logs ## Alias for container-logs

# Testing targets
test-env-up: ## Start test environment
	$(COMPOSE_CMD) -f docker-compose.test.yml up -d mock-discord

test-env-down: ## Stop test environment
	$(COMPOSE_CMD) -f docker-compose.test.yml down

test-acceptance: ## Run acceptance tests
	$(COMPOSE_CMD) -f docker-compose.test.yml up --build acceptance-tests

# Health and monitoring
health: ## Check server health
	@curl -s http://localhost:8080/health | jq '.' || echo "Server not responding"

status: ## Show container status
	$(COMPOSE_CMD) ps

# Cleanup targets
clean: ## Clean up build artifacts
	rm -f mock-discord
	go clean

clean-containers: ## Clean up container resources
	$(COMPOSE_CMD) down -v
	$(COMPOSE_CMD) -f docker-compose.test.yml down -v
	$(CONTAINER_RUNTIME) image prune -f

clean-docker: clean-containers ## Alias for clean-containers

# Development workflow
dev: deps build run ## Full development setup

# CI/CD targets
ci-test: container-build test-env-up ## CI test pipeline
	@echo "Waiting for mock server to be ready..."
	@timeout 30 bash -c 'until curl -s http://localhost:18080/health > /dev/null; do sleep 1; done'
	@echo "Running acceptance tests..."
	$(COMPOSE_CMD) -f docker-compose.test.yml up --build --exit-code-from acceptance-tests acceptance-tests
	$(MAKE) test-env-down

# Quick commands
up: container-run ## Alias for container-run
down: container-stop ## Alias for container-stop
restart: down up ## Restart containers