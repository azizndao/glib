package grouter

import (
	"fmt"
	"net/http"
)

// Error represents an error returned by a handler
type Error struct {
	Code     int   `json:"code"`
	Data     any   `json:"data,omitempty"`
	internal error `json:"-"`
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.internal != nil {
		return e.internal.Error()
	}

	return fmt.Sprintf("%d: %s", e.Code, e.Data)
}

// Usefull api error

func ErrorNotFound(data any, internal error) *Error {
	return &Error{
		Code:     http.StatusNotFound,
		Data:     data,
		internal: internal,
	}
}

func ErrorBadRequest(data any, internal error) *Error {
	return &Error{
		Code:     http.StatusBadRequest,
		Data:     data,
		internal: internal,
	}
}

func ErrorUnauthorized(data any, internal error) *Error {
	return &Error{
		Code:     http.StatusUnauthorized,
		Data:     data,
		internal: internal,
	}
}

func ErrorForbidden(data any, internal error) *Error {
	return &Error{
		Code:     http.StatusForbidden,
		Data:     data,
		internal: internal,
	}
}

func ErrorInternalServerError(data any, internal error) *Error {
	return &Error{
		Code:     http.StatusInternalServerError,
		Data:     data,
		internal: internal,
	}
}

func ErrorServiceUnavailable(data any, internal error) *Error {
	return &Error{
		Code:     http.StatusServiceUnavailable,
		Data:     data,
		internal: internal,
	}
}

func ErrorGatewayTimeout(data any, internal error) *Error {
	return &Error{
		Code:     http.StatusGatewayTimeout,
		Data:     data,
		internal: internal,
	}
}

func ErrorMethodNotAllowed(data any, internal error) *Error {
	return &Error{
		Code:     http.StatusMethodNotAllowed,
		Data:     data,
		internal: internal,
	}
}
