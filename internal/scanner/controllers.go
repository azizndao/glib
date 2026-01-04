package scanner

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// scanController scans a type declaration for @Controller annotation
func (s *Scanner) scanController(typeSpec *ast.TypeSpec, doc *ast.CommentGroup, packageName, packagePath, filePath string) (*Controller, error) {
	// Store current package context for type resolution
	s.currentPackageName = packageName
	s.currentPackagePath = packagePath

	annotations := extractAnnotations(doc)
	ctrlAnn := findAnnotation(annotations, "Controller")
	if ctrlAnn == nil {
		return nil, fmt.Errorf("controller annotation not found")
	}

	// Parse controller annotation: path=/api/v1/post tags=api,public
	ctrlParams := parseControllerAnnotation(ctrlAnn.Value)

	routePrefix := ctrlParams["path"]
	if routePrefix == "" {
		return nil, fmt.Errorf("controller path is required")
	}

	// Parse tags from controller annotation
	var tags []string
	if tagsStr := ctrlParams["tags"]; tagsStr != "" {
		tags = parseCommaSeparated(tagsStr)
	}

	// Parse struct fields for DI
	var fields []*Field
	if structType, ok := typeSpec.Type.(*ast.StructType); ok {
		fields = s.parseFields(structType.Fields)
	}

	controller := &Controller{
		Name:        typeSpec.Name.Name,
		PackageName: packageName,
		PackagePath: packagePath,
		FilePath:    filePath,
		SourceLine:  s.fset.Position(typeSpec.Pos()).Line,
		RoutePrefix: routePrefix,
		Tags:        tags,
		Fields:      fields,
	}

	return controller, nil
}

// scanHandlers scans controller methods for @Route annotations
// This should be called after all controllers are found
// packageFiles contains all files from the same package for type resolution
func (s *Scanner) scanHandlers(file *ast.File, controllers []*Controller, packageFiles []*ast.File) error {
	// Parse imports for this file to resolve package paths
	s.parseImports(file)

	// Set current file for type lookups
	s.currentFile = file

	// Build type spec map for all files in the package
	s.buildTypeSpecMapFromPackage(packageFiles)

	// Create a map of controller names for quick lookup
	controllerMap := make(map[string]*Controller)
	for _, ctrl := range controllers {
		controllerMap[ctrl.Name] = ctrl
	}

	// Iterate through function declarations
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil {
			continue
		}

		// Get receiver type name
		receiverType := s.getReceiverTypeName(funcDecl.Recv)
		if receiverType == "" {
			continue
		}

		// Check if this is a controller method
		controller, ok := controllerMap[receiverType]
		if !ok {
			continue
		}

		// Check for @Route annotation
		annotations := extractAnnotations(funcDecl.Doc)
		routeAnn := findAnnotation(annotations, "Route")
		if routeAnn == nil {
			continue
		}

		// Parse route annotation: method=POST path=/ tags=protected with=auth,admin
		routeParams := parseRouteAnnotation(routeAnn.Value)

		method := routeParams["method"]
		path := routeParams["path"]
		if method == "" || path == "" {
			continue
		}
		method = strings.ToUpper(method)

		// Parse tags from route annotation
		var tags []string
		if tagsStr := routeParams["tags"]; tagsStr != "" {
			tags = parseCommaSeparated(tagsStr)
		}

		// Parse @With annotation (explicit middleware override)
		var with []string
		if withStr := routeParams["with"]; withStr != "" {
			with = parseCommaSeparated(withStr)
		}

		// Parse handler signature
		// Restore package context for type resolution
		s.currentPackageName = controller.PackageName
		s.currentPackagePath = controller.PackagePath

		signature, err := s.parseHandlerSignature(funcDecl)
		if err != nil {
			return fmt.Errorf("invalid handler signature for %s.%s: %w", controller.Name, funcDecl.Name.Name, err)
		}

		// Build full path
		fullPath := controller.RoutePrefix + path

		handler := &Handler{
			Name:       funcDecl.Name.Name,
			Method:     method,
			Path:       path,
			FullPath:   fullPath,
			SourceLine: s.fset.Position(funcDecl.Pos()).Line,
			Tags:       tags,
			With:       with,
			Signature:  signature,
		}

		controller.Handlers = append(controller.Handlers, handler)
	}

	return nil
}

// getReceiverTypeName extracts the type name from a receiver
func (s *Scanner) getReceiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}

	field := recv.List[0]
	switch t := field.Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}

	return ""
}

// parseFields parses struct fields for dependency injection
func (s *Scanner) parseFields(fieldList *ast.FieldList) []*Field {
	if fieldList == nil {
		return nil
	}

	var fields []*Field
	for _, field := range fieldList.List {
		if len(field.Names) == 0 {
			continue // Embedded field
		}

		typeInfo := s.parseType(field.Type)
		for _, name := range field.Names {
			fields = append(fields, &Field{
				Name: name.Name,
				Type: typeInfo,
			})
		}
	}

	return fields
}

