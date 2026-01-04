package scanner

import (
	"fmt"
	"go/ast"
)

// scanProvider scans a function declaration for @Provider annotation
func (s *Scanner) scanProvider(funcDecl *ast.FuncDecl, annotation *Annotation, packageName, packagePath, filePath string) (*Provider, error) {
	// Store current package context for type resolution
	s.currentPackageName = packageName
	s.currentPackagePath = packagePath

	lifecycle := parseProviderAnnotation(annotation.Value)

	if lifecycle != "singleton" && lifecycle != "transient" {
		return nil, fmt.Errorf("invalid provider lifecycle: %s (must be 'singleton' or 'transient')", lifecycle)
	}

	// Parse function parameters (dependencies)
	var dependencies []*Field
	if funcDecl.Type.Params != nil {
		dependencies = s.parseFields(funcDecl.Type.Params)
	}

	// Parse return type and check if it returns error
	var returnType *TypeInfo
	returnsError := false
	if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
		// First return value is what it provides
		returnType = s.parseType(funcDecl.Type.Results.List[0].Type)

		// Count return values to determine if it returns error
		returnCount := 0
		for _, field := range funcDecl.Type.Results.List {
			if len(field.Names) == 0 {
				returnCount++
			} else {
				returnCount += len(field.Names)
			}
		}

		if returnCount > 2 {
			return nil, fmt.Errorf("provider must return (T, error) or (T), got %d return values", returnCount)
		}

		// If returnCount > 1, it returns (T, error)
		returnsError = returnCount > 1
	} else {
		return nil, fmt.Errorf("provider must have return value")
	}

	provider := &Provider{
		Name:         funcDecl.Name.Name,
		FunctionName: funcDecl.Name.Name,
		PackageName:  packageName,
		PackagePath:  packagePath,
		FilePath:     filePath,
		SourceLine:   s.fset.Position(funcDecl.Pos()).Line,
		Lifecycle:    lifecycle,
		ReturnType:   returnType,
		Dependencies: dependencies,
		ReturnsError: returnsError,
	}

	return provider, nil
}

// scanMiddleware scans a function declaration for @Middleware annotation
func (s *Scanner) scanMiddleware(funcDecl *ast.FuncDecl, annotation *Annotation, packageName, packagePath, filePath string) (*Middleware, error) {
	// Store current package context for type resolution
	s.currentPackageName = packageName
	s.currentPackagePath = packagePath

	// Parse middleware definition (name, target, order)
	def := parseMiddlewareDefinition(annotation.Value)

	middlewareName := def["name"]
	if middlewareName == "" {
		return nil, fmt.Errorf("middleware name cannot be empty")
	}

	target := def["target"]
	if target == "" {
		target = "all"
	}

	order := 100
	if orderStr := def["order"]; orderStr != "" {
		if _, err := fmt.Sscanf(orderStr, "%d", &order); err != nil {
			return nil, fmt.Errorf("invalid middleware order: %s", orderStr)
		}
	}

	// Parse function parameters (dependencies)
	var dependencies []*Field
	if funcDecl.Type.Params != nil {
		dependencies = s.parseFields(funcDecl.Type.Params)
	}

	// Detect signature type: standard (func(http.Handler) http.Handler) or glib (func(Request, Next) Response)
	signature := MiddlewareSignatureStandard // default to standard
	if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) == 1 {
		returnType := funcDecl.Type.Results.List[0].Type

		if funcType, ok := returnType.(*ast.FuncType); ok {
			// Check if return type is func(glib.Request, glib.Next) glib.Response
			if s.isGlibMiddlewareSignature(funcType) {
				signature = MiddlewareSignatureGlib
			} else if s.isChiMiddlewareSignature(funcType) {
				signature = MiddlewareSignatureStandard
			}
		}
	}

	middleware := &Middleware{
		Name:         middlewareName,
		FunctionName: funcDecl.Name.Name,
		PackageName:  packageName,
		PackagePath:  packagePath,
		FilePath:     filePath,
		SourceLine:   s.fset.Position(funcDecl.Pos()).Line,
		Target:       target,
		Order:        order,
		Signature:    signature,
		Dependencies: dependencies,
	}

	return middleware, nil
}

// isChiMiddlewareSignature checks if a function type matches chi middleware pattern:
// func(http.Handler) http.Handler
func (s *Scanner) isChiMiddlewareSignature(funcType *ast.FuncType) bool {
	// Check params: should have 1 parameter (http.Handler)
	if funcType.Params == nil || len(funcType.Params.List) != 1 {
		return false
	}

	// Check param: should be http.Handler
	param := funcType.Params.List[0].Type
	if !s.isHTTPHandler(param) {
		return false
	}

	// Check return: should be http.Handler
	if funcType.Results == nil || len(funcType.Results.List) != 1 {
		return false
	}

	result := funcType.Results.List[0].Type
	return s.isHTTPHandler(result)
}

// isGlibMiddlewareSignature checks if a function type matches:
// func(glib.Request, glib.Next) glib.Response
func (s *Scanner) isGlibMiddlewareSignature(funcType *ast.FuncType) bool {
	// Check params: should have 2 parameters
	if funcType.Params == nil || len(funcType.Params.List) != 2 {
		return false
	}

	// Check param 1: glib.Request (interface)
	param1 := funcType.Params.List[0].Type
	if !s.isGlibType(param1, "Request") {
		return false
	}

	// Check param 2: glib.Next (function type)
	param2 := funcType.Params.List[1].Type
	if !s.isGlibType(param2, "Next") {
		return false
	}

	// Check return: glib.Response (struct)
	if funcType.Results == nil || len(funcType.Results.List) != 1 {
		return false
	}

	result := funcType.Results.List[0].Type
	return s.isGlibType(result, "Response")
}

// isHTTPHandler checks if a type is http.Handler
func (s *Scanner) isHTTPHandler(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		// Check if it's http.Handler
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == "http" && t.Sel.Name == "Handler"
		}
	case *ast.Ident:
		// Direct reference (imported with .)
		return t.Name == "Handler"
	}
	return false
}

// isGlibType checks if a type is from glib/pkg/middleware package
func (s *Scanner) isGlibType(expr ast.Expr, typeName string) bool {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		// Check if it's middleware.TypeName or glib.TypeName
		if ident, ok := t.X.(*ast.Ident); ok {
			return (ident.Name == "middleware" || ident.Name == "glib") && t.Sel.Name == typeName
		}
	case *ast.Ident:
		// Direct reference (imported with .)
		return t.Name == typeName
	}
	return false
}
