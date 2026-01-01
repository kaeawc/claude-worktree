# Makefile for auto-worktree
# Provides convenient commands for local development

# Binary name
BINARY_NAME=auto-worktree

# Build directory
BUILD_DIR=build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Main package path
MAIN_PACKAGE=./cmd/$(BINARY_NAME)

# Installation path
INSTALL_PATH=$(shell go env GOPATH)/bin

# Build flags
LDFLAGS=-ldflags "-s -w"

# Build targets for different platforms
PLATFORMS=linux darwin windows
ARCHITECTURES=amd64 arm64

.PHONY: all build test lint fmt vet clean install help coverage staticcheck \
        build-all build-linux build-darwin build-windows deps tidy verify

# Default target
all: test build

help:
	@echo "Makefile for auto-worktree"
	@echo ""
	@echo "Usage:"
	@echo "  make build          Build the binary for current platform"
	@echo "  make test           Run tests"
	@echo "  make test-verbose   Run tests with verbose output"
	@echo "  make coverage       Run tests with coverage report"
	@echo "  make lint           Run golangci-lint"
	@echo "  make fmt            Format code with gofmt"
	@echo "  make vet            Run go vet"
	@echo "  make staticcheck    Run staticcheck"
	@echo "  make install        Install binary to GOPATH/bin"
	@echo "  make clean          Remove build artifacts"
	@echo "  make deps           Download dependencies"
	@echo "  make tidy           Tidy and verify dependencies"
	@echo "  make verify         Verify dependencies"
	@echo "  make build-all      Build for all platforms"
	@echo "  make ci             Run all CI checks locally"
	@echo ""

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for all platforms and architectures
build-all: $(PLATFORMS)

$(PLATFORMS):
	@echo "Building for $@..."
	@mkdir -p $(BUILD_DIR)/$@
	@for arch in $(ARCHITECTURES); do \
		if [ "$@" = "windows" ] && [ "$$arch" = "arm64" ]; then \
			echo "Skipping windows/arm64 (limited support)"; \
			continue; \
		fi; \
		echo "Building $@/$$arch..."; \
		output="$(BUILD_DIR)/$@/$(BINARY_NAME)-$$arch"; \
		if [ "$@" = "windows" ]; then \
			output="$$output.exe"; \
		fi; \
		GOOS=$@ GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $$output $(MAIN_PACKAGE); \
	done
	@echo "Build complete for $@"

# Convenience targets for specific platforms
build-linux:
	@$(MAKE) linux

build-darwin:
	@$(MAKE) darwin

build-windows:
	@$(MAKE) windows

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	$(GOTEST) -v -race -cover ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	$(GOTEST) -v -race -coverprofile=$(BUILD_DIR)/coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report: $(BUILD_DIR)/coverage.html"

# Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not found. Install with:"; \
		echo "  brew install golangci-lint"; \
		echo "  or go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run staticcheck
staticcheck:
	@echo "Running staticcheck..."
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not found. Install with:"; \
		echo "  go install honnef.co/go/tools/cmd/staticcheck@latest"; \
		exit 1; \
	fi

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Verify dependencies
verify:
	@echo "Verifying dependencies..."
	$(GOMOD) verify

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@mkdir -p $(INSTALL_PATH)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installed to $(INSTALL_PATH)/$(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Run all CI checks locally
ci: deps verify vet staticcheck lint test
	@echo "All CI checks passed!"
