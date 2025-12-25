// Package local implements a local filesystem storage driver with support
// for signed URLs, visibility management, and comprehensive file operations.
package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/azizndao/glib/storage"
	"github.com/azizndao/glib/storage/internal"
)

// Options configures the local storage driver.
type Options struct {
	// Root is the base directory for file storage
	Root string

	// BaseURL is the base URL for accessing files (e.g., http://localhost:8080/storage)
	BaseURL string

	// URLSecret is used for signing temporary URLs (required for TemporaryURL)
	URLSecret string

	// DefaultVisibility sets the default visibility for new files
	DefaultVisibility storage.Visibility

	// CreateRoot creates the root directory if it doesn't exist
	CreateRoot bool

	// PermissionsPublic sets file permissions for public files (default: 0644)
	PermissionsPublic os.FileMode

	// PermissionsPrivate sets file permissions for private files (default: 0600)
	PermissionsPrivate os.FileMode
}

// LocalStorage implements local filesystem storage.
type LocalStorage struct {
	root               string
	baseURL            string
	urlSecret          string
	defaultVisibility  storage.Visibility
	permissionsPublic  os.FileMode
	permissionsPrivate os.FileMode
}

// New creates a new local storage driver.
func New(opts Options) (*LocalStorage, error) {
	// Validate root path
	if opts.Root == "" {
		return nil, fmt.Errorf("storage/local: root path is required")
	}

	// Create root directory if needed
	if opts.CreateRoot {
		if err := os.MkdirAll(opts.Root, 0755); err != nil {
			return nil, fmt.Errorf("storage/local: failed to create root directory: %w", err)
		}
	}

	// Check if root exists
	if _, err := os.Stat(opts.Root); err != nil {
		return nil, fmt.Errorf("storage/local: root directory does not exist: %w", err)
	}

	// Set defaults
	if opts.DefaultVisibility == "" {
		opts.DefaultVisibility = storage.VisibilityPrivate
	}
	if opts.PermissionsPublic == 0 {
		opts.PermissionsPublic = 0644
	}
	if opts.PermissionsPrivate == 0 {
		opts.PermissionsPrivate = 0600
	}

	return &LocalStorage{
		root:               opts.Root,
		baseURL:            strings.TrimSuffix(opts.BaseURL, "/"),
		urlSecret:          opts.URLSecret,
		defaultVisibility:  opts.DefaultVisibility,
		permissionsPublic:  opts.PermissionsPublic,
		permissionsPrivate: opts.PermissionsPrivate,
	}, nil
}

// Put stores a file.
func (s *LocalStorage) Put(ctx context.Context, path string, contents io.Reader) error {
	fullPath := s.fullPath(path)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("storage/local: failed to create directory: %w", err)
	}

	// Create file with default permissions
	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, s.permissionsPrivate)
	if err != nil {
		return fmt.Errorf("storage/local: failed to create file: %w", err)
	}
	defer file.Close()

	// Copy contents
	if _, err := io.Copy(file, contents); err != nil {
		return fmt.Errorf("storage/local: failed to write file: %w", err)
	}

	return nil
}

// PutFile stores a file with metadata.
func (s *LocalStorage) PutFile(ctx context.Context, path string, file *storage.File) error {
	if err := s.Put(ctx, path, file.Reader); err != nil {
		return err
	}

	// Set visibility
	if file.Visibility != "" {
		if err := s.SetVisibility(ctx, path, file.Visibility); err != nil {
			return err
		}
	}

	return nil
}

// Get retrieves file contents.
func (s *LocalStorage) Get(ctx context.Context, path string) ([]byte, error) {
	data, err := os.ReadFile(s.fullPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.ErrFileNotFound
		}
		return nil, fmt.Errorf("storage/local: failed to read file: %w", err)
	}
	return data, nil
}

// GetStream retrieves file as a stream.
func (s *LocalStorage) GetStream(ctx context.Context, path string) (io.ReadCloser, error) {
	file, err := os.Open(s.fullPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, storage.ErrFileNotFound
		}
		return nil, fmt.Errorf("storage/local: failed to open file: %w", err)
	}
	return file, nil
}

// Exists checks if file exists.
func (s *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	_, err := os.Stat(s.fullPath(path))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("storage/local: failed to check file: %w", err)
}

// Missing checks if file doesn't exist.
func (s *LocalStorage) Missing(ctx context.Context, path string) (bool, error) {
	exists, err := s.Exists(ctx, path)
	return !exists, err
}

// Size returns file size in bytes.
func (s *LocalStorage) Size(ctx context.Context, path string) (int64, error) {
	info, err := os.Stat(s.fullPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, storage.ErrFileNotFound
		}
		return 0, fmt.Errorf("storage/local: failed to stat file: %w", err)
	}
	return info.Size(), nil
}

// LastModified returns file modification time.
func (s *LocalStorage) LastModified(ctx context.Context, path string) (time.Time, error) {
	info, err := os.Stat(s.fullPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, storage.ErrFileNotFound
		}
		return time.Time{}, fmt.Errorf("storage/local: failed to stat file: %w", err)
	}
	return info.ModTime(), nil
}

// Delete removes a file.
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	err := os.Remove(s.fullPath(path))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage/local: failed to delete file: %w", err)
	}
	return nil
}

