# glib Framework Specifications

This directory contains comprehensive specifications for the glib framework, a Laravel-inspired web framework for Go.

## v2.0.0 Modular Architecture

As of v2.0.0, glib has been restructured into independent modules for better flexibility and maintainability:

- **[http](../http)** - HTTP server with routing, middleware, and context abstraction
- **[common](../common)** - Shared utilities (errors, logging, config, DI container)
- **[foundation](../foundation)** - Application lifecycle and ServiceProvider pattern
- **[database](../database)** - Database manager and ORM with relationships
- **[validation](../validation)** - Request validation with i18n support
- **[ratelimit](../ratelimit)** - Rate limiting utilities
- **[cli](../cli)** - Code generation and project scaffolding tools

### Module Dependency Graph

```
common (pure utilities)
  ↑
  ├── foundation (DI framework) ──→ database
  │                                    ↑
  ├── http                             │
  ├── validation                       │
  └── ratelimit                        │
                                       │
  cli (independent)                    │
```

For migration information from v1.x to v2.0.0, see [MIGRATION.md](../MIGRATION.md).

For complete restructuring history, see [RESTRUCTURE-SUMMARY.md](./RESTRUCTURE-SUMMARY.md).

## Table of Contents

### Core Specifications

1. **[00-overview.md](./00-overview.md)** - Framework overview, vision, and architecture decisions
2. **[01-foundation.md](./01-foundation.md)** - Service container, providers, configuration, and application lifecycle
3. **[02-database.md](./02-database.md)** - GORM integration, ORM layer, relationships, and migrations

### Feature Specifications

1. **[03-cli.md](./03-cli.md)** - Command-line tool, code generators, and project scaffolding
2. **[04-authentication.md](./04-authentication.md)** - Authentication system, JWT, sessions, OAuth2, and authorization
3. **[05-queue-scheduling.md](./05-queue-scheduling.md)** - Job queues, multiple drivers, task scheduling
4. **[06-cache-storage.md](./06-cache-storage.md)** - Caching system and file storage abstraction

### Developer Experience

1. **[07-developer-experience.md](./07-developer-experience.md)** - Collections API, factories, seeders, testing utilities

### Additional Documentation (To Be Created)

1. **[08-package-structure.md](./08-package-structure.md)** - Project structure, organization, and conventions
2. **[09-testing-strategy.md](./09-testing-strategy.md)** - Testing approach, utilities, and best practices
3. **[10-performance.md](./10-performance.md)** - Performance goals, benchmarks, and optimization strategies

## Implementation Roadmap

### Phase 1-4: Core Modules (✅ COMPLETED - v2.0.0-alpha)

- ✅ Service container & DI (`common/container`, `foundation`)
- ✅ Service providers (`foundation`)
- ✅ Configuration management (`common/config`)
- ✅ Application lifecycle (`foundation`)
- ✅ HTTP server with routing (`http`)
- ✅ Middleware stack (`http/middleware`)
- ✅ Request validation (`validation`)
- ✅ Error handling (`common/errors`)
- ✅ Structured logging (`common/slog`)
- ✅ GORM integration (`database`)
- ✅ Active Record ORM (`database/orm`)
- ✅ Model relationships (`database/orm`)
- ✅ Soft deletes & scopes (`database/orm`)
- ✅ CLI framework (`cli`)
- ✅ Code generators (`cli/generators`)
- ✅ Project scaffolding (`cli`)

### Phase 5: Authentication (Planned - v2.1.0)

- JWT & sessions
- OAuth2 providers
- Policies & gates
- User authentication

### Phase 6: Queues & Scheduling (Planned - v2.2.0)

- Queue system
- Multiple drivers
- Job chaining
- Task scheduler

### Phase 7: Cache & Storage (Planned - v2.3.0)

- Cache drivers
- File storage
- Cloud storage

### Phase 8: Developer Experience (Planned - v2.4.0)

- Collections API
- Factories & seeders
- Enhanced testing utilities

## Reading Guide

### For Framework Users

Start with:

