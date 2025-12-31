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
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .glibrc
	glibrcContent := `{
  "version": "1.0",
  "generate": {
    "output": "generated",
    "package": "generated"
  }
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".glibrc"), []byte(glibrcContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a simple controller
	controllersDir := filepath.Join(tmpDir, "controllers")
	if err := os.MkdirAll(controllersDir, 0755); err != nil {
		t.Fatal(err)
	}

	controllerContent := `package controllers

import (
	"context"
	"github.com/azizndao/glib"
)

// @Controller path=/api/health
type HealthController struct {}

// @Route method=GET path=/
func (c *HealthController) Get(ctx context.Context) glib.Result[string] {
	return glib.OK("healthy")
}
`
	if err := os.WriteFile(filepath.Join(controllersDir, "health.go"), []byte(controllerContent), 0644); err != nil {
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
		config:  ".glibrc",
		verbose: false,
		watch:   false,
	}

	err = runGenerate(opts)
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Verify generated files exist
	generatedDir := filepath.Join(tmpDir, "generated")
	expectedFiles := []string{
		"glib.gen.go",
		"di.gen.go",
		"routes.gen.go",
		"parsers.gen.go",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(generatedDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", file)
		}
	}
}

// TestRunGenerate_NoGoMod tests error when go.mod is missing
func TestRunGenerate_NoGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .glibrc but no go.mod
	glibrcContent := `{
  "version": "1.0",
  "generate": {
    "output": "generated",
    "package": "generated"
  }
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".glibrc"), []byte(glibrcContent), 0644); err != nil {
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

	opts := &generateOptions{
		dir:     ".",
		output:  "",
		config:  ".glibrc",
		verbose: false,
		watch:   false,
	}

	err = runGenerate(opts)
	if err == nil {
		t.Error("Expected error when go.mod is missing")
	}
}

// TestRunGenerate_NoGlibrc tests error when .glibrc is missing
func TestRunGenerate_NoGlibrc(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod but no .glibrc
	goModContent := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
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

	opts := &generateOptions{
		dir:     ".",
		output:  "",
		config:  ".glibrc",
		verbose: false,
		watch:   false,
	}

	err = runGenerate(opts)
	if err == nil {
		t.Error("Expected error when .glibrc is missing")
	}
}

// TestRunGenerate_ValidationError tests error when validation fails
func TestRunGenerate_ValidationError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .glibrc
	glibrcContent := `{
  "version": "1.0",
  "generate": {
    "output": "generated",
    "package": "generated"
  }
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".glibrc"), []byte(glibrcContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create controller with invalid signature (circular dependency)
	controllersDir := filepath.Join(tmpDir, "controllers")
	if err := os.MkdirAll(controllersDir, 0755); err != nil {
		t.Fatal(err)
	}

	servicesDir := filepath.Join(tmpDir, "services")
	if err := os.MkdirAll(servicesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create circular dependency: A -> B -> A
	serviceAContent := `package services

// @Provider singleton
func NewServiceA(b *ServiceB) *ServiceA {
	return &ServiceA{}
}

type ServiceA struct {}
`
	if err := os.WriteFile(filepath.Join(servicesDir, "service_a.go"), []byte(serviceAContent), 0644); err != nil {
		t.Fatal(err)
	}

	serviceBContent := `package services

// @Provider singleton
func NewServiceB(a *ServiceA) *ServiceB {
	return &ServiceB{}
}

type ServiceB struct {}
`
	if err := os.WriteFile(filepath.Join(servicesDir, "service_b.go"), []byte(serviceBContent), 0644); err != nil {
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

	opts := &generateOptions{
		dir:     ".",
		output:  "",
		config:  ".glibrc",
		verbose: false,
		watch:   false,
	}

	err = runGenerate(opts)
	if err == nil {
		t.Error("Expected error when validation fails (circular dependency)")
	}
}

// TestRunGenerate_WatchMode tests watch mode (not implemented)
func TestRunGenerate_WatchMode(t *testing.T) {
	opts := &generateOptions{
		dir:     ".",
		output:  "",
		config:  ".glibrc",
		verbose: false,
		watch:   true,
	}

	err := runGenerate(opts)
	if err == nil {
		t.Error("Expected error for unimplemented watch mode")
	}
	if err.Error() != "watch mode not implemented yet" {
		t.Errorf("Expected 'watch mode not implemented yet' error, got: %v", err)
	}
}

// TestRunGenerate_CustomOutput tests custom output directory
func TestRunGenerate_CustomOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module testproject

go 1.21

require (
	github.com/azizndao/glib v0.1.0
)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .glibrc
	glibrcContent := `{
  "version": "1.0",
  "generate": {
    "output": "generated",
    "package": "generated"
  }
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".glibrc"), []byte(glibrcContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a simple controller
	controllersDir := filepath.Join(tmpDir, "controllers")
	if err := os.MkdirAll(controllersDir, 0755); err != nil {
		t.Fatal(err)
	}

	controllerContent := `package controllers

import (
	"context"
	"github.com/azizndao/glib"
)

// @Controller path=/api/test
type TestController struct {}

// @Route method=GET path=/
func (c *TestController) Get(ctx context.Context) glib.Result[string] {
	return glib.OK("test")
}
`
	if err := os.WriteFile(filepath.Join(controllersDir, "test.go"), []byte(controllerContent), 0644); err != nil {
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

	// Run generate with custom output
	customOutput := "custom_gen"
	opts := &generateOptions{
		dir:     ".",
		output:  customOutput,
		config:  ".glibrc",
		verbose: false,
		watch:   false,
	}

	err = runGenerate(opts)
	if err != nil {
		t.Fatalf("runGenerate failed: %v", err)
	}

	// Verify files are in custom output directory
	generatedDir := filepath.Join(tmpDir, customOutput)
	if _, err := os.Stat(filepath.Join(generatedDir, "glib.gen.go")); os.IsNotExist(err) {
		t.Errorf("Expected file in custom output directory %s", customOutput)
	}
}

// TestNewGenerateCmd tests command creation
func TestNewGenerateCmd(t *testing.T) {
	cmd := newGenerateCmd()

	if cmd.Use != "generate" {
		t.Errorf("Expected Use='generate', got '%s'", cmd.Use)
	}

	// Check flags exist
	if cmd.Flags().Lookup("dir") == nil {
		t.Error("Expected --dir flag")
	}
	if cmd.Flags().Lookup("output") == nil {
		t.Error("Expected --output flag")
	}
	if cmd.Flags().Lookup("config") == nil {
		t.Error("Expected --config flag")
	}
	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("Expected --verbose flag")
	}
	if cmd.Flags().Lookup("watch") == nil {
		t.Error("Expected --watch flag")
	}
}
