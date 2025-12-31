package generator

import (
	"testing"
)

func TestCapitalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "Hello"},
		{"world", "World"},
		{"", ""},
		{"a", "A"},
		{"ABC", "ABC"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := capitalize(tt.input)
			if result != tt.expected {
				t.Errorf("capitalize(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}
