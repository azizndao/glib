package generator

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"

	"github.com/azizndao/glib/internal/scanner"
)

// Generator generates code from scanned project
type Generator struct {
	project   *scanner.Project
	outputDir string
	pkgName   string
}

// New creates a new generator
func New(project *scanner.Project, outputDir, pkgName string) *Generator {
	return &Generator{
		project:   project,
		outputDir: outputDir,
		pkgName:   pkgName,
	}
}

// Generate generates all code files
func (g *Generator) Generate() error {
	// Create output directory
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate files
	files := []struct {
		name      string
		generator func() (string, error)
	}{
		{"glib.gen.go", g.generateBootstrap},
		{"di.gen.go", g.generateDI},
		{"routes.gen.go", g.generateRoutes},
		{"parsers.gen.go", g.generateParsers},
		{"errors.gen.go", g.generateErrors},
	}

	for _, file := range files {
		if err := g.generateFile(file.name, file.generator); err != nil {
			return fmt.Errorf("failed to generate %s: %w", file.name, err)
		}
	}

	return nil
}

// generateFile generates a single file
func (g *Generator) generateFile(filename string, gen func() (string, error)) error {
	// Generate code
	code, err := gen()
	if err != nil {
		return err
	}

	// Format code
	formatted, err := format.Source([]byte(code))
	if err != nil {
		// If formatting fails, write unformatted code for debugging
		fmt.Printf("Warning: failed to format %s: %v\n", filename, err)
		formatted = []byte(code)
	}

	// Write file
	path := filepath.Join(g.outputDir, filename)
	if err := os.WriteFile(path, formatted, 0644); err != nil {
		return err
	}

	return nil
}

// renderTemplate is a helper for template-based generation
func (g *Generator) renderTemplate(tmpl string, data any) (string, error) {
	// TODO: Use text/template if needed, for now we'll use string building
	return tmpl, nil
}
