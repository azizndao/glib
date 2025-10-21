package errors

import "net/http"

// UnprocessableEntity creates a 422 Unprocessable Entity error
func UnprocessableEntity(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusUnprocessableEntity,
		Data:     data,
		internal: internal,
	}
}

// Conflict creates a 409 Conflict error
func Conflict(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusConflict,
		Data:     data,
		internal: internal,
	}
}

// Gone creates a 410 Gone error
func Gone(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusGone,
		Data:     data,
		internal: internal,
	}
}

// NotFound creates a 404 Not Found error
func NotFound(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusNotFound,
		Data:     data,
		internal: internal,
	}
}

// BadRequest creates a 400 Bad Request error
func BadRequest(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusBadRequest,
		Data:     data,
		internal: internal,
	}
}

// Unauthorized creates a 401 Unauthorized error
func Unauthorized(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusUnauthorized,
		Data:     data,
		internal: internal,
	}
}

// Forbidden creates a 403 Forbidden error
func Forbidden(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusForbidden,
		Data:     data,
		internal: internal,
	}
}

// InternalServerError creates a 500 Internal Server Error
func InternalServerError(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusInternalServerError,
		Data:     data,
		internal: internal,
	}
}

// ServiceUnavailable creates a 503 Service Unavailable error
func ServiceUnavailable(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusServiceUnavailable,
		Data:     data,
		internal: internal,
	}
}

// GatewayTimeout creates a 504 Gateway Timeout error
func GatewayTimeout(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusGatewayTimeout,
		Data:     data,
		internal: internal,
	}
}

// MethodNotAllowed creates a 405 Method Not Allowed error
func MethodNotAllowed(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusMethodNotAllowed,
		Data:     data,
		internal: internal,
	}
}

// NotImplemented creates a 501 Not Implemented error
func NotImplemented(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusNotImplemented,
		Data:     data,
		internal: internal,
	}
}

// BadGateway creates a 502 Bad Gateway error
func BadGateway(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusBadGateway,
		Data:     data,
		internal: internal,
	}
}

// TooManyRequests creates a 429 Too Many Requests error
func TooManyRequests(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusTooManyRequests,
		Data:     data,
		internal: internal,
	}
}

// RequestEntityTooLarge creates a 413 Request Entity Too Large error
func RequestEntityTooLarge(data any, internal error) *ApiError {
	return &ApiError{
		Code:     http.StatusRequestEntityTooLarge,
		Data:     data,
		internal: internal,
	}
}
