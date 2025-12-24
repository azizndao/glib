// Package generators provides code generation utilities for the glib CLI.
package generators

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
	"unicode"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Generator handles code generation from templates
type Generator struct {
	templates *template.Template
}

// NewGenerator creates a new Generator instance with all templates loaded
func NewGenerator() (*Generator, error) {
	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(templatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Generator{
		templates: tmpl,
	}, nil
}

// Generate generates code from a template and writes it to a file
func (g *Generator) Generate(templateName, outputPath string, data interface{}) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("file already exists: %s", outputPath)
	}

	// Execute template
	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// GenerateString generates code from a template and returns it as a string
func (g *Generator) GenerateString(templateName string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.String(), nil
}

// templateFuncs returns template helper functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"toLower":    strings.ToLower,
		"toUpper":    strings.ToUpper,
		"toTitle":    strings.Title,
		"toCamel":    toCamelCase,
		"toSnake":    toSnakeCase,
		"toPlural":   toPlural,
		"toSingular": toSingular,
		"now":        time.Now,
		"timestamp":  timestamp,
	}
}

// toCamelCase converts a string to camelCase
func toCamelCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	if len(words) == 0 {
		return s
	}

	result := strings.ToLower(words[0])
	for _, word := range words[1:] {
		result += strings.Title(strings.ToLower(word))
	}
	return result
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

// toPlural converts a word to plural form (simple implementation)
func toPlural(s string) string {
	s = strings.ToLower(s)
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "z") || strings.HasSuffix(s, "ch") ||
		strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	return s + "s"
}

// toSingular converts a word to singular form (simple implementation)
func toSingular(s string) string {
	s = strings.ToLower(s)
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "es") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}

// Timestamp returns a timestamp string suitable for migration names
func Timestamp() string {
	return time.Now().Format("20060102150405")
}

// timestamp is the lowercase version for template use
func timestamp() string {
	return Timestamp()
}

// ModelData holds data for model template generation
type ModelData struct {
	Package       string
	Name          string
	TableName     string
	Fields        []Field
	Relationships []Relationship
	Imports       []string
	Comment       string
}

// Field represents a model field
type Field struct {
	Name    string
	Type    string
	Tags    string
	Comment string
}

// Relationship represents a model relationship
type Relationship struct {
	Name    string
	Type    string
	Tags    string
	Comment string
}

// ControllerData holds data for controller template generation
type ControllerData struct {
	Package  string
	Name     string
	Model    string
	Resource bool
	Imports  []string
	Comment  string
}

// MigrationData holds data for migration template generation
type MigrationData struct {
	Name      string
	TableName string
	Timestamp string
	Type      string // "sql" or "go"
}

// MiddlewareData holds data for middleware template generation
type MiddlewareData struct {
	Package string
	Name    string
	Imports []string
	Comment string
}
