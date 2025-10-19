// Package errors provides a standardized way to represent errors in HTTP handlers.
package errors

import "fmt"

// Error represents an error returned by a handler
type Error struct {
	Code     int   `json:"code"`
	Data     any   `json:"data,omitempty"`
	internal error `json:"-"`
}

// New creates a new Error with the given code, data, and internal error
func New(code int, data any, internal error) *Error {
	return &Error{
		Code:     code,
		Data:     data,
		internal: internal,
	}
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.internal != nil {
		return e.internal.Error()
	}

	return fmt.Sprintf("%d: %s", e.Code, e.Data)
}
