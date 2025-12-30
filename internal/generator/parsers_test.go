package generator

import (
	"testing"

	"github.com/azizndao/glib/internal/scanner"
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

func TestNeedsStrconvPkg(t *testing.T) {
	tests := []struct {
		name       string
		pathParams []*scanner.PathParam
		expected   bool
	}{
		{
			name: "int param",
			pathParams: []*scanner.PathParam{
				{
					Type: &scanner.TypeInfo{
						Name:        "int",
						IsPrimitive: true,
					},
				},
			},
			expected: true,
		},
		{
			name: "string param",
			pathParams: []*scanner.PathParam{
				{
					Type: &scanner.TypeInfo{
						Name:        "string",
						IsPrimitive: true,
					},
				},
			},
			expected: false,
		},
		{
			name: "uuid param",
			pathParams: []*scanner.PathParam{
				{
					Type: &scanner.TypeInfo{
						Name:        "UUID",
						PackageName: "uuid",
					},
				},
			},
			expected: false,
		},
		{
			name: "mixed params",
			pathParams: []*scanner.PathParam{
				{
					Type: &scanner.TypeInfo{
						Name:        "string",
						IsPrimitive: true,
					},
				},
				{
					Type: &scanner.TypeInfo{
						Name:        "int64",
						IsPrimitive: true,
					},
				},
			},
			expected: true,
		},
		{
			name:       "no params",
			pathParams: nil,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsStrconvPkg(tt.pathParams)
			if result != tt.expected {
				t.Errorf("needsStrconvPkg() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestNeedsUUID(t *testing.T) {
	tests := []struct {
		name       string
		pathParams []*scanner.PathParam
		expected   bool
	}{
		{
			name: "uuid param",
			pathParams: []*scanner.PathParam{
				{
					Type: &scanner.TypeInfo{
						Name:        "UUID",
						PackageName: "uuid",
					},
				},
			},
			expected: true,
		},
		{
			name: "string param",
			pathParams: []*scanner.PathParam{
				{
					Type: &scanner.TypeInfo{
						Name:        "string",
						IsPrimitive: true,
					},
				},
			},
			expected: false,
		},
		{
			name: "mixed params with uuid",
			pathParams: []*scanner.PathParam{
				{
					Type: &scanner.TypeInfo{
						Name:        "int",
						IsPrimitive: true,
					},
				},
				{
					Type: &scanner.TypeInfo{
						Name:        "UUID",
						PackageName: "uuid",
					},
				},
			},
			expected: true,
		},
		{
			name:       "no params",
			pathParams: nil,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsUUID(tt.pathParams)
			if result != tt.expected {
				t.Errorf("needsUUID() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestNeedsErrorCheck(t *testing.T) {
	tests := []struct {
		typeName string
		expected bool
	}{
		{"int", true},
		{"string", false},
		{"int64", true},
		{"float64", true},
		{"bool", true},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			result := needsErrorCheck(tt.typeName)
			if result != tt.expected {
				t.Errorf("needsErrorCheck(%s) = %v, expected %v", tt.typeName, result, tt.expected)
			}
		})
	}
}
