package scanner

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ScanStats tracks scanning statistics
type ScanStats struct {
	FilesScanned int
	CacheHits    int
	CacheMisses  int
	Providers    int
	Controllers  int
	Middleware   int
	Handlers     int
}

// Scanner scans a Go project for Glib annotations
type Scanner struct {
	fset               *token.FileSet
	modulePath         string
	projectDir         string
	currentPackageName string
	currentPackagePath string
	currentImports     map[string]string
	currentFile        *ast.File                // Current file being scanned (for type lookups)
	typeSpecs          map[string]*ast.TypeSpec // Maps type name to TypeSpec (for current file)
	cache              *FileCache
	cacheEnabled       bool
	parallel           bool
	workers            int
	mu                 sync.Mutex
	stats              ScanStats
}

type ScannerOption func(*Scanner)

func WithCache(cacheDir string) ScannerOption {
	return func(s *Scanner) {
		s.cache = NewFileCache(cacheDir)
		s.cacheEnabled = true
	}
}

// WithParallel enables parallel file scanning with specified number of workers
// If workers is 0, it will use runtime.NumCPU()/2
func WithParallel(workers int) ScannerOption {
	return func(s *Scanner) {
		s.parallel = true
		s.workers = workers
	}
}

func New(projectDir string, opts ...ScannerOption) (*Scanner, error) {
	// Find module path from go.mod
	modulePath, err := findModulePath(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find module path: %w", err)
	}

	scanner := &Scanner{
		fset:       token.NewFileSet(),
		modulePath: modulePath,
		projectDir: projectDir,
	}

	// Apply options
	for _, opt := range opts {
		opt(scanner)
	}

	return scanner, nil
}

