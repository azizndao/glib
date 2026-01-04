package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCache_Basic(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test.cache")

	cache := New[string, string](cachePath)

	// Test Set and Get
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	val, ok := cache.Get("key1")
	if !ok || val != "value1" {
		t.Errorf("expected value1, got %s", val)
	}

	val, ok = cache.Get("key2")
	if !ok || val != "value2" {
		t.Errorf("expected value2, got %s", val)
	}

	// Test Get non-existent key
	_, ok = cache.Get("nonexistent")
	if ok {
		t.Error("expected false for non-existent key")
	}

	// Test Len
	if cache.Len() != 2 {
		t.Errorf("expected length 2, got %d", cache.Len())
	}
}

func TestCache_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test.cache")

	// Create and populate cache
	cache1 := New[string, int](cachePath)
	cache1.Set("a", 1)
	cache1.Set("b", 2)
	cache1.Set("c", 3)

	// Save to disk
	if err := cache1.Save(); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("cache file was not created")
	}

	// Load into new cache
	cache2 := New[string, int](cachePath)
	if err := cache2.Load(); err != nil {
		t.Fatalf("failed to load cache: %v", err)
	}

	// Verify data
	if cache2.Len() != 3 {
		t.Errorf("expected length 3 after load, got %d", cache2.Len())
	}

	val, ok := cache2.Get("a")
	if !ok || val != 1 {
		t.Errorf("expected 1 for key 'a', got %d", val)
	}

	val, ok = cache2.Get("b")
	if !ok || val != 2 {
		t.Errorf("expected 2 for key 'b', got %d", val)
	}

	val, ok = cache2.Get("c")
	if !ok || val != 3 {
		t.Errorf("expected 3 for key 'c', got %d", val)
	}
}

func TestCache_LoadNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "nonexistent.cache")

	cache := New[string, string](cachePath)

	// Should not error when loading non-existent cache
	if err := cache.Load(); err != nil {
		t.Errorf("expected no error loading non-existent cache, got: %v", err)
	}

	// Should be empty
	if cache.Len() != 0 {
		t.Errorf("expected empty cache, got length %d", cache.Len())
	}
}

func TestCache_Delete(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test.cache")

	cache := New[string, string](cachePath)
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Delete key1
	cache.Delete("key1")

	// Verify key1 is gone
	_, ok := cache.Get("key1")
	if ok {
		t.Error("key1 should be deleted")
	}

	// Verify key2 still exists
	val, ok := cache.Get("key2")
	if !ok || val != "value2" {
		t.Error("key2 should still exist")
	}

	// Verify length
	if cache.Len() != 1 {
		t.Errorf("expected length 1 after delete, got %d", cache.Len())
	}
}

func TestCache_Clear(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test.cache")

	cache := New[string, string](cachePath)
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Clear cache
	cache.Clear()

	// Verify empty
	if cache.Len() != 0 {
		t.Errorf("expected empty cache after clear, got length %d", cache.Len())
	}

	// Verify keys are gone
	_, ok := cache.Get("key1")
	if ok {
		t.Error("cache should be empty after clear")
	}
}

func TestCache_Keys(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test.cache")

	cache := New[string, int](cachePath)
	cache.Set("a", 1)
	cache.Set("b", 2)
	cache.Set("c", 3)

	keys := cache.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}

	// Check all keys are present
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	if !keyMap["a"] || !keyMap["b"] || !keyMap["c"] {
		t.Error("not all keys returned")
	}
}

func TestCache_ForEach(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test.cache")

	cache := New[string, int](cachePath)
	cache.Set("a", 1)
	cache.Set("b", 2)
	cache.Set("c", 3)

	// Sum all values
	sum := 0
	cache.ForEach(func(key string, value int) {
		sum += value
	})

	if sum != 6 {
		t.Errorf("expected sum of 6, got %d", sum)
	}
}

func TestCache_ComplexTypes(t *testing.T) {
	type User struct {
		Name  string
		Email string
		Age   int
	}

	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test.cache")

	cache := New[int, *User](cachePath)

	// Add users
	cache.Set(1, &User{Name: "Alice", Email: "alice@example.com", Age: 30})
	cache.Set(2, &User{Name: "Bob", Email: "bob@example.com", Age: 25})

	// Save and reload
	if err := cache.Save(); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	cache2 := New[int, *User](cachePath)
	if err := cache2.Load(); err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// Verify data
	user, ok := cache2.Get(1)
	if !ok || user.Name != "Alice" || user.Age != 30 {
		t.Error("user 1 not loaded correctly")
	}

	user, ok = cache2.Get(2)
	if !ok || user.Name != "Bob" || user.Age != 25 {
		t.Error("user 2 not loaded correctly")
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test.cache")

	cache := New[int, int](cachePath)

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			cache.Set(n, n*2)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all values
	if cache.Len() != 10 {
		t.Errorf("expected 10 entries, got %d", cache.Len())
	}

	for i := 0; i < 10; i++ {
		val, ok := cache.Get(i)
		if !ok || val != i*2 {
			t.Errorf("key %d: expected %d, got %d", i, i*2, val)
		}
	}
}
