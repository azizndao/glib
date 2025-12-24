package generators

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModelGeneration(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Initialize generator
	gen, err := NewGenerator()
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Test data
	data := ModelData{
		Package:   "models",
		Name:      "User",
		TableName: "users",
		Comment:   "User represents a user record",
		Imports:   []string{},
	}

	// Generate model
	outputPath := filepath.Join(tmpDir, "user.go")
	if err := gen.Generate("model.tmpl", outputPath, data); err != nil {
		t.Fatalf("Failed to generate model: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Model file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Check for expected content
	expectedStrings := []string{
		"package models",
		"type User struct",
		"orm.Model",
		"func (User) TableName() string",
		`return "users"`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Generated content missing expected string: %s", expected)
		}
	}
}

func TestControllerGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	gen, err := NewGenerator()
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Test resource controller
	data := ControllerData{
		Package:  "controllers",
		Name:     "UserController",
		Model:    "User",
		Resource: true,
		Imports:  []string{},
	}

	outputPath := filepath.Join(tmpDir, "user_controller.go")
	if err := gen.Generate("controller.tmpl", outputPath, data); err != nil {
		t.Fatalf("Failed to generate controller: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Check for CRUD methods
	expectedMethods := []string{"Index", "Show", "Store", "Update", "Destroy"}
	for _, method := range expectedMethods {
		if !strings.Contains(contentStr, "func (ctrl *UserController) "+method) {
			t.Errorf("Generated controller missing method: %s", method)
		}
	}
}

func TestMigrationGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	gen, err := NewGenerator()
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	data := MigrationData{
		Name:      "create_users_table",
		TableName: "users",
		Timestamp: "20241224120000",
		Type:      "sql",
	}

	outputPath := filepath.Join(tmpDir, "20241224120000_create_users_table.sql")
	if err := gen.Generate("migration_sql.tmpl", outputPath, data); err != nil {
		t.Fatalf("Failed to generate migration: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Check for goose directives and table creation
	expectedStrings := []string{
		"-- +goose Up",
		"-- +goose Down",
		"CREATE TABLE users",
		"DROP TABLE users",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Generated migration missing expected string: %s", expected)
		}
	}
}

func TestMiddlewareGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	gen, err := NewGenerator()
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	data := MiddlewareData{
		Package: "middleware",
		Name:    "Auth",
		Imports: []string{},
		Comment: "Auth is a middleware function",
	}

	outputPath := filepath.Join(tmpDir, "auth.go")
	if err := gen.Generate("middleware.tmpl", outputPath, data); err != nil {
		t.Fatalf("Failed to generate middleware: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	expectedStrings := []string{
		"package middleware",
		"func Auth(next glib.Handler) glib.Handler",
		"return func(c *glib.Ctx) error",
		"return next(c)",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Generated middleware missing expected string: %s", expected)
		}
	}
}

func TestFileOverwriteProtection(t *testing.T) {
	tmpDir := t.TempDir()
	gen, err := NewGenerator()
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	data := ModelData{
		Package:   "models",
		Name:      "User",
		TableName: "users",
	}

	outputPath := filepath.Join(tmpDir, "user.go")

	// First generation should succeed
	if err := gen.Generate("model.tmpl", outputPath, data); err != nil {
		t.Fatalf("Failed first generation: %v", err)
	}

	// Second generation should fail (file exists)
	if err := gen.Generate("model.tmpl", outputPath, data); err == nil {
		t.Error("Expected error when overwriting existing file, got nil")
	}
}

func TestSnakeCaseConversion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"User", "user"},
		{"UserProfile", "user_profile"},
		{"APIKey", "a_p_i_key"},
		{"HTTPRequest", "h_t_t_p_request"},
	}

	for _, tt := range tests {
		result := toSnakeCase(tt.input)
		if result != tt.expected {
			t.Errorf("toSnakeCase(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestPluralConversion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user", "users"},
		{"post", "posts"},
		{"category", "categories"},
		{"bus", "buses"},
		{"box", "boxes"},
	}

	for _, tt := range tests {
		result := toPlural(tt.input)
		if result != tt.expected {
			t.Errorf("toPlural(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}
