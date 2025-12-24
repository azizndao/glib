package glib

// Validator defines the interface for request validation.
// This allows the core module to remain independent of specific validation implementations.
type Validator interface {
	// Validate validates a struct with a specific locale and returns validation errors
	Validate(s interface{}, locale string) error
}

// NoOpValidator is a validator that does nothing (for when validation is not needed)
type NoOpValidator struct{}

func (NoOpValidator) Validate(s interface{}, locale string) error {
	return nil
}
