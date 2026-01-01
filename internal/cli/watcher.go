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
	watcher     *fsnotify.Watcher
	debounce    time.Duration
	rootDir     string
	excludeDirs []string
	outputDir   string // Generated output directory to exclude
	changes     chan []string
	errors      chan error
	done        chan struct{}
	mu          sync.Mutex
	pending     map[string]bool
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(rootDir string, outputDir string, debounce time.Duration) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	fw := &FileWatcher{
		watcher:     watcher,
		debounce:    debounce,
		rootDir:     rootDir,
		outputDir:   outputDir,
		excludeDirs: []string{"vendor", "node_modules", ".git", ".glib", "tmp"},
		changes:     make(chan []string, 10),
		errors:      make(chan error, 10),
		done:        make(chan struct{}),
		pending:     make(map[string]bool),
	}

	return fw, nil
}

// Start begins watching for file changes
// Returns channels for changes and errors
func (fw *FileWatcher) Start() (<-chan []string, <-chan error, error) {
	// Add all directories to watch
	if err := fw.addRecursive(fw.rootDir); err != nil {
		return nil, nil, fmt.Errorf("failed to add directories: %w", err)
	}

	// Start goroutines for event processing
	go fw.watchEvents()
	go fw.debounceChanges()

	return fw.changes, fw.errors, nil
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() error {
	close(fw.done)
	return fw.watcher.Close()
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

// shouldIncludeFile checks if a file should trigger a rebuild
func (fw *FileWatcher) shouldIncludeFile(path string) bool {
	// Only watch .go files and glib.json
	if strings.HasSuffix(path, ".go") {
		// Exclude generated files
		if strings.HasSuffix(path, ".gen.go") {
			return false
		}
		// Exclude test files (they don't affect the server)
		if strings.HasSuffix(path, "_test.go") {
			return false
		}
		return true
	}

	// Watch glib.json for config changes
	if filepath.Base(path) == "glib.json" || filepath.Base(path) == ".glibrc" {
		return true
	}

	return false
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
							_ = fw.addRecursive(event.Name)
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

// debounceChanges collects pending changes and emits them after debounce period
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

// CountWatchedFiles returns the number of .go files being watched
func (fw *FileWatcher) CountWatchedFiles() (int, error) {
	count := 0
	err := filepath.Walk(fw.rootDir, func(path string, info os.FileInfo, err error) error {
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
			count++
		}

		return nil
	})

	return count, err
}
