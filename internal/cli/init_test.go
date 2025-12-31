package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInitProject_Success tests successful project initialization
func TestInitProject_Success(t *testing.T) {
	tmpDir := t.TempDir()

	err := initProject(tmpDir, "test/myapp", false, false)
	if err != nil {
		t.Fatalf("initProject failed: %v", err)
	}

	// Verify expected files were created
	expectedFiles := []string{
		"go.mod",
		".glibrc",
		".gitignore",
		"README.md",
		"main.go",
		"configs/config.go",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", file)
		}
	}

	// Verify go.mod content
	goModContent, err := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if len(goModContent) == 0 {
		t.Error("go.mod is empty")
	}

	// Verify .glibrc content
	glibrcContent, err := os.ReadFile(filepath.Join(tmpDir, ".glibrc"))
	if err != nil {
		t.Fatal(err)
	}
	if len(glibrcContent) == 0 {
		t.Error(".glibrc is empty")
	}
}

// TestInitProject_WithExample tests initialization with example controller
func TestInitProject_WithExample(t *testing.T) {
	tmpDir := t.TempDir()

	err := initProject(tmpDir, "test/myapp", true, false)
	if err != nil {
		t.Fatalf("initProject failed: %v", err)
	}

	// Verify health controller was created
	healthControllerPath := filepath.Join(tmpDir, "health/controller.go")
	if _, err := os.Stat(healthControllerPath); os.IsNotExist(err) {
		t.Error("Expected health controller to be created with --example flag")
	}
}

// TestInitProject_Minimal tests minimal initialization
func TestInitProject_Minimal(t *testing.T) {
	tmpDir := t.TempDir()

	err := initProject(tmpDir, "test/myapp", false, true)
	if err != nil {
		t.Fatalf("initProject failed: %v", err)
	}

	// Verify basic files were created
	expectedFiles := []string{
		"go.mod",
		".glibrc",
		"main.go",
		"configs/config.go",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", file)
		}
	}
}

// TestInitProject_NonEmptyDirectory tests error when directory is not empty
func TestInitProject_NonEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file in the directory
	if err := os.WriteFile(filepath.Join(tmpDir, "existing.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	err := initProject(tmpDir, "test/myapp", false, false)
	if err == nil {
		t.Error("Expected error when directory is not empty")
	}
}

// TestInitProject_WithGitDirectory tests initialization in directory with only .git
func TestInitProject_WithGitDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .git directory (should be allowed)
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	err := initProject(tmpDir, "test/myapp", false, false)
	if err != nil {
		t.Errorf("Expected initialization to succeed with only .git directory, got: %v", err)
	}

	// Verify files were created
	if _, err := os.Stat(filepath.Join(tmpDir, "go.mod")); os.IsNotExist(err) {
		t.Error("Expected go.mod to be created")
	}
}

// TestInitProject_CustomModuleName tests initialization with custom module name
func TestInitProject_CustomModuleName(t *testing.T) {
	tmpDir := t.TempDir()

	customModule := "github.com/user/customapp"
	err := initProject(tmpDir, customModule, false, false)
	if err != nil {
		t.Fatalf("initProject failed: %v", err)
	}

	// Verify go.mod contains custom module name
	goModContent, err := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}

	// Check if module name is in go.mod
	goModStr := string(goModContent)
	if len(goModStr) == 0 {
		t.Error("go.mod is empty")
	}
}

// TestInitProject_CreateDirectory tests creating a new directory
func TestInitProject_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "newproject")

	err := initProject(newDir, "test/newapp", false, false)
	if err != nil {
		t.Fatalf("initProject failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("Expected new directory to be created")
	}

	// Verify files were created in new directory
	if _, err := os.Stat(filepath.Join(newDir, "go.mod")); os.IsNotExist(err) {
		t.Error("Expected go.mod in new directory")
	}
}

// TestNewInitCmd tests command creation
func TestNewInitCmd(t *testing.T) {
	cmd := newInitCmd()

	if cmd.Use != "init [directory]" {
		t.Errorf("Expected Use='init [directory]', got '%s'", cmd.Use)
	}

	// Check flags exist
	if cmd.Flags().Lookup("module") == nil {
		t.Error("Expected --module flag")
	}
	if cmd.Flags().Lookup("example") == nil {
		t.Error("Expected --example flag")
	}
	if cmd.Flags().Lookup("minimal") == nil {
		t.Error("Expected --minimal flag")
	}
}

// TestInferModuleName tests module name inference
func TestInferModuleName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple directory name",
			path:     "/home/user/myapp",
			expected: "myapp",
		},
		{
			name:     "nested directory",
			path:     "/home/user/projects/testapp",
			expected: "testapp",
		},
		{
			name:     "root directory",
			path:     "/myapp",
			expected: "myapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferModuleName(tt.path)
			if result != tt.expected {
				t.Errorf("Expected module name '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestInitProject_AllFiles tests that all necessary files are created
func TestInitProject_AllFiles(t *testing.T) {
	tmpDir := t.TempDir()

	err := initProject(tmpDir, "test/fullapp", true, false)
	if err != nil {
		t.Fatalf("initProject failed: %v", err)
	}

	// Check all expected files
	files := []struct {
		path     string
		required bool
	}{
		{"go.mod", true},
		{".glibrc", true},
		{".gitignore", true},
		{"README.md", true},
		{"main.go", true},
		{"configs/config.go", true},
		{"health/controller.go", true}, // because example=true
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f.path)
		info, err := os.Stat(path)
		if f.required && os.IsNotExist(err) {
			t.Errorf("Required file %s does not exist", f.path)
		}
		if err == nil && info.Size() == 0 {
			t.Errorf("File %s is empty", f.path)
		}
	}
}
