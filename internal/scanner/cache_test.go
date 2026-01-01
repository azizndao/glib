package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileCache(t *testing.T) {
	// Create temp cache dir
	cacheDir := t.TempDir()

	t.Run("basic_set_and_get", func(t *testing.T) {
		cache := NewFileCache(cacheDir)

		entry := &CacheEntry{
			Hash:    "abc123",
			ModTime: time.Now(),
			Controllers: []*Controller{
				{Name: "TestController"},
			},
		}

		cache.Set("/path/to/file.go", entry)

		retrieved, ok := cache.GetByHash("/path/to/file.go", "abc123")
		if !ok {
			t.Fatal("expected to find cached entry")
		}

		if len(retrieved.Controllers) != 1 || retrieved.Controllers[0].Name != "TestController" {
			t.Error("cached data doesn't match")
		}
	})

	t.Run("cache_miss_on_wrong_hash", func(t *testing.T) {
		cache := NewFileCache(cacheDir)

		entry := &CacheEntry{
			Hash:    "abc123",
			ModTime: time.Now(),
		}

		cache.Set("/path/to/file.go", entry)

		_, ok := cache.GetByHash("/path/to/file.go", "different_hash")
		if ok {
			t.Error("should not find entry with different hash")
		}
	})

	t.Run("save_and_load", func(t *testing.T) {
		cache1 := NewFileCache(cacheDir)

		entry := &CacheEntry{
			Hash:    "def456",
			ModTime: time.Now(),
			Providers: []*Provider{
				{Name: "TestProvider"},
			},
		}

		cache1.Set("/path/to/provider.go", entry)

		if err := cache1.Save(); err != nil {
			t.Fatalf("failed to save cache: %v", err)
		}

		// Create new cache and load
		cache2 := NewFileCache(cacheDir)
		if err := cache2.Load(); err != nil {
			t.Fatalf("failed to load cache: %v", err)
		}

		retrieved, ok := cache2.GetByHash("/path/to/provider.go", "def456")
		if !ok {
			t.Fatal("expected to find cached entry after load")
		}

		if len(retrieved.Providers) != 1 || retrieved.Providers[0].Name != "TestProvider" {
			t.Error("loaded data doesn't match")
		}
	})

	t.Run("invalidate", func(t *testing.T) {
		cache := NewFileCache(cacheDir)

		entry := &CacheEntry{
			Hash:    "xyz789",
			ModTime: time.Now(),
		}

		cache.Set("/path/to/file.go", entry)

		// Verify it exists
		if _, ok := cache.GetByHash("/path/to/file.go", "xyz789"); !ok {
			t.Fatal("entry should exist before invalidation")
		}

		cache.Invalidate("/path/to/file.go")

		// Verify it's gone
		if _, ok := cache.GetByHash("/path/to/file.go", "xyz789"); ok {
			t.Error("entry should not exist after invalidation")
		}
	})

	t.Run("clear", func(t *testing.T) {
		cache := NewFileCache(cacheDir)

		cache.Set("/file1.go", &CacheEntry{Hash: "a"})
		cache.Set("/file2.go", &CacheEntry{Hash: "b"})
		cache.Set("/file3.go", &CacheEntry{Hash: "c"})

		cache.Clear()

		if _, ok := cache.GetByHash("/file1.go", "a"); ok {
			t.Error("cache should be empty after clear")
		}
	})
}

func TestScannerWithCache(t *testing.T) {
	projectDir := "../../examples/demo"

	// Skip if demo doesn't exist
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		t.Skip("demo project not found")
	}

	cacheDir := t.TempDir()

	t.Run("scan_with_cache_enabled", func(t *testing.T) {
		scanner, err := New(projectDir, WithCache(cacheDir))
		if err != nil {
			t.Fatalf("failed to create scanner: %v", err)
		}

		// First scan - should populate cache
		project1, err := scanner.Scan()
		if err != nil {
			t.Fatalf("first scan failed: %v", err)
		}

		if len(project1.Controllers) == 0 {
			t.Fatal("expected to find controllers")
		}

		// Second scan - should use cache
		scanner2, err := New(projectDir, WithCache(cacheDir))
		if err != nil {
			t.Fatalf("failed to create second scanner: %v", err)
		}

		project2, err := scanner2.Scan()
		if err != nil {
			t.Fatalf("second scan failed: %v", err)
		}

		// Results should be the same
		if len(project1.Controllers) != len(project2.Controllers) {
			t.Errorf("controller count mismatch: %d vs %d",
				len(project1.Controllers), len(project2.Controllers))
		}

		if len(project1.Providers) != len(project2.Providers) {
			t.Errorf("provider count mismatch: %d vs %d",
				len(project1.Providers), len(project2.Providers))
		}
	})

	t.Run("incremental_scan", func(t *testing.T) {
		scanner, err := New(projectDir, WithCache(cacheDir))
		if err != nil {
			t.Fatalf("failed to create scanner: %v", err)
		}

		// Initial scan
		project1, err := scanner.Scan()
		if err != nil {
			t.Fatalf("initial scan failed: %v", err)
		}

		// Incremental scan with no changes
		project2, err := scanner.ScanIncremental([]string{})
		if err != nil {
			t.Fatalf("incremental scan failed: %v", err)
		}

		if len(project1.Controllers) != len(project2.Controllers) {
			t.Errorf("incremental scan produced different results")
		}
	})
}

func TestComputeFileHash(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")

	content := []byte("package main\n\nfunc main() {}\n")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	hash1, err := computeFileHash(tmpFile)
	if err != nil {
		t.Fatalf("failed to compute hash: %v", err)
	}

	if hash1 == "" {
		t.Error("hash should not be empty")
	}

	// Compute hash again - should be same
	hash2, err := computeFileHash(tmpFile)
	if err != nil {
		t.Fatalf("failed to compute second hash: %v", err)
	}

	if hash1 != hash2 {
		t.Error("hashes should be identical for same content")
	}

	// Change content
	content = []byte("package main\n\nfunc main() { println(\"hi\") }\n")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	hash3, err := computeFileHash(tmpFile)
	if err != nil {
		t.Fatalf("failed to compute third hash: %v", err)
	}

	if hash1 == hash3 {
		t.Error("hashes should be different for different content")
	}
}

func BenchmarkScanWithCache(b *testing.B) {
	projectDir := "../../examples/demo"
	cacheDir := b.TempDir()

	// First scan to populate cache
	scanner, _ := New(projectDir, WithCache(cacheDir))
	scanner.Scan()

	b.Run("WithCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			scanner, _ := New(projectDir, WithCache(cacheDir))
			scanner.Scan()
		}
	})

	b.Run("WithoutCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			scanner, _ := New(projectDir)
			scanner.Scan()
		}
	})
}
