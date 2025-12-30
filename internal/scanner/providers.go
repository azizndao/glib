package scanner

import (
	"fmt"
	"go/ast"
)

// scanProvider scans a function declaration for @Provider annotation
func (s *Scanner) scanProvider(funcDecl *ast.FuncDecl, annotation *Annotation, packageName, packagePath, filePath string) (*Provider, error) {
	lifecycle := parseProviderAnnotation(annotation.Value)

	if lifecycle != "singleton" && lifecycle != "transient" {
		return nil, fmt.Errorf("invalid provider lifecycle: %s (must be 'singleton' or 'transient')", lifecycle)
	}

	// Parse function parameters (dependencies)
	var dependencies []*Field
	if funcDecl.Type.Params != nil {
		dependencies = s.parseFields(funcDecl.Type.Params)
	}

	// Parse return type
	var returnType *TypeInfo
	if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
		// First return value is what it provides
		returnType = s.parseType(funcDecl.Type.Results.List[0].Type)

		// Validate: must return (T, error) or (T)
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
	} else {
		return nil, fmt.Errorf("provider must have return value")
	}

	provider := &Provider{
		Name:         funcDecl.Name.Name,
		FunctionName: funcDecl.Name.Name,
		PackageName:  packageName,
		PackagePath:  packagePath,
		FilePath:     filePath,
		Lifecycle:    lifecycle,
		ReturnType:   returnType,
		Dependencies: dependencies,
		FuncDecl:     funcDecl,
		Position:     s.fset.Position(funcDecl.Pos()),
	}

	return provider, nil
}

// scanMiddleware scans a function declaration for @Middleware annotation
func (s *Scanner) scanMiddleware(funcDecl *ast.FuncDecl, annotation *Annotation, packageName, packagePath, filePath string) (*Middleware, error) {
	middlewareName := annotation.Value
	if middlewareName == "" {
		return nil, fmt.Errorf("middleware name cannot be empty")
	}

	// Parse function parameters (dependencies)
	var dependencies []*Field
	if funcDecl.Type.Params != nil {
		dependencies = s.parseFields(funcDecl.Type.Params)
	}

	// Validate return type: must return func(http.Handler) http.Handler
	if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) != 1 {
		return nil, fmt.Errorf("middleware must return func(http.Handler) http.Handler")
	}

	middleware := &Middleware{
		Name:         middlewareName,
		FunctionName: funcDecl.Name.Name,
		PackageName:  packageName,
		PackagePath:  packagePath,
		FilePath:     filePath,
		Dependencies: dependencies,
		FuncDecl:     funcDecl,
		Position:     s.fset.Position(funcDecl.Pos()),
	}

	return middleware, nil
}
