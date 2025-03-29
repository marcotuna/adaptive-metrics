# Adaptive Metrics Makefile

# Variables
APP_NAME := adaptive-metrics
BUILD_DIR := build
MAIN_PATH := ./adaptive-metrics
CONFIG_PATH := ./configs/config.yaml
GO_FILES := $(shell find . -name "*.go" -not -path "./vendor/*")
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Go related variables
GOPATH := $(shell go env GOPATH)
GO := go

.PHONY: all build clean run test test-coverage lint fmt vet docker-build docker-run help

# Default target
all: clean build

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Build complete. Binary located at: $(BUILD_DIR)/$(APP_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "Clean complete"

# Run the application
run:
	@echo "Starting $(APP_NAME)..."
	@$(GO) run $(MAIN_PATH) --config $(CONFIG_PATH)

# Run with hot reload (requires 'air' - https://github.com/cosmtrek/air)
dev:
	@command -v air > /dev/null 2>&1 || (echo "air is not installed. Installing..." && go install github.com/cosmtrek/air@latest)
	@echo "Starting $(APP_NAME) in development mode..."
	@air -c .air.toml

# Run all tests
test:
	@echo "Running tests..."
	@$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) test -coverprofile=$(BUILD_DIR)/coverage.out ./...
	@$(GO) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report generated at $(BUILD_DIR)/coverage.html"

# Run linter (requires golangci-lint)
lint:
	@command -v golangci-lint > /dev/null 2>&1 || (echo "golangci-lint is not installed. Installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@echo "Running linter..."
	@golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@$(GO) vet ./...

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(APP_NAME):$(VERSION) .
	@echo "Docker image built: $(APP_NAME):$(VERSION)"

# Run with Docker
docker-run:
	@echo "Running in Docker..."
	@docker run -p 8080:8080 --name $(APP_NAME) -v $(PWD)/configs:/app/configs $(APP_NAME):$(VERSION)

# Generate code (mocks, etc.)
generate:
	@echo "Generating code..."
	@$(GO) generate ./...

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@$(GO) mod download
	@$(GO) mod tidy

# Initialize config directory if not exists
init-config:
	@echo "Initializing config directory..."
	@mkdir -p configs/rules
	@[ -f $(CONFIG_PATH) ] || cp configs/config.yaml.example $(CONFIG_PATH) || echo "No config example found, skipping"

# Help command
help:
	@echo "Available commands:"
	@echo "  make all            - Clean and build the application"
	@echo "  make build          - Build the application"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make run            - Run the application"
	@echo "  make dev            - Run with hot reload (requires 'air')"
	@echo "  make test           - Run all tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make lint           - Run linter (requires golangci-lint)"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run with Docker"
	@echo "  make generate       - Generate code (mocks, etc.)"
	@echo "  make deps           - Install dependencies"
	@echo "  make init-config    - Initialize config directory"
	@echo "  make help           - Show this help message"