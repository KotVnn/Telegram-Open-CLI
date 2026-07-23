# ADR 002: Backend Abstraction Architecture

## Status

Accepted

## Date

2024-01-15

## Context

TOC needs to support multiple AI coding agents:
- OpenCode
- Claude Code
- Aider
- Gemini CLI
- Future agents

Each agent has different:
- CLI interfaces
- Configuration methods
- Output formats
- Capabilities

We need an architecture that:
- Allows adding new backends without changing core code
- Enables switching backends per session
- Supports testing with mock backends
- Isolates backend-specific logic

## Decision

We will use a Strategy pattern with a common Backend interface. Each backend implements the interface, and a BackendManager handles selection and lifecycle.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                   Backend Manager                           │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                   Backend Interface                     ││
│  │  - Name()                                              ││
│  │  - Initialize()                                        ││
│  │  - CreateSession()                                     ││
│  │  - SendMessage()                                       ││
│  │  - StreamMessage()                                     ││
│  │  - Capabilities()                                      ││
│  └─────────────────────────────────────────────────────────┘│
│       │              │              │              │         │
│  ┌────┴────┐    ┌────┴────┐    ┌────┴────┐    ┌────┴────┐  │
│  │OpenCode │    │ Claude  │    │  Aider  │    │ Gemini  │  │
│  │ Adapter │    │ Adapter │    │ Adapter │    │ Adapter │  │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Selection Strategy

```
User Request
    │
    ▼
┌─────────────────┐
│ Check User      │
│ Preference      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Check Project   │
│ Config          │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Use Default     │
│ Backend         │
└─────────────────┘
```

## Consequences

### Positive

1. **Extensibility** - Easy to add new backends
   - Implement Backend interface
   - Register in manager
   - Done

2. **Isolation** - Backend logic is isolated
   - Each adapter in its own package
   - No shared state between backends
   - Easy to test independently

3. **Flexibility** - Users can switch backends
   - Per session
   - Per project
   - Per request

4. **Testability** - Easy to test with mocks
   - MockBackend for unit tests
   - Real adapters for integration tests

### Negative

1. **Maintenance** - Multiple adapters to maintain
   - Mitigated by similar patterns
   - Mitigated by shared tests

2. **Feature Parity** - Not all features available in all backends
   - Mitigated by Capabilities() method
   - Mitigated by graceful degradation

### Risks

- **Medium** - Backend APIs may change frequently
- **Mitigation** - Regular testing, adapter updates, version pinning

## Implementation Details

### Registration

```go
// internal/backend/manager.go

var registry = make(map[string]func() Backend)

func Register(name string, factory func() Backend) {
    registry[name] = factory
}

func init() {
    Register("opencode", opencode.New)
    Register("claude", claude.New)
    Register("aider", aider.New)
    Register("gemini", gemini.New)
}
```

### Selection

```go
func (m *Manager) GetBackend(ctx context.Context, req *Request) (Backend, error) {
    // 1. Check user preference
    if user := GetUserFromContext(ctx); user.PreferredBackend != "" {
        return m.get(user.PreferredBackend)
    }

    // 2. Check project config
    if project := GetProjectFromContext(ctx); project.DefaultBackend != "" {
        return m.get(project.DefaultBackend)
    }

    // 3. Use default
    return m.get(m.config.DefaultBackend)
}
```

## Alternatives Considered

| Alternative | Pros | Cons | Decision |
|-------------|------|------|----------|
| Direct implementation | Simpler | Tight coupling, hard to extend | Rejected |
| Plugin system | More flexible | Overkill for current needs | Future consideration |
| Interface + switch | Simple | Doesn't scale | Rejected |

## References

- [Strategy Pattern](https://refactoring.guru/design-patterns/strategy)
- [Adapter Pattern](https://refactoring.guru/design-patterns/adapter)
- [Go Interface Design](https://go.dev/doc/effective_go#interfaces)
