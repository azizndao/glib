package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanWithStream(t *testing.T) {
	// Create a test project structure
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create a provider file
	providerCode := `package main

// @Provider singleton
func NewDatabase() *Database {
	return &Database{}
}

type Database struct{}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "provider.go"), []byte(providerCode), 0644); err != nil {
		t.Fatalf("Failed to create provider.go: %v", err)
	}

	// Create a controller file
	controllerCode := `package main

import (
	"context"
	"github.com/azizndao/glib"
)

// @Controller path=/api
type UserController struct {
	DB *Database
}

// @Route method=GET path=/users
func (c *UserController) List(ctx context.Context) glib.Result[[]string] {
	return glib.Ok([]string{})
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "controller.go"), []byte(controllerCode), 0644); err != nil {
		t.Fatalf("Failed to create controller.go: %v", err)
	}

	// Create scanner
	scanner, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	// Scan with streaming
	events := make(chan StreamEvent, 100)

	var (
		providerEvents   []StreamEvent
		controllerEvents []StreamEvent
		progressEvents   []StreamEvent
		completeEvents   []StreamEvent
		errorEvents      []StreamEvent
	)

	// Process events concurrently
	done := make(chan bool)
	go func() {
		defer close(done)
		for event := range events {
			switch event.Type {
			case EventProvider:
				providerEvents = append(providerEvents, event)
			case EventController:
				controllerEvents = append(controllerEvents, event)
			case EventProgress:
				progressEvents = append(progressEvents, event)
			case EventComplete:
				completeEvents = append(completeEvents, event)
			case EventError:
				errorEvents = append(errorEvents, event)
			}
		}
	}()

	project, err := scanner.ScanWithStream(events)
	<-done // Wait for all events to be processed

	if err != nil {
		t.Fatalf("ScanWithStream failed: %v", err)
	}

	// Verify results
	if len(providerEvents) != 1 {
		t.Errorf("Expected 1 provider event, got %d", len(providerEvents))
	}

	if len(controllerEvents) != 1 {
		t.Errorf("Expected 1 controller event, got %d", len(controllerEvents))
	}

	if len(progressEvents) == 0 {
		t.Error("Expected progress events, got none")
	}

	if len(completeEvents) != 1 {
		t.Errorf("Expected 1 complete event, got %d", len(completeEvents))
	}

	if len(errorEvents) > 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errorEvents), errorEvents[0].Error)
	}

	// Verify project was assembled correctly
	if len(project.Providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(project.Providers))
	}

	if len(project.Controllers) != 1 {
		t.Errorf("Expected 1 controller, got %d", len(project.Controllers))
	}

	if project.Providers[0].Name != "NewDatabase" {
		t.Errorf("Expected provider 'NewDatabase', got '%s'", project.Providers[0].Name)
	}
}

