// Package typeutil provides utilities for type conversion and manipulation.
//
// This package is useful when you need to convert between types that are not
// directly compatible but can be marshaled to JSON. It's particularly helpful
// when working with context values or dynamic data structures.
//
// Usage example:
//
//	// Convert a map to a struct
//	data := map[string]any{"name": "John", "age": 30}
//	type User struct {
//	    Name string `json:"name"`
//	    Age  int    `json:"age"`
//	}
//	user, err := typeutil.Convert[User](data)
//
// Note: This package uses JSON marshaling/unmarshaling internally, so it has
// some overhead. For performance-critical code, prefer direct type assertions
// or manual conversion when possible.
package typeutil

import (
	"bytes"
	"encoding/json"
)

// Convert converts any value into the desired type using JSON marshaling and unmarshaling.
// It first checks if the value is already of the target type to avoid unnecessary conversion.
// If not, it uses JSON as an intermediate format to perform the conversion.
//
// This is useful for converting between compatible types (e.g., map[string]any to structs)
// but comes with JSON marshaling overhead.
//
// Returns an error if the conversion fails (e.g., incompatible types or invalid JSON).
func Convert[T any](data any) (T, error) {
	if v, ok := data.(T); ok {
		return v, nil
	}

	var result T
	buffer := &bytes.Buffer{}
	decoder := json.NewDecoder(buffer)
	writer := json.NewEncoder(buffer)

	if err := writer.Encode(data); err != nil {
		return result, err
	}
	err := decoder.Decode(&result)
	return result, err
}

// MustConvert is like Convert but panics if the conversion fails.
// Use this only when you are certain the conversion will succeed.
//
// Example:
//
//	user := typeutil.MustConvert[User](contextValue)
func MustConvert[T any](data any) T {
	res, err := Convert[T](data)
	if err != nil {
		panic(err)
	}
	return res
}
