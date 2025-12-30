package scanner

import (
	"fmt"
	"go/ast"
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

	routePrefix := parseControllerAnnotation(ctrlAnn.Value)

	// Parse middleware annotations
	var middlewares []string
	if mwAnn := findAnnotation(annotations, "Middleware"); mwAnn != nil {
		middlewares = parseMiddlewareAnnotation(mwAnn.Value)
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
		RoutePrefix: routePrefix,
		Middlewares: middlewares,
		Fields:      fields,
		TypeSpec:    typeSpec,
		Position:    s.fset.Position(typeSpec.Pos()),
	}

	return controller, nil
}

// scanHandlers scans controller methods for @Route annotations
// This should be called after all controllers are found
func (s *Scanner) scanHandlers(file *ast.File, controllers []*Controller) error {
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

		// Parse route annotation
		method, path := parseRouteAnnotation(routeAnn.Value)
		if method == "" || path == "" {
			continue
		}

		// Parse middleware annotations
		var middlewares []string
		if mwAnn := findAnnotation(annotations, "Middleware"); mwAnn != nil {
			middlewares = parseMiddlewareAnnotation(mwAnn.Value)
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
			Name:        funcDecl.Name.Name,
			Method:      method,
			Path:        path,
			FullPath:    fullPath,
			Middlewares: middlewares,
			Signature:   signature,
			FuncDecl:    funcDecl,
			Position:    s.fset.Position(funcDecl.Pos()),
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

	case *ast.StarExpr:
		// Pointer type: *User, *gorm.DB
		typeInfo.IsPointer = true
		inner := s.parseType(t.X)
		typeInfo.Name = inner.Name
		typeInfo.PackagePath = inner.PackagePath
		typeInfo.PackageName = inner.PackageName
		typeInfo.FullName = "*" + inner.FullName
		typeInfo.IsError = inner.IsError

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
		// Qualified type: uuid.UUID, gorm.DB
		if pkgIdent, ok := t.X.(*ast.Ident); ok {
			typeInfo.PackageName = pkgIdent.Name
			typeInfo.Name = t.Sel.Name
			typeInfo.FullName = pkgIdent.Name + "." + t.Sel.Name
			typeInfo.IsContext = typeInfo.PackageName == "context" && typeInfo.Name == "Context"
		}

	case *ast.InterfaceType:
		// Interface type
		typeInfo.Name = "any"
		typeInfo.FullName = "any"
	}

	return typeInfo
}

// isPrimitive checks if a type name is a Go primitive
func isPrimitive(name string) bool {
	primitives := map[string]bool{
		"bool":       true,
		"string":     true,
		"int":        true,
		"int8":       true,
		"int16":      true,
		"int32":      true,
		"int64":      true,
		"uint":       true,
		"uint8":      true,
		"uint16":     true,
		"uint32":     true,
		"uint64":     true,
		"uintptr":    true,
		"byte":       true,
		"rune":       true,
		"float32":    true,
		"float64":    true,
		"complex64":  true,
		"complex128": true,
	}
	return primitives[name]
}

// Helper to check if type is http.ResponseWriter or *http.Request
func (s *Scanner) isHTTPType(typeInfo *TypeInfo) bool {
	if typeInfo.PackageName != "http" {
		return false
	}
	return typeInfo.Name == "ResponseWriter" || typeInfo.Name == "Request"
}
