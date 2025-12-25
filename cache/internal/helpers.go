package internal

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
)

// CopyValue copies a value from src to dest using type switching for simple types
// and JSON marshaling for complex types (slices, structs, maps).
func CopyValue(src, dest any) error {
	// Simple implementation using reflection
	if dest == nil {
		return nil
	}

	// For simple types, use type switch
	switch d := dest.(type) {
	case *any:
		*d = src
		return nil
	case *string:
		if s, ok := src.(string); ok {
			*d = s
			return nil
		}
	case *int:
		if i, ok := src.(int); ok {
			*d = i
			return nil
		}
	case *int64:
		if i, ok := src.(int64); ok {
			*d = i
			return nil
		}
	case *float64:
		if f, ok := src.(float64); ok {
			*d = f
			return nil
		}
	case *bool:
		if b, ok := src.(bool); ok {
			*d = b
			return nil
		}
	}

	// For complex types (slices, structs, maps), use JSON marshaling
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// GenerateOwnerID generates a unique owner identifier for distributed locks.
func GenerateOwnerID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
