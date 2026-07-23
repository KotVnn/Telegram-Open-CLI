.PHONY: build clean test lint fmt vet run install help

# Build variables
BINARY_NAME=toc
BUILD_DIR=./bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go variables
GO=go
GOFLAGS=-trimpath
LDFLAGS=-ldflags "-s -w -X 'github.com/KotVnn/Telegram-Open-CLI/pkg/version.Version=$(VERSION)' -X 'github.com/KotVnn/Telegram-Open-CLI/pkg/version.Commit=$(COMMIT)' -X 'github.com/KotVnn/Telegram-Open-CLI/pkg/version.BuildTime=$(BUILD_TIME)'"

# Default target
help: ## Show this help
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build

build: ## Build the binary
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)$(shell if [ "$(OS)" = "Windows_NT" ]; then echo ".exe"; fi) ./cmd/toc

build-all: ## Build for all platforms
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/toc
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/toc
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/toc
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/toc
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/toc
	GOOS=windows GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe ./cmd/toc

# Development

run: ## Run the application
	$(GO) run ./cmd/toc start

run-init: ## Run init command
	$(GO) run ./cmd/toc init

# Testing

test: ## Run all tests
	$(GO) test ./...

test-verbose: ## Run all tests with verbose output
	$(GO) test -v ./...

test-race: ## Run tests with race detection
	$(GO) test -race ./...

test-cover: ## Run tests with coverage report
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-unit: ## Run unit tests only
	$(GO) test -short ./...

test-integration: ## Run integration tests
	$(GO) test -tags=integration ./...

test-e2e: ## Run E2E tests
	$(GO) test -tags=e2e -timeout 5m ./test/e2e/...

# Code quality

lint: ## Run linter
	golangci-lint run

lint-fix: ## Run linter with auto-fix
	golangci-lint run --fix

fmt: ## Format code
	gofmt -s -w .
	goimports -w .

vet: ## Run go vet
	$(GO) vet ./...

check: fmt vet lint test ## Run all checks

# Dependencies

deps: ## Download dependencies
	$(GO) mod download

tidy: ## Tidy go modules
	$(GO) mod tidy

# Install

install: ## Install the binary
	$(GO) install $(GOFLAGS) $(LDFLAGS) ./cmd/toc

uninstall: ## Uninstall the binary
	$(GO) clean -i ./cmd/toc

# Clean

clean: ## Clean build artifacts
	$(GO) clean
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Release

release: ## Create a release (requires goreleaser)
	goreleaser release --clean

snapshot: ## Create a snapshot release
	goreleaser release --snapshot --clean

# Documentation

docs-serve: ## Serve documentation locally
	@echo "Open docs in your browser..."
	cd docs && python3 -m http.server 8000

# Git

tag: ## Create a version tag
	@read -p "Version (e.g., v0.1.0): " version; \
	git tag -a $$version -m "Release $$version"; \
	git push origin $$version
