package errs

// ErrDetails is a marker interface for telling Encore
// the type is used for reporting error details.
//
// We require a marker method (as opposed to using interface{})
// to facilitate static analysis and to ensure the type
// can be properly serialized across the network.
type ErrDetails interface {
	ErrDetails() // marker method; it need not do anything
}

// ValidationError represents a single field validation error
type ValidationError struct {
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

// ValidationErrors is a collection of validation errors that implements ErrDetails
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// ErrDetails implements the ErrDetails interface
func (v *ValidationErrors) ErrDetails() {}

// NewValidationErrors creates a new ValidationErrors instance
func NewValidationErrors(errors []ValidationError) *ValidationErrors {
	return &ValidationErrors{Errors: errors}
}

// AddError adds a validation error to the collection
func (v *ValidationErrors) AddError(field string, messages ...string) {
	v.Errors = append(v.Errors, ValidationError{
		Field:    field,
		Messages: messages,
	})
}
