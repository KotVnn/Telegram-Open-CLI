# ADR 001: Use Cobra for CLI Framework

## Status

Accepted

## Date

2024-01-15

## Context

TOC needs a CLI framework that supports:
- Subcommands (init, start, config, status)
- Flag parsing with validation
- Help text generation
- Shell completion
- Integration with configuration (Viper)

We need a mature, well-tested framework that follows Go conventions.

## Decision

We will use `github.com/spf13/cobra` as the CLI framework.

## Consequences

### Positive

1. **Industry Standard** - Cobra is used by major Go projects:
   - Kubernetes (kubectl)
   - Docker
   - GitHub CLI
   - Hugo
   - Caddy

2. **Feature Complete** - Supports all our requirements:
   - Subcommands with aliases
   - Persistent and local flags
   - Auto-generated help text
   - Shell completion (bash, zsh, fish, powershell)
   - Hook system (PreRun, PostRun)

3. **Excellent Documentation** - Comprehensive docs at cobra.dev

4. **Viper Integration** - Seamless integration with Viper for configuration

5. **Community Support** - Large community, active maintenance

### Negative

1. **Dependency** - Adds an external dependency (mitigated by its maturity)

2. **Opinionated** - Cobra imposes certain patterns (mitigated by alignment with our needs)

### Risks

- **Low** - Cobra is battle-tested and actively maintained

## Alternatives Considered

| Alternative | Pros | Cons | Decision |
|-------------|------|------|----------|
| `urfave/cli` | Simpler API | Less features, smaller community | Rejected |
| `pflag` | Lower level | No subcommand support | Rejected |
| `kong` | Type-safe | Less adoption, newer | Rejected |
| Standard `flag` | No dependency | Too low-level | Rejected |

## References

- [Cobra Documentation](https://cobra.dev)
- [spf13/go-skills](https://github.com/spf13/go-skills)
- [Effective Go](https://go.dev/doc/effective_go)
