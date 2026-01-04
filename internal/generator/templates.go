package generator

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/azizndao/glib/internal/scanner"
)

//go:embed templates/*.templ templates/parsers/*.templ
var templatesFS embed.FS

var templates *template.Template

func init() {
	var err error
	templates, err = template.New("").Funcs(templateFuncs()).ParseFS(templatesFS, "templates/*.templ", "templates/parsers/*.templ")
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
		"typeRef":         typeRef,
		"toChiMethod":     toChiMethod,
		"ToUpper":         strings.ToUpper,
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

// typeRef returns the type reference, stripping "main." prefix for main package types
func typeRef(typeInfo *scanner.TypeInfo) string {
	if typeInfo == nil {
		return "any"
	}

	// Handle builtin types (any, error, string, int, etc.) - no package prefix
	if typeInfo.IsPrimitive || typeInfo.Name == "any" || typeInfo.Name == "error" {
		prefix := ""
		if typeInfo.IsPointer {
			prefix = "*"
		}
		if typeInfo.IsSlice {
			prefix = "[]"
		}
		return prefix + typeInfo.Name
	}

	// Strip "main." prefix since main package types can't be imported
	if after, ok := strings.CutPrefix(typeInfo.FullName, "main."); ok {
		return after
	}
	return typeInfo.FullName
}

func toChiMethod(method string) string {
	method = strings.ToUpper(method)
	switch method {
	case "GET":
		return "Get"
	case "POST":
		return "Post"
	case "PUT":
		return "Put"
	case "DELETE":
		return "Delete"
	case "PATCH":
		return "Patch"
	case "HEAD":
		return "Head"
	case "OPTIONS":
		return "Options"
	case "CONNECT":
		return "Connect"
	case "TRACE":
		return "Trace"
	default:
		return "Method"
	}
}

// executeTemplate executes a template with the given data
func (g *Generator) executeTemplate(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}
	return buf.String(), nil
}