func TestStreamEventOrdering(t *testing.T) {
	// Verify that events are sent in the correct order:
	// Providers → Configs → Middleware → Controllers → Handlers → Complete

	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create files with all types
	allInOneCode := `package main

import (
	"context"
	"net/http"
	"github.com/azizndao/glib"
)

// @Provider singleton
func NewService() *Service {
	return &Service{}
}

type Service struct{}

// @Config
type AppConfig struct {
	Port int
}

// @Middleware name=auth
func Auth(w http.ResponseWriter, r *http.Request) {
}

// @Controller path=/api
type APIController struct{}

// @Route method=GET path=/test
func (c *APIController) Test(ctx context.Context) glib.Result[string] {
	return glib.Ok("ok")
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(allInOneCode), 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	scanner, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	events := make(chan StreamEvent, 100)

	var eventOrder []StreamEventType

	done := make(chan bool)
	go func() {
		defer close(done)
		for event := range events {
			eventOrder = append(eventOrder, event.Type)
		}
	}()

	_, err = scanner.ScanWithStream(events)
	<-done // Wait for all events

	if err != nil {
		t.Fatalf("ScanWithStream failed: %v", err)
	}

	// Verify order: Provider comes before Config, Config before Middleware, etc.
	providerIdx := -1
	configIdx := -1
	middlewareIdx := -1
	controllerIdx := -1
	completeIdx := -1

	for i, eventType := range eventOrder {
		switch eventType {
		case EventProvider:
			if providerIdx == -1 {
				providerIdx = i
			}
		case EventConfig:
			if configIdx == -1 {
				configIdx = i
			}
		case EventMiddleware:
			if middlewareIdx == -1 {
				middlewareIdx = i
			}
		case EventController:
			if controllerIdx == -1 {
				controllerIdx = i
			}
		case EventComplete:
			completeIdx = i
		}
	}

	if providerIdx == -1 || configIdx == -1 || middlewareIdx == -1 || controllerIdx == -1 {
		t.Fatalf("Missing expected events. Provider: %d, Config: %d, Middleware: %d, Controller: %d",
			providerIdx, configIdx, middlewareIdx, controllerIdx)
	}

	// Verify ordering
	if providerIdx > configIdx {
		t.Errorf("Provider event (%d) should come before Config event (%d)", providerIdx, configIdx)
	}
	if configIdx > middlewareIdx {
		t.Errorf("Config event (%d) should come before Middleware event (%d)", configIdx, middlewareIdx)
	}
	if middlewareIdx > controllerIdx {
		t.Errorf("Middleware event (%d) should come before Controller event (%d)", middlewareIdx, controllerIdx)
	}
	if completeIdx != len(eventOrder)-1 {
		t.Errorf("Complete event should be last, but was at index %d of %d", completeIdx, len(eventOrder))
	}
}

func TestStreamWithParallel(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create provider file
	providerCode := `package main

// @Provider singleton
func NewDB() *DB {
	return &DB{}
}

type DB struct{}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "provider.go"), []byte(providerCode), 0644); err != nil {
		t.Fatalf("Failed to create provider.go: %v", err)
	}

	// Create scanner with parallel enabled
	scanner, err := New(tmpDir, WithParallel(2))
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	events := make(chan StreamEvent, 100)

	var receivedEvents []StreamEvent

	done := make(chan bool)
	go func() {
		defer close(done)
		for event := range events {
			receivedEvents = append(receivedEvents, event)
		}
	}()

	project, err := scanner.ScanWithStream(events)
	<-done // Wait for all events

	if err != nil {
		t.Fatalf("ScanWithStream with parallel failed: %v", err)
	}

	// Verify we got events
	if len(receivedEvents) == 0 {
		t.Error("Expected events, got none")
	}

	// Verify project results match
	if len(project.Providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(project.Providers))
	}

	// Check for complete event
	hasComplete := false
	for _, event := range receivedEvents {
		if event.Type == EventComplete {
			hasComplete = true
			break
		}
	}

	if !hasComplete {
		t.Error("Expected EventComplete but didn't receive it")
	}
}

func TestStreamProgress(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	goMod := `module testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create multiple files
	for i := 0; i < 3; i++ {
		code := `package main

// @Provider singleton
func NewService` + string(rune('A'+i)) + `() *Service {
	return &Service{}
}

type Service struct{}
`
		filename := filepath.Join(tmpDir, "service"+string(rune('a'+i))+".go")
		if err := os.WriteFile(filename, []byte(code), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	scanner, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	events := make(chan StreamEvent, 100)

	var progressEvents []*ScanProgress

	done := make(chan bool)
	go func() {
		defer close(done)
		for event := range events {
			if event.Type == EventProgress && event.Progress != nil {
				progressEvents = append(progressEvents, event.Progress)
			}
		}
	}()

	_, err = scanner.ScanWithStream(events)
	<-done // Wait for all events

	if err != nil {
		t.Fatalf("ScanWithStream failed: %v", err)
	}

	// Verify we got progress updates
	if len(progressEvents) == 0 {
		t.Error("Expected progress events, got none")
	}

	// Verify progress is monotonically increasing
	for i := 1; i < len(progressEvents); i++ {
		if progressEvents[i].ScannedFiles < progressEvents[i-1].ScannedFiles {
			t.Errorf("Progress went backwards: %d -> %d",
				progressEvents[i-1].ScannedFiles, progressEvents[i].ScannedFiles)
		}
	}

	// Verify final progress matches totals
	if len(progressEvents) > 0 {
		final := progressEvents[len(progressEvents)-1]
		if final.ScannedFiles != final.TotalFiles {
			t.Errorf("Final progress mismatch: scanned %d, total %d",
				final.ScannedFiles, final.TotalFiles)
		}
	}
}
