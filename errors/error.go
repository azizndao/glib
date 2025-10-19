// Package errors provides a standardized way to represent errors in HTTP handlers.
package errors

// Error represents an error returned by a handler
type Error struct {
	Code     int   `json:"code"`
	Data     any   `json:"data,omitempty"`
	internal error `json:"-"`
}

func New(code int, data any, internal error) *Error {
	return &Error{
		Code:     code,
		Data:     data,
		internal: internal,
	}
}
