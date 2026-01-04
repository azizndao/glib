package glib

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/azizndao/glib/pkg/errs"
	"github.com/azizndao/glib/validator"
)

const (
	contentTypeJSON = "application/json"
	internalErrCode = "internal"
	internalErrMsg  = "An internal error occurred"
)

// ErrorInfo contains the error information returned to clients
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
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

		// Include details if present (validation errors, etc)
		if glibErr.Details != nil {
			errorInfo.Details = glibErr.Details
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

// WriteError writes an error response (exported for use in generated code)
func WriteError(w http.ResponseWriter, err error) {
	writeError(w, err)
}

// WriteResponseWithMetadata writes a response with metadata extraction
// Extracts header and status code fields from response struct using reflection
func WriteResponseWithMetadata(w http.ResponseWriter, httpMethod string, data any) {
	// Default status code based on HTTP method
	statusCode := getDefaultStatusCode(httpMethod)

	// Extract headers and status from struct using reflection
	if data != nil {
		val := reflect.ValueOf(data)

		// Handle pointers
		if val.Kind() == reflect.Ptr {
			if val.IsNil() {
				// Nil pointer - just write default response
				writeJSON(w, statusCode, data)
				return
			}
			val = val.Elem()
		}

		// Only process structs
		if val.Kind() == reflect.Struct {
			typ := val.Type()

			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				fieldVal := val.Field(i)

				// Check for header tag: header:"Location" or header:"ETag,omitempty"
				if headerTag := field.Tag.Get("header"); headerTag != "" {
					// Parse tag (could have omitempty)
					headerName := headerTag
					omitEmpty := false

					if strings.HasSuffix(headerTag, ",omitempty") {
						omitEmpty = true
						headerName = strings.TrimSuffix(headerTag, ",omitempty")
					}

					// Get field value as string
					headerValue := getFieldValueAsString(fieldVal)

					// Set header if not empty or if not omitempty
					if headerValue != "" || !omitEmpty {
						w.Header().Set(headerName, headerValue)
					}
				}

				// Check for response:"httpstatus" tag
				if responseTag := field.Tag.Get("response"); responseTag == "httpstatus" {
					// Field must be int type
					if fieldVal.Kind() == reflect.Int {
						statusCode = int(fieldVal.Int())
					}
				}
			}
		}
	}

	writeJSON(w, statusCode, data)
}

// getFieldValueAsString converts a reflect.Value to string representation
func getFieldValueAsString(val reflect.Value) string {
	switch val.Kind() {
	case reflect.String:
		return val.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return string(rune(val.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return string(rune(val.Uint()))
	case reflect.Bool:
		if val.Bool() {
			return "true"
		}
		return "false"
	case reflect.Ptr:
		if val.IsNil() {
			return ""
		}
		return getFieldValueAsString(val.Elem())
	default:
		// For complex types, use fmt.Sprint equivalent
		return ""
	}
}

// getDefaultStatusCode returns the default HTTP status code for a given method
func getDefaultStatusCode(method string) int {
	switch method {
	case "POST":
		return http.StatusCreated // 201
	case "DELETE":
		return http.StatusNoContent // 204
	case "PUT", "PATCH":
		return http.StatusOK // 200
	default: // GET, HEAD, OPTIONS, etc.
		return http.StatusOK // 200
	}
}

// WriteParamError writes a parameter validation error
// This is used by generated parsers for parameter parsing errors
func WriteParamError(w http.ResponseWriter, paramName, reason string) {
	// Create Goyave-style error for query params (most common case for param errors)
	goyaveErr := validator.NewValidationErrors()
	goyaveErr.AddQueryError([]string{paramName}, reason)

	writeJSON(w, http.StatusBadRequest, map[string]any{"error": goyaveErr})
}
