.PHONY: help lint test build check clean install-tools
.PHONY: lint-go lint-python format-python test-go test-python build-go
.PHONY: docker-build docker-up docker-down

# Default target
.DEFAULT_GOAL := help

##@ General

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

check: lint test build ## Run all checks (lint + test + build)
	@echo "✅ All checks passed!"

lint: lint-go lint-python ## Run all linters

test: test-go test-python ## Run all tests

build: build-go ## Build all services

clean: ## Clean build artifacts and caches
	@echo "🧹 Cleaning build artifacts..."
	@rm -f go-service/orbitstream
	@find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	@find . -type d -name "*.egg-info" -exec rm -rf {} + 2>/dev/null || true
	@find . -type f -name "*.pyc" -delete 2>/dev/null || true
	@echo "✅ Clean complete"

##@ Go Service

lint-go: ## Lint Go code with golangci-lint
	@echo "🔍 Linting Go code..."
	@cd go-service && golangci-lint run --timeout=5m
	@echo "✅ Go linting passed"

test-go: ## Run Go tests
	@echo "🧪 Running Go tests..."
	@cd go-service && go test ./... -v -cover -short
	@echo "✅ Go tests passed"

build-go: ## Build Go service
	@echo "🔨 Building Go service..."
	@cd go-service && go build -o orbitstream .
	@echo "✅ Go build complete: go-service/orbitstream"

##@ Python Simulator

lint-python: ## Lint Python code with ruff and black
	@echo "🔍 Linting Python code..."
	@cd python-simulator && ruff check .
	@cd python-simulator && black --check .
	@echo "✅ Python linting passed"

format-python: ## Auto-format Python code with black
	@echo "🎨 Formatting Python code..."
	@cd python-simulator && black .
	@cd python-simulator && ruff check --fix .
	@echo "✅ Python formatting complete"

test-python: ## Run Python tests
	@echo "🧪 Running Python tests..."
	@cd python-simulator && PYTHONPATH=$$(pwd) pytest -v
	@echo "✅ Python tests passed"

##@ Docker

docker-build: ## Build Docker images
	@echo "🐳 Building Docker images..."
	@docker compose build
	@echo "✅ Docker images built"

docker-up: ## Start all services with Docker Compose
	@echo "🚀 Starting services..."
	@docker compose up -d
	@echo "✅ Services started"

docker-down: ## Stop all services
	@echo "🛑 Stopping services..."
	@docker compose down
	@echo "✅ Services stopped"

##@ Tools Installation

install-tools: ## Install development tools (golangci-lint, ruff, black, pre-commit)
	@echo "📦 Installing development tools..."
	@echo "Installing golangci-lint..."
	@command -v golangci-lint >/dev/null 2>&1 || \
		(curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin)
	@echo "Installing Python tools..."
	@pip install --upgrade ruff black pytest pre-commit
	@echo "✅ All tools installed"
	@echo ""
	@echo "To enable pre-commit hooks, run:"
	@echo "  pre-commit install"