1. `00-overview.md` - Understand the vision and architecture
2. [Root README](../README.md) - Module overview and quick start
3. [MIGRATION.md](../MIGRATION.md) - Upgrading from v1.x to v2.0.0
4. Module READMEs - Detailed documentation:
   - [http](../http/README.md) - HTTP server
   - [database](../database/README.md) - Database & ORM
   - [foundation](../foundation/README.md) - DI framework
   - [common](../common/README.md) - Utilities
5. `04-authentication.md` - Future auth system (planned)

### For Contributors

Read in order:

1. [RESTRUCTURE-SUMMARY.md](./RESTRUCTURE-SUMMARY.md) - Understand v2.0 modularization
2. All specifications to understand the complete picture
3. Module READMEs for implementation details
4. Pick a feature from planned phases and contribute

### For Architects

Focus on:

1. `00-overview.md` - Architecture decisions
2. [RESTRUCTURE-SUMMARY.md](./RESTRUCTURE-SUMMARY.md) - Modular design rationale
3. `01-foundation.md` - Core patterns (note: now split across modules)
4. Module dependency graph (see above)

## Status

### v2.0.0-alpha Status (Current Release)

**Core Modules (✅ Complete)**

- ✅ HTTP Server - Routing, middleware, context abstraction
- ✅ Common Utilities - Errors, logging, config, DI container
- ✅ Foundation - ServiceProvider pattern, application lifecycle
- ✅ Database - GORM integration, ORM, relationships, soft deletes
- ✅ Validation - Request validation with i18n
- ✅ CLI - Code generators, scaffolding

**Documentation (✅ Complete)**

- ✅ Root README.md - Modular architecture overview
- ✅ MIGRATION.md - v1.x → v2.0.0 upgrade guide
- ✅ http/README.md - HTTP server documentation (~1,200 lines)
- ✅ common/README.md - Utilities documentation (~900 lines)
- ✅ foundation/README.md - DI framework documentation (~850 lines)
- ✅ database/README.md - Database & ORM documentation (~1,100 lines)
- ✅ database/orm/README.md - ORM detailed guide
- ✅ RESTRUCTURE-SUMMARY.md - Complete modularization history

### Core Specifications (✅ Complete)

- ✅ 00-overview.md - Architecture decisions and vision
- ✅ 01-foundation.md - Service container, providers, configuration  
  _(Note: Implementation now split across `common/container` and `foundation`)_
- ✅ 02-database.md - GORM integration, ORM, migrations  
  _(Note: Implemented in `database` module)_
- ✅ 03-cli.md - CLI tool and code generators  
  _(Note: Implemented in `cli` module)_
- ✅ 04-authentication.md - Auth system, JWT, sessions, OAuth2 _(Planned)_
- ✅ 05-queue-scheduling.md - Queues and task scheduling _(Planned)_
- ✅ 06-cache-storage.md - Cache and file storage _(Planned)_
- ✅ 07-developer-experience.md - Collections, factories, testing _(Planned)_

### Supporting Documentation (Planned ⏳)

- ⏳ 08-package-structure.md - Package organization _(Superseded by module READMEs)_
- ⏳ 09-testing-strategy.md - Testing approach
- ⏳ 10-performance.md - Performance benchmarks

### Implementation Status

- 🎉 **v2.0.0-alpha Released**: Fully modular architecture complete
- ✅ **Core Functionality**: HTTP, Database, ORM, DI, CLI all working
- ✅ **Documentation**: Comprehensive README for each module (~4,000 lines total)
- 📝 **Next Priority**: Begin Phase 5 (Authentication) after v2.0.0 stable release
- 🚀 **Release Plan**:
  - v2.0.0-beta (after community feedback)
  - v2.0.0 stable
  - v2.1.0 (Authentication)
  - v2.2.0 (Queues & Scheduling)
  - v2.3.0 (Cache & Storage)

## Contributing

When adding or modifying specifications:

1. **Be Detailed**: Include code examples, API designs, and rationale
2. **Be Practical**: Focus on real-world use cases
3. **Be Consistent**: Follow Go idioms and established patterns
4. **Cross-Reference**: Link to related specifications
5. **Include Tests**: Show how features will be tested
6. **Document Migrations**: Explain backward compatibility

## Questions or Feedback

Open an issue in the main repository with the `spec` label.
