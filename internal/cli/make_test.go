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

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Load config from defaults
	cfg := getDefaultConfig()

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

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Load config from defaults
	cfg := getDefaultConfig()

	// Make controller with custom path
	opts := &makeOptions{
		path:      "custom/path/posts",
		prefix:    "/api/posts",
		noExample: false,
	}

	err = makeController("posts", opts, cfg)
	if err != nil {
		t.Fatalf("makeController failed: %v", err)
	}

	// Verify controller was created at custom path
	controllerPath := filepath.Join(tmpDir, "custom/path/posts/controller.go")
	if _, err := os.Stat(controllerPath); os.IsNotExist(err) {
		t.Error("Expected controller.go to be created at custom path")
	}
}

// TestMakeProvider_Success tests successful provider creation
func TestMakeProvider_Success(t *testing.T) {
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

	// Load config from defaults
	cfg := getDefaultConfig()

	// Make provider
	opts := &makeOptions{
		path:      "",
		noExample: false,
	}

	err = makeProvider("database", opts, cfg)
	if err != nil {
		t.Fatalf("makeProvider failed: %v", err)
	}

	// Verify provider file was created
	providerPath := filepath.Join(tmpDir, "providers/database.go")
	if _, err := os.Stat(providerPath); os.IsNotExist(err) {
		t.Error("Expected database.go to be created")
	}

	// Verify content
	content, err := os.ReadFile(providerPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) == 0 {
		t.Error("database.go is empty")
	}
}

// TestMakeMiddleware_Success tests successful middleware creation
func TestMakeMiddleware_Success(t *testing.T) {
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

	// Load config from defaults
	cfg := getDefaultConfig()

	// Make middleware
	opts := &makeOptions{
		path: "",
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
	content, err := os.ReadFile(middlewarePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) == 0 {
		t.Error("auth.go is empty")
	}
}
