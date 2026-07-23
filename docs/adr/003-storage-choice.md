# ADR 003: Use SQLite for Storage

## Status

Accepted

## Date

2024-01-15

## Context

TOC needs local storage for:
- Sessions
- Messages
- Users
- Projects
- Configuration

Requirements:
- Simple setup (no external services)
- Good performance for single-user/small team
- ACID compliance
- Easy backup
- Cross-platform support

## Decision

We will use SQLite via GORM for primary storage.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                 Storage Interface                       ││
│  │  - SaveSession()                                       ││
│  │  - GetSession()                                        ││
│  │  - ListSessions()                                      ││
│  │  - SaveMessage()                                       ││
│  │  - etc.                                                ││
│  └─────────────────────────────────────────────────────────┘│
│                          │                                   │
│                          ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                 GORM Layer                              ││
│  │  - Query building                                      ││
│  │  - Migrations                                          ││
│  │  - Connection pooling                                  ││
│  └─────────────────────────────────────────────────────────┘│
│                          │                                   │
│                          ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                 SQLite                                  ││
│  │  - ~/.toc/toc.db                                       ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Consequences

### Positive

1. **Zero Configuration** - No setup required
   - Single file database
   - No server process
   - Works out of the box

2. **Portability** - Cross-platform
   - Works on Windows, macOS, Linux
   - Single file easy to backup/transfer

3. **Performance** - Good for expected load
   - WAL mode for concurrent reads
   - Sufficient for single-user/small team

4. **Reliability** - Battle-tested
   - ACID compliance
   - Used by billions of devices
   - Extensive testing

5. **Simplicity** - Easy to work with
   - GORM simplifies queries
   - Auto-migrations
   - Connection pooling built-in

### Negative

1. **Concurrent Writes** - Limited write performance
   - Mitigated by WAL mode
   - Mitigated by connection pooling
   - Acceptable for expected load

2. **Scalability** - Not suitable for distributed systems
   - Mitigated by single-user focus
   - Future: Can migrate to PostgreSQL if needed

3. **Features** - Missing some advanced features
   - No full-text search (mitigated by LIKE queries)
   - No JSON queries (mitigated by GORM)

### Risks

- **Low** - SQLite is battle-tested and mature

## Mitigation

1. **WAL Mode** - Better concurrent read performance
   ```go
   db, _ := gorm.Open(sqlite.Open("toc.db"), &gorm.Config{})
   db.Exec("PRAGMA journal_mode=WAL")
   ```

2. **Connection Pooling** - Managed by GORM
   ```go
   sqlDB, _ := db.DB()
   sqlDB.SetMaxOpenConns(1)
   ```

3. **Future Migration** - Can switch to PostgreSQL
   - Storage interface abstracts implementation
   - GORM supports multiple databases
   - Migration scripts can be added

## Alternatives Considered

| Alternative | Pros | Cons | Decision |
|-------------|------|------|----------|
| BoltDB/bbolt | Embedded, fast | No SQL queries | Rejected |
| BadgerDB | Fast, LSM-tree | Overkill | Rejected |
| JSON files | Simple | No concurrency, no queries | Rejected |
| PostgreSQL | Powerful | Requires external service | Future |
| MySQL | Popular | Requires external service | Rejected |
| MongoDB | Flexible | Requires external service | Rejected |

## Schema

```sql
-- Sessions table
CREATE TABLE sessions (
    id VARCHAR(36) PRIMARY KEY,
    backend VARCHAR(50) NOT NULL,
    project_id VARCHAR(36),
    title VARCHAR(255),
    status VARCHAR(20) NOT NULL,
    model VARCHAR(100),
    agent VARCHAR(50),
    working_dir VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP
);

-- Messages table
CREATE TABLE messages (
    id VARCHAR(36) PRIMARY KEY,
    session_id VARCHAR(36) NOT NULL,
    role VARCHAR(20) NOT NULL,
    content TEXT,
    files TEXT,
    tokens TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);

-- Users table
CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    username VARCHAR(100),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(20) DEFAULT 'user',
    allowed_backends TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Projects table
CREATE TABLE projects (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    path VARCHAR(500),
    description TEXT,
    default_backend VARCHAR(50),
    default_model VARCHAR(100),
    owner_id BIGINT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id)
);
```

## References

- [SQLite](https://www.sqlite.org)
- [GORM](https://gorm.io)
- [SQLite WAL Mode](https://www.sqlite.org/wal.html)