// Scan scans the project and returns the IR
func (s *Scanner) Scan() (*Project, error) {
	// Use parallel scanning if enabled
	if s.parallel {
		return s.ScanParallel()
	}

	project := &Project{
		Module: s.modulePath,
	}

	// Track file paths and package structure (not AST files)
	type fileInfo struct {
		path        string
		packagePath string
	}

	var allFiles []fileInfo
	controllerFiles := make(map[string][]fileInfo) // packagePath -> files with controllers

	// First pass: collect all controllers, providers, middleware
	err := filepath.Walk(s.projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip generated files
		if strings.HasSuffix(path, ".gen.go") {
			return nil
		}

		// Skip vendor, node_modules, etc.
		if strings.Contains(path, "/vendor/") ||
			strings.Contains(path, "/node_modules/") ||
			strings.Contains(path, "/.git/") {
			return nil
		}

		// Parse the file
		file, err := parser.ParseFile(s.fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		// Calculate package path
		relPath, err := filepath.Rel(s.projectDir, filepath.Dir(path))
		if err != nil {
			return err
		}
		packagePath := s.modulePath
		if relPath != "." {
			packagePath = s.modulePath + "/" + strings.ReplaceAll(relPath, string(os.PathSeparator), "/")
		}

		// Store file info (not AST)
		fInfo := fileInfo{
			path:        path,
			packagePath: packagePath,
		}
		allFiles = append(allFiles, fInfo)

		// Scan for annotations
		if err := s.scanFile(file, path, project); err != nil {
			return fmt.Errorf("failed to scan %s: %w", path, err)
		}

		// Track files that have controllers
		hasController := false
		for _, ctrl := range project.Controllers {
			if ctrl.FilePath == path {
				hasController = true
				break
			}
		}
		if hasController {
			controllerFiles[packagePath] = append(controllerFiles[packagePath], fInfo)
		}

		// AST is now GC-able after this function returns
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Second pass: scan handlers for controllers
	// Re-parse only files needed for handler scanning (files with controllers + their package files)
	for packagePath, pkgFileInfos := range controllerFiles {
		// Find all controllers in this package
		var packageControllers []*Controller
		for _, ctrl := range project.Controllers {
			if ctrl.PackagePath == packagePath {
				packageControllers = append(packageControllers, ctrl)
			}
		}

		if len(packageControllers) == 0 {
			continue
		}

		// Re-parse all files in this package for type resolution
		var packageAstFiles []*ast.File
		filesInPackage := make(map[string]*ast.File)

		// Get all files in this package (not just ones with controllers)
		for _, fInfo := range allFiles {
			if fInfo.packagePath == packagePath {
				file, err := parser.ParseFile(s.fset, fInfo.path, nil, parser.ParseComments)
				if err != nil {
					continue // Skip files that can't be parsed
				}
				packageAstFiles = append(packageAstFiles, file)
				filesInPackage[fInfo.path] = file
			}
		}

		// Scan handlers for each file that has controllers
		for _, fInfo := range pkgFileInfos {
			file, ok := filesInPackage[fInfo.path]
			if !ok {
				continue
			}

			// Find controllers in this specific file
			var fileControllers []*Controller
			for _, ctrl := range packageControllers {
				if ctrl.FilePath == fInfo.path {
					fileControllers = append(fileControllers, ctrl)
				}
			}

			if len(fileControllers) > 0 {
				if err := s.scanHandlers(file, fileControllers, packageAstFiles); err != nil {
					return nil, fmt.Errorf("failed to scan handlers in %s: %w", fInfo.path, err)
				}
			}
		}

		// AST files are now GC-able after processing this package
	}

	// Update statistics
	s.stats.FilesScanned = len(allFiles)
	s.stats.Providers = len(project.Providers)
	s.stats.Controllers = len(project.Controllers)
	s.stats.Middleware = len(project.Middleware)
	for _, ctrl := range project.Controllers {
		s.stats.Handlers += len(ctrl.Handlers)
	}

	return project, nil
}

// scanFile scans a single file for annotations
func (s *Scanner) scanFile(file *ast.File, filePath string, project *Project) error {
	// Lock for thread-safe access to mutable scanner state
	s.mu.Lock()
	defer s.mu.Unlock()

	packageName := file.Name.Name

	// Parse imports for type resolution
	s.parseImports(file)

	// Calculate package path relative to module
	relPath, err := filepath.Rel(s.projectDir, filepath.Dir(filePath))
	if err != nil {
		return err
	}
	packagePath := s.modulePath
	if relPath != "." {
		packagePath = s.modulePath + "/" + strings.ReplaceAll(relPath, string(os.PathSeparator), "/")
	}

	// Scan for @Config annotated structs
	configs, err := s.scanConfig(file, packagePath, filePath)
	if err != nil {
		return err
	}
	if len(configs) > 0 {
		project.Configs = append(project.Configs, configs...)
	}

	// Iterate through declarations
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			// Type declarations (structs) - potential controllers
			if d.Tok == token.TYPE {
				for _, spec := range d.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						annotations := extractAnnotations(d.Doc)

						// Check for @Controller
						if ctrlAnn := findAnnotation(annotations, "Controller"); ctrlAnn != nil {
							controller, err := s.scanController(typeSpec, d.Doc, packageName, packagePath, filePath)
							if err != nil {
								return err
							}
							project.Controllers = append(project.Controllers, controller)
						}
					}
				}
			}

		case *ast.FuncDecl:
			// Skip methods (functions with receivers) - they're not providers/middleware
			if d.Recv != nil {
				continue
			}

			annotations := extractAnnotations(d.Doc)

			// Check for @Provider
			if provAnn := findAnnotation(annotations, "Provider"); provAnn != nil {
				provider, err := s.scanProvider(d, provAnn, packageName, packagePath, filePath)
				if err != nil {
					return err
				}
				project.Providers = append(project.Providers, provider)
			}

			// Check for @Middleware
			if mwAnn := findAnnotation(annotations, "Middleware"); mwAnn != nil {
				middleware, err := s.scanMiddleware(d, mwAnn, packageName, packagePath, filePath)
				if err != nil {
					return err
				}
				project.Middleware = append(project.Middleware, middleware)
			}
		}
	}

	return nil
}

// findModulePath reads go.mod to find the module path
func findModulePath(projectDir string) (string, error) {
	goModPath := filepath.Join(projectDir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("go.mod not found: %w", err)
	}

	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	return "", fmt.Errorf("module directive not found in go.mod")
}

// Stats returns scanning statistics
func (s *Scanner) Stats() ScanStats {
	return s.stats
}
