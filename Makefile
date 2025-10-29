# Makefile for darrot Discord TTS bot

# Variables
BINARY_NAME=darrot
CONTAINER_NAME=darrot:test
MOCK_DISCORD_IMAGE=mock-discord:test
ACCEPTANCE_TEST_IMAGE=acceptance-tests:test
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Container runtime (podman or docker)
CONTAINER_RUNTIME?=podman

# Build flags
LDFLAGS=-ldflags="-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Directories and files
SRC_FILES := $(shell find . -name '*.go' -not -path './tests/*' -not -path './.git/*')
TEST_SRC_FILES := $(shell find tests -name '*.go' 2>/dev/null || true)
DOCKER_FILES := Dockerfile tests/mock-discord/Dockerfile tests/acceptance/Dockerfile.test
CONFIG_FILES := go.mod go.sum
BUILD_DEPS := $(SRC_FILES) $(CONFIG_FILES) Dockerfile

# Test configuration
CONTAINER_TEST_CONFIG := tests/container/structure-test.yaml
RESULTS_DIR := tests/results
ACCEPTANCE_RESULTS_DIR := tests/acceptance/test-results

# Timestamp files for tracking builds
BUILD_STAMPS_DIR := .build-stamps
CONTAINER_STAMP := $(BUILD_STAMPS_DIR)/container.stamp
MOCK_DISCORD_STAMP := $(BUILD_STAMPS_DIR)/mock-discord.stamp
ACCEPTANCE_TEST_STAMP := $(BUILD_STAMPS_DIR)/acceptance-test.stamp
CONTAINER_TEST_TOOL_STAMP := $(BUILD_STAMPS_DIR)/container-test-tool.stamp

# Default target
.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# Create build stamps directory
$(BUILD_STAMPS_DIR):
	@mkdir -p $(BUILD_STAMPS_DIR)

# Development targets
.PHONY: build
build: ## Build the application binary
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/darrot

.PHONY: test
test: ## Run all tests
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test ## Run tests and generate coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: lint
lint: ## Run linting tools
	go fmt ./...
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; else echo "golangci-lint not found, skipping"; fi

.PHONY: clean
clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf $(BUILD_STAMPS_DIR)
	rm -rf $(RESULTS_DIR)
	rm -rf $(ACCEPTANCE_RESULTS_DIR)
	$(CONTAINER_RUNTIME) rmi $(CONTAINER_NAME) $(MOCK_DISCORD_IMAGE) $(ACCEPTANCE_TEST_IMAGE) 2>/dev/null || true

# Container build targets with dependency tracking
$(CONTAINER_STAMP): $(BUILD_DEPS) | $(BUILD_STAMPS_DIR)
	@echo "Building container image: $(CONTAINER_NAME)"
	$(CONTAINER_RUNTIME) build -t $(CONTAINER_NAME) .
	@touch $(CONTAINER_STAMP)

$(MOCK_DISCORD_STAMP): tests/mock-discord/Dockerfile $(shell find tests/mock-discord -name '*.go' 2>/dev/null || true) | $(BUILD_STAMPS_DIR)
	@echo "Building mock Discord server image: $(MOCK_DISCORD_IMAGE)"
	$(CONTAINER_RUNTIME) build -t $(MOCK_DISCORD_IMAGE) -f tests/mock-discord/Dockerfile tests/mock-discord/
	@touch $(MOCK_DISCORD_STAMP)

$(ACCEPTANCE_TEST_STAMP): tests/acceptance/Dockerfile.test $(TEST_SRC_FILES) | $(BUILD_STAMPS_DIR)
	@echo "Building acceptance test image: $(ACCEPTANCE_TEST_IMAGE)"
	$(CONTAINER_RUNTIME) build -t $(ACCEPTANCE_TEST_IMAGE) -f tests/acceptance/Dockerfile.test tests/acceptance/
	@touch $(ACCEPTANCE_TEST_STAMP)

