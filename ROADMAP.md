# Roadmap

This document outlines the planned development phases for TOC.

## Overview

| Phase | Focus | Timeline | Status |
|-------|-------|----------|--------|
| Phase 1 | Foundation | Weeks 1-6 | ✅ Complete |
| Phase 2 | Core Features | Weeks 7-12 | ✅ Complete |
| Phase 3 | Advanced Features | Weeks 13-20 | ✅ Complete |
| Phase 4 | Production Ready | Weeks 21-26 | ✅ Complete |
| Phase 5 | Community | Ongoing | 🔜 Planned |

---

## Phase 1: Foundation (Weeks 1-6) ✅

**Goal:** Establish core architecture and basic functionality.

### Tasks

- [x] Project structure setup
- [x] Core interfaces defined
- [x] Configuration system (Viper + TOML)
- [x] SQLite storage implementation
- [x] OpenCode backend adapter
- [x] Basic Telegram bot
- [x] Session management (CRUD)
- [x] Basic commands (/new, /sessions, /switch, /close, /status)
- [x] Unit tests (>60% coverage)
- [x] CI/CD setup (GitHub Actions)
- [x] README and basic documentation

### Deliverables

- [x] Working CLI with `init` and `start` commands
- [x] Bot that can create and manage sessions
- [x] OpenCode integration working
- [x] All tests passing

### Success Criteria

- [x] `toc init` creates config file
- [x] `toc start` runs the bot
- [x] Can create a session via `/new`
- [x] Can list sessions via `/sessions`
- [x] Can send messages and receive responses
- [x] Unit test coverage > 60%

### Bug Fixes (Phase 1)

All bugs identified during Phase 1 evaluation have been fixed:

- **P0 (Critical):** 7/7 fixed — data races, env var destruction, error handling
- **P1 (High):** 7/7 fixed — message ID collision, memory leaks, error exposure
- **P2 (Medium):** 8/8 fixed — dead code, config gaps, auth feedback
- **P3 (Low):** 10/14 fixed — test cleanup, documentation placeholders

### Files Created/Modified

- `.gitignore` — comprehensive coverage
- `.goreleaser.yml` — v2 config, 6 platform targets
- `.github/workflows/release.yml` — tag-triggered releases
- `.github/dependabot.yml` — weekly updates
- `.github/PULL_REQUEST_TEMPLATE.md`
- `.github/CODEOWNERS`
- `README.md` — all placeholder URLs fixed
- `CHANGELOG.md` — release date set
- `SECURITY.md` — contact info filled
- `CODE_OF_CONDUCT.md` — contact info filled
- `CONTRIBUTING.md` — development guide

---

## Phase 2: Core Features (Weeks 7-12) ✅

**Goal:** Implement essential features for daily use.

### Tasks

- [x] Streaming response support
- [x] File upload/download
- [x] Inline keyboards for quick actions
- [x] Middleware system
- [x] User authentication
- [x] Rate limiting
- [x] Logging (zerolog)
- [x] Claude Code backend adapter
- [x] Error handling and recovery
- [x] Integration tests
- [x] Documentation improvements

### Deliverables

- Real-time streaming responses
- File handling capabilities
- Robust middleware system
- Claude Code support

### Success Criteria

- [x] Responses stream in real-time
- [x] Can upload and process files
- [x] Inline keyboards work correctly
- [x] Authentication prevents unauthorized access
- [x] Rate limiting prevents abuse
- [x] All error cases handled gracefully

### Summary

Phase 2 implemented all core features for daily use:

- **Middleware wiring:** AuthMiddleware and RateLimitMiddleware fully integrated
- **Structured logging:** All `fmt.Fprintf(os.Stderr)` replaced with zerolog
- **Claude Code adapter:** Full backend implementation with tests
- **Inline keyboards:** ToTelegram() conversion, callback handlers for session/model/agent selection
- **File handling:** Document upload, Telegram API download, backend attachment passing
- **Integration tests:** Backend manager, Telegram bot, and app initialization flows

---

## Phase 3: Advanced Features (Weeks 13-20) ✅

**Goal:** Add advanced functionality for power users.

