// Package errors provides a standardized way to represent errors in HTTP handlers.
package errors

import "fmt"

// APIError represents an error returned by a handler
type APIError struct {
	Code     int   `json:"code"`
	Data     any   `json:"data,omitempty"`
	internal error `json:"-"`
}

// NewAPI creates a new Error with the given code, data, and internal error
func NewAPI(code int, data any, internal error) *APIError {
	return &APIError{
		Code:     code,
		Data:     data,
		internal: internal,
	}
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.internal != nil {
		return e.internal.Error()
	}

	return fmt.Sprintf("%d: %s", e.Code, e.Data)
}
