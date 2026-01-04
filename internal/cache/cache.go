// Package cache provides a generic file-based caching mechanism with gob encoding.
// It's used by scanner and validator packages to cache results for incremental processing.
package cache

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Cache provides thread-safe caching with gob-based persistence
type Cache[K comparable, V any] struct {
	entries  map[K]V
	mu       sync.RWMutex
	filePath string
}

// New creates a new cache with the specified file path for persistence
func New[K comparable, V any](filePath string) *Cache[K, V] {
	return &Cache[K, V]{
		entries:  make(map[K]V),
		filePath: filePath,
	}
}

// Load loads cache entries from disk using gob encoding
func (c *Cache[K, V]) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	file, err := os.Open(c.filePath)
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

// Save persists cache entries to disk using gob encoding
func (c *Cache[K, V]) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write to temp file first
	tempPath := c.filePath + ".tmp"
	file, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(c.entries); err != nil {
		file.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to encode cache: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close cache file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, c.filePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	return nil
}

// Get retrieves a value from the cache
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.entries[key]
	return val, ok
}

// Set stores a value in the cache
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = value
}

// Delete removes a value from the cache
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// Clear removes all entries from the cache
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[K]V)
}

// Len returns the number of entries in the cache
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// Keys returns all keys in the cache
func (c *Cache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]K, 0, len(c.entries))
	for k := range c.entries {
		keys = append(keys, k)
	}
	return keys
}

// ForEach iterates over all entries in the cache
// The callback function receives a copy of the key and value
func (c *Cache[K, V]) ForEach(fn func(key K, value V)) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for k, v := range c.entries {
		fn(k, v)
	}
}