// parseType parses a type expression into TypeInfo
func (s *Scanner) parseType(expr ast.Expr) *TypeInfo {
	typeInfo := &TypeInfo{}

	switch t := expr.(type) {
	case *ast.Ident:
		// Simple type: int, string, User, etc.
		typeInfo.Name = t.Name
		typeInfo.IsPrimitive = isPrimitive(t.Name)
		typeInfo.IsError = t.Name == "error"

		// If it's not a primitive/error and not a builtin, it's from the current package
		if !typeInfo.IsPrimitive && !typeInfo.IsError && s.currentPackageName != "" {
			typeInfo.PackageName = s.currentPackageName
			typeInfo.PackagePath = s.currentPackagePath
			typeInfo.FullName = s.currentPackageName + "." + t.Name
		} else {
			typeInfo.FullName = t.Name
		}

	case *ast.IndexExpr:
		// Generic type: Result[T], Option[User], etc.
		baseType := s.parseType(t.X)
		paramType := s.parseType(t.Index)

		typeInfo.Name = baseType.Name
		typeInfo.PackageName = baseType.PackageName
		typeInfo.PackagePath = baseType.PackagePath
		typeInfo.IsGeneric = true
		typeInfo.TypeParams = []*TypeInfo{paramType}
		typeInfo.FullName = baseType.FullName + "[" + paramType.FullName + "]"

	case *ast.IndexListExpr:
		// Generic type with multiple params: Map[K, V], etc.
		baseType := s.parseType(t.X)

		typeInfo.Name = baseType.Name
		typeInfo.PackageName = baseType.PackageName
		typeInfo.PackagePath = baseType.PackagePath
		typeInfo.IsGeneric = true

		var paramNames []string
		for _, index := range t.Indices {
			paramType := s.parseType(index)
			typeInfo.TypeParams = append(typeInfo.TypeParams, paramType)
			paramNames = append(paramNames, paramType.FullName)
		}

		typeInfo.FullName = baseType.FullName + "[" + joinStrings(paramNames, ", ") + "]"

	case *ast.StarExpr:
		// Pointer type: *User, *gorm.DB
		typeInfo.IsPointer = true
		inner := s.parseType(t.X)
		typeInfo.Name = inner.Name
		typeInfo.PackagePath = inner.PackagePath
		typeInfo.PackageName = inner.PackageName
		typeInfo.FullName = "*" + inner.FullName
		typeInfo.IsError = inner.IsError
		typeInfo.IsPrimitive = inner.IsPrimitive

	case *ast.ArrayType:
		// Slice type: []User
		if t.Len == nil { // Slice, not array
			typeInfo.IsSlice = true
			inner := s.parseType(t.Elt)
			typeInfo.Name = inner.Name
			typeInfo.PackagePath = inner.PackagePath
			typeInfo.PackageName = inner.PackageName
			typeInfo.FullName = "[]" + inner.FullName
		}

	case *ast.SelectorExpr:
		// Qualified type: uuid.UUID, gorm.DB, models.Post
		if pkgIdent, ok := t.X.(*ast.Ident); ok {
			typeInfo.PackageName = pkgIdent.Name
			typeInfo.Name = t.Sel.Name
			typeInfo.FullName = pkgIdent.Name + "." + t.Sel.Name
			typeInfo.IsContext = typeInfo.PackageName == "context" && typeInfo.Name == "Context"

			// Look up package path from imports
			if importPath, ok := s.currentImports[pkgIdent.Name]; ok {
				typeInfo.PackagePath = importPath
			}
		}

	case *ast.InterfaceType:
		// Interface type
		typeInfo.Name = "any"
		typeInfo.FullName = "any"
	}

	return typeInfo
}

// joinStrings joins strings with separator (helper for generic types)
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	var result strings.Builder
	result.WriteString(strs[0])
	for i := 1; i < len(strs); i++ {
		result.WriteString(sep)
		result.WriteString(strs[i])
	}
	return result.String()
}

// parseImports extracts import mappings from a file
func (s *Scanner) parseImports(file *ast.File) {
	s.currentImports = make(map[string]string)

	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}

		// Remove quotes from import path
		importPath := strings.Trim(imp.Path.Value, `"`)

		// Determine the package name
		var pkgName string
		if imp.Name != nil {
			// Named import: import foo "github.com/bar/baz"
			pkgName = imp.Name.Name
		} else {
			// Default import: use last segment of path
			parts := strings.Split(importPath, "/")
			pkgName = parts[len(parts)-1]
		}

		s.currentImports[pkgName] = importPath
	}
}

// buildTypeSpecMapFromPackage builds a map of type names to their TypeSpec for all files in the package
func (s *Scanner) buildTypeSpecMapFromPackage(packageFiles []*ast.File) {
	s.typeSpecs = make(map[string]*ast.TypeSpec)

	// Extract only type declarations, keeping minimal AST references
	for _, file := range packageFiles {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				s.typeSpecs[typeSpec.Name.Name] = typeSpec
			}
		}
	}
}
