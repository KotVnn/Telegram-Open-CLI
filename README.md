# TOC - Telegram OpenCode CLI

[![CI](https://github.com/KotVnn/toc/actions/workflows/ci.yml/badge.svg)](https://github.com/KotVnn/toc/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/KotVnn/toc)](https://goreportcard.com/report/github.com/KotVnn/toc)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> Control AI Coding Agents through Telegram. One install, one command, full control.

TOC lets you control AI coding agents (OpenCode, Claude Code, Aider, Gemini CLI) remotely via Telegram. Install once, run `toc start`, and start coding from anywhere.

## Features

- **Multi-Backend Support** - Works with OpenCode, Claude Code, Aider, and Gemini CLI
- **Remote Control** - Control your coding agents from Telegram on any device
- **Session Management** - Create, switch, and manage multiple coding sessions
- **Streaming Responses** - See AI responses in real-time
- **File Support** - Upload and download files through Telegram
- **Multi-Project** - Work on multiple projects simultaneously
- **Plugin System** - Extend functionality with custom plugins

## Quick Start

### Install

```bash
# Using Go
go install github.com/KotVnn/toc@latest

# Using Homebrew (macOS/Linux)
brew install KotVnn/toc

# Using Scoop (Windows)
scoop install toc

# Using Chocolatey (Windows)
choco install toc

# Download binary
# Visit https://github.com/KotVnn/toc/releases
```

### Setup

```bash
# Initialize configuration
toc init

# This creates ~/.toc/config.toml
# Edit it with your Telegram bot token

# Start the bot
toc start
```

### Create a Telegram Bot

1. Open Telegram and search for [@BotFather](https://t.me/BotFather)
2. Send `/newbot` and follow the prompts
3. Copy the bot token
4. Add it to `~/.toc/config.toml`:
   ```toml
   [telegram]
   token = "your-bot-token-here"
   ```

## Usage

### Telegram Commands

| Command | Description |
|---------|-------------|
| `/start` | Start a conversation with the bot |
| `/help` | Show available commands |
| `/new [name]` | Create a new coding session |
| `/sessions` | List all active sessions |
| `/switch [id]` | Switch to a different session |
| `/close [id]` | Close a session |
| `/project [name]` | Create or switch project |
| `/status` | Show current session status |
| `/settings` | Configure your preferences |

### Example Workflow

```
You: /new my-api
Bot: ✅ Created session "my-api" using OpenCode

You: Fix the authentication bug in user.go
Bot: 🔄 Processing...
Bot: ✅ Found the issue in user.go:42
Bot: 
     ```diff
     - if err := validateToken(token); err != nil {
     + if err := validateToken(token); err != nil {
     +     log.Printf("Token validation failed: %v", err)
     ```
     Applied changes. Want me to run tests?

You: Yes, run the tests
Bot: 🔄 Running tests...
Bot: ✅ All tests passing (12/12)
```

## Configuration

Config file: `~/.toc/config.toml`

```toml
version = "1"

[default]
backend = "opencode"

[telegram]
token = ""                          # Required: Your bot token

[storage]
path = ""                           # Default: ~/.toc/toc.db

[backends.opencode]
enabled = true
command = "opencode"
args = ["run", "--format", "json"]
timeout = "30m"

[backends.claude]
enabled = false
command = "claude"
args = ["-p"]
timeout = "30m"

[backends.aider]
enabled = false
command = "aider"
args = ["--yes"]
timeout = "30m"

[backends.gemini]
enabled = false
command = "gemini"
timeout = "30m"

[security]
require_auth = true
max_sessions_per_user = 5

[logging]
level = "info"
format = "json"
output = "stdout"
```

### Environment Variables

All configuration options can be set via environment variables with `TOC_` prefix:

```bash
export TOC_TELEGRAM_TOKEN="your-token"
export TOC_DEFAULT_BACKEND="opencode"
export TOC_LOG_LEVEL="debug"
```

## Supported Backends

| Backend | Status | Notes |
|---------|--------|-------|
| OpenCode | ✅ Supported | Primary backend, full feature support |
| Claude Code | 🔜 Planned | Coming in Phase 2 |
| Aider | 🔜 Planned | Coming in Phase 3 |
| Gemini CLI | 🔜 Planned | Coming in Phase 3 |

## Documentation

- [Architecture](ARCHITECTURE.md) - System design and patterns
- [Contributing](CONTRIBUTING.md) - How to contribute
- [Changelog](CHANGELOG.md) - Release history
- [Roadmap](ROADMAP.md) - Future plans

## Development

```bash
# Clone the repository
git clone https://github.com/KotVnn/toc.git
cd toc

# Install dependencies
go mod download

# Build
go build -o toc ./cmd/toc

# Run tests
go test ./...

# Run with coverage
go test -cover ./...
```

See [docs/setup.md](docs/setup.md) for detailed development setup.

## Project Structure

```
toc/
├── cmd/toc/                  # CLI entry point
├── internal/
│   ├── app/                  # Application lifecycle
│   ├── config/               # Configuration
│   ├── backend/              # AI backend adapters
│   ├── telegram/             # Telegram bot
│   ├── storage/              # Data persistence
│   ├── user/                 # User management
│   ├── project/              # Project management
│   └── plugin/               # Plugin system
├── pkg/
│   ├── version/              # Version info
│   └── errors/               # Custom errors
└── docs/                     # Documentation
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Security

For security concerns, please see [SECURITY.md](SECURITY.md).

## License

This project is licensed under the MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [OpenCode](https://opencode.ai) - The open source AI coding agent
- [go-telegram/bot](https://github.com/go-telegram/bot) - Telegram Bot API for Go
- [Cobra](https://github.com/spf13/cobra) - CLI framework