$(CONTAINER_TEST_TOOL_STAMP): | $(BUILD_STAMPS_DIR)
	@echo "Checking container-structure-test installation..."
	@if ! command -v container-structure-test >/dev/null 2>&1; then \
		echo "Installing container-structure-test..."; \
		./scripts/install-container-structure-test.sh; \
	fi
	@touch $(CONTAINER_TEST_TOOL_STAMP)

# Container targets
.PHONY: container-build
container-build: $(CONTAINER_STAMP) ## Build container image (with dependency tracking)

.PHONY: container-build-force
container-build-force: ## Force rebuild container image
	@rm -f $(CONTAINER_STAMP)
	@$(MAKE) container-build

.PHONY: container-test
container-test: $(CONTAINER_STAMP) $(CONTAINER_TEST_TOOL_STAMP) ## Run container structure tests
	@CONTAINER_IMAGE=$(CONTAINER_NAME) \
	 TEST_CONFIG=$(CONTAINER_TEST_CONFIG) \
	 RESULTS_DIR=$(RESULTS_DIR) \
	 CONTAINER_RUNTIME=$(CONTAINER_RUNTIME) \
	 ./scripts/container-test-simple.sh

.PHONY: container-test-quick
container-test-quick: $(CONTAINER_STAMP) ## Run quick container validation tests
	@CONTAINER_IMAGE=$(CONTAINER_NAME) \
	 CONTAINER_RUNTIME=$(CONTAINER_RUNTIME) \
	 ./scripts/test-container-quick.sh

.PHONY: container-test-install
container-test-install: $(CONTAINER_TEST_TOOL_STAMP) ## Install container-structure-test tool

.PHONY: container-run
container-run: $(CONTAINER_STAMP) ## Run container locally
	$(CONTAINER_RUNTIME) run --rm -it \
		-v $(PWD)/data:/app/data \
		-v $(PWD)/darrot-config.yaml:/app/darrot-config.yaml:ro \
		$(CONTAINER_NAME)

.PHONY: container-shell
container-shell: $(CONTAINER_STAMP) ## Get shell access to container
	$(CONTAINER_RUNTIME) run --rm -it --entrypoint /bin/sh $(CONTAINER_NAME)

# Acceptance testing targets
.PHONY: acceptance-build
acceptance-build: $(CONTAINER_STAMP) $(MOCK_DISCORD_STAMP) $(ACCEPTANCE_TEST_STAMP) ## Build all acceptance test images

.PHONY: acceptance-test
acceptance-test: acceptance-build ## Run acceptance tests
	@echo "Running acceptance tests..."
	@mkdir -p $(ACCEPTANCE_RESULTS_DIR)
	@PROJECT_NAME="darrot-acceptance-tests-$$(date +%s)"; \
	COMPOSE_FILE="docker-compose.test.yml"; \
	COMPOSE_CMD="docker-compose"; \
	if [ "$(CONTAINER_RUNTIME)" = "podman" ]; then \
		if command -v podman-compose >/dev/null 2>&1; then \
			COMPOSE_CMD="podman-compose"; \
			COMPOSE_FILE="podman-compose.test.yml"; \
		else \
			export DOCKER_HOST="unix:///run/user/$$(id -u)/podman/podman.sock"; \
		fi; \
	fi; \
	cd tests/acceptance && \
	$$COMPOSE_CMD -f $$COMPOSE_FILE -p $$PROJECT_NAME up -d mock-discord && \
	sleep 5 && \
	$$COMPOSE_CMD -f $$COMPOSE_FILE -p $$PROJECT_NAME up -d darrot-bot && \
	sleep 10 && \
	$$COMPOSE_CMD -f $$COMPOSE_FILE -p $$PROJECT_NAME run --rm \
		-e TEST_SUITE=all \
		-e TEST_TIMEOUT=300s \
		acceptance-tests; \
	TEST_RESULT=$$?; \
	$$COMPOSE_CMD -f $$COMPOSE_FILE -p $$PROJECT_NAME down -v --remove-orphans 2>/dev/null || true; \
	exit $$TEST_RESULT

