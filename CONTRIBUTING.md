# Contributing to TOC

Thank you for considering contributing to TOC! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Setup](#development-setup)
- [Code Style](#code-style)
- [Commit Messages](#commit-messages)
- [Pull Request Process](#pull-request-process)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Features](#suggesting-features)
- [Questions](#questions)

---

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

---

## Getting Started

1. Fork the repository
2. Clone your fork
3. Create a feature branch
4. Make your changes
5. Submit a pull request

```bash
# Fork on GitHub, then:
git clone https://github.com/KotVnn/Telegram-Open-CLI.git
cd toc
git remote add upstream https://github.com/KotVnn/Telegram-Open-CLI.git
git checkout -b feature/my-feature
```

---

## How to Contribute

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates.

**When filing a bug report, include:**

1. **Clear title** - Summarize the issue
2. **Steps to reproduce** - How can we reproduce the issue?
3. **Expected behavior** - What should happen?
4. **Actual behavior** - What actually happens?
5. **Environment** - OS, Go version, TOC version
6. **Logs** - Relevant log output (if any)

**Template:**

```markdown
## Bug Report

### Description
A clear description of the bug.

### Steps to Reproduce
1. Run `toc start`
2. Send `/new test`
3. Send a message
4. See error

### Expected Behavior
The message should be processed.

### Actual Behavior
Error: "connection refused"

### Environment
- OS: Windows 11
- Go: 1.22
- TOC: v0.1.0

### Logs
```
2024-01-15 10:30:00 ERR connection refused
```
```

### Suggesting Features

**When suggesting a feature, include:**

1. **Problem statement** - What problem does this solve?
2. **Proposed solution** - How should it work?
3. **Alternatives considered** - Other approaches?
4. **Use cases** - Who benefits?

**Template:**

```markdown
## Feature Request

### Problem
I often need to switch between backends manually.

### Proposed Solution
Add a `/switch` command to change backends on the fly.

### Alternatives
- Use config file (less convenient)
- Restart bot (wasteful)

### Use Cases
- Testing different backends
- Using cheaper models for simple tasks
```

### Contributing Code

1. **Find an issue** - Look for `good first issue` or `help wanted` labels
2. **Discuss** - Comment on the issue to let others know you're working on it
3. **Fork & branch** - Create a feature branch from `main`
4. **Implement** - Write your code following the style guide
5. **Test** - Add tests and ensure they pass
6. **Document** - Update documentation if needed
7. **Submit PR** - Create a pull request with clear description

---

## Development Setup

See [docs/setup.md](docs/setup.md) for detailed development environment setup.

### Quick Start

```bash
# Clone and build
git clone https://github.com/KotVnn/Telegram-Open-CLI.git
cd toc
go mod download
go build -o toc.exe ./cmd/toc

# Run tests
go test ./...

# Start development
go run ./cmd/toc init
go run ./cmd/toc start
```

---

## Code Style

### Go Code

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `gofmt` and `gofumpt` for formatting
- Use `goimports` for import organization
- Run `golangci-lint` before committing

### Formatting

```bash
# Format code
gofmt -s -w .
goimports -w .

# Lint
golangci-lint run

# Vet
go vet ./...
```

### Import Order

```go
import (
    // Standard library
    "context"
    "fmt"
    "time"

    // External packages
    "github.com/rs/zerolog"
    "github.com/stretchr/testify/assert"

    // Internal packages
    "github.com/KotVnn/Telegram-Open-CLI/internal/backend"
    "github.com/KotVnn/Telegram-Open-CLI/internal/config"
)
```

### Naming Conventions

- **Packages**: Short, lowercase, single word (`backend`, `config`)
- **Types**: PascalCase (`Session`, `BackendConfig`)
- **Functions**: PascalCase for exported, camelCase for unexported
- **Constants**: PascalCase for exported, camelCase for unexported
- **Interfaces**: PascalCase, often `-er` suffix (`Backend`, `Storage`)

### Comments

```go
// Backend defines the interface for AI coding agent backends.
// Each backend must implement this interface to be used with TOC.
type Backend interface {
    // Name returns the backend identifier (e.g., "opencode").
    Name() string

    // Initialize sets up the backend with the provided configuration.
    // This must be called before any other backend methods.
    Initialize(ctx context.Context, config BackendConfig) error
}
```

---

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

| Type | Description | Example |
|------|-------------|---------|
| `feat` | New feature | `feat: add Claude backend adapter` |
| `fix` | Bug fix | `fix: handle session not found error` |
| `docs` | Documentation | `docs: update README installation` |
| `style` | Formatting | `style: format code with gofmt` |
| `refactor` | Code change | `refactor: extract session manager` |
| `test` | Tests | `test: add unit tests for backend` |
| `chore` | Maintenance | `chore: update dependencies` |
| `perf` | Performance | `perf: optimize message parsing` |

### Examples

```
feat(backend): add streaming response support

Implement streaming responses for OpenCode backend.
Responses are now displayed in real-time as they arrive.

Closes #123
```

```
fix(telegram): handle rate limit errors

Add retry logic with exponential backoff when Telegram
API rate limits are encountered.

Fixes #456
```

---

## Pull Request Process

### Before Submitting

- [ ] Code follows style guidelines
- [ ] Tests added/updated
- [ ] Documentation updated (if needed)
- [ ] All tests pass (`go test ./...`)
- [ ] Linter passes (`golangci-lint run`)
- [ ] Commit messages follow conventions

### PR Description

```markdown
## Description

Brief description of changes.

## Type of Change

- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing

Describe tests added/updated.

## Checklist

- [ ] Code follows style guidelines
- [ ] Tests pass locally
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
```

### Review Process

1. Automated CI checks must pass
2. At least one maintainer review required
3. Address review feedback
4. Squash and merge

---

## Questions?

Feel free to:
- Open an issue for questions
- Start a discussion on GitHub
- Reach out on Telegram (if available)

## Plugin Development

TOC supports plugins to extend functionality. See `examples/plugins/example/` for a reference implementation.

### Plugin Interface

```go
type Plugin interface {
    Name() string
    Description() string
    Version() string
    Initialize(ctx context.Context, config PluginConfig) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Capabilities() *PluginCapabilities
}
```

### Creating a Plugin

1. Create a new directory in `examples/plugins/`
2. Implement the `Plugin` interface
3. Register hooks with the plugin manager
4. Add documentation

Thank you for contributing!
