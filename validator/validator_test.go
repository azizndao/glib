package validator_test

import (
	"errors"
	"testing"

	"github.com/azizndao/glib/errs"
	"github.com/azizndao/glib/validator"
)

type TestRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3,max=50"`
	Age      int    `json:"age" validate:"required,gte=18,lte=120"`
}

func TestValidator(t *testing.T) {
	validator := validator.NewValidator()

	tests := []struct {
		name      string
		input     TestRequest
		wantError bool
	}{
		{
			name: "valid request",
			input: TestRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Age:      25,
			},
			wantError: false,
		},
		{
			name: "invalid email",
			input: TestRequest{
				Email:    "not-an-email",
				Username: "testuser",
				Age:      25,
			},
			wantError: true,
		},
		{
			name: "username too short",
			input: TestRequest{
				Email:    "test@example.com",
				Username: "ab",
				Age:      25,
			},
			wantError: true,
		},
		{
			name: "age too low",
			input: TestRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Age:      15,
			},
			wantError: true,
		},
		{
			name: "missing required fields",
			input: TestRequest{
				Email: "",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.input)
			if tt.wantError && err == nil {
				t.Error("expected validation error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			// If error is expected, check that it's a validation error
			if tt.wantError && err != nil {
				var glibErr *errs.Error
				if !errors.As(err, &glibErr) {
					t.Errorf("expected *errs.Error, got %T", err)
					return
				}
				if glibErr.Code != errs.InvalidArgument {
					t.Errorf("expected InvalidArgument code, got %s", glibErr.Code)
				}
			}
		})
	}
}

func TestValidatorWithPointers(t *testing.T) {
	type OptionalRequest struct {
		Email *string `json:"email" validate:"omitempty,email"`
		Age   *int    `json:"age" validate:"omitempty,gte=18"`
	}

	validator := validator.NewValidator()

	// Valid: nil pointers are allowed with omitempty
	req1 := OptionalRequest{}
	if err := validator.Validate(req1); err != nil {
		t.Errorf("expected no error for nil pointers, got: %v", err)
	}

	// Valid: valid email
	email := "test@example.com"
	age := 25
	req2 := OptionalRequest{Email: &email, Age: &age}
	if err := validator.Validate(req2); err != nil {
		t.Errorf("expected no error for valid values, got: %v", err)
	}

	// Invalid: bad email
	badEmail := "not-an-email"
	req3 := OptionalRequest{Email: &badEmail}
	if err := validator.Validate(req3); err == nil {
		t.Error("expected validation error for invalid email")
	}
}
