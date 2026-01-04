package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/azizndao/glib/internal/scanner"
)

// I18nGenerator generates i18n translation code
type I18nGenerator struct {
	project    *scanner.Project
	pkgName    string
	config     I18nConfig
	baseLocale *scanner.LocaleFile
}

// I18nConfig holds i18n generation configuration
type I18nConfig struct {
	Enabled          bool
	LocalesDir       string
	DefaultLocale    string
	SupportedLocales []string
	DetectFrom       []string
	QueryParam       string
}

// I18nSection represents a nested translation section
type I18nSection struct {
	Name       string                  // e.g., "Errors", "ErrorsAuth"
	StructName string                  // e.g., "TranslatorErrors", "TranslatorErrorsAuth"
	Path       []string                // e.g., ["errors"], ["errors", "auth"]
	Children   map[string]*I18nSection // Nested subsections
	Methods    []*I18nMethod           // Translation methods in this section
	Parent     *I18nSection            // Parent section (nil for root)
}

// I18nMethod represents a generated translation method
type I18nMethod struct {
	Name     string                      // e.g., "TokenExpired"
	Key      string                      // e.g., "errors.auth.token_expired"
	Template string                      // e.g., "Session expired %d minutes ago"
	Params   []*scanner.TranslationParam // Type-safe parameters
	Comment  string                      // Generated documentation
}

// NewI18nGenerator creates a new i18n generator
func NewI18nGenerator(project *scanner.Project, pkgName string, config I18nConfig) *I18nGenerator {
	return &I18nGenerator{
		project: project,
		pkgName: pkgName,
		config:  config,
	}
}

// Generate generates the i18n translation code as multiple files
func (g *I18nGenerator) Generate() (map[string]string, error) {
	if len(g.project.LocaleFiles) == 0 {
		return nil, fmt.Errorf("no locale files found")
	}

	// Find base locale (default locale)
	g.baseLocale = g.findBaseLocale()
	if g.baseLocale == nil {
		return nil, fmt.Errorf("default locale %s not found", g.config.DefaultLocale)
	}

	// Validate translation completeness
	if err := scanner.ValidateTranslationCompleteness(g.project.LocaleFiles, g.config.DefaultLocale); err != nil {
		return nil, fmt.Errorf("translation validation failed: %w", err)
	}

	// Validate format verbs match across locales
	if err := scanner.ValidateFormatVerbs(g.project.LocaleFiles); err != nil {
		return nil, fmt.Errorf("format verb validation failed: %w", err)
	}

	// Build nested structure
	structure := g.buildNestedStructure()

	// Generate multiple files
	files := make(map[string]string)

	// Generate main translator.go
	mainCode, err := g.generateMainFile(structure)
	if err != nil {
		return nil, err
	}
	files["translator.go"] = mainCode

	// Generate separate file for each root section
	for _, child := range structure.Children {
		filename := strings.ToLower(child.Name) + ".go"
		code, err := g.generateSectionFile(child)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s: %w", filename, err)
		}
		files[filename] = code
	}

	return files, nil
}

// prepareTemplateData prepares data for the i18n template
func (g *I18nGenerator) prepareTemplateData(structure *I18nSection) map[string]any {
	// Collect all sections (flatten the tree for template)
	var sections []map[string]any
	g.collectSections(structure, &sections)

	// Collect root children
	var rootChildren []map[string]any
	for _, child := range structure.Children {
		rootChildren = append(rootChildren, map[string]any{
			"Name":       child.Name,
			"StructName": child.StructName,
			"Children":   g.getChildrenData(child),
		})
	}

	// Check detection sources
	hasQuery := false
	hasHeader := false
	for _, source := range g.config.DetectFrom {
		if source == "query" {
			hasQuery = true
		}
		if source == "header" {
			hasHeader = true
		}
	}

	return map[string]any{
		"RootChildren":       rootChildren,
		"Sections":           sections,
		"HasQueryDetection":  hasQuery,
		"HasHeaderDetection": hasHeader,
		"QueryParam":         g.config.QueryParam,
		"SupportedLocales":   g.config.SupportedLocales,
		"DefaultLocale":      g.config.DefaultLocale,
	}
}

// generateMainFile generates the main translator.go file
func (g *I18nGenerator) generateMainFile(structure *I18nSection) (string, error) {
	// Collect root children
	var rootChildren []map[string]any
	for _, child := range structure.Children {
		rootChildren = append(rootChildren, map[string]any{
			"Name":       child.Name,
			"StructName": child.StructName,
			"Children":   g.getChildrenData(child),
		})
	}

	// Check detection sources
	hasQuery := false
	hasHeader := false
	for _, source := range g.config.DetectFrom {
		if source == "query" {
			hasQuery = true
		}
		if source == "header" {
			hasHeader = true
		}
	}

	data := map[string]any{
		"RootChildren":       rootChildren,
		"HasQueryDetection":  hasQuery,
		"HasHeaderDetection": hasHeader,
		"QueryParam":         g.config.QueryParam,
		"SupportedLocales":   g.config.SupportedLocales,
		"DefaultLocale":      g.config.DefaultLocale,
	}

	gen := &Generator{
		project: g.project,
		pkgName: "i18n",
	}

	return gen.executeTemplate("i18n_main.templ", data)
}

