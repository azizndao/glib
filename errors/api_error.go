// Package errors provides a standardized way to represent errors in HTTP handlers.
package errors

import "fmt"

// ApiError represents an error returned by a handler
type ApiError struct {
	Code     int   `json:"code"`
	Data     any   `json:"data,omitempty"`
	internal error `json:"-"`
}

// NewApi creates a new Error with the given code, data, and internal error
func NewApi(code int, data any, internal error) *ApiError {
	return &ApiError{
		Code:     code,
		Data:     data,
		internal: internal,
	}
}

// Error implements the error interface
func (e *ApiError) Error() string {
	if e.internal != nil {
		return e.internal.Error()
	}

	return fmt.Sprintf("%d: %s", e.Code, e.Data)
}
