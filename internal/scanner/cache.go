package scanner

import (
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileCache manages cached scan results for incremental scanning
type FileCache struct {
	entries  map[string]*CacheEntry // filePath -> entry
	mu       sync.RWMutex
	cacheDir string // Where to persist cache
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
	return &FileCache{
		entries:  make(map[string]*CacheEntry),
		cacheDir: cacheDir,
	}
}

// Load loads cache from disk
func (c *FileCache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cachePath := filepath.Join(c.cacheDir, "scan.cache")
	file, err := os.Open(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache file yet, that's ok
		}
		return fmt.Errorf("failed to open cache file: %w", err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&c.entries); err != nil {
		return fmt.Errorf("failed to decode cache: %w", err)
	}

	return nil
}

// Save persists cache to disk
func (c *FileCache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cachePath := filepath.Join(c.cacheDir, "scan.cache")
	file, err := os.Create(cachePath)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(c.entries); err != nil {
		return fmt.Errorf("failed to encode cache: %w", err)
	}

	return nil
}

// Get retrieves a cached entry if valid
func (c *FileCache) Get(filePath string, modTime time.Time) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[filePath]
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
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[filePath]
	if !exists {
		return nil, false
	}

	if entry.Hash != hash {
		return nil, false
	}

	return entry, true
}

// Set stores a cache entry
func (c *FileCache) Set(filePath string, entry *CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[filePath] = entry
}

// Invalidate removes a cache entry
func (c *FileCache) Invalidate(filePath string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, filePath)
}

// Clear removes all cache entries
func (c *FileCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
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