// generateSectionFile generates a file for a specific section
func (g *I18nGenerator) generateSectionFile(section *I18nSection) (string, error) {
	// Collect all sections in this tree
	var sections []map[string]any
	g.collectSections(section, &sections)

	data := map[string]any{
		"RootSection": map[string]any{
			"Name":       section.Name,
			"StructName": section.StructName,
			"Children":   g.getChildrenData(section),
		},
		"Sections": sections,
	}

	gen := &Generator{
		project: g.project,
		pkgName: "i18n",
	}

	return gen.executeTemplate("i18n_section.templ", data)
}

// getChildrenData extracts children data for template
func (g *I18nGenerator) getChildrenData(section *I18nSection) []map[string]any {
	var children []map[string]any
	for _, child := range section.Children {
		children = append(children, map[string]any{
			"Name":       child.Name,
			"StructName": child.StructName,
		})
	}
	return children
}

// collectSections recursively collects all sections for template
func (g *I18nGenerator) collectSections(section *I18nSection, sections *[]map[string]any) {
	// Skip root
	if section.Parent == nil {
		for _, child := range section.Children {
			g.collectSections(child, sections)
		}
		return
	}

	// Prepare methods data
	var methods []map[string]any
	for _, method := range section.Methods {
		methods = append(methods, map[string]any{
			"Name":       method.Name,
			"Key":        method.Key,
			"Comment":    method.Comment,
			"Params":     method.Params,
			"StructName": section.StructName, // Add struct name to each method
		})
	}

	// Prepare children data
	var children []map[string]any
	for _, child := range section.Children {
		children = append(children, map[string]any{
			"Name":       child.Name,
			"StructName": child.StructName,
		})
	}

	*sections = append(*sections, map[string]any{
		"StructName": section.StructName,
		"PathString": strings.Join(section.Path, "."),
		"Children":   children,
		"Methods":    methods,
	})

	// Recursively collect children
	for _, child := range section.Children {
		g.collectSections(child, sections)
	}
}

// Remove old generateCode and related string-building methods since we're using templates now

// findBaseLocale finds the default locale file
func (g *I18nGenerator) findBaseLocale() *scanner.LocaleFile {
	for _, locale := range g.project.LocaleFiles {
		if locale.Code == g.config.DefaultLocale {
			return locale
		}
	}
	return nil
}

// buildNestedStructure organizes translations into nested sections
func (g *I18nGenerator) buildNestedStructure() *I18nSection {
	root := &I18nSection{
		Name:       "Translator",
		StructName: "Translator",
		Path:       []string{},
		Children:   make(map[string]*I18nSection),
		Methods:    []*I18nMethod{},
		Parent:     nil,
	}

	// Add all translations to structure
	for _, translation := range g.baseLocale.Translations {
		g.addTranslationToStructure(root, translation)
	}

	// Sort children and methods for deterministic output
	g.sortStructure(root)

	return root
}

// addTranslationToStructure adds a translation to the nested structure
func (g *I18nGenerator) addTranslationToStructure(root *I18nSection, translation *scanner.Translation) {
	// Navigate to the correct section
	currentSection := root
	for i, part := range translation.Section {
		sectionName := capitalizeFirst(part)

		// Check if section already exists
		child, exists := currentSection.Children[part]
		if !exists {
			// Create new section
			pathSoFar := append([]string{}, translation.Section[:i+1]...)
			structName := g.buildStructName(pathSoFar)

			child = &I18nSection{
				Name:       sectionName,
				StructName: structName,
				Path:       pathSoFar,
				Children:   make(map[string]*I18nSection),
				Methods:    []*I18nMethod{},
				Parent:     currentSection,
			}
			currentSection.Children[part] = child
		}

		currentSection = child
	}

	// Add method to the section
	method := g.buildMethod(translation)
	currentSection.Methods = append(currentSection.Methods, method)
}

// buildStructName generates struct name from path (e.g., ["errors", "auth"] -> "TranslatorErrorsAuth")
func (g *I18nGenerator) buildStructName(path []string) string {
	parts := []string{"Translator"}
	for _, p := range path {
		parts = append(parts, capitalizeFirst(p))
	}
	return strings.Join(parts, "")
}

// buildMethod creates an I18nMethod from a Translation
func (g *I18nGenerator) buildMethod(translation *scanner.Translation) *I18nMethod {
	methodName := toPascalCase(translation.Name)

	return &I18nMethod{
		Name:     methodName,
		Key:      translation.Key,
		Template: translation.Template,
		Params:   translation.Parameters,
		Comment:  translation.Template, // Just the template string, template will add formatting
	}
}

// sortStructure recursively sorts sections and methods for deterministic output
func (g *I18nGenerator) sortStructure(section *I18nSection) {
	// Sort methods by name
	sort.Slice(section.Methods, func(i, j int) bool {
		return section.Methods[i].Name < section.Methods[j].Name
	})

	// Recursively sort children
	for _, child := range section.Children {
		g.sortStructure(child)
	}
}

// Helper: convert snake_case or kebab-case to PascalCase
func toPascalCase(s string) string {
	// Replace underscores and hyphens with spaces
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")

	// Split into words and capitalize each
	words := strings.Fields(s)
	for i, word := range words {
		words[i] = capitalizeFirst(word)
	}

	return strings.Join(words, "")
}
