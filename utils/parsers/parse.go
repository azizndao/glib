package parsers

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// ParseInt converts a string to int with error context.
func ParseInt(value, fieldName string) (int, error) {
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid integer: %w", fieldName, err)
	}
	return result, nil
}

// ParseInt64 converts a string to int64 with error context.
func ParseInt64(value, fieldName string) (int64, error) {
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid int64: %w", fieldName, err)
	}
	return result, nil
}

// ParseUint64 converts a string to uint64 with error context.
func ParseUint64(value, fieldName string) (uint64, error) {
	result, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid uint64: %w", fieldName, err)
	}
	return result, nil
}

// ParseFloat64 converts a string to float64 with error context.
func ParseFloat64(value, fieldName string) (float64, error) {
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid float64: %w", fieldName, err)
	}
	return result, nil
}

// ParseBool converts a string to bool with error context.
func ParseBool(value, fieldName string) (bool, error) {
	result, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s: invalid boolean: %w", fieldName, err)
	}
	return result, nil
}

// ParseUUID converts a string to UUID with error context.
func ParseUUID(value, fieldName string) (uuid.UUID, error) {
	result, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: invalid UUID: %w", fieldName, err)
	}
	return result, nil
}

// ParseDuration converts a string to time.Duration with error context.
// Accepts duration strings like "300ms", "1.5h", "2h45m", etc.
func ParseDuration(value, fieldName string) (time.Duration, error) {
	result, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid duration: %w", fieldName, err)
	}
	return result, nil
}

// ParseURL converts a string to *url.URL with error context.
func ParseURL(value, fieldName string) (*url.URL, error) {
	result, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid URL: %w", fieldName, err)
	}
	return result, nil
}

// Soft-parse functions for query/header parameters
// These return zero values on error instead of returning errors,
// allowing the validator to collect all errors at once.

// ParseIntOrZero converts a string to int, returns 0 on error.
// Used for query/header parameters that will be validated by the validator.
func ParseIntOrZero(value string) int {
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return result
}

// ParseInt64OrZero converts a string to int64, returns 0 on error.
func ParseInt64OrZero(value string) int64 {
	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return result
}

// ParseUint64OrZero converts a string to uint64, returns 0 on error.
func ParseUint64OrZero(value string) uint64 {
	result, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return result
}

// ParseFloat64OrZero converts a string to float64, returns 0.0 on error.
func ParseFloat64OrZero(value string) float64 {
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0.0
	}
	return result
}

// ParseBoolOrFalse converts a string to bool, returns false on error.
func ParseBoolOrFalse(value string) bool {
	result, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return result
}

// ParseUUIDOrNil converts a string to UUID, returns uuid.Nil on error.
func ParseUUIDOrNil(value string) uuid.UUID {
	result, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil
	}
	return result
}

// ParseDurationOrZero converts a string to time.Duration, returns 0 on error.
func ParseDurationOrZero(value string) time.Duration {
	result, err := time.ParseDuration(value)
	if err != nil {
		return 0
	}
	return result
}

// ParseURLOrNil converts a string to *url.URL, returns nil on error.
func ParseURLOrNil(value string) *url.URL {
	result, err := url.Parse(value)
	if err != nil {
		return nil
	}
	return result
}
