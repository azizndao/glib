# Glib Storage

A flexible, modular file storage abstraction for Go with support for multiple drivers (Local, S3/MinIO, and more).

## Features

- **Multiple Storage Drivers** - Local filesystem, S3-compatible storage (AWS S3, MinIO, DigitalOcean Spaces, etc.)
- **Unified Interface** - Same API across all drivers
- **Temporary URLs** - Generate signed URLs with expiration for secure file access
- **Visibility Control** - Public/private file access management
- **Streaming Support** - Efficient handling of large files
- **Directory Operations** - Create, list, and delete directories
- **Zero Dependency Bloat** - Core module has no driver dependencies

## Architecture

The storage module follows a modular architecture similar to Go's `database/sql`:

```
storage/          # Core module (interfaces only)
storage/local/    # Local filesystem driver
storage/s3/       # S3-compatible driver (MinIO SDK)
```

Applications import only the drivers they need, avoiding unnecessary dependencies.

## Installation

### Core Module

```bash
go get github.com/azizndao/glib/storage
```

### Local Storage Driver

```bash
go get github.com/azizndao/glib/storage/local
```

### S3 Storage Driver

```bash
go get github.com/azizndao/glib/storage/s3
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "strings"
    
    "github.com/azizndao/glib/storage"
    "github.com/azizndao/glib/storage/local"
)

func main() {
    ctx := context.Background()
    
    // Create storage manager
    manager := storage.NewManager()
    
    // Register local disk
    manager.RegisterDisk("local", func() storage.Storage {
        disk, _ := local.New(local.Options{
            Root:       "storage/app",
            BaseURL:    "http://localhost:8080/storage",
            URLSecret:  "your-secret-key",
            CreateRoot: true,
        })
        return disk
    })
    
    // Get disk
    disk, err := manager.Disk("local")
    if err != nil {
        log.Fatal(err)
    }
    
    // Store a file
    content := strings.NewReader("Hello, World!")
    if err := disk.Put(ctx, "hello.txt", content); err != nil {
        log.Fatal(err)
    }
    
    // Retrieve a file
    data, err := disk.Get(ctx, "hello.txt")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(data)) // Output: Hello, World!
    
    // Generate a temporary URL (expires in 1 hour)
    url, err := disk.TemporaryURL(ctx, "hello.txt", time.Now().Add(1*time.Hour))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Temporary URL:", url)
}
```

## Driver Configuration

### Local Storage

```go
import "github.com/azizndao/glib/storage/local"

disk, err := local.New(local.Options{
    Root:              "storage/app",           // Base directory (required)
    BaseURL:           "http://localhost:8080", // Base URL for file access
    URLSecret:         "secret-key",            // Secret for signing temporary URLs
    CreateRoot:        true,                    // Create root dir if missing
    DefaultVisibility: storage.VisibilityPrivate,
    PermissionsPublic: 0644,                   // File permissions for public files
    PermissionsPrivate: 0600,                   // File permissions for private files
})
```

### S3-Compatible Storage

```go
import "github.com/azizndao/glib/storage/s3"

// AWS S3
disk, err := s3.New(s3.Options{
    Endpoint:  "s3.us-east-1.amazonaws.com",
    AccessKey: "your-access-key",
    SecretKey: "your-secret-key",
    Bucket:    "my-bucket",
    Region:    "us-east-1",
    UseSSL:    true,
})

// MinIO
disk, err := s3.New(s3.Options{
    Endpoint:  "minio.example.com:9000",
    AccessKey: "minioadmin",
    SecretKey: "minioadmin",
    Bucket:    "uploads",
    UseSSL:    false, // or true if using HTTPS
})

// DigitalOcean Spaces
disk, err := s3.New(s3.Options{
    Endpoint:  "nyc3.digitaloceanspaces.com",
    AccessKey: "your-access-key",
    SecretKey: "your-secret-key",
    Bucket:    "my-space",
    Region:    "nyc3",
    UseSSL:    true,
})
```

## API Reference

### Storage Interface

