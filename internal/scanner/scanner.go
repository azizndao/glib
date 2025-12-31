package scanner

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Scanner scans a Go project for Glib annotations
type Scanner struct {
	fset               *token.FileSet
	modulePath         string                   // e.g., "github.com/user/myapp"
	projectDir         string                   // e.g., "/path/to/myapp"
	currentPackageName string                   // Current package being scanned
	currentPackagePath string                   // Current package import path
	currentImports     map[string]string        // Maps package name to import path for current file
	currentFile        *ast.File                // Current file being scanned (for type lookups)
	typeSpecs          map[string]*ast.TypeSpec // Maps type name to TypeSpec (for current file)
}

// New creates a new scanner
func New(projectDir string) (*Scanner, error) {
	// Find module path from go.mod
	modulePath, err := findModulePath(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to find module path: %w", err)
	}

	return &Scanner{
		fset:       token.NewFileSet(),
		modulePath: modulePath,
		projectDir: projectDir,
	}, nil
}

// Scan scans the project and returns the IR
func (s *Scanner) Scan() (*Project, error) {
	project := &Project{
		Module: s.modulePath,
	}

	// First pass: collect all controllers, providers, middleware
	fileMap := make(map[string]*ast.File) // Track parsed files

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

		fileMap[path] = file

		// Scan for annotations
		if err := s.scanFile(file, path, project); err != nil {
			return fmt.Errorf("failed to scan %s: %w", path, err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Second pass: scan handlers for all controllers
	// Group files by package path
	packageFiles := make(map[string]map[string]*ast.File) // packagePath -> filePath -> file
	for filePath, file := range fileMap {
		// Calculate package path relative to module
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

	// Scan handlers with all package files available
	for filePath, file := range fileMap {
		// Find controllers in this file
		var fileControllers []*Controller
		for _, ctrl := range project.Controllers {
			if ctrl.FilePath == filePath {
				fileControllers = append(fileControllers, ctrl)
			}
		}

		if len(fileControllers) > 0 {
			// Get all files from the same package
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

	return project, nil
}

// scanFile scans a single file for annotations
func (s *Scanner) scanFile(file *ast.File, filePath string, project *Project) error {
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

	// Scan for Config struct (no annotation needed)
	if project.Config == nil {
		config, err := s.scanConfig(file, packagePath, filePath)
		if err != nil {
			return err
		}
		if config != nil {
			project.Config = config
		}
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

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	return "", fmt.Errorf("module directive not found in go.mod")
}
