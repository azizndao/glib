package scanner

import (
	"fmt"
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ScanIncremental performs incremental scanning on changed files only
// changedFiles should be absolute paths to files that changed
func (s *Scanner) ScanIncremental(changedFiles []string) (*Project, error) {
	if !s.cacheEnabled || s.cache == nil {
		// No cache available, fall back to full scan
		return s.Scan()
	}

	project := &Project{
		Module: s.modulePath,
	}

	// Invalidate cache for changed files
	for _, path := range changedFiles {
		s.cache.Invalidate(path)
	}

	// Scan all files, using cache for unchanged ones
	fileMap := make(map[string]*ast.File) // Track parsed files (for handlers pass)

	err := s.walkGoFiles(func(path string, info os.FileInfo) error {

		// Check if file is in cache and unchanged
		s.stats.FilesScanned++
		if cached, hash, err := s.cache.isFileCached(path); err == nil && cached {
			if entry, ok := s.cache.GetByHash(path, hash); ok {
				// Use cached results
				s.stats.CacheHits++
				project.Controllers = append(project.Controllers, entry.Controllers...)
				project.Providers = append(project.Providers, entry.Providers...)
				project.Middleware = append(project.Middleware, entry.Middleware...)
				project.Configs = append(project.Configs, entry.Configs...)

				// If this file has controllers, we need to parse it for handlers pass
				if len(entry.Controllers) > 0 {
					file, err := parser.ParseFile(s.fset, path, nil, parser.ParseComments)
					if err != nil {
						return fmt.Errorf("failed to parse %s: %w", path, err)
					}
					fileMap[path] = file
				}
				return nil
			}
		}

		// File is new or changed, parse and scan it
		s.stats.CacheMisses++
		file, err := parser.ParseFile(s.fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		fileMap[path] = file

		// Track what we found before scanning
		beforeControllers := len(project.Controllers)
		beforeProviders := len(project.Providers)
		beforeMiddleware := len(project.Middleware)
		beforeConfigs := len(project.Configs)

		// Scan for annotations
		if err := s.scanFile(file, path, project); err != nil {
			return fmt.Errorf("failed to scan %s: %w", path, err)
		}

		// Cache the results
		hash, _ := computeFileHash(path)
		entry := &CacheEntry{
			Hash:        hash,
			ModTime:     info.ModTime(),
			LastScan:    time.Now(),
			Controllers: project.Controllers[beforeControllers:],
			Providers:   project.Providers[beforeProviders:],
			Middleware:  project.Middleware[beforeMiddleware:],
			Configs:     project.Configs[beforeConfigs:],
		}
		s.cache.Set(path, entry)

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Save cache
	_ = s.cache.Save()

	// Second pass: scan handlers for all controllers
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
				return nil, fmt.Errorf("failed to scan handlers in %s: %w", filePath, err)
			}
		}
	}

	// Scan locale files if i18n is enabled
	if s.i18nEnabled && s.i18nLocaleDir != "" {
		localesPath := filepath.Join(s.projectDir, s.i18nLocaleDir)
		if _, err := os.Stat(localesPath); err == nil {
			localeFiles, err := ScanLocales(localesPath)
			if err != nil {
				return nil, fmt.Errorf("failed to scan locales: %w", err)
			}
			project.LocaleFiles = localeFiles
		}
	}

	// Update statistics
	s.stats.FilesScanned = len(fileMap)
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

// InvalidateCache invalidates the cache for specific files
func (s *Scanner) InvalidateCache(files []string) {
	if s.cache != nil {
		for _, file := range files {
			s.cache.Invalidate(file)
		}
	}
}

// ClearCache clears all cached data
func (s *Scanner) ClearCache() error {
	if s.cache != nil {
		s.cache.Clear()
		return s.cache.Save()
	}
	return nil
}
