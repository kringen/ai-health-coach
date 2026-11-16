# Makefile for AI Health Coach application
# This file provides common development tasks and automation

# Variables
APP_NAME := health-coach
BINARY_NAME := health-coach
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev-$(shell date +%Y%m%d-%H%M%S)")
BUILD_TIME := $(shell date +%FT%T%z)
DOCKER_IMAGE := $(APP_NAME):$(VERSION)
DOCKER_REGISTRY := docker.io/kringen
GO_VERSION := 1.21

# Go related variables
GOBASE := $(shell pwd)
GOPATH := $(shell go env GOPATH)
GOBIN := $(GOBASE)/bin
GOFILES := $(wildcard *.go)

# Container and compose commands
CONTAINER_ENGINE ?= podman
COMPOSE_CMD ?= podman-compose
COMPOSE_FILE := docker-compose.yaml
COMPOSE_OVERRIDE := docker-compose.override.yaml
CONTAINERFILE := Dockerfile

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# Make sure GOBIN is in PATH
export PATH := $(GOBIN):$(PATH)

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
.PHONY: help
help:
	@echo "Available commands:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Build the Go application binary
.PHONY: build
build:
	@echo "$(BLUE)[INFO]$(NC) Building $(APP_NAME)..."
	@mkdir -p $(GOBIN)
	@go build -ldflags="-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" \
		-o $(GOBIN)/$(BINARY_NAME) cmd/server/main.go
	@echo "$(GREEN)[SUCCESS]$(NC) Binary built: $(GOBIN)/$(BINARY_NAME)"

## clean: Clean build artifacts and temporary files
.PHONY: clean
clean:
	@echo "$(BLUE)[INFO]$(NC) Cleaning build artifacts..."
	@rm -rf $(GOBIN)
	@rm -f coverage.out coverage.html test-report.html
	@go clean
	@$(CONTAINER_ENGINE) system prune -f
	@echo "$(GREEN)[SUCCESS]$(NC) Clean completed"

## test: Run all tests
.PHONY: test
test:
	@echo "$(BLUE)[INFO]$(NC) Running tests..."
	@./scripts/test.sh

## test-unit: Run only unit tests
.PHONY: test-unit
test-unit:
	@echo "$(BLUE)[INFO]$(NC) Running unit tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)[SUCCESS]$(NC) Unit tests completed. Coverage report: coverage.html"

## test-integration: Run integration tests
.PHONY: test-integration
test-integration:
	@echo "$(BLUE)[INFO]$(NC) Running integration tests..."
	@go test -tags=integration -v ./...

## lint: Run linting checks
.PHONY: lint
lint:
	@echo "$(BLUE)[INFO]$(NC) Running linting checks..."
	@golangci-lint run ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Linting completed"

## fmt: Format Go code
.PHONY: fmt
fmt:
	@echo "$(BLUE)[INFO]$(NC) Formatting Go code..."
	@go fmt ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Code formatted"

## vet: Run go vet
.PHONY: vet
vet:
	@echo "$(BLUE)[INFO]$(NC) Running go vet..."
	@go vet ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Go vet completed"

## deps: Download and tidy dependencies
.PHONY: deps
deps:
	@echo "$(BLUE)[INFO]$(NC) Downloading dependencies..."
	@go mod download
	@go mod tidy
	@go mod verify
	@echo "$(GREEN)[SUCCESS]$(NC) Dependencies updated"

## docker-build: Build Docker image
.PHONY: docker-build
docker-build:
	@echo "$(BLUE)[INFO]$(NC) Building Docker image: $(DOCKER_IMAGE)"
	@$(CONTAINER_ENGINE) build --file $(CONTAINERFILE) --build-arg VERSION=$(VERSION) -t $(DOCKER_IMAGE) .
	@$(CONTAINER_ENGINE) tag $(DOCKER_IMAGE) $(APP_NAME):latest
	@echo "$(GREEN)[SUCCESS]$(NC) Docker image built: $(DOCKER_IMAGE)"

