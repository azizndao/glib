package generator

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/azizndao/glib/internal/scanner"
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
		"parsePathParam":  parsePathParam,
		"needsStrconvPkg": needsStrconvPkg,
		"needsErrorCheck": needsErrorCheck,
		"needsUUID":       needsUUID,
		"typeRef":         typeRef,
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

// parsePathParam generates the conversion code for a path parameter
// Returns the Go code to convert a string path value to the target type
func parsePathParam(paramName, paramType string) string {
	switch paramType {
	case "string":
		return fmt.Sprintf("%s := %sStr", paramName, paramName)
	case "int":
		return fmt.Sprintf(`%s, err := strconv.Atoi(%sStr)
		if err != nil {
			writeParamError(w, "%s", %sStr, "must be a valid integer")
			return
		}`, paramName, paramName, paramName, paramName)
	case "int64":
		return fmt.Sprintf(`%s, err := strconv.ParseInt(%sStr, 10, 64)
		if err != nil {
			writeParamError(w, "%s", %sStr, "must be a valid 64-bit integer")
			return
		}`, paramName, paramName, paramName, paramName)
	case "int32":
		return fmt.Sprintf(`%sTmp, err := strconv.ParseInt(%sStr, 10, 32)
		if err != nil {
			writeParamError(w, "%s", %sStr, "must be a valid 32-bit integer")
			return
		}
		%s := int32(%sTmp)`, paramName, paramName, paramName, paramName, paramName, paramName)
	case "uint":
		return fmt.Sprintf(`%sTmp, err := strconv.ParseUint(%sStr, 10, 0)
		if err != nil {
			writeParamError(w, "%s", %sStr, "must be a valid unsigned integer")
			return
		}
		%s := uint(%sTmp)`, paramName, paramName, paramName, paramName, paramName, paramName)
	case "uint64":
		return fmt.Sprintf(`%s, err := strconv.ParseUint(%sStr, 10, 64)
		if err != nil {
			writeParamError(w, "%s", %sStr, "must be a valid 64-bit unsigned integer")
			return
		}`, paramName, paramName, paramName, paramName)
	case "bool":
		return fmt.Sprintf(`%s, err := strconv.ParseBool(%sStr)
		if err != nil {
			writeParamError(w, "%s", %sStr, "must be a valid boolean (true/false)")
			return
		}`, paramName, paramName, paramName, paramName)
	case "float64":
		return fmt.Sprintf(`%s, err := strconv.ParseFloat(%sStr, 64)
		if err != nil {
			writeParamError(w, "%s", %sStr, "must be a valid number")
			return
		}`, paramName, paramName, paramName, paramName)
	case "float32":
		return fmt.Sprintf(`%sTmp, err := strconv.ParseFloat(%sStr, 32)
		if err != nil {
			writeParamError(w, "%s", %sStr, "must be a valid 32-bit number")
			return
		}
		%s := float32(%sTmp)`, paramName, paramName, paramName, paramName, paramName, paramName)
	default:
		// For non-primitive types, just assign the string
		return fmt.Sprintf("%s := %sStr", paramName, paramName)
	}
}

// needsStrconvPkg checks if any path params need strconv package
func needsStrconvPkg(pathParams []*scanner.PathParam) bool {
	for _, param := range pathParams {
		if param.Type.IsPrimitive && param.Type.Name != "string" {
			return true
		}
	}
	return false
}

// needsErrorCheck checks if a type needs error checking (primitives other than string)
func needsErrorCheck(typeName string) bool {
	return typeName != "string"
}

// needsUUID checks if any path params use uuid.UUID
func needsUUID(pathParams []*scanner.PathParam) bool {
	for _, param := range pathParams {
		if param.Type.PackageName == "uuid" && param.Type.Name == "UUID" {
			return true
		}
	}
	return false
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
	if strings.HasPrefix(typeInfo.FullName, "main.") {
		return strings.TrimPrefix(typeInfo.FullName, "main.")
	}
	return typeInfo.FullName
}

// executeTemplate executes a template with the given data
func (g *Generator) executeTemplate(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}
	return buf.String(), nil
}
