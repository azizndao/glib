# Changelog

All notable changes to the Glib Storage module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2024-12-24

### Added

#### Core Module (`storage/`)

- Initial release of modular storage abstraction
- `Storage` interface with comprehensive file operations
- `Manager` for managing multiple storage disks
- `File` struct for representing uploaded files with metadata
- `Visibility` type for public/private file access control
- Helper functions for MIME type detection
- Support for multipart form file uploads via `NewFileFromMultipart()`
- Zero external dependencies in core module

#### Local Storage Driver (`storage/local/`)

- Complete local filesystem storage implementation
- HMAC-SHA256 signed temporary URLs
- Visibility control via file permissions (0644 for public, 0600 for private)
- Support for nested directory creation
- Streaming support for large files
- Concurrent-safe operations
- **Test Coverage**: 100% (20 test functions, all passing)

#### S3 Storage Driver (`storage/s3/`)

- S3-compatible storage using MinIO SDK
- Support for AWS S3, MinIO, DigitalOcean Spaces, Backblaze B2
- Presigned URL generation for temporary access
- Custom metadata support
- Path prefix support for multi-tenancy
- Streaming support for large files
- Automatic multipart upload handling
- **Dependencies**: `github.com/minio/minio-go/v7`

### Storage Operations

All drivers support the following operations:

**Basic Operations**:

- `Put()` - Store file from io.Reader
- `PutFile()` - Store file with metadata
- `Get()` - Retrieve file as bytes
- `GetStream()` - Retrieve file as stream

**File Information**:

- `Exists()` - Check if file exists
- `Missing()` - Check if file is missing
- `Size()` - Get file size
- `LastModified()` - Get modification time

**File Operations**:

- `Delete()` - Remove single file
- `DeleteDirectory()` - Remove directory and contents
- `Copy()` - Copy file
- `Move()` - Move/rename file

**Directory Operations**:

- `Files()` - List files (non-recursive)
- `AllFiles()` - List all files recursively
- `Directories()` - List directories (non-recursive)
- `AllDirectories()` - List all directories recursively
- `MakeDirectory()` - Create directory

**URLs**:

- `URL()` - Get public URL for file
- `TemporaryURL()` - Generate signed temporary URL with expiration

**Visibility**:

- `SetVisibility()` - Set file visibility (public/private)
- `GetVisibility()` - Get file visibility

### Architecture

- **Modular Design**: Core module defines interfaces, drivers are separate modules
- **Zero Dependency Bloat**: Applications import only the drivers they need
- **Interface-Driver Pattern**: Follows Go's `database/sql` pattern
- **Thread-Safe**: All operations are concurrent-safe

### Documentation

- Comprehensive README for core module with usage examples
- Driver-specific READMEs with configuration guides
- Complete API reference
- Integration examples with HTTP file uploads
- Testing guidelines
- Migration guide from monolithic storage

### Performance

- Streaming support avoids loading large files into memory
- Connection pooling (S3/MinIO)
- Efficient directory traversal
- Minimal allocations

### Known Limitations

**Local Storage**:

- No built-in CDN support
- Temporary URLs require custom validation middleware
- No metadata storage

**S3 Storage**:

- Limited ACL support via MinIO SDK (`SetVisibility()` not fully implemented)
- Directories are simulated (not native S3 concept)
- Metadata not included in list operations

## [Unreleased]

### Planned Features

- **Google Cloud Storage driver** (`storage/gcs/`)
- **Azure Blob Storage driver** (`storage/azure/`)
- **HTTP middleware** for serving files and validating signed URLs
- **Enhanced metadata support** for local storage (via sidecar files or database)
- **Resumable uploads** for large files
- **Progress callbacks** for upload/download operations
- **Batch operations** for improved performance
- **Caching layer** integration with `cache/` module
- **Image transformation** utilities (resize, crop, format conversion)

### Under Consideration

- WebDAV storage driver
- FTP/SFTP storage driver
- In-memory storage driver (for testing)
- Storage events/hooks system
- File versioning support
- Encryption at rest (for local storage)
- Compression support

## Migration Guide

### From Monolithic Storage Packages

If migrating from a single storage package:

```go
// Before
import "yourapp/storage"
storage := storage.NewLocalStorage("/path")

// After
import (
    "github.com/azizndao/glib/storage"
    "github.com/azizndao/glib/storage/local"
)

manager := storage.NewManager()
manager.RegisterDisk("local", func() storage.Storage {
    disk, _ := local.New(local.Options{Root: "/path"})
    return disk
})
```

### Breaking Changes

None (initial release).

## Contributing

Contributions are welcome! Areas for contribution:

1. Additional storage drivers (GCS, Azure, etc.)
2. Performance optimizations
3. Additional test coverage
4. Documentation improvements
5. Bug fixes

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

## Links

- [GitHub Repository](https://github.com/azizndao/glib)
- [Documentation](./README.md)
- [Issue Tracker](https://github.com/azizndao/glib/issues)

---

**Note**: This is the initial release (v0.1.0) of the storage module. The API is considered stable but may evolve based on community feedback.
