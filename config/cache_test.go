package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/azizndao/glib/config"
)

func TestRepository_Cache(t *testing.T) {
	repo := config.New()
	repo.Set("app.name", "TestApp")
	repo.Set("app.debug", true)
	repo.Set("database.host", "localhost")
	repo.Set("database.port", 5432)

	// Create temp directory for cache
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "config.json")

	// Cache the config
	err := repo.Cache(cachePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("expected cache file to exist")
	}

	// Verify IsCached returns true
	if !repo.IsCached() {
		t.Error("expected IsCached to return true")
	}

	// Verify CachePath returns correct path
	if repo.CachePath() != cachePath {
		t.Errorf("expected cache path %s, got %s", cachePath, repo.CachePath())
	}
}

func TestRepository_LoadCache(t *testing.T) {
	// Create and cache config
	repo1 := config.New()
	repo1.Set("app.name", "TestApp")
	repo1.Set("app.debug", true)
	repo1.Set("database.host", "localhost")
	repo1.Set("database.port", 5432)
	repo1.Set("nested.deep.value", "found")

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "config.json")

	err := repo1.Cache(cachePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Load cache into new repository
	repo2 := config.New()
	err = repo2.LoadCache(cachePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify values match
	if repo2.GetString("app.name") != "TestApp" {
		t.Error("expected app.name to be TestApp")
	}

	if !repo2.GetBool("app.debug") {
		t.Error("expected app.debug to be true")
	}

	if repo2.GetString("database.host") != "localhost" {
		t.Error("expected database.host to be localhost")
	}

	if repo2.GetInt("database.port") != 5432 {
		t.Error("expected database.port to be 5432")
	}

	if repo2.GetString("nested.deep.value") != "found" {
		t.Error("expected nested.deep.value to be found")
	}

	// Verify IsCached
	if !repo2.IsCached() {
		t.Error("expected IsCached to return true")
	}
}

func TestRepository_LoadCache_NonExistent(t *testing.T) {
	repo := config.New()
	err := repo.LoadCache("/nonexistent/path/config.json")
	if err == nil {
		t.Error("expected error when loading non-existent cache")
	}
}

func TestRepository_LoadCache_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	err := os.WriteFile(cachePath, []byte("invalid json{{{"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	repo := config.New()
	err = repo.LoadCache(cachePath)
	if err == nil {
		t.Error("expected error when loading invalid JSON")
	}
}

func TestRepository_ClearCache(t *testing.T) {
	repo := config.New()
	repo.Set("app.name", "TestApp")

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "config.json")

	// Cache the config
	err := repo.Cache(cachePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify cache exists
	if !repo.IsCached() {
		t.Error("expected IsCached to return true")
	}

	// Clear cache
	err = repo.ClearCache()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify cache file is deleted
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("expected cache file to be deleted")
	}

	// Verify IsCached returns false
	if repo.IsCached() {
		t.Error("expected IsCached to return false")
	}

	// Verify CachePath is empty
	if repo.CachePath() != "" {
		t.Error("expected CachePath to return empty string")
	}
}

func TestRepository_CacheAndLoad_ComplexStructure(t *testing.T) {
	repo1 := config.New()

	// Set complex nested structure
	repo1.Set("database.connections.mysql.host", "mysql.example.com")
	repo1.Set("database.connections.mysql.port", 3306)
	repo1.Set("database.connections.postgres.host", "postgres.example.com")
	repo1.Set("database.connections.postgres.port", 5432)
	repo1.Set("services.api.timeout", "30s")
	repo1.Set("services.api.retry", 3)
	repo1.Set("features.auth.enabled", true)
	repo1.Set("features.cache.driver", "redis")

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "config.json")

	// Cache
	err := repo1.Cache(cachePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Load into new repo
	repo2 := config.New()
	err = repo2.LoadCache(cachePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify all values
	tests := []struct {
		key      string
		expected any
		getter   func(string) any
	}{
		{"database.connections.mysql.host", "mysql.example.com", func(k string) any { return repo2.GetString(k) }},
		{"database.connections.mysql.port", 3306, func(k string) any { return repo2.GetInt(k) }},
		{"database.connections.postgres.host", "postgres.example.com", func(k string) any { return repo2.GetString(k) }},
		{"database.connections.postgres.port", 5432, func(k string) any { return repo2.GetInt(k) }},
		{"services.api.timeout", "30s", func(k string) any { return repo2.GetString(k) }},
		{"services.api.retry", 3, func(k string) any { return repo2.GetInt(k) }},
		{"features.auth.enabled", true, func(k string) any { return repo2.GetBool(k) }},
		{"features.cache.driver", "redis", func(k string) any { return repo2.GetString(k) }},
	}

	for _, tt := range tests {
		got := tt.getter(tt.key)
		if got != tt.expected {
			t.Errorf("key %s: expected %v, got %v", tt.key, tt.expected, got)
		}
	}
}

func TestRepository_CacheCreatesDirectory(t *testing.T) {
	repo := config.New()
	repo.Set("app.name", "TestApp")

	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "nested", "deep", "config.json")

	// Cache should create nested directories
	err := repo.Cache(cachePath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify directory was created
	dir := filepath.Dir(cachePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("expected directory to be created")
	}

	// Verify file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("expected cache file to exist")
	}
}
