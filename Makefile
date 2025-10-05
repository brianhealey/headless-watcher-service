.PHONY: run build test clean install help download-models

# Variables
BINARY_NAME=sensecap-server
PORT?=8834
TOKEN?=
PIPER_MODEL_DIR=internal/models/piper

# Piper voice configuration (override with PIPER_VOICE=en_US-amy-medium make download-models)
PIPER_VOICE ?= en_US-lessac-medium

# Parse voice config: en_US-lessac-medium -> en/en_US/lessac/medium
VOICE_PARTS = $(subst -, ,$(PIPER_VOICE))
PIPER_LANG := $(word 1,$(VOICE_PARTS))
PIPER_VOICE_NAME := $(word 2,$(VOICE_PARTS))
PIPER_QUALITY := $(word 3,$(VOICE_PARTS))
PIPER_LOCALE := $(word 1,$(subst _, ,$(PIPER_LANG)))
PIPER_MODEL_URL=https://huggingface.co/rhasspy/piper-voices/resolve/v1.0.0/$(PIPER_LOCALE)/$(PIPER_LANG)/$(PIPER_VOICE_NAME)/$(PIPER_QUALITY)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

download-models: ## Download Piper TTS model (use PIPER_VOICE=en_US-amy-medium to change voice)
	@echo "Downloading Piper TTS model: $(PIPER_VOICE)..."
	@echo "URL: $(PIPER_MODEL_URL)"
	@mkdir -p $(PIPER_MODEL_DIR)
	@if [ ! -f "$(PIPER_MODEL_DIR)/$(PIPER_VOICE).onnx" ]; then \
		echo "Downloading $(PIPER_VOICE).onnx..."; \
		curl -L -o $(PIPER_MODEL_DIR)/$(PIPER_VOICE).onnx $(PIPER_MODEL_URL)/$(PIPER_VOICE).onnx; \
	else \
		echo "Model already exists: $(PIPER_MODEL_DIR)/$(PIPER_VOICE).onnx"; \
	fi
	@if [ ! -f "$(PIPER_MODEL_DIR)/$(PIPER_VOICE).onnx.json" ]; then \
		echo "Downloading $(PIPER_VOICE).onnx.json..."; \
		curl -L -o $(PIPER_MODEL_DIR)/$(PIPER_VOICE).onnx.json $(PIPER_MODEL_URL)/$(PIPER_VOICE).onnx.json; \
	else \
		echo "Model config already exists: $(PIPER_MODEL_DIR)/$(PIPER_VOICE).onnx.json"; \
	fi
	@echo "Model download complete"

install: download-models ## Install dependencies and download models
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
