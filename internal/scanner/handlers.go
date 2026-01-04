package scanner

import (
	"fmt"
	"go/ast"
	"reflect"
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
// Supports three handler patterns:
//   - (T, error): func(ctx context.Context, ...params) (T, error)  (returns data and error)
//   - error: func(ctx context.Context, ...params) error            (returns error only, no content)
//   - Raw HTTP: func(w http.ResponseWriter, r *http.Request)       (raw)
func (s *Scanner) analyzeSignature(sig *HandlerSignature, params, returns []*TypeInfo, funcDecl *ast.FuncDecl) (*HandlerSignature, error) {
	// Check for Pattern: Raw HTTP handler
	// Must have exactly 2 params: (http.ResponseWriter, *http.Request)
	if len(params) == 2 && s.isRawHTTPHandler(params) {
		// No return value required for raw handlers
		if len(returns) > 0 {
			return nil, fmt.Errorf("raw HTTP handler must not return any value, got %d return values", len(returns))
		}

		sig.Pattern = PatternRawHTTP
		sig.ReturnsError = false

		return sig, nil
	}

	// Must have at least one parameter (context.Context)
	if len(params) == 0 {
		return nil, fmt.Errorf("handler must have at least one parameter (context.Context) or be a raw handler (w, r)")
	}

	// First parameter must be context.Context
	if !params[0].IsContext {
		return nil, fmt.Errorf("first parameter must be context.Context, got %s (or use raw handler with http.ResponseWriter and *http.Request)", params[0].FullName)
	}

	// Check return types to determine pattern
	if len(returns) == 2 {
		// Pattern: (T, error) - Returns data and error
		if !returns[1].IsError {
			return nil, fmt.Errorf("second return value must be error, got %s", returns[1].FullName)
		}

		sig.Pattern = PatternDataError
		sig.ReturnsError = true

		// Analyze response struct for header and status tags
		metadata, err := s.analyzeResponseStruct(returns[0])
		if err != nil {
			return nil, fmt.Errorf("failed to analyze response struct: %w", err)
		}
		sig.ResponseMetadata = metadata

	} else if len(returns) == 1 {
		// Pattern: error - Returns error only (no content)
		if !returns[0].IsError {
			return nil, fmt.Errorf("handler must return (T, error), error, or be a raw handler (w, r), got single return of type %s", returns[0].FullName)
		}

		sig.Pattern = PatternErrorOnly
		sig.ReturnsError = true

	} else {
		return nil, fmt.Errorf("handler must return (T, error), error, or be a raw handler (w, r), got %d return values", len(returns))
	}

	// Parse remaining parameters (path params, query params, headers, and request body)
	paramNames := s.extractParamNames(funcDecl)
	sig.PathParams = []*PathParam{}
	sig.QueryParams = []*QueryParam{}
	sig.HeaderParams = []*HeaderParam{}

	for i := 1; i < len(params); i++ {
		if i >= len(paramNames) {
			continue
		}

		paramName := paramNames[i]
		paramType := params[i]

		// First, check if this is a struct with query/header tags
		// Check both pointer and non-pointer structs from same package
		typeName := paramType.Name

		// For pointer types, get the underlying type name
		if paramType.IsPointer && paramType.Name != "" {
			typeName = paramType.Name
		}

		// Only check structs from same package (not primitives, not external packages)
		if !paramType.IsSlice && !paramType.IsPrimitive &&
			(paramType.PackageName == "" || paramType.PackageName == s.currentPackageName) &&
			typeName != "" {

			// Try to parse struct tags
			queryParams, headerParams, hasJSONTags := s.parseStructTags(typeName)

			if len(queryParams) > 0 {
				// This struct has query parameters
				sig.QueryParams = append(sig.QueryParams, queryParams...)
			}

			if len(headerParams) > 0 {
				// This struct has header parameters
				sig.HeaderParams = append(sig.HeaderParams, headerParams...)
			}

			// If struct has query/header tags, store the struct type
			if len(queryParams) > 0 || len(headerParams) > 0 {
				sig.ParamsStructType = paramType
				// Check if params struct has validation tags
				sig.NeedsParamsValidation = s.hasValidationTags(typeName)

				// If struct also has JSON tags, treat JSON fields as request body
				if hasJSONTags {
					sig.RequestType = paramType
					sig.NeedsValidation = s.hasValidationTags(typeName)
				}
				continue
			}

			// If struct has JSON tags (and no query/header tags), treat as request body
			if hasJSONTags {
				sig.RequestType = paramType
				// Check if struct has validation tags
				sig.NeedsValidation = s.hasValidationTags(typeName)
			}
		}

		// Check if this is a path param (primitive type with path-like name)
		if isPathParamName(paramName) && (paramType.IsPrimitive || paramType.PackageName == "uuid") {
			sig.PathParams = append(sig.PathParams, &PathParam{
				Name:     paramName,
				Type:     paramType,
				Position: i,
			})
		} else if sig.RequestType == nil {
			// If no struct tags and not a path param, treat as request body
			// (backward compatible: structs without tags are JSON bodies)
			sig.RequestType = paramType
			// Check if struct has validation tags
			sig.NeedsValidation = s.hasValidationTags(paramType.Name)
		}
	}

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

// Common path parameter name patterns
const (
	pathParamSuffixID  = "Id"
	pathParamSuffixKey = "Key"
)

var commonPathParamNames = map[string]bool{
	"id":     true,
	"uuid":   true,
	"key":    true,
	"slug":   true,
	"userId": true,
	"postId": true,
}

// isPathParamName checks if a parameter name looks like a path parameter
func isPathParamName(name string) bool {
	return commonPathParamNames[name] ||
		len(name) >= 2 && name[len(name)-2:] == pathParamSuffixID ||
		len(name) >= 3 && name[len(name)-3:] == pathParamSuffixKey
}

// parseStructTags analyzes a struct type for query/header/json tags
// Returns query params, header params, and whether it has JSON tags (body)
func (s *Scanner) parseStructTags(typeName string) (queryParams []*QueryParam, headerParams []*HeaderParam, hasJSONTags bool) {
	// Look up TypeSpec in current file
	typeSpec, ok := s.typeSpecs[typeName]
	if !ok {
		return nil, nil, false
	}

	// Must be a struct type
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil, nil, false
	}

	// Iterate through struct fields
	for _, field := range structType.Fields.List {
		if field.Tag == nil {
			continue
		}

		// Parse struct tag
		tagValue := field.Tag.Value
		// Remove backticks
		if len(tagValue) >= 2 && tagValue[0] == '`' && tagValue[len(tagValue)-1] == '`' {
			tagValue = tagValue[1 : len(tagValue)-1]
		}

		// Get field name
		if len(field.Names) == 0 {
			continue
		}
		fieldName := field.Names[0].Name

		// Parse field type
		fieldType := s.parseType(field.Type)

		// Check for query tag
		if queryTag := parseStructTag(tagValue, "query"); queryTag != "" {
			queryParams = append(queryParams, &QueryParam{
				FieldName:  fieldName,
				ParamName:  queryTag,
				Type:       fieldType,
				IsOptional: fieldType.IsPointer,
			})
		}

		// Check for header tag
		if headerTag := parseStructTag(tagValue, "header"); headerTag != "" {
			headerParams = append(headerParams, &HeaderParam{
				FieldName:  fieldName,
				HeaderName: headerTag,
				Type:       fieldType,
				IsOptional: fieldType.IsPointer,
			})
		}

		// Check for json tag
		if jsonTag := parseStructTag(tagValue, "json"); jsonTag != "" {
			hasJSONTags = true
		}
	}

	return queryParams, headerParams, hasJSONTags
}

// parseStructTag extracts a specific tag value from struct tag string using reflect.StructTag
// Example: `query:"page" json:"title"` with key="query" returns "page"
func parseStructTag(tagString, key string) string {
	tag := reflect.StructTag(tagString)
	return tag.Get(key)
}

// hasValidationTags checks if a struct has validate tags
func (s *Scanner) hasValidationTags(typeName string) bool {
	// Look up TypeSpec in current file
	typeSpec, ok := s.typeSpecs[typeName]
	if !ok {
		return false
	}

	// Must be a struct type
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return false
	}

	// Check if any field has a validate tag
	for _, field := range structType.Fields.List {
		if field.Tag == nil {
			continue
		}

		// Parse struct tag
		tagValue := field.Tag.Value
		// Remove backticks
		if len(tagValue) >= 2 && tagValue[0] == '`' && tagValue[len(tagValue)-1] == '`' {
			tagValue = tagValue[1 : len(tagValue)-1]
		}

		// Check for validate tag
		if validateTag := parseStructTag(tagValue, "validate"); validateTag != "" {
			return true
		}
	}

	return false
}