.PHONY: acceptance-test-core
acceptance-test-core: acceptance-build ## Run core functionality acceptance tests
	@$(MAKE) acceptance-test TEST_SUITE=core

.PHONY: acceptance-test-concurrent
acceptance-test-concurrent: acceptance-build ## Run concurrent acceptance tests
	@$(MAKE) acceptance-test TEST_SUITE=concurrent

.PHONY: acceptance-test-error
acceptance-test-error: acceptance-build ## Run error resilience acceptance tests
	@$(MAKE) acceptance-test TEST_SUITE=error

.PHONY: acceptance-clean
acceptance-clean: ## Clean up acceptance test containers and volumes
	@echo "Cleaning up acceptance test containers..."
	@for runtime in podman docker; do \
		if command -v $$runtime >/dev/null 2>&1; then \
			$$runtime ps -a --filter "name=darrot-acceptance" -q | xargs -r $$runtime rm -f; \
			$$runtime volume ls --filter "name=darrot-acceptance" -q | xargs -r $$runtime volume rm; \
		fi; \
	done

# Combined targets
.PHONY: all
all: lint test container-test ## Run all checks (lint, test, container-test)

.PHONY: ci
ci: lint test container-build container-test ## Run CI pipeline locally

.PHONY: test-all
test-all: test container-test acceptance-test ## Run all tests (unit, container, acceptance)

# Development workflow
.PHONY: dev-setup
dev-setup: container-test-install ## Set up development environment
	go mod download
	@echo "Development environment ready!"

.PHONY: pre-commit
pre-commit: lint test ## Run pre-commit checks
	@echo "Pre-commit checks passed!"

# Debug and maintenance targets
.PHONY: show-deps
show-deps: ## Show build dependencies
	@echo "Source files that trigger container rebuild:"
	@echo "$(SRC_FILES)" | tr ' ' '\n'
	@echo ""
	@echo "Test files that trigger test image rebuild:"
	@echo "$(TEST_SRC_FILES)" | tr ' ' '\n'

.PHONY: clean-stamps
clean-stamps: ## Clean build stamps (force rebuild on next make)
	rm -rf $(BUILD_STAMPS_DIR)

.PHONY: status
status: ## Show build status
	@echo "Build Status:"
	@echo "============="
	@if [ -f "$(CONTAINER_STAMP)" ]; then \
		echo "✓ Container image: built ($(shell stat -c %y $(CONTAINER_STAMP) 2>/dev/null || stat -f %Sm $(CONTAINER_STAMP) 2>/dev/null || echo 'unknown'))"; \
	else \
		echo "✗ Container image: not built"; \
	fi
	@if [ -f "$(MOCK_DISCORD_STAMP)" ]; then \
		echo "✓ Mock Discord image: built ($(shell stat -c %y $(MOCK_DISCORD_STAMP) 2>/dev/null || stat -f %Sm $(MOCK_DISCORD_STAMP) 2>/dev/null || echo 'unknown'))"; \
	else \
		echo "✗ Mock Discord image: not built"; \
	fi
	@if [ -f "$(ACCEPTANCE_TEST_STAMP)" ]; then \
		echo "✓ Acceptance test image: built ($(shell stat -c %y $(ACCEPTANCE_TEST_STAMP) 2>/dev/null || stat -f %Sm $(ACCEPTANCE_TEST_STAMP) 2>/dev/null || echo 'unknown'))"; \
	else \
		echo "✗ Acceptance test image: not built"; \
	fi
	@if [ -f "$(CONTAINER_TEST_TOOL_STAMP)" ]; then \
		echo "✓ Container test tool: installed"; \
	else \
		echo "✗ Container test tool: not installed"; \
	fi