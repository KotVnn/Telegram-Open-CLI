# Roadmap

This document outlines the planned development phases for TOC.

## Overview

| Phase | Focus | Timeline | Status |
|-------|-------|----------|--------|
| Phase 1 | Foundation | Weeks 1-6 | ✅ Complete |
| Phase 2 | Core Features | Weeks 7-12 | 🔜 Planned |
| Phase 3 | Advanced Features | Weeks 13-20 | 🔜 Planned |
| Phase 4 | Production Ready | Weeks 21-26 | 🔜 Planned |
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

## Phase 2: Core Features (Weeks 7-12)

**Goal:** Implement essential features for daily use.

### Tasks

- [ ] Streaming response support
- [ ] File upload/download
- [ ] Inline keyboards for quick actions
- [ ] Middleware system
- [ ] User authentication
- [ ] Rate limiting
- [ ] Logging (zerolog)
- [ ] Claude Code backend adapter
- [ ] Error handling and recovery
- [ ] Integration tests
- [ ] Documentation improvements

### Deliverables

- Real-time streaming responses
- File handling capabilities
- Robust middleware system
- Claude Code support

### Success Criteria

- [ ] Responses stream in real-time
- [ ] Can upload and process files
- [ ] Inline keyboards work correctly
- [ ] Authentication prevents unauthorized access
- [ ] Rate limiting prevents abuse
- [ ] All error cases handled gracefully

---

## Phase 3: Advanced Features (Weeks 13-20)

**Goal:** Add advanced functionality for power users.

### Tasks

- [ ] Multi-project support
- [ ] Workspace management
- [ ] User management (CRUD)
- [ ] Permission system (RBAC)
- [ ] Plugin system
- [ ] Aider backend adapter
- [ ] Gemini backend adapter
- [ ] Tool calling support
- [ ] MCP integration
- [ ] Webhook mode for Telegram
- [ ] Performance optimization

### Deliverables

- Multi-project workflow
- Plugin architecture
- Four backend adapters
- Advanced permissions

### Success Criteria

- [ ] Can manage multiple projects
- [ ] Plugins can extend functionality
- [ ] All four backends working
- [ ] Permissions enforced correctly
- [ ] Webhook mode stable

---

## Phase 4: Production Ready (Weeks 21-26)

**Goal:** Prepare for production use and public release.

### Tasks

- [ ] Metrics (Prometheus)
- [ ] Docker support
- [ ] Binary releases (GoReleaser)
- [ ] Shell completions
- [ ] Documentation website
- [ ] Performance testing
- [ ] Security audit
- [ ] Auto-update check
- [ ] Man pages
- [ ] Example plugins
- [ ] Migration guides

### Deliverables

- Production-ready binaries
- Docker image
- Comprehensive documentation
- Performance benchmarks

### Success Criteria

- [ ] Cross-platform binaries available
- [ ] Docker image < 50MB
- [ ] Shell completions work
- [ ] Documentation complete
- [ ] Performance benchmarks documented
- [ ] Security audit passed

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
