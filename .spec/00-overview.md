# glib Framework - Complete Specification

## v2.0.0 Update: Modular Architecture

**Status:** ✅ Implemented in v2.0.0-alpha

As of v2.0.0, glib has been fully modularized into independent packages. This specification document describes the original vision - see [RESTRUCTURE-SUMMARY.md](./RESTRUCTURE-SUMMARY.md) for implementation details.

### Current Module Structure

```
glib (v2.0.0)
├── http/            # HTTP server (github.com/azizndao/glib)
├── common/          # Utilities (github.com/azizndao/glib/common)
├── foundation/      # DI framework (github.com/azizndao/glib/foundation)
├── database/        # Database & ORM (github.com/azizndao/glib/database)
├── validation/      # Validation (github.com/azizndao/glib/validation)
├── ratelimit/       # Rate limiting (github.com/azizndao/glib/ratelimit)
└── cli/             # Dev tools (github.com/azizndao/glib/cli)
```

For detailed module documentation:

- [http/README.md](../http/README.md)
- [common/README.md](../common/README.md)
- [foundation/README.md](../foundation/README.md)
- [database/README.md](../database/README.md)

---

## Vision

Transform **glib** from an HTTP framework into a comprehensive, Laravel-inspired backend framework for Go that provides everything needed for full-stack backend development with minimal external dependencies.

## Core Philosophy

1. **Convention over Configuration**: Sensible defaults with override capability
2. **Developer Experience First**: Elegant APIs, powerful CLI, excellent documentation
3. **Minimal Dependencies**: Build what we can, integrate carefully when needed
4. **Go Idiomatic**: Respect Go's design principles and idioms
5. **Progressive Enhancement**: Simple things are simple, complex things are possible
6. **Production Ready**: Battle-tested patterns, comprehensive error handling

## Architecture Decisions

### Dependency Injection / Service Container

**Decision**: Custom lightweight container (Option B)

- Build our own minimal DI container
- Keep dependencies minimal
- Interface-based registration
- Type-safe with generics support
- Manual resolution in most cases
- Clear and explicit over magic

**Rationale**: Maintains control, keeps framework lightweight, follows Go idioms

### ORM Strategy

**Decision**: GORM Integration (Option B)

- Use GORM v2 as the ORM foundation
- Proven, feature-rich, community support
- Wrap with Laravel-style Active Record API
- Add relationship helpers and query builders

**Rationale**: GORM is battle-tested and feature-complete. Building a custom ORM would take months and likely result in bugs. We can provide a better API on top of GORM while leveraging its maturity.

### Project Structure & CLI

**Decision**: Built-in CLI Tool (Option A)

- Ship a `glib` command-line tool with the framework
- Commands: `glib new`, `glib make:*`, `glib migrate`, etc.
- Artisan-inspired code generation
- Scaffolding for consistency

**Rationale**: CLI tools dramatically improve developer productivity and ensure consistency across projects.

### Authentication Strategy

**Decision**: Full Auth Package (Option A)

- JWT tokens (stateless API auth)
- Session-based auth (traditional web)
- OAuth2 support (social login)
- Password reset flows
- Complete authentication system

**Rationale**: Authentication is foundational for most applications. Providing a complete, secure solution out of the box saves weeks of development time.

### Queue System Backend

**Decision**: Multiple Drivers (Option C)

- Database driver (PostgreSQL/MySQL)
- Redis driver (production scale)
- In-memory driver (development/testing)
- Extensible for future drivers (SQS, RabbitMQ)

**Rationale**: Flexibility without forcing infrastructure requirements. Database driver works immediately, Redis scales for production, in-memory simplifies testing.

## Technology Stack

### Core Dependencies

- **Router**: Chi v5 (already integrated)
- **ORM**: GORM v2
- **Validation**: go-playground/validator (already integrated)
- **CLI**: spf13/cobra or urfave/cli
- **Redis**: go-redis/redis (for cache/queue)
- **AWS SDK**: for S3 storage support

### Standard Library Usage