// analyzeResponseStruct extracts header and httpstatus tags from response type
func (s *Scanner) analyzeResponseStruct(responseType *TypeInfo) (*ResponseMetadata, error) {
	// Nil response is valid (e.g., for error-only handlers)
	if responseType == nil {
		return nil, nil
	}

	// Only analyze struct types from the same package
	if responseType.IsPrimitive || responseType.IsSlice {
		return nil, nil
	}

	// External packages or empty package name - skip analysis
	if responseType.PackageName != "" && responseType.PackageName != s.currentPackageName {
		return nil, nil
	}

	// Get struct type name (handle pointers)
	typeName := responseType.Name
	if typeName == "" {
		return nil, nil
	}

	// Look up TypeSpec in current file
	typeSpec, ok := s.typeSpecs[typeName]
	if !ok {
		// Not found in current file - not an error, just no metadata
		return nil, nil
	}

	// Must be a struct type
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil, nil
	}

	metadata := &ResponseMetadata{
		HeaderFields: []*HeaderField{},
	}

	// Iterate through struct fields
	for _, field := range structType.Fields.List {
		if field.Tag == nil {
			continue
		}

		// Parse struct tag
		tagValue := field.Tag.Value
		// Remove backticks
		if len(tagValue) >= 2 && tagValue[0] == '`' && tagValue[len(tagValue)-1] == '`' {
			tagValue = tagValue[1 : len(tagValue)-1]
		}

		// Get field name
		if len(field.Names) == 0 {
			continue
		}
		fieldName := field.Names[0].Name

		// Parse field type
		fieldType := s.parseType(field.Type)

		// Check for header tag: header:"Location" or header:"ETag,omitempty"
		if headerTag := parseStructTag(tagValue, "header"); headerTag != "" {
			// Parse omitempty from header tag
			omitEmpty := false
			headerName := headerTag
			if len(headerTag) > 10 && headerTag[len(headerTag)-10:] == ",omitempty" {
				omitEmpty = true
				headerName = headerTag[:len(headerTag)-10]
			}

			metadata.HeaderFields = append(metadata.HeaderFields, &HeaderField{
				FieldName:  fieldName,
				HeaderName: headerName,
				Type:       fieldType,
				OmitEmpty:  omitEmpty,
			})
		}

		// Check for response:"httpstatus" tag
		if responseTag := parseStructTag(tagValue, "response"); responseTag == "httpstatus" {
			// Verify field type is int
			if fieldType.Name != "int" || !fieldType.IsPrimitive {
				return nil, fmt.Errorf("field %s with response:\"httpstatus\" tag must be of type int, got %s", fieldName, fieldType.FullName)
			}
			metadata.StatusCodeField = fieldName
		}
	}

	// Return nil if no metadata found
	if len(metadata.HeaderFields) == 0 && metadata.StatusCodeField == "" {
		return nil, nil
	}

	return metadata, nil
}