## docker-push: Push Docker image to registry
.PHONY: docker-push
docker-push: docker-build
	@echo "$(BLUE)[INFO]$(NC) Pushing Docker image to registry..."
	@docker tag $(DOCKER_IMAGE) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)
	@$(CONTAINER_ENGINE) push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)
	@$(CONTAINER_ENGINE) tag $(APP_NAME):latest $(DOCKER_REGISTRY)/$(APP_NAME):latest
	@$(CONTAINER_ENGINE) push $(DOCKER_REGISTRY)/$(APP_NAME):latest
	@echo "$(GREEN)[SUCCESS]$(NC) Docker image and latest tag pushed"

## up: Start all services with Podman Compose
.PHONY: up
up:
	@echo "$(BLUE)[INFO]$(NC) Starting all services..."
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) up -d
	@echo "$(GREEN)[SUCCESS]$(NC) Services started. Access the app at http://localhost:8080"

## down: Stop all services
.PHONY: down
down:
	@echo "$(BLUE)[INFO]$(NC) Stopping all services..."
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) down
	@echo "$(GREEN)[SUCCESS]$(NC) Services stopped"

## logs: Show logs from all services
.PHONY: logs
logs:
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) logs -f

## logs-app: Show logs from the application only
.PHONY: logs-app
logs-app:
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) logs -f health-coach

## restart: Restart all services
.PHONY: restart
restart: down up

## status: Show status of all services
.PHONY: status
status:
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) ps

## db-migrate: Run database migrations
.PHONY: db-migrate
db-migrate:
	@echo "$(BLUE)[INFO]$(NC) Running database migrations..."
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) exec health-coach ./health-coach -migrate
	@echo "$(GREEN)[SUCCESS]$(NC) Database migrations completed"

## db-seed: Seed the database with sample data
.PHONY: db-seed
db-seed:
	@echo "$(BLUE)[INFO]$(NC) Seeding database with sample data..."
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) exec postgres psql -U healthtracker -d healthtracker -f /docker-entrypoint-initdb.d/seed.sql
	@echo "$(GREEN)[SUCCESS]$(NC) Database seeded"

## db-reset: Reset the database (WARNING: This will delete all data)
.PHONY: db-reset
db-reset:
	@echo "$(RED)[WARNING]$(NC) This will delete all database data. Are you sure? [y/N]"
	@read -r REPLY; \
	if [ "$$REPLY" = "y" ] || [ "$$REPLY" = "Y" ]; then \
		echo "$(BLUE)[INFO]$(NC) Resetting database..."; \
		$(COMPOSE_CMD) -f $(COMPOSE_FILE) down -v; \
		$(COMPOSE_CMD) -f $(COMPOSE_FILE) up -d postgres; \
		sleep 10; \
		$(COMPOSE_CMD) -f $(COMPOSE_FILE) up -d; \
		echo "$(GREEN)[SUCCESS]$(NC) Database reset completed"; \
	else \
		echo "$(YELLOW)[INFO]$(NC) Database reset cancelled"; \
	fi

## dev: Start development environment
.PHONY: dev
dev:
	@echo "$(BLUE)[INFO]$(NC) Starting development environment..."
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) up -d postgres
	@sleep 5
	@echo "$(GREEN)[SUCCESS]$(NC) Development services started"
	@echo "$(BLUE)[INFO]$(NC) Starting application in development mode..."
	@export ENVIRONMENT=development DB_HOST=localhost OLLAMA_BASE_URL=http://localhost:11434 && \
		go run cmd/server/main.go

## dev-tools: Start development tools (adminer, etc.)
.PHONY: dev-tools
dev-tools:
	@echo "$(BLUE)[INFO]$(NC) Starting development tools..."
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) --profile dev-tools up -d
	@echo "$(GREEN)[SUCCESS]$(NC) Development tools started:"
	@echo "  - Adminer: http://localhost:8081"

## monitoring: Start monitoring stack
.PHONY: monitoring
monitoring:
	@echo "$(BLUE)[INFO]$(NC) Starting monitoring stack..."
	@$(COMPOSE_CMD) -f $(COMPOSE_FILE) --profile monitoring up -d
	@echo "$(GREEN)[SUCCESS]$(NC) Monitoring stack started:"
	@echo "  - Prometheus: http://localhost:9090"
	@echo "  - Grafana: http://localhost:3000 (admin/admin)"

