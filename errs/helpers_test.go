package errs

import (
	"errors"
	"net/http"
	"testing"
)

func TestNewBadRequest(t *testing.T) {
	err := NewBadRequest().WithMessage("invalid input")

	// Verify it's an *Error
	var glibErr *Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *Error type")
	}

	// Verify error code
	if glibErr.Code != InvalidArgument {
		t.Errorf("expected code %v, got %v", InvalidArgument, glibErr.Code)
	}

	// Verify message
	if glibErr.Message != "invalid input" {
		t.Errorf("expected message 'invalid input', got %q", glibErr.Message)
	}

	// Verify HTTP status
	if glibErr.Code.HTTPStatus() != http.StatusBadRequest {
		t.Errorf("expected HTTP status %d, got %d", http.StatusBadRequest, glibErr.Code.HTTPStatus())
	}
}

func TestNewUnauthorized(t *testing.T) {
	err := NewUnauthorized().WithMessage("authentication required")

	var glibErr *Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *Error type")
	}

	if glibErr.Code != Unauthenticated {
		t.Errorf("expected code %v, got %v", Unauthenticated, glibErr.Code)
	}

	if glibErr.Code.HTTPStatus() != http.StatusUnauthorized {
		t.Errorf("expected HTTP status %d, got %d", http.StatusUnauthorized, glibErr.Code.HTTPStatus())
	}
}

func TestNewForbidden(t *testing.T) {
	err := NewForbidden().WithMessage("access denied")

	var glibErr *Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *Error type")
	}

	if glibErr.Code != PermissionDenied {
		t.Errorf("expected code %v, got %v", PermissionDenied, glibErr.Code)
	}

	if glibErr.Code.HTTPStatus() != http.StatusForbidden {
		t.Errorf("expected HTTP status %d, got %d", http.StatusForbidden, glibErr.Code.HTTPStatus())
	}
}

func TestNewNotFound(t *testing.T) {
	err := NewNotFound().WithMessage("resource not found")

	var glibErr *Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *Error type")
	}

	if glibErr.Code != NotFound {
		t.Errorf("expected code %v, got %v", NotFound, glibErr.Code)
	}

	if glibErr.Message != "resource not found" {
		t.Errorf("expected message 'resource not found', got %q", glibErr.Message)
	}

	if glibErr.Code.HTTPStatus() != http.StatusNotFound {
		t.Errorf("expected HTTP status %d, got %d", http.StatusNotFound, glibErr.Code.HTTPStatus())
	}
}

func TestNewConflict(t *testing.T) {
	err := NewConflict().WithMessage("resource already exists")

	var glibErr *Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *Error type")
	}

	if glibErr.Code != AlreadyExists {
		t.Errorf("expected code %v, got %v", AlreadyExists, glibErr.Code)
	}

	if glibErr.Code.HTTPStatus() != http.StatusConflict {
		t.Errorf("expected HTTP status %d, got %d", http.StatusConflict, glibErr.Code.HTTPStatus())
	}
}

func TestNewInternal(t *testing.T) {
	err := NewInternal().WithMessage("internal server error")

	var glibErr *Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *Error type")
	}

	if glibErr.Code != Internal {
		t.Errorf("expected code %v, got %v", Internal, glibErr.Code)
	}

	if glibErr.Code.HTTPStatus() != http.StatusInternalServerError {
		t.Errorf("expected HTTP status %d, got %d", http.StatusInternalServerError, glibErr.Code.HTTPStatus())
	}
}

func TestWithMessage(t *testing.T) {
	err := NewNotFound().WithMessage("user not found")

	// Verify it returns an error (not *Builder)
	if err == nil {
		t.Fatal("WithMessage should return non-nil error")
	}

	// Verify type
	var glibErr *Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *Error type")
	}

	// Verify message is set
	if glibErr.Message != "user not found" {
		t.Errorf("expected message 'user not found', got %q", glibErr.Message)
	}
}

func TestBuilderChaining(t *testing.T) {
	// Test that we can chain Msg() before WithMessage()
	err := NewBadRequest().Msg("base message").WithMessage("override message")

	var glibErr *Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *Error type")
	}

	// WithMessage should override previous Msg
	if glibErr.Message != "override message" {
		t.Errorf("expected 'override message', got %q", glibErr.Message)
	}
}

func TestErrorHelpers_WithMetadata(t *testing.T) {
	// Test that we can add metadata before WithMessage
	builder := NewBadRequest().Meta("userId", "123", "action", "create")
	err := builder.WithMessage("validation failed")

	var glibErr *Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *Error type")
	}

	// Verify metadata
	if glibErr.Meta["userId"] != "123" {
		t.Errorf("expected userId=123, got %v", glibErr.Meta["userId"])
	}

	if glibErr.Meta["action"] != "create" {
		t.Errorf("expected action=create, got %v", glibErr.Meta["action"])
	}
}

// ValidationDetails is a test type that implements ErrDetails
type ValidationDetails struct {
	Field  string
	Reason string
}

func (ValidationDetails) ErrDetails() {}

func TestErrorHelpers_WithDetails(t *testing.T) {
	details := ValidationDetails{
		Field:  "email",
		Reason: "invalid format",
	}

	err := NewBadRequest().Details(details).WithMessage("validation error")

	var glibErr *Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *Error type")
	}

	// Verify details
	if glibErr.Details == nil {
		t.Fatal("expected details to be set")
	}

	detailsVal, ok := glibErr.Details.(ValidationDetails)
	if !ok {
		t.Fatalf("expected details to be ValidationDetails, got %T", glibErr.Details)
	}

	if detailsVal.Field != "email" {
		t.Errorf("expected field=email, got %v", detailsVal.Field)
	}
}

func TestErrorHelpers_AllCodes(t *testing.T) {
	tests := []struct {
		name       string
		builderFn  func() *Builder
		expectCode ErrCode
		expectHTTP int
	}{
		{"BadRequest", NewBadRequest, InvalidArgument, http.StatusBadRequest},
		{"Unauthorized", NewUnauthorized, Unauthenticated, http.StatusUnauthorized},
		{"Forbidden", NewForbidden, PermissionDenied, http.StatusForbidden},
		{"NotFound", NewNotFound, NotFound, http.StatusNotFound},
		{"Conflict", NewConflict, AlreadyExists, http.StatusConflict},
		{"Internal", NewInternal, Internal, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.builderFn().WithMessage("test message")

			var glibErr *Error
			if !errors.As(err, &glibErr) {
				t.Fatal("expected *Error type")
			}

			if glibErr.Code != tt.expectCode {
				t.Errorf("expected code %v, got %v", tt.expectCode, glibErr.Code)
			}

			if glibErr.Code.HTTPStatus() != tt.expectHTTP {
				t.Errorf("expected HTTP status %d, got %d", tt.expectHTTP, glibErr.Code.HTTPStatus())
			}

			if glibErr.Message != "test message" {
				t.Errorf("expected message 'test message', got %q", glibErr.Message)
			}
		})
	}
}
