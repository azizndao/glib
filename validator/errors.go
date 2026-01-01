package validator

// FieldValidation represents a field's validation state with errors and nested fields/elements
type FieldValidation struct {
	Errors   []string                    `json:"errors,omitempty"`
	Fields   map[string]*FieldValidation `json:"fields,omitempty"`
	Elements map[string]*FieldValidation `json:"elements,omitempty"`
}

// ValidationSection represents validation errors for a section (body, query, headers)
type ValidationSection struct {
	Fields map[string]*FieldValidation `json:"fields,omitempty"`
}

// ValidationErrors represents validation errors in Goyave-style nested format
// Separates body, query, and header validations
type ValidationErrors struct {
	Body    map[string]*FieldValidation `json:"body,omitempty"`
	Query   map[string]*FieldValidation `json:"query,omitempty"`
	Headers map[string]*FieldValidation `json:"headers,omitempty"`
}

// ErrDetails implements the ErrDetails interface
func (g *ValidationErrors) ErrDetails() {}

// NewValidationErrors creates a new GoyaveValidationErrors instance
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{}
}

// AddBodyError adds a validation error to the body section
func (g *ValidationErrors) AddBodyError(path []string, message string) {
	if g.Body == nil {
		g.Body = make(map[string]*FieldValidation)
	}
	addFieldError(g.Body, path, message)
}

// AddQueryError adds a validation error to the query section
func (g *ValidationErrors) AddQueryError(path []string, message string) {
	if g.Query == nil {
		g.Query = make(map[string]*FieldValidation)
	}
	addFieldError(g.Query, path, message)
}

// AddHeaderError adds a validation error to the headers section
func (g *ValidationErrors) AddHeaderError(path []string, message string) {
	if g.Headers == nil {
		g.Headers = make(map[string]*FieldValidation)
	}
	addFieldError(g.Headers, path, message)
}

// addFieldError recursively adds an error to nested field structure
func addFieldError(fields map[string]*FieldValidation, path []string, message string) {
	if len(path) == 0 {
		return
	}

	fieldName := path[0]
	if fields[fieldName] == nil {
		fields[fieldName] = &FieldValidation{}
	}

	if len(path) == 1 {
		// Leaf node - add error
		fields[fieldName].Errors = append(fields[fieldName].Errors, message)
	} else {
		// Nested path - recurse
		if fields[fieldName].Fields == nil {
			fields[fieldName].Fields = make(map[string]*FieldValidation)
		}
		addFieldError(fields[fieldName].Fields, path[1:], message)
	}
}

// NestedValidationErrors represents validation errors in simple nested/dot-notation format (deprecated)
type NestedValidationErrors struct {
	Errors map[string][]string `json:"errors"`
}

// ErrDetails implements the ErrDetails interface
func (n *NestedValidationErrors) ErrDetails() {}

// NewNestedValidationErrors creates a new NestedValidationErrors instance
func NewNestedValidationErrors(errors map[string][]string) *NestedValidationErrors {
	return &NestedValidationErrors{Errors: errors}
}
