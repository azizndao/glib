package generator

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl templates/parsers/*.tmpl
var templatesFS embed.FS

var templates *template.Template

func init() {
	var err error
	templates, err = template.New("").Funcs(templateFuncs()).ParseFS(templatesFS, "templates/*.tmpl", "templates/parsers/*.tmpl")
	if err != nil {
		panic(fmt.Sprintf("failed to parse templates: %v", err))
	}
}

// templateFuncs returns custom template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"capitalizeFirst": capitalizeFirst,
		"lowercaseFirst":  lowercaseFirst,
		"join":            strings.Join,
		"hasPrefix":       strings.HasPrefix,
		"hasSuffix":       strings.HasSuffix,
		"trimPrefix":      strings.TrimPrefix,
		"trimSuffix":      strings.TrimSuffix,
	}
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// lowercaseFirst lowercases the first letter of a string
func lowercaseFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// executeTemplate executes a template with the given data
func (g *Generator) executeTemplate(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}
	return buf.String(), nil
}
