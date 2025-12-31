package errs

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestErrorResponseSerialization tests that errors with ValidationErrors serialize correctly
func TestErrorResponseSerialization(t *testing.T) {
	// Create an error with validation details
	validationErrs := NewValidationErrors([]ValidationError{
		{Field: "email", Messages: []string{"must be a valid email"}},
		{Field: "password", Messages: []string{"must be at least 8 characters", "must contain a number"}},
	})

	err := B().
		Code(InvalidArgument).
		Msg("Validation failed").
		Details(validationErrs).
		Err()

	// Verify it's an *Error
	var glibErr *Error
	if !errors.As(err, &glibErr) {
		t.Fatal("Expected *Error type")
	}

	// Verify the error has details
	if glibErr.Details == nil {
		t.Fatal("Expected details to be set")
	}

	// Verify details can be converted back
	details, ok := glibErr.Details.(*ValidationErrors)
	if !ok {
		t.Fatal("Expected *ValidationErrors type")
	}

	if len(details.Errors) != 2 {
		t.Errorf("Expected 2 validation errors, got %d", len(details.Errors))
	}

	// Test JSON serialization
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(glibErr.Code.HTTPStatus())

	response := map[string]any{
		"error": map[string]any{
			"code":    glibErr.Code.String(),
			"message": glibErr.Message,
			"details": details.Errors,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Fatalf("Failed to encode response: %v", err)
	}

	// Verify status code
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Parse response
	var parsedResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &parsedResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify structure
	errorObj, ok := parsedResp["error"].(map[string]any)
	if !ok {
		t.Fatal("Expected 'error' object in response")
	}

	if code, _ := errorObj["code"].(string); code != "invalid_argument" {
		t.Errorf("Expected code 'invalid_argument', got '%s'", code)
	}

	if msg, _ := errorObj["message"].(string); msg != "Validation failed" {
		t.Errorf("Expected message 'Validation failed', got '%s'", msg)
	}

	detailsArray, ok := errorObj["details"].([]any)
	if !ok {
		t.Fatal("Expected 'details' array in response")
	}

	if len(detailsArray) != 2 {
		t.Errorf("Expected 2 details, got %d", len(detailsArray))
	}
}

// TestErrorWithoutDetails tests that errors without details don't include the details field
func TestErrorWithoutDetails(t *testing.T) {
	err := B().
		Code(NotFound).
		Msg("Resource not found").
		Err()

	var glibErr *Error
	if !errors.As(err, &glibErr) {
		t.Fatal("Expected *Error type")
	}

	// Verify no details
	if glibErr.Details != nil {
		t.Error("Expected no details")
	}

	// Test JSON serialization
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(glibErr.Code.HTTPStatus())

	response := map[string]any{
		"error": map[string]any{
			"code":    glibErr.Code.String(),
			"message": glibErr.Message,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Fatalf("Failed to encode response: %v", err)
	}

	// Verify status code
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	// Parse response
	var parsedResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &parsedResp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify structure
	errorObj, ok := parsedResp["error"].(map[string]any)
	if !ok {
		t.Fatal("Expected 'error' object in response")
	}

	if _, hasDetails := errorObj["details"]; hasDetails {
		t.Error("Expected no 'details' field in response")
	}
}

// TestMultipleValidationErrors tests adding errors incrementally
func TestMultipleValidationErrors(t *testing.T) {
	validationErrs := &ValidationErrors{}

	// Add errors one by one
	validationErrs.AddError("username", "is required")
	validationErrs.AddError("email", "must be a valid email", "must be unique")
	validationErrs.AddError("age", "must be positive")

	if len(validationErrs.Errors) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(validationErrs.Errors))
	}

	// Check specific fields
	if validationErrs.Errors[0].Field != "username" {
		t.Errorf("Expected field 'username', got '%s'", validationErrs.Errors[0].Field)
	}

	if len(validationErrs.Errors[1].Messages) != 2 {
		t.Errorf("Expected 2 messages for email, got %d", len(validationErrs.Errors[1].Messages))
	}

	// Create error with these validation errors
	err := B().
		Code(InvalidArgument).
		Msg("Multiple validation errors").
		Details(validationErrs).
		Err()

	var glibErr *Error
	if !errors.As(err, &glibErr) {
		t.Fatal("Expected *Error type")
	}

	details, ok := glibErr.Details.(*ValidationErrors)
	if !ok {
		t.Fatal("Expected *ValidationErrors type")
	}

	if len(details.Errors) != 3 {
		t.Errorf("Expected 3 validation errors in details, got %d", len(details.Errors))
	}
}
