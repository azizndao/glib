package scanner

import (
	"fmt"
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Streaming Scanner Usage Example:
//
//	scanner, _ := New(projectDir)
//	events := make(chan StreamEvent, 100)
//
//	// Process events as they arrive
//	go func() {
//		for event := range events {
//			switch event.Type {
//			case EventProvider:
//				fmt.Printf("Provider: %s\n", event.Provider.Name)
//				// Start DI resolution immediately!
//			case EventProgress:
//				fmt.Printf("Progress: %d/%d files\n",
//					event.Progress.ScannedFiles, event.Progress.TotalFiles)
//			case EventComplete:
//				fmt.Println("Scan complete!")
//			}
//		}
//	}()
//
//	project, _ := scanner.ScanWithStream(events)
//
// Events are streamed in order: Providers → Configs → Middleware → Controllers → Handlers → Complete
// This enables progressive processing - consumers can start working with providers before
// the full scan completes, significantly reducing time-to-first-result for large projects.

// StreamEvent represents a scan event that can be streamed
type StreamEvent struct {
	Type       StreamEventType
	Controller *Controller
	Provider   *Provider
	Middleware *Middleware
	Config     *Config
	Error      error
	FilePath   string
	Progress   *ScanProgress
}

// StreamEventType represents the type of stream event
type StreamEventType int

const (
	EventProvider   StreamEventType = iota // Provider found
	EventConfig                            // Config found
	EventController                        // Controller found (no handlers yet)
	EventMiddleware                        // Middleware found
	EventHandler                           // Handler found (sent after all controllers)
	EventProgress                          // Progress update
	EventError                             // Error occurred
	EventComplete                          // Scan complete
)

// ScanProgress tracks scanning progress
type ScanProgress struct {
	TotalFiles       int
	ScannedFiles     int
	ControllersFound int
	ProvidersFound   int
	MiddlewareFound  int
	ConfigsFound     int
}

// ScanWithStream scans the project and streams results as they're found
// Events are sent in order: Providers → Configs → Controllers → Handlers → Complete
// This allows consumers to start DI resolution while scanning continues
func (s *Scanner) ScanWithStream(events chan<- StreamEvent) (*Project, error) {
	defer close(events)

	project := &Project{
		Module: s.modulePath,
	}

	// Use parallel scanning if enabled
	if s.parallel {
		return s.scanParallelWithStream(events)
	}

	// Count total files first for progress reporting
	totalFiles := 0
	s.walkGoFiles(func(path string, info os.FileInfo) error {
		totalFiles++
		return nil
	})

	progress := &ScanProgress{
		TotalFiles: totalFiles,
	}

	// First pass: collect controllers, providers, middleware, configs
	// Stream them immediately as found
	fileMap := make(map[string]*ast.File)
	var mu sync.Mutex

	err := s.walkGoFiles(func(path string, info os.FileInfo) error {

		// Parse the file
		file, err := parser.ParseFile(s.fset, path, nil, parser.ParseComments)
		if err != nil {
			events <- StreamEvent{
				Type:     EventError,
				Error:    fmt.Errorf("failed to parse %s: %w", path, err),
				FilePath: path,
			}
			return nil // Continue scanning other files
		}

		fileMap[path] = file

		// Scan file and stream results immediately
		tempProject := &Project{Module: s.modulePath}
		if err := s.scanFile(file, path, tempProject); err != nil {
			events <- StreamEvent{
				Type:     EventError,
				Error:    fmt.Errorf("failed to scan %s: %w", path, err),
				FilePath: path,
			}
			return nil
		}

		// Stream providers first (needed for DI)
		for _, provider := range tempProject.Providers {
			mu.Lock()
			project.Providers = append(project.Providers, provider)
			progress.ProvidersFound++
			mu.Unlock()

			events <- StreamEvent{
				Type:     EventProvider,
				Provider: provider,
				FilePath: path,
			}
		}

		// Stream configs second (needed for DI)
		for _, config := range tempProject.Configs {
			mu.Lock()
			project.Configs = append(project.Configs, config)
			progress.ConfigsFound++
			mu.Unlock()

			events <- StreamEvent{
				Type:     EventConfig,
				Config:   config,
				FilePath: path,
			}
		}

		// Stream middleware third
		for _, middleware := range tempProject.Middleware {
			mu.Lock()
			project.Middleware = append(project.Middleware, middleware)
			progress.MiddlewareFound++
			mu.Unlock()

			events <- StreamEvent{
				Type:       EventMiddleware,
				Middleware: middleware,
				FilePath:   path,
			}
		}

		// Stream controllers (without handlers yet)
		for _, controller := range tempProject.Controllers {
			mu.Lock()
			project.Controllers = append(project.Controllers, controller)
			progress.ControllersFound++
			mu.Unlock()

			events <- StreamEvent{
				Type:       EventController,
				Controller: controller,
				FilePath:   path,
			}
		}

		// Update progress
		mu.Lock()
		progress.ScannedFiles++
		mu.Unlock()

		events <- StreamEvent{
			Type: EventProgress,
			Progress: &ScanProgress{
				TotalFiles:       progress.TotalFiles,
				ScannedFiles:     progress.ScannedFiles,
				ControllersFound: progress.ControllersFound,
				ProvidersFound:   progress.ProvidersFound,
				MiddlewareFound:  progress.MiddlewareFound,
				ConfigsFound:     progress.ConfigsFound,
			},
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Second pass: scan handlers (must be done after all controllers found)
	// This is NOT streamed yet as handlers need full package context
	packageFiles := make(map[string]map[string]*ast.File)
	for filePath, file := range fileMap {
		relPath, err := filepath.Rel(s.projectDir, filepath.Dir(filePath))
		if err != nil {
			continue
		}
		packagePath := s.modulePath
		if relPath != "." {
			packagePath = s.modulePath + "/" + strings.ReplaceAll(relPath, string(os.PathSeparator), "/")
		}

		if packageFiles[packagePath] == nil {
			packageFiles[packagePath] = make(map[string]*ast.File)
		}
		packageFiles[packagePath][filePath] = file
	}

	// Scan handlers for each file
	for filePath, file := range fileMap {
		// Find controllers in this file
		var fileControllers []*Controller
		for _, ctrl := range project.Controllers {
			if ctrl.FilePath == filePath {
				fileControllers = append(fileControllers, ctrl)
			}
		}

		if len(fileControllers) > 0 {
			// Get package files
			var pkgFiles []*ast.File
			for _, ctrl := range fileControllers {
				if files, ok := packageFiles[ctrl.PackagePath]; ok {
					for _, f := range files {
						pkgFiles = append(pkgFiles, f)
					}
					break
				}
			}

			// Scan handlers
			if err := s.scanHandlers(file, fileControllers, pkgFiles); err != nil {
				events <- StreamEvent{
					Type:     EventError,
					Error:    fmt.Errorf("failed to scan handlers in %s: %w", filePath, err),
					FilePath: filePath,
				}
			} else {
				// Stream handler events
				for _, ctrl := range fileControllers {
					for _, handler := range ctrl.Handlers {
						events <- StreamEvent{
							Type:       EventHandler,
							Controller: ctrl,
							FilePath:   filePath,
						}
						_ = handler // Handler is part of controller
					}
				}
			}
		}
	}

	// Scan locale files if i18n is enabled
	if err := s.scanLocaleFiles(project); err != nil {
		events <- StreamEvent{
			Type:  EventError,
			Error: err,
		}
	}

	// Send completion event
	events <- StreamEvent{
		Type: EventComplete,
		Progress: &ScanProgress{
			TotalFiles:       progress.TotalFiles,
			ScannedFiles:     progress.ScannedFiles,
			ControllersFound: progress.ControllersFound,
			ProvidersFound:   progress.ProvidersFound,
			MiddlewareFound:  progress.MiddlewareFound,
			ConfigsFound:     progress.ConfigsFound,
		},
	}

	return project, nil
}

// scanParallelWithStream implements parallel scanning with streaming
func (s *Scanner) scanParallelWithStream(events chan<- StreamEvent) (*Project, error) {
	// For now, delegate to regular parallel scan
	// TODO: Implement true streaming parallel scan in future
	project, err := s.ScanParallel()

	if err != nil {
		events <- StreamEvent{
			Type:  EventError,
			Error: err,
		}
		return nil, err
	}

	// Stream all results
	for _, provider := range project.Providers {
		events <- StreamEvent{
			Type:     EventProvider,
			Provider: provider,
		}
	}

	for _, config := range project.Configs {
		events <- StreamEvent{
			Type:   EventConfig,
			Config: config,
		}
	}

	for _, middleware := range project.Middleware {
		events <- StreamEvent{
			Type:       EventMiddleware,
			Middleware: middleware,
		}
	}

	for _, controller := range project.Controllers {
		events <- StreamEvent{
			Type:       EventController,
			Controller: controller,
		}
	}

	events <- StreamEvent{
		Type: EventComplete,
	}

	return project, nil
}
