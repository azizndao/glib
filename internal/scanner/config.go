package scanner

import (
	"fmt"
	"go/ast"
	"reflect"
	"strings"
)

// scanConfig scans for type Config struct declaration
func (s *Scanner) scanConfig(file *ast.File, packagePath, filePath string) (*Config, error) {
	var config *Config

	// Look for type Config struct
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Check if it's named "Config"
			if typeSpec.Name.Name != "Config" {
				continue
			}

			// Check if it's a struct
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Found Config struct
			config = &Config{
				Name:        "Config",
				PackageName: file.Name.Name,
				PackagePath: packagePath,
				FilePath:    filePath,
				TypeSpec:    typeSpec,
				Position:    s.fset.Position(typeSpec.Pos()),
			}

			// Parse struct fields
			fields, err := s.scanConfigFields(structType, "")
			if err != nil {
				return nil, fmt.Errorf("failed to parse Config fields: %w", err)
			}
			config.Fields = fields

			return config, nil
		}
	}

	return nil, nil
}

// scanConfigFields recursively scans struct fields and extracts config metadata
func (s *Scanner) scanConfigFields(structType *ast.StructType, envPrefix string) ([]*ConfigField, error) {
	var fields []*ConfigField

	for _, field := range structType.Fields.List {
		// Skip fields without names (embedded fields)
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name

		// Skip unexported fields
		if !ast.IsExported(fieldName) {
			continue
		}

		// Parse struct tags
		envName := ""
		defaultValue := ""
		required := false

		if field.Tag != nil {
			tagValue := strings.Trim(field.Tag.Value, "`")
			tag := reflect.StructTag(tagValue)

			// Parse env tag: `env:"PORT"`
			if envTag := tag.Get("env"); envTag != "" {
				envName = envTag
			}

			// Parse default tag: `default:"8080"`
			defaultValue = tag.Get("default")

			// Parse required tag: `required:"true"`
			if reqTag := tag.Get("required"); reqTag == "true" {
				required = true
			}
		}

		// If no env tag, auto-generate from field name with prefix
		if envName == "" {
			envName = envPrefix + toSnakeCase(fieldName)
		}
		// If env tag exists, use it as-is (user controls the full name)

		// Parse field type
		typeInfo := s.parseConfigType(field.Type)

		configField := &ConfigField{
			Name:         fieldName,
			Type:         typeInfo,
			EnvName:      strings.ToUpper(envName),
			DefaultValue: defaultValue,
			Required:     required,
			Position:     s.fset.Position(field.Pos()),
		}

		// Check if field is a nested struct
		if structType, ok := field.Type.(*ast.StructType); ok {
			configField.IsNested = true
			// Recursively scan nested struct fields
			nestedPrefix := configField.EnvName + "_"
			nestedFields, err := s.scanConfigFields(structType, nestedPrefix)
			if err != nil {
				return nil, err
			}
			configField.Fields = nestedFields
		}

		fields = append(fields, configField)
	}

	return fields, nil
}

// parseConfigType parses a field type expression and returns TypeInfo
func (s *Scanner) parseConfigType(expr ast.Expr) *TypeInfo {
	switch t := expr.(type) {
	case *ast.Ident:
		// Built-in types: string, int, bool, etc.
		typeName := t.Name
		return &TypeInfo{
			Name:        typeName,
			IsPrimitive: isPrimitiveType(typeName),
			FullName:    typeName,
		}

	case *ast.StarExpr:
		// Pointer type: *int, *string
		inner := s.parseConfigType(t.X)
		if inner != nil {
			inner.IsPointer = true
			inner.FullName = "*" + inner.FullName
		}
		return inner

	case *ast.StructType:
		// Nested struct type
		return &TypeInfo{
			Name:        "struct",
			IsPrimitive: false,
			FullName:    "struct",
		}

	case *ast.SelectorExpr:
		// Qualified type: time.Duration, url.URL
		pkgIdent, ok := t.X.(*ast.Ident)
		if !ok {
			return nil
		}

		pkgName := pkgIdent.Name
		typeName := t.Sel.Name

		// Look up full import path
		pkgPath := s.currentImports[pkgName]

		return &TypeInfo{
			Name:        typeName,
			PackageName: pkgName,
			PackagePath: pkgPath,
			IsPrimitive: false,
			FullName:    pkgName + "." + typeName,
		}

	case *ast.ArrayType:
		// Slice type: []string, []int
		elemType := s.parseConfigType(t.Elt)
		if elemType != nil {
			elemType.IsSlice = true
			elemType.FullName = "[]" + elemType.FullName
		}
		return elemType

	default:
		return &TypeInfo{
			Name:     "unknown",
			FullName: "unknown",
		}
	}
}

// toSnakeCase converts CamelCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToUpper(result.String())
}

// isPrimitiveType checks if a type is a Go primitive
func isPrimitiveType(typeName string) bool {
	primitives := map[string]bool{
		"string":  true,
		"int":     true,
		"int8":    true,
		"int16":   true,
		"int32":   true,
		"int64":   true,
		"uint":    true,
		"uint8":   true,
		"uint16":  true,
		"uint32":  true,
		"uint64":  true,
		"float32": true,
		"float64": true,
		"bool":    true,
		"byte":    true,
		"rune":    true,
	}
	return primitives[typeName]
}
