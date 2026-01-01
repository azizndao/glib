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
	project       *scanner.Project
	outputDir     string
	pkgName       string
	validationCfg ValidationConfig
}

// ValidationConfig holds validation settings from config.toml
type ValidationConfig struct {
	Enabled         bool
	Languages       []string
	DefaultLanguage string
}

// New creates a new generator
func New(project *scanner.Project, outputDir, pkgName string) *Generator {
	return NewWithValidation(project, outputDir, pkgName, ValidationConfig{
		Languages:       []string{"en"},
		DefaultLanguage: "en",
	})
}

// NewWithValidation creates a new generator with validation config
func NewWithValidation(project *scanner.Project, outputDir, pkgName string, validationCfg ValidationConfig) *Generator {
	return &Generator{
		project:       project,
		outputDir:     outputDir,
		pkgName:       pkgName,
		validationCfg: validationCfg,
	}
}

// Generate generates all code files
func (g *Generator) Generate() error {
	// Validate field name uniqueness before generating
	if err := g.validateUniqueNames(); err != nil {
		return err
	}

	// Create output directory
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate files
	files := []struct {
		name      string
		generator func() (string, error)
	}{
		{"di.gen.go", g.generateDI},
		{"routes.gen.go", g.generateRoutes},
		{"parsers.gen.go", g.generateParsers},
	}

	// Add validator only if validation is enabled
	if g.validationCfg.Enabled {
		files = append([]struct {
			name      string
			generator func() (string, error)
		}{{"validator.gen.go", g.generateValidator}}, files...)
	}

	// Add config loader to generated files if Configs exist
	if len(g.project.Configs) > 0 {
		files = append(files, struct {
			name      string
			generator func() (string, error)
		}{"config.gen.go", g.generateConfigLoader})
	}

	for _, file := range files {
		if err := g.generateFile(file.name, file.generator); err != nil {
			return fmt.Errorf("failed to generate %s: %w", file.name, err)
		}
	}

	// Generate .env.example if Configs exist
	if len(g.project.Configs) > 0 {
		if err := g.generateEnvExample(); err != nil {
			fmt.Printf("Warning: failed to generate .env.example: %v\n", err)
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

// generateConfigLoader generates the config loader in the generated package
func (g *Generator) generateConfigLoader() (string, error) {
	if len(g.project.Configs) == 0 {
		return "", fmt.Errorf("no configs found")
	}

	// Get config package path from first config (all configs should be in same package)
	configPackagePath := g.project.Configs[0].PackagePath

	configGen := NewConfigGenerator(g.project, g.pkgName)

	// Pass full package path (e.g., "glib/demo/configs") not just package name
	return configGen.GenerateConfigLoaderForPackage(configPackagePath)
}

// generateEnvExample generates the .env.example file
func (g *Generator) generateEnvExample() error {
	configGen := NewConfigGenerator(g.project, g.pkgName)
	content, err := configGen.GenerateEnvExample()
	if err != nil {
		return err
	}

	if content == "" {
		return nil
	}

	// Write to project root (parent of output dir)
	projectRoot := filepath.Dir(g.outputDir)
	path := filepath.Join(projectRoot, ".env.example")
	return os.WriteFile(path, []byte(content), 0644)
}

// generateValidator generates the validator initialization code
func (g *Generator) generateValidator() (string, error) {
	// Use configured default language, or default to English
	defaultLang := g.validationCfg.DefaultLanguage
	if defaultLang == "" {
		defaultLang = "en"
	}

	// Use configured languages, or use the default language
	languages := g.validationCfg.Languages
	if len(languages) == 0 {
		languages = []string{defaultLang}
	}

	// Prepare template data
	data := map[string]any{
		"PackageName":     g.pkgName,
		"Languages":       languages,
		"DefaultLanguage": defaultLang,
	}

	return g.executeTemplate("validator.templ", data)
}

// validateUniqueNames checks for field name collisions across providers, controllers, and configs
func (g *Generator) validateUniqueNames() error {
	seen := make(map[string]string) // fieldName -> source

	// Check providers
	for _, prov := range g.project.Providers {
		name := g.providerFieldName(prov)
		if source, exists := seen[name]; exists {
			return fmt.Errorf("field name collision: '%s' (provider:%s conflicts with %s)",
				name, prov.Name, source)
		}
		seen[name] = "provider:" + prov.Name
	}

	// Check controllers
	for _, ctrl := range g.project.Controllers {
		name := g.controllerFieldName(ctrl)
		if source, exists := seen[name]; exists {
			return fmt.Errorf("field name collision: '%s' (controller:%s conflicts with %s)",
				name, ctrl.Name, source)
		}
		seen[name] = "controller:" + ctrl.Name
	}

	// Check configs
	for _, cfg := range g.project.Configs {
		name := capitalize(cfg.Name)
		if source, exists := seen[name]; exists {
			return fmt.Errorf("field name collision: '%s' (config:%s conflicts with %s)",
				name, cfg.Name, source)
		}
		seen[name] = "config:" + cfg.Name
	}

	// Check middleware
	for _, mw := range g.project.Middleware {
		name := g.middlewareFieldName(mw)
		if source, exists := seen[name]; exists {
			return fmt.Errorf("field name collision: '%s' (middleware:%s conflicts with %s)",
				name, mw.Name, source)
		}
		seen[name] = "middleware:" + mw.Name
	}

	return nil
}
