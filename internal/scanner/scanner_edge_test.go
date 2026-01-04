package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanner_EdgeCases(t *testing.T) {
	t.Run("empty_project_directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create go.mod
		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test/empty\ngo 1.21\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		scanner, err := New(tmpDir)
		if err != nil {
			t.Fatalf("New() should work with empty project: %v", err)
		}

		project, err := scanner.Scan()
		if err != nil {
			t.Fatalf("Scan() should work with empty project: %v", err)
		}

		if len(project.Controllers) != 0 {
			t.Errorf("Expected 0 controllers, got %d", len(project.Controllers))
		}
		if len(project.Providers) != 0 {
			t.Errorf("Expected 0 providers, got %d", len(project.Providers))
		}
	})

	t.Run("project_with_only_comments", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create go.mod
		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test/comments\ngo 1.21\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create file with only comments
		mainGo := filepath.Join(tmpDir, "main.go")
		content := `package main

// This is a comment
// @Controller but not really
// Just comments here

/* Multi-line comment
   @Provider also not real
*/
`
		if err := os.WriteFile(mainGo, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		scanner, err := New(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		project, err := scanner.Scan()
		if err != nil {
			t.Fatalf("Should handle comments-only file: %v", err)
		}

		if len(project.Controllers) != 0 {
			t.Errorf("Should not find controllers in comments")
		}
	})

	t.Run("deeply_nested_packages", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create go.mod
		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test/nested\ngo 1.21\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create deeply nested structure
		deepDir := filepath.Join(tmpDir, "a", "b", "c", "d", "e")
		if err := os.MkdirAll(deepDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create a provider in the deep directory
		deepFile := filepath.Join(deepDir, "service.go")
		content := `package e

// @Provider singleton
func NewService() *Service {
	return &Service{}
}

type Service struct {}
`
		if err := os.WriteFile(deepFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		scanner, err := New(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		project, err := scanner.Scan()
		if err != nil {
			t.Fatalf("Should handle deeply nested packages: %v", err)
		}

		if len(project.Providers) != 1 {
			t.Errorf("Expected 1 provider in nested package, got %d", len(project.Providers))
		}
	})

	t.Run("file_with_build_tags", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create go.mod
		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test/buildtags\ngo 1.21\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create file with build tags
		taggedFile := filepath.Join(tmpDir, "tagged.go")
		content := `//go:build linux
// +build linux

package main

// @Provider singleton
func NewLinuxService() *LinuxService {
	return &LinuxService{}
}

type LinuxService struct {}
`
		if err := os.WriteFile(taggedFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		scanner, err := New(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		project, err := scanner.Scan()
		if err != nil {
			t.Fatalf("Should handle build tags: %v", err)
		}

		// Build tags don't affect annotation scanning
		if len(project.Providers) != 1 {
			t.Errorf("Expected 1 provider even with build tags, got %d", len(project.Providers))
		}
	})

	t.Run("unicode_in_annotations", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create go.mod
		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test/unicode\ngo 1.21\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create file with unicode in comments
		unicodeFile := filepath.Join(tmpDir, "unicode.go")
		content := `package main

import "net/http"

// @Controller path=/api/用户 tags=中文,日本語
type UserController struct {}

// @Route method=GET path=/
func (c *UserController) List(w http.ResponseWriter, r *http.Request) {}
`
		if err := os.WriteFile(unicodeFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		scanner, err := New(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		project, err := scanner.Scan()
		if err != nil {
			t.Fatalf("Should handle unicode in annotations: %v", err)
		}

		if len(project.Controllers) != 1 {
			t.Fatal("Expected 1 controller")
		}

		ctrl := project.Controllers[0]
		if ctrl.RoutePrefix != "/api/用户" {
			t.Errorf("Unicode path not preserved: got %s", ctrl.RoutePrefix)
		}
	})

	t.Run("annotation_with_special_characters", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create go.mod
		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test/special\ngo 1.21\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create file with special chars
		specialFile := filepath.Join(tmpDir, "special.go")
		content := `package main

import "net/http"

// @Controller path=/api/v1/posts tags=api,web,backend description=Handle all CRUD operations for posts
type PostController struct {}

// @Route method=GET path=/{id:[0-9]+}
func (c *PostController) Get(w http.ResponseWriter, r *http.Request) {}
`
		if err := os.WriteFile(specialFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		scanner, err := New(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		project, err := scanner.Scan()
		if err != nil {
			t.Fatalf("Should handle special characters: %v", err)
		}

		if len(project.Controllers) != 1 {
			t.Fatal("Expected 1 controller")
		}

		ctrl := project.Controllers[0]
		if len(ctrl.Handlers) != 1 {
			t.Fatalf("Expected 1 handler, got %d", len(ctrl.Handlers))
		}
	})

	t.Run("mixed_line_endings", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create go.mod
		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test/lineendings\ngo 1.21\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create file with mixed line endings (CRLF and LF)
		mixedFile := filepath.Join(tmpDir, "mixed.go")
		content := "package main\r\n\r\n// @Provider singleton\r\nfunc NewService() *Service {\n\treturn &Service{}\n}\n\ntype Service struct {}\n"
		if err := os.WriteFile(mixedFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		scanner, err := New(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		project, err := scanner.Scan()
		if err != nil {
			t.Fatalf("Should handle mixed line endings: %v", err)
		}

		if len(project.Providers) != 1 {
			t.Errorf("Expected 1 provider with mixed line endings, got %d", len(project.Providers))
		}
	})

	t.Run("very_long_file_path", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create go.mod
		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test/longpath\ngo 1.21\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create a very long path (but reasonable)
		longPath := filepath.Join(tmpDir, "very", "long", "nested", "package", "structure", "with", "many", "levels", "to", "test", "path", "handling")
		if err := os.MkdirAll(longPath, 0o755); err != nil {
			t.Fatal(err)
		}

		longFile := filepath.Join(longPath, "service.go")
		content := `package handling

// @Provider singleton
func NewService() *Service { return &Service{} }
type Service struct {}
`
		if err := os.WriteFile(longFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		scanner, err := New(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		project, err := scanner.Scan()
		if err != nil {
			t.Fatalf("Should handle long paths: %v", err)
		}

		if len(project.Providers) != 1 {
			t.Errorf("Expected 1 provider with long path, got %d", len(project.Providers))
		}
	})

	t.Run("symlink_to_directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create go.mod
		goMod := filepath.Join(tmpDir, "go.mod")
		if err := os.WriteFile(goMod, []byte("module test/symlink\ngo 1.21\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create actual directory
		actualDir := filepath.Join(tmpDir, "actual")
		if err := os.MkdirAll(actualDir, 0o755); err != nil {
			t.Fatal(err)
		}

		actualFile := filepath.Join(actualDir, "service.go")
		content := `package actual

// @Provider singleton
func NewService() *Service { return &Service{} }
type Service struct {}
`
		if err := os.WriteFile(actualFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		// Create symlink (skip if not supported)
		symlinkPath := filepath.Join(tmpDir, "linked")
		if err := os.Symlink(actualDir, symlinkPath); err != nil {
			t.Skip("Symlinks not supported on this system")
		}

		scanner, err := New(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		project, err := scanner.Scan()
		if err != nil {
			t.Fatalf("Should handle symlinks: %v", err)
		}

		// Should find the provider (symlinks are followed by filepath.Walk)
		if len(project.Providers) < 1 {
			t.Errorf("Expected at least 1 provider through symlink, got %d", len(project.Providers))
		}
	})
}

func TestScanner_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test/concurrent\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create multiple files
	for i := range 10 {
		file := filepath.Join(tmpDir, "service"+string(rune('0'+i))+".go")
		content := `package main

// @Provider singleton
func NewService` + string(rune('A'+i)) + `() *Service` + string(rune('A'+i)) + ` {
	return &Service` + string(rune('A'+i)) + `{}
}

type Service` + string(rune('A'+i)) + ` struct {}
`
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Scan multiple times concurrently with separate scanner instances
	// (Scanner is not thread-safe by design, each goroutine needs its own instance)
	const goroutines = 5
	results := make(chan error, goroutines)

	for range goroutines {
		go func() {
			scanner, err := New(tmpDir)
			if err != nil {
				results <- err
				return
			}
			_, err = scanner.Scan()
			results <- err
		}()
	}

	// Check all succeeded
	for i := range goroutines {
		if err := <-results; err != nil {
			t.Errorf("Concurrent scan %d failed: %v", i, err)
		}
	}
}

func TestScanner_ExcludePatterns(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test/exclude\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create files in different directories
	// vendor, node_modules, .git should be excluded by default
	// .glib, tmp also excluded
	// generated, normal should NOT be excluded (user directories)
	dirs := []string{"vendor", "node_modules", ".git", ".glib", "tmp", "normal"}
	for _, dir := range dirs {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			t.Fatal(err)
		}

		file := filepath.Join(dirPath, "service.go")
		content := `package ` + dir + `

// @Provider singleton
func NewService() *Service { return &Service{} }
type Service struct {}
`
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	scanner, err := New(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	project, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should only find the provider in "normal" directory
	// vendor, node_modules, .git, .glib, tmp should be excluded
	if len(project.Providers) != 1 {
		t.Errorf("Expected 1 provider (only from normal/), got %d", len(project.Providers))
		for _, p := range project.Providers {
			t.Logf("Found provider in package: %s", p.PackageName)
		}
	}

	if len(project.Providers) > 0 && project.Providers[0].PackageName != "normal" {
		t.Errorf("Expected provider from 'normal' package, got '%s'", project.Providers[0].PackageName)
	}
}
