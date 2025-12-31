package scanner

import (
	"fmt"
	"go/ast"
)

// parseHandlerSignature analyzes a handler function signature and determines its pattern
// See 02-HANDLERS.md for the 9 supported patterns
func (s *Scanner) parseHandlerSignature(funcDecl *ast.FuncDecl) (*HandlerSignature, error) {
	sig := &HandlerSignature{}

	// Parse receiver
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		recvField := funcDecl.Recv.List[0]
		sig.Receiver = &Field{
			Name: "", // Receiver name not important
			Type: s.parseType(recvField.Type),
		}
	}

	// Parse parameters
	params := funcDecl.Type.Params
	if params == nil {
		return nil, fmt.Errorf("handler must have parameters")
	}

	var paramTypes []*TypeInfo
	for _, field := range params.List {
		typeInfo := s.parseType(field.Type)

		// Handle multiple names with same type: func(a, b int)
		if len(field.Names) == 0 {
			paramTypes = append(paramTypes, typeInfo)
		} else {
			for range field.Names {
				paramTypes = append(paramTypes, typeInfo)
			}
		}
	}

	// Parse return types
	var returnTypes []*TypeInfo
	if funcDecl.Type.Results != nil {
		for _, field := range funcDecl.Type.Results.List {
			typeInfo := s.parseType(field.Type)

			if len(field.Names) == 0 {
				returnTypes = append(returnTypes, typeInfo)
			} else {
				for range field.Names {
					returnTypes = append(returnTypes, typeInfo)
				}
			}
		}
	}

	// Analyze signature to determine pattern
	return s.analyzeSignature(sig, paramTypes, returnTypes, funcDecl)
}

// analyzeSignature determines the handler pattern from parameters and returns
// Supports two handler patterns:
//   - Result: func(ctx context.Context, ...params) glib.Result[T]  (type-safe)
//   - Raw HTTP: func(w http.ResponseWriter, r *http.Request)        (raw)
func (s *Scanner) analyzeSignature(sig *HandlerSignature, params, returns []*TypeInfo, funcDecl *ast.FuncDecl) (*HandlerSignature, error) {
	// Check for Pattern: Raw HTTP handler
	// Must have exactly 2 params: (http.ResponseWriter, *http.Request)
	if len(params) == 2 && s.isRawHTTPHandler(params) {
		// No return value required for raw handlers
		if len(returns) > 0 {
			return nil, fmt.Errorf("raw HTTP handler must not return any value, got %d return values", len(returns))
		}

		sig.Pattern = PatternRawHTTP
		sig.HasRawHTTP = true
		sig.HasContext = false // Context accessed via r.Context()
		sig.ReturnsError = false

		return sig, nil
	}

	// Pattern: Type-safe handler with Result[T]
	// Must have at least one parameter (context.Context)
	if len(params) == 0 {
		return nil, fmt.Errorf("handler must have at least one parameter (context.Context) or be a raw handler (w, r)")
	}

	// First parameter must be context.Context
	if !params[0].IsContext {
		return nil, fmt.Errorf("first parameter must be context.Context, got %s (or use raw handler with http.ResponseWriter and *http.Request)", params[0].FullName)
	}
	sig.HasContext = true

	// Must return exactly one value: glib.Result[T]
	if len(returns) != 1 {
		return nil, fmt.Errorf("handler must return exactly one value of type glib.Result[T], got %d return values", len(returns))
	}

	returnType := returns[0]

	// Check if return type is glib.Result[T]
	if !s.isGlibResult(returnType) {
		return nil, fmt.Errorf("handler must return glib.Result[T], got %s", returnType.FullName)
	}

	// Extract generic type parameter T from Result[T]
	if len(returnType.TypeParams) > 0 {
		sig.ResponseType = returnType.TypeParams[0]
	}

	// Parse remaining parameters (path params and request body)
	paramNames := s.extractParamNames(funcDecl)
	sig.PathParams = []*PathParam{}

	for i := 1; i < len(params); i++ {
		if i >= len(paramNames) {
			continue
		}

		paramName := paramNames[i]
		paramType := params[i]

		// Determine if this is a path param or request body
		// Path params are identified by common names or position
		if isPathParamName(paramName) {
			sig.PathParams = append(sig.PathParams, &PathParam{
				Name:     paramName,
				Type:     paramType,
				Position: i,
			})
		} else {
			// Last non-path-param parameter is the request body
			sig.RequestType = paramType
		}
	}

	// Pattern: Unified Result[T] pattern
	sig.Pattern = PatternResult
	sig.ReturnsError = false // Result[T] handles errors internally

	return sig, nil
}

// isRawHTTPHandler checks if params match (http.ResponseWriter, *http.Request)
func (s *Scanner) isRawHTTPHandler(params []*TypeInfo) bool {
	if len(params) != 2 {
		return false
	}

	// First param must be http.ResponseWriter
	if params[0].Name != "ResponseWriter" || params[0].PackageName != "http" {
		return false
	}

	// Second param must be *http.Request
	if params[1].Name != "Request" || params[1].PackageName != "http" || !params[1].IsPointer {
		return false
	}

	return true
}

// isGlibResult checks if a type is glib.Result[T]
func (s *Scanner) isGlibResult(typeInfo *TypeInfo) bool {
	if !typeInfo.IsGeneric {
		return false
	}

	if typeInfo.Name != "Result" {
		return false
	}

	// Check if it's from glib package
	return typeInfo.PackageName == "glib" ||
		typeInfo.PackagePath == "github.com/azizndao/glib"
}

// extractParamNames extracts parameter names from function declaration
func (s *Scanner) extractParamNames(funcDecl *ast.FuncDecl) []string {
	var names []string

	if funcDecl.Type.Params == nil {
		return names
	}

	for _, field := range funcDecl.Type.Params.List {
		if len(field.Names) == 0 {
			// Unnamed parameter
			names = append(names, "")
		} else {
			for _, name := range field.Names {
				names = append(names, name.Name)
			}
		}
	}

	return names
}

// isPathParamName checks if a parameter name looks like a path parameter
func isPathParamName(name string) bool {
	// Common path parameter names
	pathParamNames := map[string]bool{
		"id":     true,
		"uuid":   true,
		"key":    true,
		"slug":   true,
		"userId": true,
		"postId": true,
	}

	return pathParamNames[name] ||
		len(name) >= 2 && name[len(name)-2:] == "Id" ||
		len(name) >= 3 && name[len(name)-3:] == "Key"
}
