# Local Storage Driver

Local filesystem storage driver for Glib Storage with support for signed temporary URLs and visibility management.

## Installation

```bash
go get github.com/azizndao/glib/storage/local
```

## Features

- ✅ File CRUD operations
- ✅ Directory operations
- ✅ Streaming support for large files
- ✅ Signed temporary URLs with HMAC-SHA256
- ✅ Visibility control via file permissions
- ✅ Zero external dependencies (except storage core)

## Usage

```go
import "github.com/azizndao/glib/storage/local"

disk, err := local.New(local.Options{
    Root:              "storage/app",
    BaseURL:           "http://localhost:8080/storage",
    URLSecret:         "your-32-char-secret-key-here",
    CreateRoot:        true,
    DefaultVisibility: storage.VisibilityPrivate,
    PermissionsPublic: 0644,
    PermissionsPrivate: 0600,
})
```

## Configuration Options

| Option               | Type          | Required | Default             | Description                                                           |
| -------------------- | ------------- | -------- | ------------------- | --------------------------------------------------------------------- |
| `Root`               | `string`      | Yes      | -                   | Base directory for file storage                                       |
| `BaseURL`            | `string`      | No       | `""`                | Base URL for file access (e.g., `http://localhost:8080/storage`)      |
| `URLSecret`          | `string`      | No       | `""`                | Secret key for signing temporary URLs (required for `TemporaryURL()`) |
| `CreateRoot`         | `bool`        | No       | `false`             | Create root directory if it doesn't exist                             |
| `DefaultVisibility`  | `Visibility`  | No       | `VisibilityPrivate` | Default visibility for new files                                      |
| `PermissionsPublic`  | `os.FileMode` | No       | `0644`              | File permissions for public files                                     |
| `PermissionsPrivate` | `os.FileMode` | No       | `0600`              | File permissions for private files                                    |

## Signed Temporary URLs

The local driver generates HMAC-SHA256 signed URLs for temporary file access:

```go
// Generate temporary URL (expires in 1 hour)
url, err := disk.TemporaryURL(ctx, "private/document.pdf", time.Now().Add(1*time.Hour))
// Returns: http://localhost:8080/storage/private/document.pdf?expires=1234567890&signature=abc123...
```

### Validating Signed URLs

You'll need to validate signed URLs in your HTTP handler:

```go
import "github.com/azizndao/glib/storage/internal"

func ServeFile(w http.ResponseWriter, r *http.Request) {
    path := r.URL.Path
    expires := r.URL.Query().Get("expires")
    signature := r.URL.Query().Get("signature")

    if signature != "" {
        valid, err := internal.VerifySignature(path, expires, signature, "your-secret-key")
        if err != nil || !valid {
            http.Error(w, "Invalid or expired URL", http.StatusForbidden)
            return
        }
    }

    // Serve file...
}
```

## Visibility Management

Files can be public or private:

```go
// Public files (0644) - readable by everyone
disk.SetVisibility(ctx, "public/logo.png", storage.VisibilityPublic)

// Private files (0600) - readable only by owner
disk.SetVisibility(ctx, "private/secret.txt", storage.VisibilityPrivate)
```

## Performance

- **Streaming**: Large files are streamed efficiently without loading into memory
- **Concurrency**: All operations are thread-safe
- **Disk I/O**: Uses standard Go file operations for maximum compatibility

## Limitations

- No built-in CDN support (use reverse proxy like nginx)
- Temporary URLs require custom middleware for validation
- Metadata storage not supported (consider using a database for metadata)

## Example: Complete Setup

```go
package main

import (
    "context"
    "log"
    "net/http"
    "strings"
    "time"

    "github.com/azizndao/glib/storage"
    "github.com/azizndao/glib/storage/local"
    "github.com/azizndao/glib/storage/internal"
)

func main() {
    // Create disk
    disk, err := local.New(local.Options{
        Root:       "./storage",
        BaseURL:    "http://localhost:8080/files",
        URLSecret:  "your-secret-key-min-32-chars",
        CreateRoot: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    // HTTP handler for serving files
    http.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
        path := strings.TrimPrefix(r.URL.Path, "/files/")
        expires := r.URL.Query().Get("expires")
        signature := r.URL.Query().Get("signature")

        // Validate signature if present
        if signature != "" {
            valid, _ := internal.VerifySignature(path, expires, signature, "your-secret-key-min-32-chars")
            if !valid {
                http.Error(w, "Invalid URL", http.StatusForbidden)
                return
            }
        }

        // Stream file
        ctx := r.Context()
        stream, err := disk.GetStream(ctx, path)
        if err != nil {
            http.Error(w, "File not found", http.StatusNotFound)
            return
        }
        defer stream.Close()

        http.ServeContent(w, r, path, time.Now(), stream.(io.ReadSeeker))
    })

    // Store a file
    ctx := context.Background()
    content := strings.NewReader("Hello, World!")
    if err := disk.Put(ctx, "hello.txt", content); err != nil {
        log.Fatal(err)
    }

    // Generate temporary URL
    url, _ := disk.TemporaryURL(ctx, "hello.txt", time.Now().Add(1*time.Hour))
    log.Println("Temporary URL:", url)

    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Testing

```go
func TestLocalStorage(t *testing.T) {
    disk, err := local.New(local.Options{
        Root:       t.TempDir(),
        CreateRoot: true,
    })
    require.NoError(t, err)

    ctx := context.Background()

    // Test file operations
    err = disk.Put(ctx, "test.txt", strings.NewReader("content"))
    assert.NoError(t, err)

    data, err := disk.Get(ctx, "test.txt")
    assert.NoError(t, err)
    assert.Equal(t, []byte("content"), data)
}
```

## See Also

- [Storage Core Documentation](../README.md)
- [S3 Driver Documentation](../s3/README.md)
