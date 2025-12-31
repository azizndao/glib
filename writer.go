package glib

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/azizndao/glib/pkg/errs"
)

const (
	contentTypeJSON = "application/json"
	internalErrCode = "internal"
	internalErrMsg  = "An internal error occurred"
)

// ErrorDetail represents a single error detail entry
type ErrorDetail struct {
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

// ErrorInfo contains the error information returned to clients
type ErrorInfo struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// ErrorResponse is the top-level error response structure
type ErrorResponse struct {
	Error ErrorInfo `json:"error"`
}

// Write writes the Result[T] to the HTTP response
// This is the main method used by generated handlers
func (r Result[T]) Write(w http.ResponseWriter) {
	// If there's an error, write error response
	if r.err != nil {
		writeError(w, r.err)
		return
	}

	// Write custom headers
	for key, values := range r.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// For no-content responses, don't write body
	if r.StatusCode == http.StatusNoContent {
		w.WriteHeader(r.StatusCode)
		return
	}

	// Write success response with data
	writeJSON(w, r.StatusCode, r.Data)
}

// writeJSON writes a JSON response with proper error handling
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("Failed to encode JSON response: %v", err)
		}
	}
}

// writeError writes an error response
func writeError(w http.ResponseWriter, err error) {
	var glibErr *errs.Error
	if errors.As(err, &glibErr) {
		// Structured glib error - return user-facing details
		errorInfo := ErrorInfo{
			Code:    glibErr.Code.String(),
			Message: glibErr.Message,
		}

		// Include details (validation errors) if present
		if glibErr.Details != nil {
			errorInfo.Details = convertDetails(glibErr.Details)
		}

		response := ErrorResponse{Error: errorInfo}
		writeJSON(w, glibErr.Code.HTTPStatus(), response)
		return
	}

	// Generic error - don't expose internal details
	response := ErrorResponse{
		Error: ErrorInfo{
			Code:    internalErrCode,
			Message: internalErrMsg,
		},
	}
	writeJSON(w, http.StatusInternalServerError, response)
}

// WriteParamError writes a parameter validation error
// This is used by generated parsers for parameter parsing errors
func WriteParamError(w http.ResponseWriter, paramName, reason string) {
	response := ErrorResponse{
		Error: ErrorInfo{
			Code:    "invalid_argument",
			Message: "Invalid request parameters",
			Details: []ErrorDetail{
				{
					Field:    paramName,
					Messages: []string{reason},
				},
			},
		},
	}
	writeJSON(w, http.StatusBadRequest, response)
}

// convertDetails converts errs.ErrDetails to our ErrorDetail format
func convertDetails(details errs.ErrDetails) []ErrorDetail {
	// If details is nil, return empty slice
	if details == nil {
		return nil
	}

	// Try to convert ValidationErrors
	if validationErrs, ok := details.(*errs.ValidationErrors); ok {
		result := make([]ErrorDetail, len(validationErrs.Errors))
		for i, err := range validationErrs.Errors {
			result[i] = ErrorDetail{
				Field:    err.Field,
				Messages: err.Messages,
			}
		}
		return result
	}

	// For other ErrDetails types, return nil
	// Users can extend this function to handle custom error detail types
	return nil
}