```go
type Storage interface {
    // Basic Operations
    Put(ctx context.Context, path string, contents io.Reader) error
    PutFile(ctx context.Context, path string, file *File) error
    Get(ctx context.Context, path string) ([]byte, error)
    GetStream(ctx context.Context, path string) (io.ReadCloser, error)
    
    // File Information
    Exists(ctx context.Context, path string) (bool, error)
    Missing(ctx context.Context, path string) (bool, error)
    Size(ctx context.Context, path string) (int64, error)
    LastModified(ctx context.Context, path string) (time.Time, error)
    
    // File Operations
    Delete(ctx context.Context, path string) error
    DeleteDirectory(ctx context.Context, path string) error
    Copy(ctx context.Context, from, to string) error
    Move(ctx context.Context, from, to string) error
    
    // Directory Operations
    Files(ctx context.Context, directory string) ([]string, error)
    AllFiles(ctx context.Context, directory string) ([]string, error)
    Directories(ctx context.Context, directory string) ([]string, error)
    AllDirectories(ctx context.Context, directory string) ([]string, error)
    MakeDirectory(ctx context.Context, path string) error
    
    // URLs
    URL(ctx context.Context, path string) (string, error)
    TemporaryURL(ctx context.Context, path string, expiration time.Time) (string, error)
    
    // Visibility
    SetVisibility(ctx context.Context, path string, visibility Visibility) error
    GetVisibility(ctx context.Context, path string) (Visibility, error)
}
```

## Usage Examples

### Storing Files

```go
// From io.Reader
content := strings.NewReader("file content")
disk.Put(ctx, "documents/file.txt", content)

// From File struct (with metadata)
file := storage.NewFile("avatar.jpg", 1024, reader)
file.WithVisibility(storage.VisibilityPublic)
file.WithMetadata("user_id", "123")

disk.PutFile(ctx, "avatars/user-123.jpg", file)

// From multipart form upload
formFile, _ := c.FormFile("upload")
file, _ := storage.NewFileFromMultipart(formFile)
disk.PutFile(ctx, "uploads/"+file.Name, file)
```

### Retrieving Files

```go
// Get entire file as bytes
data, err := disk.Get(ctx, "documents/file.txt")

// Stream large files
stream, err := disk.GetStream(ctx, "videos/movie.mp4")
defer stream.Close()

// Copy to writer
io.Copy(outputWriter, stream)
```

### File Operations

```go
// Check existence
exists, err := disk.Exists(ctx, "file.txt")

// Get file info
size, _ := disk.Size(ctx, "file.txt")
modTime, _ := disk.LastModified(ctx, "file.txt")

// Copy file
disk.Copy(ctx, "file.txt", "backup/file.txt")

// Move file
disk.Move(ctx, "temp/file.txt", "permanent/file.txt")

// Delete file
disk.Delete(ctx, "file.txt")

// Delete entire directory
disk.DeleteDirectory(ctx, "temp")
```

### Directory Operations

```go
// List files (non-recursive)
files, err := disk.Files(ctx, "documents")

// List all files recursively
allFiles, err := disk.AllFiles(ctx, "documents")

// List directories
dirs, err := disk.Directories(ctx, "uploads")

// Create directory
disk.MakeDirectory(ctx, "uploads/2024/01")
```

### URL Generation

```go
// Public URL
url, err := disk.URL(ctx, "images/logo.png")
// Returns: http://localhost:8080/storage/images/logo.png

// Temporary signed URL (expires in 1 hour)
tempURL, err := disk.TemporaryURL(ctx, "private/document.pdf", time.Now().Add(1*time.Hour))
// Returns: http://localhost:8080/storage/private/document.pdf?expires=1234567890&signature=abc123...
```

### Visibility Management

```go
// Set file visibility
disk.SetVisibility(ctx, "file.txt", storage.VisibilityPublic)
disk.SetVisibility(ctx, "secret.txt", storage.VisibilityPrivate)

// Get file visibility
visibility, err := disk.GetVisibility(ctx, "file.txt")
if visibility == storage.VisibilityPublic {
    fmt.Println("File is publicly accessible")
}
```

## HTTP File Uploads

