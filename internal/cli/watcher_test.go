package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

// getDefaultWatchConfig returns default watch config for tests
func getDefaultWatchConfig(debounce time.Duration) *WatchConfig {
	return &WatchConfig{
		Debounce:     debounce,
		ExcludeDirs:  []string{"vendor", "node_modules", ".git", ".glib", "tmp"},
		IncludeFiles: []string{"*.go"},
		ExcludeFiles: []string{"*_test.go", "*.gen.go"},
	}
}

func TestFileWatcher_FilterFiles(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Go file", "main.go", true},
		{"Go file in subdir", "controllers/auth.go", true},
		{"Test file", "main_test.go", false},
		{"Generated file", "routes.gen.go", false},
		{"JSON file", "package.json", false},
		{"Markdown", "README.md", false},
	}

	fw := &FileWatcher{
		rootDir:      ".",
		outputDir:    "generated",
		includeFiles: []string{"*.go"},
		excludeFiles: []string{"*_test.go", "*.gen.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fw.shouldIncludeFile(tt.path)
			if result != tt.expected {
				t.Errorf("shouldIncludeFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFileWatcher_ExcludeDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	fw := &FileWatcher{
		rootDir:     tmpDir,
		outputDir:   "generated",
		excludeDirs: []string{"vendor", "node_modules", ".git", ".glib", "tmp"},
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Root dir", tmpDir, false},
		{"Vendor dir", filepath.Join(tmpDir, "vendor"), true},
		{"Vendor subdir", filepath.Join(tmpDir, "vendor", "github.com"), true},
		{"Node modules", filepath.Join(tmpDir, "node_modules"), true},
		{"Git dir", filepath.Join(tmpDir, ".git"), true},
		{"Glib cache", filepath.Join(tmpDir, ".glib"), true},
		{"Tmp dir", filepath.Join(tmpDir, "tmp"), true},
		{"Generated dir", filepath.Join(tmpDir, "generated"), true},
		{"Normal dir", filepath.Join(tmpDir, "controllers"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fw.shouldExcludeDir(tt.path)
			if result != tt.expected {
				t.Errorf("shouldExcludeDir(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFileWatcher_DetectChanges(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create watcher with short debounce for testing
	fw, err := NewFileWatcher(tmpDir, "generated", getDefaultWatchConfig(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()

	changes, errors, err := fw.Start()
	if err != nil {
		t.Fatal(err)
	}

	// Modify the file
	time.Sleep(50 * time.Millisecond) // Let watcher initialize
	if err := os.WriteFile(testFile, []byte("package main\n\n// Modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for change event (with timeout)
	select {
	case changedFiles := <-changes:
		found := slices.Contains(changedFiles, testFile)
		if !found {
			t.Errorf("Expected change to %s, but got changes to: %v", testFile, changedFiles)
		}
	case err := <-errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for file change event")
	}
}

func TestFileWatcher_Debouncing(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.go")
	file2 := filepath.Join(tmpDir, "file2.go")
	if err := os.WriteFile(file1, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create watcher with 200ms debounce
	fw, err := NewFileWatcher(tmpDir, "generated", getDefaultWatchConfig(200*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()

	changes, errors, err := fw.Start()
	if err != nil {
		t.Fatal(err)
	}

	// Modify both files rapidly
	time.Sleep(50 * time.Millisecond) // Let watcher initialize
	if err := os.WriteFile(file1, []byte("package main\n// Modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(file2, []byte("package main\n// Modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Should get ONE batch with both files
	select {
	case changedFiles := <-changes:
		if len(changedFiles) != 2 {
			t.Errorf("Expected 2 files in batch, got %d: %v", len(changedFiles), changedFiles)
		}
	case err := <-errors:
		t.Fatalf("Unexpected error: %v", err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for debounced changes")
	}

	// Should NOT get a second batch immediately
	select {
	case changedFiles := <-changes:
		t.Errorf("Expected no more changes, but got: %v", changedFiles)
	case <-time.After(100 * time.Millisecond):
		// Good - no extra events
	}
}

func TestFileWatcher_IgnoreGeneratedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a generated file
	generatedFile := filepath.Join(tmpDir, "routes.gen.go")
	if err := os.WriteFile(generatedFile, []byte("package generated"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a normal file
	normalFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(normalFile, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	fw, err := NewFileWatcher(tmpDir, "generated", getDefaultWatchConfig(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()

	changes, _, err := fw.Start()
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(50 * time.Millisecond) // Let watcher initialize

	// Modify the generated file (should be ignored)
	if err := os.WriteFile(generatedFile, []byte("package generated\n// Modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Modify the normal file (should trigger)
	if err := os.WriteFile(normalFile, []byte("package main\n// Modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Should only get change for normal file
	select {
	case changedFiles := <-changes:
		if len(changedFiles) != 1 {
			t.Fatalf("Expected 1 file, got %d: %v", len(changedFiles), changedFiles)
		}
		if changedFiles[0] != normalFile {
			t.Errorf("Expected change to %s, got %s", normalFile, changedFiles[0])
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for file change")
	}
}

func TestFileWatcher_CountFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some test files
	files := map[string]string{
		"main.go":          "package main",
		"main_test.go":     "package main", // Should be excluded
		"routes.gen.go":    "package gen",  // Should be excluded
		"controllers/a.go": "package controllers",
		"services/b.go":    "package services",
		"vendor/vendor.go": "package vendor", // Should be excluded
		"README.md":        "# README",       // Should be excluded
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	fw, err := NewFileWatcher(tmpDir, "generated", getDefaultWatchConfig(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()

	count, err := fw.CountWatchedFiles()
	if err != nil {
		t.Fatal(err)
	}

	// Should count: main.go (1) + controllers/a.go (1) + services/b.go (1) = 3
	// Should exclude: main_test.go, routes.gen.go, vendor/vendor.go, README.md
	expected := 3
	if count != expected {
		t.Errorf("Expected %d watched files, got %d", expected, count)
	}
}
