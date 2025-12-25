# Changelog

All notable changes to the Glib Queue package will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of queue package
- Laravel-style fluent API for job dispatching
- Job interface with retry, timeout, and failure handling
- Queue Manager with driver pattern support
- Redis driver using Asynq
- Worker implementation with concurrent job processing
- Job serialization using GOB encoding
- Job registration system
- Delayed job execution
- Job chaining support
- Job batching support
- Unique job support via UniqueJob interface
- Event system for job lifecycle hooks
- Queue statistics and monitoring
- Graceful worker shutdown
- Configurable queue priorities
- Job middleware support
- Example application demonstrating all features
- Comprehensive documentation

### Features

#### Core
- `Job` interface for defining background jobs
- `BaseJob` struct providing sensible defaults
- `Manager` for managing multiple queue connections
- `Queue` interface for queue operations
- `Worker` for processing queued jobs

#### Job Features
- Automatic retry with configurable attempts
- Custom retry delay strategies
- Job timeout support
- Job-specific queue routing
- Failure callbacks
- Conditional job execution
- Job release (re-queue) support

#### Dispatcher API
- `Dispatch()` - Fluent job dispatcher
- `OnQueue()` - Specify queue name
- `OnConnection()` - Specify connection
- `Delay()` - Delay job execution
- `DelayUntil()` - Schedule for specific time
- `MaxRetries()` - Configure retry attempts
- `Timeout()` - Set job timeout
- `WithTaskID()` - Custom task identifier
- `InGroup()` - Group related jobs
- `DispatchIf()` - Conditional dispatch
- `DispatchUnless()` - Inverted conditional dispatch
- `DispatchSync()` - Synchronous execution for testing

#### Job Patterns
- Job chaining with `NewChain()`
- Job batching with `NewBatch()`
- Unique jobs with `UniqueJob` interface
- Job middleware with `Middleware` interface

#### Worker Features
- Concurrent job processing
- Queue priority support (weighted and strict modes)
- Graceful shutdown with timeout
- Health checks
- Custom retry delay functions
- Failure detection functions
- Job middleware pipeline

#### Monitoring
- Queue statistics (pending, active, scheduled, failed counts)
- Job state tracking
- Processing metrics
- Integration with Asynq monitoring tools

#### Events
- `job.dispatched` - When a job is queued
- `job.processing` - When a job starts processing
- `job.processed` - When a job completes successfully
- `job.failed` - When a job fails
- `job.retrying` - When a job is being retried

### Documentation
- Comprehensive README with examples
- Quick start guide
- API documentation
- Best practices guide
- Production deployment guide
- Troubleshooting guide
- Example application with multiple job types

## [0.1.0] - 2024-12-25

### Added
- Initial development release
- Core queue functionality
- Asynq integration
- Basic job dispatching and processing
- Worker implementation
- Documentation

[Unreleased]: https://github.com/azizndao/glib/compare/queue-v0.1.0...HEAD
[0.1.0]: https://github.com/azizndao/glib/releases/tag/queue-v0.1.0