- `database/sql` - Database connections
- `encoding/json` - JSON handling
- `net/http` - HTTP server
- `context` - Context management
- `crypto/*` - Cryptography
- `time` - Time handling

### Optional Dependencies (User Choice)

- PostgreSQL driver: `lib/pq`
- MySQL driver: `go-sql-driver/mysql`
- SQLite driver: `modernc.org/sqlite`

## Project Structure

```
glib/
├── .spec/                          # Specification documents (this folder)
├── container/                      # Service container & DI
├── foundation/                     # Application foundation
├── config/                        # Configuration system
├── database/                      # Database connection management
├── orm/                          # ORM layer (GORM wrapper)
├── auth/                         # Authentication & authorization
├── queue/                        # Job queue system
├── schedule/                     # Task scheduling
├── cache/                        # Caching system
├── storage/                      # File storage abstraction
├── support/                      # Collections, helpers, utilities
├── testing/                      # Testing utilities
├── cmd/glib/                     # CLI tool
├── examples/                     # Example applications
├── docs/                         # Documentation
└── README.md
```

## Implementation Phases

### Phase 1: Foundation & Core Architecture (Weeks 1-2)

- Service container
- Service providers
- Enhanced configuration system
- Application lifecycle management

### Phase 2: Database Layer (Weeks 3-6)

- GORM integration & connection management
- Model base & Active Record pattern
- Relationships (HasOne, HasMany, BelongsTo, ManyToMany)
- Migrations system
- Query builder enhancements

### Phase 3: CLI Tool (Weeks 7-8)

- CLI framework setup
- Code generators (model, controller, middleware, etc.)
- Project scaffolding (`glib new`)
- Database commands (migrate, seed, etc.)

### Phase 4: Authentication & Authorization (Weeks 9-13)

- Authentication foundation
- JWT implementation
- Session-based auth
- Policies & gates
- OAuth2 providers
- Password reset flows

### Phase 5: Queue System & Scheduling (Weeks 14-16)

- Queue foundation & interfaces
- Database queue driver
- Redis queue driver
- In-memory driver for testing
- Job chaining & batching
- Task scheduling (cron-like)

### Phase 6: Caching & Storage (Weeks 17-18)

- Cache system with multiple drivers
- In-memory cache
- Redis cache
- File storage abstraction
- Local filesystem driver
- S3 storage driver

### Phase 7: Developer Experience (Weeks 19-20)

- Collections API (using generics)
- Model factories & seeders
- Testing utilities
- HTTP test helpers
- Documentation & examples

## Success Criteria

A project is considered successful when:

1. **Complete CRUD API** can be built in under 30 minutes
2. **Authentication** is plug-and-play with JWT/sessions
3. **Background jobs** work out of the box with zero config
4. **Tests** are easy to write and fast to run
5. **Documentation** is comprehensive and clear
6. **Migration from Laravel** is intuitive for PHP developers
7. **Performance** meets or exceeds other Go frameworks
8. **Community adoption** demonstrates value

## Documentation Structure

Each phase will have detailed documentation:

- API specifications
- Code examples
- Migration guides
- Best practices
- Common patterns
- Troubleshooting

## Related Specifications

- [Phase 1: Foundation](./01-foundation.md)
- [Phase 2: Database Layer](./02-database.md)
- [Phase 3: CLI Tool](./03-cli.md)
- [Phase 4: Authentication](./04-authentication.md)
- [Phase 5: Queue System](./05-queue.md)
- [Phase 6: Caching & Storage](./06-cache-storage.md)
- [Phase 7: Developer Experience](./07-developer-experience.md)
- [Package Structure](./08-package-structure.md)
- [Testing Strategy](./09-testing-strategy.md)
- [Performance Goals](./10-performance.md)

## Version Compatibility

- **Go Version**: 1.25+ (for generics support)
- **GORM Version**: v2.x
- **Semantic Versioning**: Follow semver strictly
- **Backward Compatibility**: Major versions only for breaking changes

## Contributing Guidelines

To be established:

- Code style guide
- PR process
- Testing requirements
- Documentation standards

## License

MIT License (maintaining current license)
