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
func (s *Scanner) analyzeSignature(sig *HandlerSignature, params, returns []*TypeInfo, funcDecl *ast.FuncDecl) (*HandlerSignature, error) {
	// Check return types
	sig.ReturnsError = len(returns) > 0 && returns[len(returns)-1].IsError
	if len(returns) > 0 && !returns[0].IsError {
		sig.ResponseType = returns[0]
	}

	if len(params) == 0 {
		return nil, fmt.Errorf("handler must have at least one parameter")
	}

	// Check for raw HTTP pattern (Pattern 1-2)
	if len(params) >= 2 && s.isHTTPType(params[0]) && s.isHTTPType(params[1]) {
		sig.HasRawHTTP = true
		sig.Pattern = 1
		if len(returns) > 0 {
			sig.Pattern = 2
		}
		return sig, nil
	}

	// Check for context.Context
	if params[0].IsContext {
		sig.HasContext = true
	}

	// Determine pattern based on signature
	if len(params) == 1 {
		// Pattern 3: func(ctx context.Context)
		// Pattern 4: func(ctx context.Context) error
		// Pattern 5: func(ctx context.Context) (*Response, error)
		if sig.HasContext {
			if len(returns) == 0 {
				sig.Pattern = 3
			} else if len(returns) == 1 && sig.ReturnsError {
				sig.Pattern = 4
			} else if len(returns) == 2 && sig.ReturnsError {
				sig.Pattern = 5
			}
		}
	} else if len(params) == 2 {
		// Pattern 6: func(ctx context.Context, req Request) (*Response, error)
		// Pattern 7: func(ctx context.Context, id uuid.UUID) (*Response, error)
		if sig.HasContext {
			// Check if second param is a path parameter or request body
			secondParam := params[1]

			// Try to determine if it's a path param by checking against function param names
			paramNames := s.extractParamNames(funcDecl)
			if len(paramNames) >= 2 {
				paramName := paramNames[1]

				// Common path parameter names
				if isPathParamName(paramName) {
					sig.PathParams = []*PathParam{
						{
							Name:     paramName,
							Type:     secondParam,
							Position: 1,
						},
					}
					sig.Pattern = 7
				} else {
					// It's a request body
					sig.RequestType = secondParam
					sig.Pattern = 6
				}
			} else {
				// Default to request body if we can't determine
				sig.RequestType = secondParam
				sig.Pattern = 6
			}
		}
	} else if len(params) == 3 {
		// Pattern 8: func(ctx context.Context, id uuid.UUID, req Request) (*Response, error)
		if sig.HasContext {
			paramNames := s.extractParamNames(funcDecl)
			if len(paramNames) >= 3 {
				sig.PathParams = []*PathParam{
					{
						Name:     paramNames[1],
						Type:     params[1],
						Position: 1,
					},
				}
				sig.RequestType = params[2]
				sig.Pattern = 8
			}
		}
	} else if len(params) >= 4 {
		// Pattern 9: func(ctx context.Context, id1, id2 uuid.UUID, req Request) (*Response, error)
		if sig.HasContext {
			paramNames := s.extractParamNames(funcDecl)

			// All params except first (ctx) and last (req) are path params
			for i := 1; i < len(params)-1; i++ {
				if i < len(paramNames) {
					sig.PathParams = append(sig.PathParams, &PathParam{
						Name:     paramNames[i],
						Type:     params[i],
						Position: i,
					})
				}
			}

			sig.RequestType = params[len(params)-1]
			sig.Pattern = 9
		}
	}

	if sig.Pattern == 0 {
		return nil, fmt.Errorf("unsupported handler signature pattern")
	}

	return sig, nil
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
