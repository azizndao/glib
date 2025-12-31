package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

// TestScanRealProject tests scanning a realistic project structure
func TestScanRealProject(t *testing.T) {
	// Create a temporary project structure
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module testproject

go 1.21

require (
	github.com/google/uuid v1.6.0
	gorm.io/gorm v1.25.0
)
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create services directory
	servicesDir := filepath.Join(tmpDir, "services")
	if err := os.MkdirAll(servicesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a realistic provider
	providerContent := `package services

import "gorm.io/gorm"

type Database struct {
	DB *gorm.DB
}

// @Provider singleton
func NewDatabase() (*gorm.DB, error) {
	// Implementation
	return nil, nil
}

type UserService struct {
	db *gorm.DB
}

// @Provider singleton
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

type Logger struct{}

// @Provider transient
func NewLogger() *Logger {
	return &Logger{}
}
`
	if err := os.WriteFile(filepath.Join(servicesDir, "database.go"), []byte(providerContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create controllers directory
	controllersDir := filepath.Join(tmpDir, "controllers")
	if err := os.MkdirAll(controllersDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a realistic controller
	controllerContent := `package controllers

import (
	"context"
	"testproject/services"
	
	"github.com/azizndao/glib"
	"github.com/google/uuid"
)

type User struct {
	ID   uuid.UUID
	Name string
}

// @Controller path=/api/users tags=api
type UserController struct {
	UserService *services.UserService
	Logger      *services.Logger
}

// @Route method=GET path=/
func (c *UserController) List(ctx context.Context) glib.Result[[]User] {
	return glib.OK([]User{})
}

// @Route method=GET path=/{id}
func (c *UserController) Get(ctx context.Context, id uuid.UUID) glib.Result[*User] {
	return glib.OK(&User{ID: id})
}

// @Route method=POST path=/ tags=protected
func (c *UserController) Create(ctx context.Context, req CreateUserRequest) glib.Result[*User] {
	return glib.Created(&User{})
}

type CreateUserRequest struct {
	Name string ` + "`json:\"name\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(controllersDir, "user_controller.go"), []byte(controllerContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create middleware directory
	middlewareDir := filepath.Join(tmpDir, "middleware")
	if err := os.MkdirAll(middlewareDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a realistic middleware
	middlewareContent := `package middleware

import (
	"net/http"
)

// @Middleware name=auth target=protected order=100
func Auth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}

// @Middleware name=logger target=all order=1
func Logger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
`
	if err := os.WriteFile(filepath.Join(middlewareDir, "middleware.go"), []byte(middlewareContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create config directory
	configsDir := filepath.Join(tmpDir, "configs")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a realistic config
	configContent := `package configs

// @Config
type Config struct {
	Port     int    ` + "`env:\"PORT\" default:\"8080\"`" + `
	Database string ` + "`env:\"DATABASE_URL\" required:\"true\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(configsDir, "config.go"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Now scan the project
	scanner, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	project, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Failed to scan project: %v", err)
	}

	// Verify results
	if project.Module != "testproject" {
		t.Errorf("Expected module 'testproject', got '%s'", project.Module)
	}

	// Verify providers
	if len(project.Providers) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(project.Providers))
	}

	providerNames := make(map[string]bool)
	for _, prov := range project.Providers {
		providerNames[prov.Name] = true

		// Verify lifecycle
		if prov.Name == "NewLogger" && prov.Lifecycle != "transient" {
			t.Errorf("Expected NewLogger to be transient, got %s", prov.Lifecycle)
		}
		if prov.Name == "NewDatabase" && prov.Lifecycle != "singleton" {
			t.Errorf("Expected NewDatabase to be singleton, got %s", prov.Lifecycle)
		}
	}

	if !providerNames["NewDatabase"] {
		t.Error("NewDatabase provider not found")
	}
	if !providerNames["NewUserService"] {
		t.Error("NewUserService provider not found")
	}
	if !providerNames["NewLogger"] {
		t.Error("NewLogger provider not found")
	}

	// Verify controllers
	if len(project.Controllers) != 1 {
		t.Fatalf("Expected 1 controller, got %d", len(project.Controllers))
	}

	ctrl := project.Controllers[0]
	if ctrl.Name != "UserController" {
		t.Errorf("Expected controller name 'UserController', got '%s'", ctrl.Name)
	}
	if ctrl.RoutePrefix != "/api/users" {
		t.Errorf("Expected route prefix '/api/users', got '%s'", ctrl.RoutePrefix)
	}

	// Verify handlers
	if len(ctrl.Handlers) != 3 {
		t.Errorf("Expected 3 handlers, got %d", len(ctrl.Handlers))
	}

	handlersByName := make(map[string]*Handler)
	for _, h := range ctrl.Handlers {
		handlersByName[h.Name] = h
	}

	// Verify List handler
	if list, ok := handlersByName["List"]; ok {
		if list.Method != "GET" {
			t.Errorf("Expected List method GET, got %s", list.Method)
		}
		if list.Path != "/" {
			t.Errorf("Expected List path '/', got '%s'", list.Path)
		}
		if list.FullPath != "/api/users/" {
			t.Errorf("Expected List full path '/api/users/', got '%s'", list.FullPath)
		}
	} else {
		t.Error("List handler not found")
	}

	// Verify Get handler with path parameter
	if get, ok := handlersByName["Get"]; ok {
		if get.Method != "GET" {
			t.Errorf("Expected Get method GET, got %s", get.Method)
		}
		if get.Path != "/{id}" {
			t.Errorf("Expected Get path '/{id}', got '%s'", get.Path)
		}
		if len(get.Signature.PathParams) != 1 {
			t.Errorf("Expected 1 path parameter, got %d", len(get.Signature.PathParams))
		}
		if len(get.Signature.PathParams) > 0 && get.Signature.PathParams[0].Name != "id" {
			t.Errorf("Expected path param 'id', got '%s'", get.Signature.PathParams[0].Name)
		}
	} else {
		t.Error("Get handler not found")
	}

	// Verify Create handler with tags
	if create, ok := handlersByName["Create"]; ok {
		if create.Method != "POST" {
			t.Errorf("Expected Create method POST, got %s", create.Method)
		}
		if len(create.Tags) != 1 || create.Tags[0] != "protected" {
			t.Errorf("Expected Create tags [protected], got %v", create.Tags)
		}
	} else {
		t.Error("Create handler not found")
	}

	// Verify middleware
	if len(project.Middleware) != 2 {
		t.Errorf("Expected 2 middleware, got %d", len(project.Middleware))
	}

	middlewareByName := make(map[string]*Middleware)
	for _, mw := range project.Middleware {
		middlewareByName[mw.Name] = mw
	}

	// Verify auth middleware
	if auth, ok := middlewareByName["auth"]; ok {
		if auth.Target != "protected" {
			t.Errorf("Expected auth target 'protected', got '%s'", auth.Target)
		}
		if auth.Order != 100 {
			t.Errorf("Expected auth order 100, got %d", auth.Order)
		}
	} else {
		t.Error("auth middleware not found")
	}

	// Verify logger middleware
	if logger, ok := middlewareByName["logger"]; ok {
		if logger.Target != "all" {
			t.Errorf("Expected logger target 'all', got '%s'", logger.Target)
		}
		if logger.Order != 1 {
			t.Errorf("Expected logger order 1, got %d", logger.Order)
		}
	} else {
		t.Error("logger middleware not found")
	}

	// Verify configs
	if len(project.Configs) != 1 {
		t.Fatalf("Expected 1 config, got %d", len(project.Configs))
	}

	config := project.Configs[0]
	if config.Name != "Config" {
		t.Errorf("Expected config name 'Config', got '%s'", config.Name)
	}
	if len(config.Fields) != 2 {
		t.Errorf("Expected 2 config fields, got %d", len(config.Fields))
	}
}

// TestScanEmptyProject tests scanning an empty project
func TestScanEmptyProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal go.mod
	goModContent := `module emptyproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	scanner, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	project, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Failed to scan empty project: %v", err)
	}

	if project.Module != "emptyproject" {
		t.Errorf("Expected module 'emptyproject', got '%s'", project.Module)
	}
	if len(project.Providers) != 0 {
		t.Errorf("Expected 0 providers, got %d", len(project.Providers))
	}
	if len(project.Controllers) != 0 {
		t.Errorf("Expected 0 controllers, got %d", len(project.Controllers))
	}
	if len(project.Middleware) != 0 {
		t.Errorf("Expected 0 middleware, got %d", len(project.Middleware))
	}
	if len(project.Configs) != 0 {
		t.Errorf("Expected 0 configs, got %d", len(project.Configs))
	}
}

// TestScanInvalidProject tests scanning with invalid go.mod
func TestScanInvalidProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Don't create go.mod
	_, err := New(tmpDir)
	if err == nil {
		t.Error("Expected error when scanning directory without go.mod")
	}
}

// TestScanWithSyntaxErrors tests scanning with Go syntax errors
func TestScanWithSyntaxErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module syntaxerror

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create file with syntax error
	invalidContent := `package main

// @Controller path=/api
type Controller struct {
	// Missing closing brace
`
	if err := os.WriteFile(filepath.Join(tmpDir, "invalid.go"), []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	scanner, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	_, err = scanner.Scan()
	// Should not panic, but might have errors
	// The scanner should skip files with syntax errors gracefully
	if err != nil {
		// This is acceptable - syntax errors can cause scan to fail
		t.Logf("Expected behavior: scan failed on syntax error: %v", err)
	}
}

// TestScanMultipleControllers tests scanning multiple controllers
func TestScanMultipleControllers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goModContent := `module multicontroller

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create controllers directory
	controllersDir := filepath.Join(tmpDir, "controllers")
	if err := os.MkdirAll(controllersDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create multiple controllers
	controller1Content := `package controllers

import (
	"context"
	"github.com/azizndao/glib"
)

// @Controller path=/api/posts tags=api
type PostController struct {}

// @Route method=GET path=/
func (c *PostController) List(ctx context.Context) glib.Result[any] {
	return glib.OK(nil)
}
`
	if err := os.WriteFile(filepath.Join(controllersDir, "post.go"), []byte(controller1Content), 0644); err != nil {
		t.Fatal(err)
	}

	controller2Content := `package controllers

import (
	"context"
	"github.com/azizndao/glib"
)

// @Controller path=/api/comments tags=api
type CommentController struct {}

// @Route method=GET path=/
func (c *CommentController) List(ctx context.Context) glib.Result[any] {
	return glib.OK(nil)
}
`
	if err := os.WriteFile(filepath.Join(controllersDir, "comment.go"), []byte(controller2Content), 0644); err != nil {
		t.Fatal(err)
	}

	scanner, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}
	project, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Failed to scan project: %v", err)
	}

	if len(project.Controllers) != 2 {
		t.Fatalf("Expected 2 controllers, got %d", len(project.Controllers))
	}

	controllerNames := make(map[string]bool)
	for _, ctrl := range project.Controllers {
		controllerNames[ctrl.Name] = true
	}

	if !controllerNames["PostController"] {
		t.Error("PostController not found")
	}
	if !controllerNames["CommentController"] {
		t.Error("CommentController not found")
	}
}
