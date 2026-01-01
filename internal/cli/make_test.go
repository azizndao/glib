package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMakeController_Success tests successful controller creation
func TestMakeController_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config.toml
	glibrcContent := `version = "1.0"

[make]
controllers = "controllers"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(glibrcContent), 0644); err != nil {
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

	// Load config
	cfg, err := loadGlibrc()
	if err != nil {
		t.Fatalf("Failed to load config.toml: %v", err)
	}

	// Make controller
	opts := &makeOptions{
		path:      "",
		prefix:    "/api/posts",
		noExample: false,
	}

	err = makeController("posts", opts, cfg)
	if err != nil {
		t.Fatalf("makeController failed: %v", err)
	}

	// Verify controller file was created
	controllerPath := filepath.Join(tmpDir, "controllers/posts/controller.go")
	if _, err := os.Stat(controllerPath); os.IsNotExist(err) {
		t.Error("Expected controller.go to be created")
	}

	// Verify models file was created
	modelsPath := filepath.Join(tmpDir, "controllers/posts/models.go")
	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
		t.Error("Expected models.go to be created")
	}

	// Verify content
	controllerContent, err := os.ReadFile(controllerPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(controllerContent) == 0 {
		t.Error("controller.go is empty")
	}
}

// TestMakeController_CustomPath tests controller creation with custom path
func TestMakeController_CustomPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config.toml
	glibrcContent := `version = "1.0"

[make]
controllers = "controllers"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(glibrcContent), 0644); err != nil {
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

	cfg, err := loadGlibrc()
	if err != nil {
		t.Fatalf("Failed to load config.toml: %v", err)
	}

	// Make controller with custom path
	customPath := "custom/path/mycontroller"
	opts := &makeOptions{
		path:      customPath,
		prefix:    "/api/custom",
		noExample: false,
	}

	err = makeController("custom", opts, cfg)
	if err != nil {
		t.Fatalf("makeController failed: %v", err)
	}

	// Verify controller was created in custom path
	controllerPath := filepath.Join(tmpDir, customPath, "controller.go")
	if _, err := os.Stat(controllerPath); os.IsNotExist(err) {
		t.Error("Expected controller.go in custom path")
	}
}

// TestMakeController_NoExample tests controller creation without examples
func TestMakeController_NoExample(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config.toml
	glibrcContent := `version = "1.0"

[make]
controllers = "controllers"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(glibrcContent), 0644); err != nil {
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

	cfg, err := loadGlibrc()
	if err != nil {
		t.Fatalf("Failed to load config.toml: %v", err)
	}

	opts := &makeOptions{
		path:      "",
		prefix:    "/api/test",
		noExample: true,
	}

	err = makeController("test", opts, cfg)
	if err != nil {
		t.Fatalf("makeController failed: %v", err)
	}

	// Verify files were created
	controllerPath := filepath.Join(tmpDir, "controllers/test/controller.go")
	if _, err := os.Stat(controllerPath); os.IsNotExist(err) {
		t.Error("Expected controller.go to be created")
	}
}

// TestMakeProvider_Success tests successful provider creation
func TestMakeProvider_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config.toml
	glibrcContent := `version = "1.0"

[make]
providers = "services"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(glibrcContent), 0644); err != nil {
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

	cfg, err := loadGlibrc()
	if err != nil {
		t.Fatalf("Failed to load config.toml: %v", err)
	}

	opts := &makeOptions{
		path:      "",
		noExample: false,
	}

	err = makeProvider("database", opts, cfg)
	if err != nil {
		t.Fatalf("makeProvider failed: %v", err)
	}

	// Verify provider file was created
	providerPath := filepath.Join(tmpDir, "services/database.go")
	if _, err := os.Stat(providerPath); os.IsNotExist(err) {
		t.Error("Expected database.go to be created")
	}

	// Verify content
	providerContent, err := os.ReadFile(providerPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(providerContent) == 0 {
		t.Error("database.go is empty")
	}
}

// TestMakeMiddleware_Success tests successful middleware creation
func TestMakeMiddleware_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config.toml
	glibrcContent := `version = "1.0"

[make]
middleware = "middleware"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(glibrcContent), 0644); err != nil {
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

	cfg, err := loadGlibrc()
	if err != nil {
		t.Fatalf("Failed to load config.toml: %v", err)
	}

	opts := &makeOptions{
		path:      "",
		noExample: false,
	}

	err = makeMiddleware("auth", opts, cfg)
	if err != nil {
		t.Fatalf("makeMiddleware failed: %v", err)
	}

	// Verify middleware file was created
	middlewarePath := filepath.Join(tmpDir, "middleware/auth.go")
	if _, err := os.Stat(middlewarePath); os.IsNotExist(err) {
		t.Error("Expected auth.go to be created")
	}

	// Verify content
	middlewareContent, err := os.ReadFile(middlewarePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(middlewareContent) == 0 {
		t.Error("auth.go is empty")
	}
}

// TestLoadGlibrc_Success tests loading config.toml
func TestLoadGlibrc_Success(t *testing.T) {
	tmpDir := t.TempDir()

	glibrcContent := `version = "1.0"

[generate]
output = "generated"
package = "generated"

[make]
controllers = "controllers"
providers = "services"
middleware = "middleware"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(glibrcContent), 0644); err != nil {
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

	cfg, err := loadGlibrc()
	if err != nil {
		t.Fatalf("loadGlibrc failed: %v", err)
	}

	if cfg.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", cfg.Version)
	}
	if cfg.Generate.Output != "generated" {
		t.Errorf("Expected output 'generated', got '%s'", cfg.Generate.Output)
	}
	if cfg.Make.Controllers != "controllers" {
		t.Errorf("Expected controllers 'controllers', got '%s'", cfg.Make.Controllers)
	}
}

// TestLoadGlibrc_NotFound tests error when config.toml is missing
func TestLoadGlibrc_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to temp directory without config.toml
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	_, err = loadGlibrc()
	if err == nil {
		t.Error("Expected error when config.toml is missing")
	}
}

// TestLoadGlibrc_InvalidJSON tests error with invalid JSON
func TestLoadGlibrc_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid TOML
	invalidContent := `version = "1.0"
invalid toml syntax here
[generate
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(invalidContent), 0644); err != nil {
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

	_, err = loadGlibrc()
	if err == nil {
		t.Error("Expected error with invalid TOML")
	}
}

// TestNewMakeCmd tests command creation
func TestNewMakeCmd(t *testing.T) {
	cmd := newMakeCmd()

	if cmd.Use != "make <type> <name>" {
		t.Errorf("Expected Use='make <type> <name>', got '%s'", cmd.Use)
	}

	// Check flags exist
	if cmd.Flags().Lookup("path") == nil {
		t.Error("Expected --path flag")
	}
	if cmd.Flags().Lookup("prefix") == nil {
		t.Error("Expected --prefix flag")
	}
	if cmd.Flags().Lookup("no-example") == nil {
		t.Error("Expected --no-example flag")
	}
}