## deploy: Deploy to Kubernetes
.PHONY: deploy
deploy:
	@echo "$(BLUE)[INFO]$(NC) Deploying to Kubernetes..."
	@./scripts/deploy.sh
	@echo "$(GREEN)[SUCCESS]$(NC) Deployment completed"

## k8s-apply: Apply Kubernetes manifests (base)
.PHONY: k8s-apply
k8s-apply:
	@echo "$(BLUE)[INFO]$(NC) Applying Kubernetes manifests..."
	@kubectl apply -k k8s/base
	@echo "$(GREEN)[SUCCESS]$(NC) Kubernetes manifests applied"

## k8s-delete: Delete Kubernetes resources
.PHONY: k8s-delete
k8s-delete:
	@echo "$(BLUE)[INFO]$(NC) Deleting Kubernetes resources..."
	@kubectl delete -k k8s/base
	@echo "$(GREEN)[SUCCESS]$(NC) Kubernetes resources deleted"

## security: Run security checks
.PHONY: security
security:
	@echo "$(BLUE)[INFO]$(NC) Running security checks..."
	@gosec ./...
	@govulncheck ./...
	@echo "$(GREEN)[SUCCESS]$(NC) Security checks completed"

## install-tools: Install development tools
.PHONY: install-tools
install-tools:
	@echo "$(BLUE)[INFO]$(NC) Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "$(GREEN)[SUCCESS]$(NC) Development tools installed"

## benchmark: Run benchmark tests
.PHONY: benchmark
benchmark:
	@echo "$(BLUE)[INFO]$(NC) Running benchmark tests..."
	@go test -bench=. -benchmem ./...

## coverage: Generate and view coverage report
.PHONY: coverage
coverage: test-unit
	@echo "$(BLUE)[INFO]$(NC) Opening coverage report..."
	@which open >/dev/null 2>&1 && open coverage.html || \
	 which xdg-open >/dev/null 2>&1 && xdg-open coverage.html || \
	 echo "$(YELLOW)[INFO]$(NC) Coverage report: coverage.html"

## release: Create a new release
.PHONY: release
release:
	@echo "$(BLUE)[INFO]$(NC) Creating release $(VERSION)..."
	@./scripts/release.sh $(VERSION)

## check: Run all checks (formatting, linting, tests)
.PHONY: check
check: fmt vet lint test security
	@echo "$(GREEN)[SUCCESS]$(NC) All checks passed!"

## all: Build, test, and create Docker image
.PHONY: all
all: clean deps check build docker-build
	@echo "$(GREEN)[SUCCESS]$(NC) Build pipeline completed!"

# Development workflow targets
## quick: Quick development build and test
.PHONY: quick
quick: fmt build test-unit
	@echo "$(GREEN)[SUCCESS]$(NC) Quick check completed!"

## full: Full development workflow
.PHONY: full
full: clean deps check build docker-build
	@echo "$(GREEN)[SUCCESS]$(NC) Full workflow completed!"

# Environment information
## info: Show environment information
.PHONY: info
info:
	@echo "$(BLUE)[INFO]$(NC) Environment Information:"
	@echo "  Go Version: $(shell go version)"
	@echo "  App Name: $(APP_NAME)"
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Container Image: $(DOCKER_IMAGE)"
	@echo "  Container Engine: $(CONTAINER_ENGINE)"
	@echo "  Compose Command: $(COMPOSE_CMD)"
	@echo "  Registry: $(DOCKER_REGISTRY)"
	@echo "  GOPATH: $(GOPATH)"
	@echo "  GOBIN: $(GOBIN)"

# Pre-commit hooks
## pre-commit: Run pre-commit checks
.PHONY: pre-commit
pre-commit: fmt vet lint test-unit
	@echo "$(GREEN)[SUCCESS]$(NC) Pre-commit checks passed!"

# Make scripts executable
$(shell chmod +x scripts/*.sh)
