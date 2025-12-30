package errs_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/goyave/glib/v2/pkg/errs"
)

func TestCode_HTTPStatus(t *testing.T) {
	tests := []struct {
		code     errs.Code
		expected int
	}{
		{errs.InvalidArgument, 400},
		{errs.Unauthenticated, 401},
		{errs.PermissionDenied, 403},
		{errs.NotFound, 404},
		{errs.AlreadyExists, 409},
		{errs.Internal, 500},
		{errs.Unavailable, 503},
	}

	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			if got := tt.code.HTTPStatus(); got != tt.expected {
				t.Errorf("HTTPStatus() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCode_String(t *testing.T) {
	tests := []struct {
		code     errs.Code
		expected string
	}{
		{errs.InvalidArgument, "invalid_argument"},
		{errs.Unauthenticated, "unauthenticated"},
		{errs.PermissionDenied, "permission_denied"},
		{errs.NotFound, "not_found"},
		{errs.AlreadyExists, "already_exists"},
		{errs.Internal, "internal"},
		{errs.Unavailable, "unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.code.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuilder_Basic(t *testing.T) {
	err := errs.B().
		Code(errs.NotFound).
		Msg("resource not found").
		Err()

	var glibErr *errs.Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *errs.Error")
	}

	if glibErr.Code() != errs.NotFound {
		t.Errorf("Code() = %v, want %v", glibErr.Code(), errs.NotFound)
	}

	if glibErr.Message() != "resource not found" {
		t.Errorf("Message() = %v, want %v", glibErr.Message(), "resource not found")
	}
}

func TestBuilder_WithMeta(t *testing.T) {
	err := errs.B().
		Code(errs.InvalidArgument).
		Msg("validation failed").
		Meta("field", "email").
		Meta("reason", "invalid format").
		Err()

	var glibErr *errs.Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *errs.Error")
	}

	meta := glibErr.Meta()
	if meta["field"] != "email" {
		t.Errorf("Meta[field] = %v, want %v", meta["field"], "email")
	}
	if meta["reason"] != "invalid format" {
		t.Errorf("Meta[reason] = %v, want %v", meta["reason"], "invalid format")
	}
}

func TestBuilder_WithDetails(t *testing.T) {
	details := map[string][]string{
		"email": {"must be valid email", "required"},
		"age":   {"must be at least 18"},
	}

	err := errs.B().
		Code(errs.InvalidArgument).
		Msg("validation failed").
		Details(details).
		Err()

	var glibErr *errs.Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *errs.Error")
	}

	gotDetails := glibErr.Details()
	if len(gotDetails["email"]) != 2 {
		t.Errorf("Details[email] length = %v, want 2", len(gotDetails["email"]))
	}
	if len(gotDetails["age"]) != 1 {
		t.Errorf("Details[age] length = %v, want 1", len(gotDetails["age"]))
	}
}

func TestBuilder_WithCause(t *testing.T) {
	cause := errors.New("database connection failed")
	err := errs.B().
		Code(errs.Internal).
		Msg("failed to fetch user").
		Cause(cause).
		Err()

	var glibErr *errs.Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *errs.Error")
	}

	if !errors.Is(err, cause) {
		t.Error("expected error chain to contain cause")
	}

	unwrapped := glibErr.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestBuilder_Msgf(t *testing.T) {
	err := errs.B().
		Code(errs.NotFound).
		Msgf("user with id %d not found", 123).
		Err()

	var glibErr *errs.Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *errs.Error")
	}

	expected := "user with id 123 not found"
	if glibErr.Message() != expected {
		t.Errorf("Message() = %v, want %v", glibErr.Message(), expected)
	}
}

func TestError_Error(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := errs.B().
			Code(errs.NotFound).
			Msg("not found").
			Err()

		if err.Error() != "not found" {
			t.Errorf("Error() = %v, want %v", err.Error(), "not found")
		}
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := errs.B().
			Code(errs.Internal).
			Msg("wrapper").
			Cause(cause).
			Err()

		expected := "wrapper: underlying error"
		if err.Error() != expected {
			t.Errorf("Error() = %v, want %v", err.Error(), expected)
		}
	})
}

func TestError_Is(t *testing.T) {
	err1 := errs.B().Code(errs.NotFound).Msg("not found").Err()
	err2 := errs.B().Code(errs.NotFound).Msg("different message").Err()
	err3 := errs.B().Code(errs.Internal).Msg("internal error").Err()

	if !errors.Is(err1, err2) {
		t.Error("expected errors with same code to match")
	}

	if errors.Is(err1, err3) {
		t.Error("expected errors with different codes to not match")
	}
}

func TestNew(t *testing.T) {
	err := errs.New("simple error")

	var glibErr *errs.Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *errs.Error")
	}

	if glibErr.Code() != errs.Internal {
		t.Errorf("Code() = %v, want %v", glibErr.Code(), errs.Internal)
	}

	if glibErr.Message() != "simple error" {
		t.Errorf("Message() = %v, want %v", glibErr.Message(), "simple error")
	}
}

func TestNewf(t *testing.T) {
	err := errs.Newf("error: %s", "something went wrong")

	var glibErr *errs.Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *errs.Error")
	}

	expected := "error: something went wrong"
	if glibErr.Message() != expected {
		t.Errorf("Message() = %v, want %v", glibErr.Message(), expected)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("original error")
	err := errs.Wrap(cause, "wrapped")

	var glibErr *errs.Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *errs.Error")
	}

	if !errors.Is(err, cause) {
		t.Error("expected error chain to contain cause")
	}

	if glibErr.Message() != "wrapped" {
		t.Errorf("Message() = %v, want %v", glibErr.Message(), "wrapped")
	}
}

func TestWrapf(t *testing.T) {
	cause := errors.New("original error")
	err := errs.Wrapf(cause, "wrapped: %d", 123)

	var glibErr *errs.Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *errs.Error")
	}

	if !errors.Is(err, cause) {
		t.Error("expected error chain to contain cause")
	}

	expected := "wrapped: 123"
	if glibErr.Message() != expected {
		t.Errorf("Message() = %v, want %v", glibErr.Message(), expected)
	}
}

func TestBuilder_DefaultCode(t *testing.T) {
	err := errs.B().Msg("no code specified").Err()

	var glibErr *errs.Error
	if !errors.As(err, &glibErr) {
		t.Fatal("expected *errs.Error")
	}

	if glibErr.Code() != errs.Internal {
		t.Errorf("Code() = %v, want %v (default)", glibErr.Code(), errs.Internal)
	}
}

func ExampleB() {
	err := errs.B().
		Code(errs.NotFound).
		Msg("user not found").
		Meta("user_id", "123").
		Err()

	fmt.Println(err.Error())
	// Output: user not found
}

func ExampleBuilder_Details() {
	err := errs.B().
		Code(errs.InvalidArgument).
		Msg("validation failed").
		Details(map[string][]string{
			"email": {"must be valid email", "required"},
			"age":   {"must be at least 18"},
		}).
		Err()

	var glibErr *errs.Error
	if errors.As(err, &glibErr) {
		fmt.Println(glibErr.Message())
		fmt.Printf("Validation errors: %d fields\n", len(glibErr.Details()))
	}
	// Output:
	// validation failed
	// Validation errors: 2 fields
}

func ExampleWrap() {
	originalErr := errors.New("database connection failed")
	err := errs.Wrap(originalErr, "failed to fetch user")

	fmt.Println(err.Error())
	// Output: failed to fetch user: database connection failed
}
