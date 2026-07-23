# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project structure
- Core backend interface and manager
- OpenCode backend adapter
- Telegram bot integration
- SQLite storage implementation
- Session management (CRUD operations)
- Basic middleware system (auth, logging, rate limiting)
- Configuration system (TOML, env vars, flags)
- CLI commands: init, start, version
- Keyboard builders for Telegram
- Streaming response support
- File upload/download support
- User management
- Project management
- Unit tests (>80% coverage)
- Integration test framework
- CI/CD pipeline (GitHub Actions)
- Documentation:
  - README.md
  - ARCHITECTURE.md
  - AGENTS.md
  - Contributing guidelines
  - Security policy
  - Code of conduct
  - API documentation
  - Backend adapter guide
  - Telegram adapter guide
  - Middleware guide
  - Development setup guide
  - Testing guide
  - Architecture Decision Records

### Changed
- N/A

### Deprecated
- N/A

### Removed
- N/A

### Fixed
- N/A

### Security
- N/A

---

## [0.1.0] - 2026-07-22

### Added

#### Core Features
- Support for OpenCode backend
- Telegram bot with basic commands
- Session management
- Configuration via TOML
- SQLite storage

#### CLI Commands
- `toc init` - Initialize configuration
- `toc start` - Start the bot
- `toc version` - Show version info

#### Telegram Commands
- `/start` - Start conversation
- `/help` - Show help
- `/new [name]` - Create new session
- `/sessions` - List sessions
- `/switch [id]` - Switch session
- `/close [id]` - Close session
- `/status` - Show status

#### Backend Support
- OpenCode (full support)
- Claude Code (planned)
- Aider (planned)
- Gemini CLI (planned)

#### Infrastructure
- GitHub Actions CI
- GoReleaser configuration
- Makefile for common tasks

---

## Version History

### Future Plans

See [ROADMAP.md](ROADMAP.md) for planned features.

---

## Release Notes Format

For each release, we document:

1. **Added** - New features
2. **Changed** - Changes to existing functionality
3. **Deprecated** - Features that will be removed
4. **Removed** - Features that have been removed
5. **Fixed** - Bug fixes
6. **Security** - Vulnerability fixes

---

## Contributors

Thank you to all contributors who have helped make TOC better!

See the full list of contributors on [GitHub](https://github.com/KotVnn/Telegram-Open-CLI/graphs/contributors).
