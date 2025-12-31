package errs

import (
	"testing"
)

func TestValidationErrors(t *testing.T) {
	// Test creating validation errors
	validationErrs := NewValidationErrors([]ValidationError{
		{Field: "email", Messages: []string{"must be a valid email"}},
		{Field: "password", Messages: []string{"must be at least 8 characters", "must contain a number"}},
	})

	if len(validationErrs.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(validationErrs.Errors))
	}

	if validationErrs.Errors[0].Field != "email" {
		t.Errorf("expected field 'email', got '%s'", validationErrs.Errors[0].Field)
	}

	if len(validationErrs.Errors[1].Messages) != 2 {
		t.Errorf("expected 2 messages for password, got %d", len(validationErrs.Errors[1].Messages))
	}
}

func TestValidationErrors_AddError(t *testing.T) {
	validationErrs := &ValidationErrors{}

	validationErrs.AddError("username", "is required")
	validationErrs.AddError("age", "must be positive", "must be less than 150")

	if len(validationErrs.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(validationErrs.Errors))
	}

	if validationErrs.Errors[0].Field != "username" {
		t.Errorf("expected field 'username', got '%s'", validationErrs.Errors[0].Field)
	}

	if len(validationErrs.Errors[1].Messages) != 2 {
		t.Errorf("expected 2 messages for age, got %d", len(validationErrs.Errors[1].Messages))
	}
}

func TestValidationErrors_ImplementsErrDetails(t *testing.T) {
	var _ ErrDetails = (*ValidationErrors)(nil)
}

func TestValidationErrorsInBuilder(t *testing.T) {
	validationErrs := NewValidationErrors([]ValidationError{
		{Field: "email", Messages: []string{"invalid format"}},
	})

	err := B().
		Code(InvalidArgument).
		Msg("Validation failed").
		Details(validationErrs).
		Err()

	glibErr, ok := err.(*Error)
	if !ok {
		t.Fatal("expected *Error type")
	}

	if glibErr.Code != InvalidArgument {
		t.Errorf("expected InvalidArgument code, got %v", glibErr.Code)
	}

	if glibErr.Details == nil {
		t.Fatal("expected details to be set")
	}

	details, ok := glibErr.Details.(*ValidationErrors)
	if !ok {
		t.Fatal("expected *ValidationErrors type")
	}

	if len(details.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(details.Errors))
	}

	if details.Errors[0].Field != "email" {
		t.Errorf("expected field 'email', got '%s'", details.Errors[0].Field)
	}
}