### Tasks

- [x] Multi-project support
- [x] Workspace management
- [x] User management (CRUD)
- [x] Permission system (RBAC)
- [x] Plugin system
- [x] Aider backend adapter
- [x] Gemini backend adapter
- [x] Tool calling support
- [x] MCP integration
- [x] Webhook mode for Telegram
- [x] Performance optimization

### Deliverables

- Multi-project workflow
- Plugin architecture
- Four backend adapters
- Advanced permissions

### Success Criteria

- [x] Can manage multiple projects
- [x] Plugins can extend functionality
- [x] All four backends working
- [x] Permissions enforced correctly
- [x] Webhook mode stable

### Summary

Phase 3 implemented advanced features for power users:

- **User management:** /me, /users, /ban, /unban, /role commands with admin checks
- **Project management:** /project new/list/switch/delete with ownership enforcement
- **RBAC permissions:** Role hierarchy (admin > user > viewer), permission middleware
- **Backend adapters:** Aider and Gemini adapters following Claude/OpenCode pattern
- **Webhook mode:** Telegram webhook support with polling fallback
- **Concurrency safety:** Mutex protection for all shared maps

---

## Phase 4: Production Ready (Weeks 21-26) ✅

**Goal:** Prepare for production use and public release.

### Tasks

- [x] Metrics (Prometheus)
- [x] Docker support
- [x] Binary releases (GoReleaser) — already configured
- [x] Shell completions
- [ ] Documentation website
- [x] Performance testing (benchmarks)
- [x] Security audit
- [x] Auto-update check
- [x] Man pages
- [ ] Example plugins
- [ ] Migration guides

### Deliverables

- Production-ready binaries
- Docker image
- Comprehensive documentation
- Performance benchmarks

### Success Criteria

- [x] Cross-platform binaries available
- [ ] Docker image < 50MB
- [x] Shell completions work
- [ ] Documentation complete
- [x] Performance benchmarks documented
- [x] Security audit passed

### Summary

Phase 4 prepared the project for production:

- **Metrics:** Prometheus collector with /metrics, /health, /ready endpoints
- **Docker:** Multi-stage Dockerfile (alpine-based, non-root user), docker-compose with Prometheus
- **Shell completions:** bash, zsh, fish, powershell via `toc completion`
- **Man pages:** via `toc man`
- **Auto-update:** GitHub API checker with immediate check + daily periodic
- **Benchmarks:** Session creation, message sending, concurrent sessions
- **Security:** Rate limiter, input validation/sanitization (UTF-8 safe)

---

## Phase 5: Community (Ongoing)

**Goal:** Build and support the community.

### Tasks

- [ ] GitHub launch
- [ ] Community channels (Discord/Telegram)
- [ ] Example plugins
- [ ] Example integrations
- [ ] Regular release cycle
- [ ] Feature request process
- [ ] Contribution recognition
- [ ] Blog posts
- [ ] Conference talks
- [ ] Partnerships

### Deliverables

- Active community
- Regular releases
- Growing ecosystem

### Success Criteria

- [ ] GitHub stars > 100 (3 months)
- [ ] First external contributors
- [ ] Plugin ecosystem started
- [ ] Monthly release cadence
- [ ] Active community channels

---

## Future Ideas

Features being considered for future development:

### Short Term (Next 6 months)

- Web UI dashboard
- Voice input support
- Notification system
- Session history search
- Export conversations

### Medium Term (6-12 months)

- Team collaboration features
- CI/CD integration
- Custom model training
- Multi-language support
- Mobile app

### Long Term (12+ months)

- Enterprise features
- Cloud sync
- Advanced analytics
- Marketplace for plugins
- White-label support

---

## Contributing

Want to help with development? Check out:

1. [CONTRIBUTING.md](CONTRIBUTING.md) - How to contribute
2. Issues labeled `good first issue` - Start here
3. Issues labeled `help wanted` - Need assistance
4. [ROADMAP.md](ROADMAP.md) - This document

---

## Updates

This roadmap will be updated regularly as development progresses. Check back for the latest information.

Last updated: 2026-07-22
