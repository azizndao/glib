// Package scanner provides code analysis and project scanning capabilities for the glib framework.
// It walks Go source files, extracts annotations, and builds a structured representation of
// controllers, providers, handlers, middleware, and configuration types.
package scanner

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/azizndao/glib/internal/cache"
)

// FileCache manages cached scan results for incremental scanning
type FileCache struct {
	*cache.Cache[string, *CacheEntry]        // Embedded generic cache
	cacheDir                          string // Where to persist cache
}

// CacheEntry stores cached scan data for a single file
type CacheEntry struct {
	Hash        string    // SHA256 of file content
	ModTime     time.Time // Last modified time
	LastScan    time.Time // When we scanned it
	Controllers []*Controller
	Providers   []*Provider
	Middleware  []*Middleware
	Configs     []*Config
}

// NewFileCache creates a new file cache
func NewFileCache(cacheDir string) *FileCache {
	cachePath := filepath.Join(cacheDir, "scan.cache")
	return &FileCache{
		Cache:    cache.New[string, *CacheEntry](cachePath),
		cacheDir: cacheDir,
	}
}

// Get retrieves a cached entry if valid (with modTime check)
func (c *FileCache) Get(filePath string, modTime time.Time) (*CacheEntry, bool) {
	entry, exists := c.Cache.Get(filePath)
	if !exists {
		return nil, false
	}

	// Check if file was modified since cache
	if !entry.ModTime.Equal(modTime) {
		return nil, false
	}

	return entry, true
}

// GetByHash retrieves a cached entry by content hash (more accurate)
func (c *FileCache) GetByHash(filePath string, hash string) (*CacheEntry, bool) {
	entry, exists := c.Cache.Get(filePath)
	if !exists {
		return nil, false
	}

	if entry.Hash != hash {
		return nil, false
	}

	return entry, true
}

// Invalidate removes a cache entry
func (c *FileCache) Invalidate(filePath string) {
	c.Delete(filePath)
}

// computeFileHash computes SHA256 hash of file content
func computeFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// isFileCached checks if a file is cached and valid
func (c *FileCache) isFileCached(filePath string) (bool, string, error) {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return false, "", err
	}

	// Quick check: mod time
	if entry, ok := c.Get(filePath, info.ModTime()); ok {
		// Double check with hash for accuracy
		hash, err := computeFileHash(filePath)
		if err != nil {
			return false, "", err
		}

		if entry.Hash == hash {
			return true, hash, nil
		}
	}

	// Compute hash for new entry
	hash, err := computeFileHash(filePath)
	if err != nil {
		return false, "", err
	}

	// Check if hash matches
	if entry, ok := c.GetByHash(filePath, hash); ok {
		return true, entry.Hash, nil
	}

	return false, hash, nil
}