```go
import (
    "github.com/azizndao/glib/http"
    "github.com/azizndao/glib/storage"
)

func UploadHandler(c *http.Context) error {
    // Get uploaded file
    formFile, err := c.FormFile("file")
    if err != nil {
        return err
    }
    
    // Convert to storage.File
    file, err := storage.NewFileFromMultipart(formFile)
    if err != nil {
        return err
    }
    
    // Set visibility
    file.WithVisibility(storage.VisibilityPublic)
    
    // Store file
    disk, _ := manager.Disk("local")
    err = disk.PutFile(c.Request.Context(), "uploads/"+file.Name, file)
    if err != nil {
        return err
    }
    
    // Generate URL
    url, _ := disk.URL(c.Request.Context(), "uploads/"+file.Name)
    
    return c.JSON(200, map[string]string{
        "url": url,
    })
}
```

## Temporary URL Validation

For local storage, you'll need to validate signed URLs in your HTTP handler:

```go
import "github.com/azizndao/glib/storage/internal"

func ServeFile(c *http.Context) error {
    path := c.Param("path")
    expires := c.Query("expires")
    signature := c.Query("signature")
    
    // Validate signature if present
    if signature != "" {
        valid, err := internal.VerifySignature(path, expires, signature, "your-secret-key")
        if err != nil || !valid {
            return c.JSON(403, map[string]string{"error": "Invalid or expired URL"})
        }
    }
    
    // Serve file
    disk, _ := manager.Disk("local")
    stream, err := disk.GetStream(c.Request.Context(), path)
    if err != nil {
        return c.JSON(404, map[string]string{"error": "File not found"})
    }
    defer stream.Close()
    
    return c.Stream(200, "application/octet-stream", stream)
}
```

## Driver Comparison

| Feature                | Local | S3/MinIO |
|------------------------|-------|----------|
| File Operations        | ✅     | ✅        |
| Streaming              | ✅     | ✅        |
| Directory Operations   | ✅     | ✅*       |
| Temporary URLs         | ✅     | ✅        |
| Visibility Control     | ✅     | ⚠️**      |
| Metadata               | ❌     | ✅        |
| Large Files (>5GB)     | ✅     | ✅        |

\* S3 doesn't have true directories, but they're simulated with prefixes  
\*\* MinIO SDK has limited ACL support; visibility management is simplified

## Performance Tips

1. **Use Streaming for Large Files** - Avoid loading entire files into memory
2. **Batch Operations** - Group multiple file operations when possible
3. **Connection Pooling** - MinIO client handles this automatically
4. **Temporary URLs** - Use signed URLs instead of proxying large files through your app

## Error Handling

```go
data, err := disk.Get(ctx, "file.txt")
if err != nil {
    if errors.Is(err, storage.ErrFileNotFound) {
        // Handle missing file
    } else if errors.Is(err, storage.ErrPermissionDenied) {
        // Handle permission error
    } else {
        // Handle other errors
    }
}
```

## Testing

```go
import "github.com/azizndao/glib/storage/local"

func TestFileStorage(t *testing.T) {
    // Use temporary directory for tests
    tmpDir := t.TempDir()
    
    disk, err := local.New(local.Options{
        Root:       tmpDir,
        CreateRoot: true,
    })
    require.NoError(t, err)
    
    // Test file operations
    ctx := context.Background()
    err = disk.Put(ctx, "test.txt", strings.NewReader("content"))
    assert.NoError(t, err)
}
```

## Migration from Monolithic Storage

If you're migrating from a monolithic storage package:

**Before:**
```go
import "yourapp/storage"

disk := storage.NewLocal("/path/to/storage")
```

**After:**
```go
import (
    "github.com/azizndao/glib/storage"
    "github.com/azizndao/glib/storage/local"
)

manager := storage.NewManager()
manager.RegisterDisk("local", func() storage.Storage {
    disk, _ := local.New(local.Options{Root: "/path/to/storage"})
    return disk
})
disk, _ := manager.Disk("local")
```

## Contributing

Contributions are welcome! Please see the [contributing guide](../../CONTRIBUTING.md).

## License

MIT License - see [LICENSE](../../LICENSE) for details.
