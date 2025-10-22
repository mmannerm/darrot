# Makefile for darrot Discord TTS bot

# Variables
BINARY_NAME=darrot
CONTAINER_NAME=darrot:test
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Container runtime (podman or docker)
CONTAINER_RUNTIME?=podman

# Build flags
LDFLAGS=-ldflags="-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Default target
.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

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
	golangci-lint run

.PHONY: clean
clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	$(CONTAINER_RUNTIME) rmi $(CONTAINER_NAME) 2>/dev/null || true

# Container targets
.PHONY: container-build
container-build: ## Build container image
	$(CONTAINER_RUNTIME) build -t $(CONTAINER_NAME) .

.PHONY: container-test
container-test: container-build ## Run container structure tests
	@echo "Running container structure tests..."
	@CONTAINER_RUNTIME=$(CONTAINER_RUNTIME) ./scripts/run-container-tests.sh

.PHONY: container-test-quick
container-test-quick: ## Run quick container validation tests
	@echo "Running quick container tests..."
	@CONTAINER_RUNTIME=$(CONTAINER_RUNTIME) ./scripts/test-container-quick.sh

.PHONY: container-test-install
container-test-install: ## Install container-structure-test tool
	@echo "Installing container-structure-test..."
	@./scripts/install-container-structure-test.sh

.PHONY: container-run
container-run: container-build ## Run container locally
	$(CONTAINER_RUNTIME) run --rm -it \
		-v $(PWD)/data:/app/data \
		-v $(PWD)/darrot-config.yaml:/app/darrot-config.yaml:ro \
		$(CONTAINER_NAME)

.PHONY: container-shell
container-shell: container-build ## Get shell access to container
	$(CONTAINER_RUNTIME) run --rm -it --entrypoint /bin/sh $(CONTAINER_NAME)

# Combined targets
.PHONY: all
all: lint test container-test ## Run all checks (lint, test, container-test)

.PHONY: ci
ci: lint test container-build container-test ## Run CI pipeline locally

# Development workflow
.PHONY: dev-setup
dev-setup: container-test-install ## Set up development environment
	go mod download
	@echo "Development environment ready!"

.PHONY: pre-commit
pre-commit: lint test ## Run pre-commit checks
	@echo "Pre-commit checks passed!"