package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRunValidate_Success tests successful validation
func TestRunValidate_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
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
)

// @Controller path=/api/test
type TestController struct {}

// @Route method=GET path=/
func (c *TestController) Get(ctx context.Context) (string, error) {
	return "test", nil
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

	// Run validate
	opts := &validateOptions{
		dir:     ".",
		verbose: false,
	}

	err = runValidate(opts)
	if err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}
}

// TestRunValidate_CircularDependency tests circular dependency detection
func TestRunValidate_CircularDependency(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create services with circular dependency
	servicesDir := filepath.Join(tmpDir, "services")
	if err := os.MkdirAll(servicesDir, 0755); err != nil {
		t.Fatal(err)
	}

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

	opts := &validateOptions{
		dir:     ".",
		verbose: false,
	}

	err = runValidate(opts)
	if err == nil {
		t.Error("Expected validation to fail due to circular dependency")
	}
}

// TestRunValidate_NoGoMod tests error when go.mod is missing
func TestRunValidate_NoGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp directory without go.mod
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	opts := &validateOptions{
		dir:     ".",
		verbose: false,
	}

	err = runValidate(opts)
	if err == nil {
		t.Error("Expected error when go.mod is missing")
	}
}

// TestRunValidate_EmptyProject tests validation of empty project
func TestRunValidate_EmptyProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module emptyproject

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

	opts := &validateOptions{
		dir:     ".",
		verbose: false,
	}

	err = runValidate(opts)
	if err != nil {
		t.Errorf("Expected empty project to validate successfully, got: %v", err)
	}
}

// TestRunValidate_DuplicateRoutes tests duplicate route detection
func TestRunValidate_DuplicateRoutes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create controllers with duplicate routes
	controllersDir := filepath.Join(tmpDir, "controllers")
	if err := os.MkdirAll(controllersDir, 0755); err != nil {
		t.Fatal(err)
	}

	controller1Content := `package controllers

import (
	"context"
)

// @Controller path=/api/test
type Controller1 struct {}

// @Route method=GET path=/
func (c *Controller1) Get(ctx context.Context) (string, error) {
	return "test1", nil
}
`
	if err := os.WriteFile(filepath.Join(controllersDir, "controller1.go"), []byte(controller1Content), 0644); err != nil {
		t.Fatal(err)
	}

	controller2Content := `package controllers

import (
	"context"
)

// @Controller path=/api/test
type Controller2 struct {}

// @Route method=GET path=/
func (c *Controller2) Get(ctx context.Context) (string, error) {
	return "test2", nil
}
`
	if err := os.WriteFile(filepath.Join(controllersDir, "controller2.go"), []byte(controller2Content), 0644); err != nil {
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

	opts := &validateOptions{
		dir:     ".",
		verbose: false,
	}

	err = runValidate(opts)
	if err == nil {
		t.Error("Expected validation to fail due to duplicate routes")
	}
}

// TestRunValidate_VerboseMode tests verbose output
func TestRunValidate_VerboseMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
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
)

// @Controller path=/api/test
type TestController struct {}

// @Route method=GET path=/
func (c *TestController) Get(ctx context.Context) (string, error) {
	return "test", nil
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

	// Run validate in verbose mode
	opts := &validateOptions{
		dir:     ".",
		verbose: true,
	}

	err = runValidate(opts)
	if err != nil {
		t.Errorf("Expected validation to pass in verbose mode, got error: %v", err)
	}
}

// TestNewValidateCmd tests command creation
func TestNewValidateCmd(t *testing.T) {
	cmd := newValidateCmd()

	if cmd.Use != "validate" {
		t.Errorf("Expected Use='validate', got '%s'", cmd.Use)
	}

	// Check flags exist
	if cmd.Flags().Lookup("dir") == nil {
		t.Error("Expected --dir flag")
	}
	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("Expected --verbose flag")
	}
}