// DeleteDirectory removes a directory and all its contents.
func (s *LocalStorage) DeleteDirectory(ctx context.Context, path string) error {
	err := os.RemoveAll(s.fullPath(path))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage/local: failed to delete directory: %w", err)
	}
	return nil
}

// Copy copies a file.
func (s *LocalStorage) Copy(ctx context.Context, from, to string) error {
	source, err := os.Open(s.fullPath(from))
	if err != nil {
		if os.IsNotExist(err) {
			return storage.ErrFileNotFound
		}
		return fmt.Errorf("storage/local: failed to open source: %w", err)
	}
	defer source.Close()

	destPath := s.fullPath(to)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("storage/local: failed to create destination directory: %w", err)
	}

	dest, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("storage/local: failed to create destination: %w", err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, source); err != nil {
		return fmt.Errorf("storage/local: failed to copy file: %w", err)
	}

	return nil
}

// Move moves a file.
func (s *LocalStorage) Move(ctx context.Context, from, to string) error {
	fromPath := s.fullPath(from)
	toPath := s.fullPath(to)

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(toPath), 0755); err != nil {
		return fmt.Errorf("storage/local: failed to create destination directory: %w", err)
	}

	err := os.Rename(fromPath, toPath)
	if err != nil {
		if os.IsNotExist(err) {
			return storage.ErrFileNotFound
		}
		return fmt.Errorf("storage/local: failed to move file: %w", err)
	}

	return nil
}

// Files lists all files in a directory (non-recursive).
func (s *LocalStorage) Files(ctx context.Context, directory string) ([]string, error) {
	fullPath := s.fullPath(directory)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("storage/local: failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(directory, entry.Name()))
		}
	}

	return files, nil
}

// AllFiles lists all files recursively.
func (s *LocalStorage) AllFiles(ctx context.Context, directory string) ([]string, error) {
	var files []string
	fullPath := s.fullPath(directory)

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(s.root, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}

		return nil
	})

	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("storage/local: failed to walk directory: %w", err)
	}

	return files, nil
}

// Directories lists all directories (non-recursive).
func (s *LocalStorage) Directories(ctx context.Context, directory string) ([]string, error) {
	fullPath := s.fullPath(directory)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("storage/local: failed to read directory: %w", err)
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(directory, entry.Name()))
		}
	}

	return dirs, nil
}

// AllDirectories lists all directories recursively.
func (s *LocalStorage) AllDirectories(ctx context.Context, directory string) ([]string, error) {
	var dirs []string
	fullPath := s.fullPath(directory)

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && path != fullPath {
			relPath, err := filepath.Rel(s.root, path)
			if err != nil {
				return err
			}
			dirs = append(dirs, relPath)
		}

		return nil
	})

	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("storage/local: failed to walk directory: %w", err)
	}

	return dirs, nil
}

// MakeDirectory creates a directory.
func (s *LocalStorage) MakeDirectory(ctx context.Context, path string) error {
	err := os.MkdirAll(s.fullPath(path), 0755)
	if err != nil {
		return fmt.Errorf("storage/local: failed to create directory: %w", err)
	}
	return nil
}

// URL gets the URL for a file.
func (s *LocalStorage) URL(ctx context.Context, path string) (string, error) {
	if s.baseURL == "" {
		return "", storage.ErrNotImplemented
	}
	return fmt.Sprintf("%s/%s", s.baseURL, strings.TrimPrefix(path, "/")), nil
}

// TemporaryURL generates a temporary URL with expiration.
func (s *LocalStorage) TemporaryURL(ctx context.Context, path string, expiration time.Time) (string, error) {
	if s.urlSecret == "" {
		return "", fmt.Errorf("storage/local: URLSecret is required for temporary URLs")
	}

	if s.baseURL == "" {
		return "", storage.ErrNotImplemented
	}

	// Generate signature
	signature := internal.GenerateSignature(path, expiration, s.urlSecret)

	// Build URL with query parameters
	url := fmt.Sprintf("%s/%s?expires=%d&signature=%s",
		s.baseURL,
		strings.TrimPrefix(path, "/"),
		expiration.Unix(),
		signature,
	)

	return url, nil
}

// SetVisibility sets file visibility (public/private).
func (s *LocalStorage) SetVisibility(ctx context.Context, path string, visibility storage.Visibility) error {
	fullPath := s.fullPath(path)

	var perms os.FileMode
	if visibility == storage.VisibilityPublic {
		perms = s.permissionsPublic
	} else {
		perms = s.permissionsPrivate
	}

	err := os.Chmod(fullPath, perms)
	if err != nil {
		if os.IsNotExist(err) {
			return storage.ErrFileNotFound
		}
		return fmt.Errorf("storage/local: failed to set visibility: %w", err)
	}

	return nil
}

// GetVisibility gets file visibility.
func (s *LocalStorage) GetVisibility(ctx context.Context, path string) (storage.Visibility, error) {
	info, err := os.Stat(s.fullPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return "", storage.ErrFileNotFound
		}
		return "", fmt.Errorf("storage/local: failed to stat file: %w", err)
	}

	// Check permissions
	mode := info.Mode().Perm()

	// If world-readable, consider it public
	if mode&0004 != 0 {
		return storage.VisibilityPublic, nil
	}

	return storage.VisibilityPrivate, nil
}

// fullPath returns the full filesystem path.
func (s *LocalStorage) fullPath(path string) string {
	return filepath.Join(s.root, path)
}
