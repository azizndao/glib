// Package config provides configuration management
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Cache serializes the configuration to a JSON file for faster loading.
// This is typically used in production to avoid re-loading and processing
// all config files on each application boot.
func (r *Repository) Cache(path string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Serialize config to JSON
	data, err := json.MarshalIndent(r.items, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	r.cached = true
	r.cachePath = path

	return nil
}

// LoadCache loads configuration from a cached JSON file.
// This is much faster than loading from individual config files.
// Returns an error if the cache file doesn't exist or is invalid.
func (r *Repository) LoadCache(path string) error {
	// Read cache file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	// Parse JSON
	var items map[string]any
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	// Replace current config
	r.mu.Lock()
	r.items = items
	r.cached = true
	r.cachePath = path
	r.mu.Unlock()

	return nil
}

// IsCached returns true if the configuration is loaded from cache.
func (r *Repository) IsCached() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cached
}

// CachePath returns the path to the cache file if cached.
func (r *Repository) CachePath() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cachePath
}

// ClearCache removes the cache file and marks config as not cached.
func (r *Repository) ClearCache() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cachePath != "" {
		if err := os.Remove(r.cachePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove cache file: %w", err)
		}
	}

	r.cached = false
	r.cachePath = ""

	return nil
}
