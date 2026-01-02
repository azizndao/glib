// Package cli provides the command-line interface for Glib code generation.
//
// The FileWatcher provides native file watching with debouncing for development mode.
// When files change, they are collected during the debounce period and emitted as a batch.
// This prevents excessive rebuilds when multiple files are saved rapidly.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher watches for file changes in the project directory
type FileWatcher struct {
	watcher      *fsnotify.Watcher
	debounce     time.Duration
	rootDir      string
	excludeDirs  []string
	outputDir    string   // Generated output directory to exclude
	includeFiles []string // File patterns to include (e.g., "*.go")
	excludeFiles []string // File patterns to exclude (e.g., "*_test.go", "*.gen.go")
	changes      chan []string
	errors       chan error
	done         chan struct{}
	mu           sync.Mutex
	pending      map[string]bool
	wg           sync.WaitGroup // Track goroutines for clean shutdown
	fileCount    int            // Cached file count for O(1) lookups
}

// WatchConfig holds configuration for the file watcher
type WatchConfig struct {
	Debounce     time.Duration
	ExcludeDirs  []string
	IncludeFiles []string
	ExcludeFiles []string
}

// NewFileWatcher creates a new file watcher with the given configuration
func NewFileWatcher(rootDir string, outputDir string, config *WatchConfig) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	fw := &FileWatcher{
		watcher:      watcher,
		debounce:     config.Debounce,
		rootDir:      rootDir,
		outputDir:    outputDir,
		excludeDirs:  config.ExcludeDirs,
		includeFiles: config.IncludeFiles,
		excludeFiles: config.ExcludeFiles,
		changes:      make(chan []string, 10),
		errors:       make(chan error, 10),
		done:         make(chan struct{}),
		pending:      make(map[string]bool),
	}

	// Pre-count files for O(1) CountWatchedFiles()
	err = fw.countFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to count files: %w", err)
	}

	return fw, nil
}

// countFiles walks the directory tree and counts matching files
func (fw *FileWatcher) countFiles() error {
	fw.fileCount = 0
	return filepath.Walk(fw.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if fw.shouldExcludeDir(path) {
				return filepath.SkipDir
			}
			return nil
		}

		if fw.shouldIncludeFile(path) {
			fw.fileCount++
		}

		return nil
	})
}

// Start begins watching for file changes
// Returns channels for changes and errors
func (fw *FileWatcher) Start() (<-chan []string, <-chan error, error) {
	// Add all directories to watch
	if err := fw.addRecursive(fw.rootDir); err != nil {
		return nil, nil, fmt.Errorf("failed to add directories: %w", err)
	}

	// Start goroutines for event processing
	fw.wg.Add(2)
	go func() {
		defer fw.wg.Done()
		fw.watchEvents()
	}()
	go func() {
		defer fw.wg.Done()
		fw.debounceChanges()
	}()

	return fw.changes, fw.errors, nil
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() error {
	close(fw.done)
	err := fw.watcher.Close()
	fw.wg.Wait() // Wait for goroutines to finish
	return err
}

// addRecursive adds a directory and all its subdirectories to the watcher
func (fw *FileWatcher) addRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		// Skip excluded directories
		if fw.shouldExcludeDir(path) {
			return filepath.SkipDir
		}

		// Add directory to watcher
		if err := fw.watcher.Add(path); err != nil {
			return fmt.Errorf("failed to watch %s: %w", path, err)
		}

		return nil
	})
}

// shouldExcludeDir checks if a directory should be excluded from watching
func (fw *FileWatcher) shouldExcludeDir(path string) bool {
	// Get relative path
	relPath, err := filepath.Rel(fw.rootDir, path)
	if err != nil {
		return false
	}

	// Check if path starts with excluded directory
	for _, excluded := range fw.excludeDirs {
		if relPath == excluded || strings.HasPrefix(relPath, excluded+string(filepath.Separator)) {
			return true
		}
	}

	// Exclude generated output directory
	if fw.outputDir != "" && (relPath == fw.outputDir || strings.HasPrefix(relPath, fw.outputDir+string(filepath.Separator))) {
		return true
	}

	return false
}

// shouldIncludeFile checks if a file should trigger a rebuild based on configured patterns
func (fw *FileWatcher) shouldIncludeFile(path string) bool {
	filename := filepath.Base(path)

	// First check exclude patterns (exclude takes precedence)
	for _, pattern := range fw.excludeFiles {
		if matchPattern(filename, pattern) {
			return false
		}
	}

	// Then check include patterns
	for _, pattern := range fw.includeFiles {
		if matchPattern(filename, pattern) {
			return true
		}
	}

	return false
}

// matchPattern checks if a filename matches a pattern (supports * wildcard)
func matchPattern(filename, pattern string) bool {
	// Simple pattern matching: exact match or wildcard suffix
	if pattern == filename {
		return true
	}

	// Handle wildcard patterns like "*.go", "*_test.go", "*.gen.go"
	if after, ok := strings.CutPrefix(pattern, "*"); ok {
		return strings.HasSuffix(filename, after)
	}

	// Handle exact filename matches
	return pattern == filename
}

// watchEvents processes fsnotify events
func (fw *FileWatcher) watchEvents() {
	for {
		select {
		case <-fw.done:
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Handle file changes (Write, Create, Remove, Rename)
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				if fw.shouldIncludeFile(event.Name) {
					fw.mu.Lock()
					fw.pending[event.Name] = true
					fw.mu.Unlock()
				}

				// If a new directory was created, add it to watcher
				if event.Has(fsnotify.Create) {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						if !fw.shouldExcludeDir(event.Name) {
							if err := fw.addRecursive(event.Name); err != nil {
								select {
								case fw.errors <- fmt.Errorf("failed to watch new directory %s: %w", event.Name, err):
								case <-fw.done:
								}
							}
						}
					}
				}
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			select {
			case fw.errors <- err:
			case <-fw.done:
				return
			}
		}
	}
}

// debounceChanges collects pending changes and emits them after debounce period.
//
// Note: If the changes channel is full and blocks during send, new events can still
// be added to the pending map. This is intentional - we want to capture all changes
// even if processing is slow. The next debounce cycle will include any new changes.
func (fw *FileWatcher) debounceChanges() {
	ticker := time.NewTicker(fw.debounce)
	defer ticker.Stop()

	for {
		select {
		case <-fw.done:
			return

		case <-ticker.C:
			fw.mu.Lock()
			if len(fw.pending) > 0 {
				// Collect all pending changes
				changes := make([]string, 0, len(fw.pending))
				for path := range fw.pending {
					changes = append(changes, path)
				}

				// Clear pending map
				fw.pending = make(map[string]bool)
				fw.mu.Unlock()

				// Send changes
				select {
				case fw.changes <- changes:
				case <-fw.done:
					return
				}
			} else {
				fw.mu.Unlock()
			}
		}
	}
}

// CountWatchedFiles returns the number of .go files being watched (cached, O(1))
func (fw *FileWatcher) CountWatchedFiles() (int, error) {
	return fw.fileCount, nil
}
