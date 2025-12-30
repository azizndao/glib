package scanner

import (
	"testing"
)

func TestIsPathParamName(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"id", true},
		{"uuid", true},
		{"key", true},
		{"slug", true},
		{"userId", true},
		{"postId", true},
		{"commentId", true},
		{"sessionKey", true},
		{"req", false},
		{"request", false},
		{"data", false},
		{"body", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPathParamName(tt.name)
			if result != tt.expected {
				t.Errorf("isPathParamName(%s) = %v, expected %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestIsPrimitive(t *testing.T) {
	primitives := []string{"bool", "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune", "float32", "float64", "complex64", "complex128"}

	for _, p := range primitives {
		t.Run(p, func(t *testing.T) {
			if !isPrimitive(p) {
				t.Errorf("%s should be primitive", p)
			}
		})
	}

	nonPrimitives := []string{"User", "Request", "UUID", "Time", "error"}
	for _, np := range nonPrimitives {
		t.Run(np, func(t *testing.T) {
			if isPrimitive(np) {
				t.Errorf("%s should NOT be primitive", np)
			}
		})
	}
}
