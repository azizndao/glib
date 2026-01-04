package scanner

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ScanParallel performs parallel scanning using a worker pool
func (s *Scanner) ScanParallel() (*Project, error) {
	if !s.parallel {
		// Parallel not enabled, use regular scan
		return s.Scan()
	}

	project := &Project{
		Module: s.modulePath,
	}

	// Create worker pool
	pool := NewWorkerPool(s, s.workers)
	pool.Start()

	// Collect file paths to process
	var filePaths []string
	fileInfos := make(map[string]os.FileInfo)

	err := s.walkGoFiles(func(path string, info os.FileInfo) error {

		filePaths = append(filePaths, path)
		fileInfos[path] = info
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Submit jobs to worker pool
	go func() {
		for _, path := range filePaths {
			info := fileInfos[path]

			// Check cache if enabled
			if s.cacheEnabled && s.cache != nil {
				if cached, hash, err := s.cache.isFileCached(path); err == nil && cached {
					if entry, ok := s.cache.GetByHash(path, hash); ok {
						s.mu.Lock()
						s.stats.CacheHits++
						s.stats.FilesScanned++
						s.mu.Unlock()
						pool.SubmitCached(path, entry)
						continue
					}
				}
			}

			// Submit for parsing
			s.mu.Lock()
			s.stats.CacheMisses++
			s.stats.FilesScanned++
			s.mu.Unlock()
			pool.Submit(path, info)
		}
		pool.Close()
	}()

	// Collect results
	fileMap := make(map[string]*ast.File) // For handlers pass
	var mu sync.Mutex

	// Process results concurrently
	var wg sync.WaitGroup
	wg.Add(2) // One for results, one for errors

	// Error collector
	var scanErrors []error
	go func() {
		defer wg.Done()
		for err := range pool.Errors() {
			mu.Lock()
			scanErrors = append(scanErrors, err)
			mu.Unlock()
		}
	}()

	// Result collector
	go func() {
		defer wg.Done()
		for result := range pool.Results() {
			mu.Lock()

			// Add to project
			project.Controllers = append(project.Controllers, result.Controllers...)
			project.Providers = append(project.Providers, result.Providers...)
			project.Middleware = append(project.Middleware, result.Middleware...)
			project.Configs = append(project.Configs, result.Configs...)

			// Keep file for handlers pass if it has controllers
			if len(result.Controllers) > 0 && result.File != nil {
				fileMap[result.FilePath] = result.File
			}

			// Cache the result if enabled
			if s.cacheEnabled && s.cache != nil && result.Hash != "" {
				entry := &CacheEntry{
					Hash:        result.Hash,
					ModTime:     result.ModTime,
					LastScan:    time.Now(),
					Controllers: result.Controllers,
					Providers:   result.Providers,
					Middleware:  result.Middleware,
					Configs:     result.Configs,
				}
				s.cache.Set(result.FilePath, entry)
			}

			mu.Unlock()
		}
	}()

	wg.Wait()

	// Check for errors
	if len(scanErrors) > 0 {
		return nil, fmt.Errorf("scan errors: %v", scanErrors[0])
	}

	// Save cache if enabled
	if s.cacheEnabled && s.cache != nil {
		_ = s.cache.Save()
	}

	// Second pass: scan handlers for all controllers (still sequential for now)
	if err := s.scanHandlersForControllers(project, fileMap); err != nil {
		return nil, err
	}

	// Update statistics
	s.stats.Providers = len(project.Providers)
	s.stats.Controllers = len(project.Controllers)
	s.stats.Middleware = len(project.Middleware)
	for _, ctrl := range project.Controllers {
		s.stats.Handlers += len(ctrl.Handlers)
	}

	// Scan locale files if i18n is enabled
	if err := s.scanLocaleFiles(project); err != nil {
		return nil, err
	}

	return project, nil
}

// scanHandlersForControllers scans handlers for all controllers
func (s *Scanner) scanHandlersForControllers(project *Project, fileMap map[string]*ast.File) error {
	// Group files by package path
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

	// Scan handlers
	for filePath, file := range fileMap {
		var fileControllers []*Controller
		for _, ctrl := range project.Controllers {
			if ctrl.FilePath == filePath {
				fileControllers = append(fileControllers, ctrl)
			}
		}

		if len(fileControllers) > 0 {
			var pkgFiles []*ast.File
			for _, ctrl := range fileControllers {
				if files, ok := packageFiles[ctrl.PackagePath]; ok {
					for _, f := range files {
						pkgFiles = append(pkgFiles, f)
					}
					break
				}
			}

			if err := s.scanHandlers(file, fileControllers, pkgFiles); err != nil {
				return fmt.Errorf("failed to scan handlers in %s: %w", filePath, err)
			}
		}
	}

	return nil
}
