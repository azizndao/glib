package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRunGenerate_Success tests successful code generation
func TestRunGenerate_Success(t *testing.T) {
	// Create temporary project
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module testproject

go 1.21

require (
	github.com/azizndao/glib v0.1.0
)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a simple controller
	controllersDir := filepath.Join(tmpDir, "controllers")
	if err := os.MkdirAll(controllersDir, 0o755); err != nil {
		t.Fatal(err)
	}

	controllerContent := `package controllers

import (
	"context"
)

// @Controller path=/api/health
type HealthController struct {}

// @Route method=GET path=/
func (c *HealthController) Get(ctx context.Context) (string, error) {
	return "healthy", nil
}
`
	if err := os.WriteFile(filepath.Join(controllersDir, "health.go"), []byte(controllerContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Run generate
	opts := &generateOptions{
		dir:     ".",
		output:  "",
		verbose: false,
		watch:   false,
	}

	err = runGenerate(opts)
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Verify generated files exist
	expectedFiles := []string{
		"generated/di.gen.go",
		"generated/routes.gen.go",
		"generated/parsers.gen.go",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected generated file %s does not exist", file)
		}
	}
}

// TestRunGenerate_NoGoMod tests error when go.mod is missing
func TestRunGenerate_NoGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	opts := &generateOptions{
		dir:     ".",
		output:  "",
		verbose: false,
		watch:   false,
	}

	err = runGenerate(opts)
	if err == nil {
		t.Error("Expected error when go.mod is missing")
	}
}
