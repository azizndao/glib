// Package storage provides a flexible file storage abstraction with support
// for multiple drivers (Local, S3, MinIO, DigitalOcean Spaces, etc.).
//
// Example usage:
//
//	// Create storage manager
//	manager := storage.NewManager()
//	manager.RegisterDisk("local", func() storage.Storage {
//	    return local.New(local.Options{
//	        Root:    "storage/app",
//	        BaseURL: "http://localhost:8080/storage",
//	    })
//	})
//
//	// Get disk
//	disk, _ := manager.Disk("local")
//
//	// Store file
//	err := disk.Put(ctx, "avatars/user.jpg", file)
//
//	// Get file
//	data, _ := disk.Get(ctx, "avatars/user.jpg")
//
//	// Generate temporary URL
//	url, _ := disk.TemporaryURL(ctx, "avatars/user.jpg", time.Now().Add(1*time.Hour))
//
//	// List files
//	files, _ := disk.Files(ctx, "avatars")
package storage

import (
	"context"
	"io"
	"time"
)

// Storage represents a file storage driver.
type Storage interface {
	// Put stores a file
	Put(ctx context.Context, path string, contents io.Reader) error

	// PutFile stores a file with metadata
	PutFile(ctx context.Context, path string, file *File) error

	// Get retrieves file contents
	Get(ctx context.Context, path string) ([]byte, error)

	// GetStream retrieves file as a stream
	GetStream(ctx context.Context, path string) (io.ReadCloser, error)

	// Exists checks if file exists
	Exists(ctx context.Context, path string) (bool, error)

	// Missing checks if file doesn't exist
	Missing(ctx context.Context, path string) (bool, error)

	// Size returns file size in bytes
	Size(ctx context.Context, path string) (int64, error)

	// LastModified returns file modification time
	LastModified(ctx context.Context, path string) (time.Time, error)

	// Delete removes a file
	Delete(ctx context.Context, path string) error

	// DeleteDirectory removes a directory and all its contents
	DeleteDirectory(ctx context.Context, path string) error

	// Copy copies a file
	Copy(ctx context.Context, from, to string) error

	// Move moves a file
	Move(ctx context.Context, from, to string) error

	// Files lists all files in a directory (non-recursive)
	Files(ctx context.Context, directory string) ([]string, error)

	// AllFiles lists all files recursively
	AllFiles(ctx context.Context, directory string) ([]string, error)

	// Directories lists all directories (non-recursive)
	Directories(ctx context.Context, directory string) ([]string, error)

	// AllDirectories lists all directories recursively
	AllDirectories(ctx context.Context, directory string) ([]string, error)

	// MakeDirectory creates a directory
	MakeDirectory(ctx context.Context, path string) error

	// URL gets the URL for a file
	URL(ctx context.Context, path string) (string, error)

	// TemporaryURL generates a temporary URL with expiration
	TemporaryURL(ctx context.Context, path string, expiration time.Time) (string, error)

	// SetVisibility sets file visibility (public/private)
	SetVisibility(ctx context.Context, path string, visibility Visibility) error

	// GetVisibility gets file visibility
	GetVisibility(ctx context.Context, path string) (Visibility, error)
}

// Driver is a factory function that creates a storage instance.
type Driver func() Storage

// FileInfo represents file metadata.
type FileInfo struct {
	Path         string
	Size         int64
	MimeType     string
	LastModified time.Time
	Visibility   Visibility
	Metadata     map[string]string
}
