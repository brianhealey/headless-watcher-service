.PHONY: run build test clean install help

# Variables
BINARY_NAME=sensecap-server
PORT?=8834
TOKEN?=

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install dependencies
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) ./cmd/server
	@echo "Build complete: ./$(BINARY_NAME)"

run: ## Run the application (use PORT=8080 TOKEN=xxx to override)
	@echo "Starting server on port $(PORT)..."
	@if [ -n "$(TOKEN)" ]; then \
		go run ./cmd/server -port $(PORT) -token $(TOKEN); \
	else \
		go run ./cmd/server -port $(PORT); \
	fi

run-auth: ## Run with authentication (requires TOKEN=xxx)
	@if [ -z "$(TOKEN)" ]; then \
		echo "Error: TOKEN is required. Use: make run-auth TOKEN=my-secret-token"; \
		exit 1; \
	fi
	@echo "Starting server with authentication on port $(PORT)..."
	go run ./cmd/server -port $(PORT) -token $(TOKEN)

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf dist/
	@echo "Clean complete"

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

lint: fmt vet ## Run formatters and linters

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p $(PORT):3000 -e AUTH_TOKEN=$(TOKEN) $(BINARY_NAME):latest

# Development shortcuts
dev: ## Run in development mode (no auth)
	@echo "Starting in development mode (no authentication)..."
	go run ./cmd/server -port $(PORT)

prod: ## Run in production mode (requires TOKEN)
	@if [ -z "$(TOKEN)" ]; then \
		echo "Error: TOKEN is required for production. Use: make prod TOKEN=my-secret-token"; \
		exit 1; \
	fi
	@echo "Starting in production mode..."
	go run ./cmd/server -port $(PORT) -token $(TOKEN)
