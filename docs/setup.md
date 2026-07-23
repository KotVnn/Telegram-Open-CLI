# Development Setup

This guide explains how to set up a development environment for TOC.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Building](#building)
- [Configuration](#configuration)
- [Running](#running)
- [IDE Setup](#ide-setup)
- [Debugging](#debugging)
- [Common Issues](#common-issues)

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22+ | Language runtime |
| Git | 2.x | Version control |
| golangci-lint | Latest | Linting |
| gopls | Latest | Language server |
| delve | Latest | Debugger |

### Optional

| Tool | Purpose |
|------|---------|
| Docker | Container testing |
| SQLite CLI | Database inspection |
| ngrok | Webhook testing |

---

## Installation

### 1. Clone Repository

```bash
git clone https://github.com/KotVnn/toc.git
cd toc
```

### 2. Install Go Dependencies

```bash
go mod download
```

### 3. Install Development Tools

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install gopls
go install golang.org/x/tools/gopls@latest

# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Install goimports
go install golang.org/x/tools/cmd/goimports@latest
```

### 4. Verify Installation

```bash
# Check Go version
go version

# Check golangci-lint
golangci-lint version

# Build the project
go build -o toc.exe ./cmd/toc

# Run tests
go test ./...
```

---

## Building

### Development Build

```bash
# Simple build
go build -o toc.exe ./cmd/toc

# Build with race detector
go build -race -o toc.exe ./cmd/toc

# Build with version info
go build -ldflags "-X github.com/KotVnn/Telegram-Open-CLI/pkg/version.Version=dev" -o toc.exe ./cmd/toc
```

### Using Makefile

```bash
# Build
make build

# Build for all platforms
make build-all

# Clean
make clean
```

### Cross-Compilation

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o toc ./cmd/toc

# macOS arm64
GOOS=darwin GOARCH=arm64 go build -o toc ./cmd/toc

# Windows amd64
GOOS=windows GOARCH=amd64 go build -o toc.exe ./cmd/toc
```

---

## Configuration

### Create Config File

```bash
# Initialize configuration
./toc.exe init

# This creates ~/.toc/config.toml
```

### Edit Config File

```toml
version = "1"

[default]
backend = "opencode"

[telegram]
token = "YOUR_BOT_TOKEN_HERE"

[storage]
path = ""

[backends.opencode]
enabled = true
command = "opencode"
args = ["run", "--format", "json"]
timeout = "30m"

[logging]
level = "debug"
format = "console"
output = "stdout"
```

### Environment Variables

For development, you can use environment variables:

```bash
# PowerShell
$env:TOC_TELEGRAM_TOKEN = "your-token"
$env:TOC_LOG_LEVEL = "debug"
$env:TOC_DEFAULT_BACKEND = "opencode"

# Bash/Linux/macOS
export TOC_TELEGRAM_TOKEN="your-token"
export TOC_LOG_LEVEL="debug"
```

---

## Running

### Development Mode

```bash
# Run directly
go run ./cmd/toc start

# Run with debug logging
go run ./cmd/toc start --log-level debug

# Run with specific config
go run ./cmd/toc start --config ./dev-config.toml
```

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/backend/...

# With verbose output
go test -v ./...

# With race detection
go test -race ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## IDE Setup

### VS Code

Install extensions:
- [Go](https://marketplace.visualstudio.com/items?itemName=golang.go)
- [Error Lens](https://marketplace.visualstudio.com/items?itemName=usernamehw.errorlens)
- [GitLens](https://marketplace.visualstudio.com/items?itemName=eamodio.gitlens)

Recommended `.vscode/settings.json`:

```json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.lintFlags": ["--fast"],
    "go.testTimeout": "30s",
    "go.formatTool": "goimports",
    "editor.formatOnSave": true,
    "[go]": {
        "editor.defaultFormatter": "golang.go",
        "editor.codeActionsOnSave": {
            "source.organizeImports": "explicit"
        }
    },
    "gopls": {
        "ui.semanticTokens": true,
        "ui.completion.usePlaceholders": true
    }
}
```

Recommended `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch TOC",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/toc",
            "args": ["start"]
        },
        {
            "name": "Launch TOC (init)",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/toc",
            "args": ["init"]
        },
        {
            "name": "Test Current File",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${fileDirname}"
        }
    ]
}
```

### GoLand

1. Open Settings > Go > Gopls
2. Enable Gopls
3. Set build tags for testing

Run configurations:
- Go Build: `cmd/toc` with args `start`
- Go Test: Test current file/package

---

## Debugging

### Using Delve

```bash
# Debug the application
dlv debug ./cmd/toc -- start

# Debug with args
dlv debug ./cmd/toc -- --config ./dev-config.toml start

# Debug tests
dlv test ./internal/backend/...

# Connect to running process
dlv attach <pid>
```

### Delve Commands

```
break main.handleStart    # Set breakpoint
continue                  # Continue execution
next                      # Step over
step                      # Step into
print session             # Print variable
goroutines                # List goroutines
```

### VS Code Debugging

1. Set breakpoints in code
2. Press F5 or use "Launch TOC" configuration
3. Use Debug Console to inspect variables
4. Use Call Stack to navigate

### Remote Debugging

```bash
# Start remote debug server
dlv exec ./toc --headless --listen=:2345 --api-version=2 --accept-multiclient start

# Connect from VS Code using "Remote" configuration
```

---

## Common Issues

### Build Errors

**"command not found: go"**
- Ensure Go is installed and in PATH
- Run `go env` to verify

**"cannot find package"**
- Run `go mod download`
- Check go.mod for correct module path

**"undefined: ..."**
- Check import paths
- Run `goimports -w .` to fix imports

### Runtime Errors

**"token invalid"**
- Verify bot token in config
- Ensure no extra whitespace in token

**"command not found: opencode"**
- Install the AI backend first
- Check backend command in config

**"database is locked"**
- Ensure only one instance is running
- Check SQLite file permissions

### Performance Issues

**Slow builds**
- Use `-count=1` to disable test caching
- Run `go clean -cache` if needed

**High memory usage**
- Check for goroutine leaks
- Use `runtime.NumGoroutine()` to monitor

---

## Useful Commands

```bash
# Format code
gofmt -s -w .
goimports -w .

# Lint
golangci-lint run

# Vet
go vet ./...

# Tidy modules
go mod tidy

# View dependencies
go mod graph

# Test coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```
