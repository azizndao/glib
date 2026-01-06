package scanner

import (
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"strings"
)

// ensurePackageFilesForTypeResolution ensures all .go files in packages with controllers
// are parsed and available in packageFiles for type resolution.
//
// This is critical because handler signature analysis needs access to all type definitions
// in a package, not just those in controller files. For example, if a controller references
// PostPaginationParams defined in models.go, we need models.go to be parsed.
//
// Parameters:
//   - packageFiles: Map of packagePath -> (filePath -> *ast.File) to populate
//   - packagesWithControllers: Set of package paths that contain controllers
//   - existingFileMap: Files already parsed (will not be re-parsed)
func (s *Scanner) ensurePackageFilesForTypeResolution(
	packageFiles map[string]map[string]*ast.File,
	packagesWithControllers map[string]bool,
	existingFileMap map[string]*ast.File,
) error {
	// For each package with controllers, parse ALL .go files in that package directory
	for packagePath := range packagesWithControllers {
		// Determine the directory path for this package
		relPath := strings.TrimPrefix(packagePath, s.modulePath)
		relPath = strings.TrimPrefix(relPath, "/")
		dirPath := filepath.Join(s.projectDir, filepath.FromSlash(relPath))

		// Read all .go files in this directory
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue // Skip if we can't read the directory
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			fileName := entry.Name()
			if !strings.HasSuffix(fileName, ".go") {
				continue
			}

			// Skip test files and generated files
			if strings.HasSuffix(fileName, "_test.go") || strings.HasSuffix(fileName, ".gen.go") {
				continue
			}

			filePath := filepath.Join(dirPath, fileName)

			// Skip if already in existingFileMap (avoid re-parsing)
			if existingFileMap != nil {
				if _, exists := existingFileMap[filePath]; exists {
					continue
				}
			}

			// Skip if already in packageFiles
			if files, ok := packageFiles[packagePath]; ok {
				if _, exists := files[filePath]; exists {
					continue
				}
			}

			// Parse this additional file for type definitions
			file, err := parser.ParseFile(s.fset, filePath, nil, parser.ParseComments)
			if err != nil {
				// Skip files with parse errors
				continue
			}

			// Add to packageFiles for type resolution
			if packageFiles[packagePath] == nil {
				packageFiles[packagePath] = make(map[string]*ast.File)
			}
			packageFiles[packagePath][filePath] = file
		}
	}

	return nil
}
