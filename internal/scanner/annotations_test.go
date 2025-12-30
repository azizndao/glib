package scanner

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestExtractAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		comment  string
		expected []Annotation
	}{
		{
			name:    "controller annotation",
			comment: "// @Controller path=/api/v1/posts tags=api,public",
			expected: []Annotation{
				{Type: "Controller", Value: "path=/api/v1/posts tags=api,public"},
			},
		},
		{
			name:    "route annotation",
			comment: "// @Route method=GET path=/{id}",
			expected: []Annotation{
				{Type: "Route", Value: "method=GET path=/{id}"},
			},
		},
		{
			name:    "provider singleton",
			comment: "// @Provider singleton",
			expected: []Annotation{
				{Type: "Provider", Value: "singleton"},
			},
		},
		{
			name:    "provider transient",
			comment: "// @Provider transient",
			expected: []Annotation{
				{Type: "Provider", Value: "transient"},
			},
		},
		{
			name:    "middleware annotation",
			comment: "// @Middleware name=auth target=protected order=10",
			expected: []Annotation{
				{Type: "Middleware", Value: "name=auth target=protected order=10"},
			},
		},
		{
			name:     "no annotation",
			comment:  "// Regular comment",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			code := "package test\n\n" + tt.comment + "\ntype Foo struct{}"
			f, err := parser.ParseFile(fset, "", code, parser.ParseComments)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			var cg *ast.CommentGroup
			if len(f.Comments) > 0 {
				cg = f.Comments[0]
			}

			anns := extractAnnotations(cg)

			if len(anns) != len(tt.expected) {
				t.Fatalf("expected %d annotations, got %d", len(tt.expected), len(anns))
			}

			for i, ann := range anns {
				if ann.Type != tt.expected[i].Type {
					t.Errorf("expected type %s, got %s", tt.expected[i].Type, ann.Type)
				}
				if ann.Value != tt.expected[i].Value {
					t.Errorf("expected value %s, got %s", tt.expected[i].Value, ann.Value)
				}
			}
		})
	}
}

func TestParseRouteAnnotation(t *testing.T) {
	tests := []struct {
		value          string
		expectedMethod string
		expectedPath   string
	}{
		{"method=GET path=/users", "GET", "/users"},
		{"method=POST path=/users", "POST", "/users"},
		{"method=PUT path=/users/{id}", "PUT", "/users/{id}"},
		{"method=DELETE path=/users/{id}", "DELETE", "/users/{id}"},
		{"method=PATCH path=/users/{id}", "PATCH", "/users/{id}"},
		{"method=get path=/users", "get", "/users"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := parseRouteAnnotation(tt.value)
			method := result["method"]
			path := result["path"]
			if method != tt.expectedMethod {
				t.Errorf("expected method %s, got %s", tt.expectedMethod, method)
			}
			if path != tt.expectedPath {
				t.Errorf("expected path %s, got %s", tt.expectedPath, path)
			}
		})
	}
}

func TestParseMiddlewareAnnotation(t *testing.T) {
	tests := []struct {
		value    string
		expected []string
	}{
		{"auth", []string{"auth"}},
		{"auth,ratelimit", []string{"auth", "ratelimit"}},
		{"auth, ratelimit, cors", []string{"auth", "ratelimit", "cors"}},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := parseMiddlewareAnnotation(tt.value)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d items, got %d", len(tt.expected), len(result))
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("expected %s, got %s", tt.expected[i], v)
				}
			}
		})
	}
}

func TestParseProviderAnnotation(t *testing.T) {
	tests := []struct {
		value    string
		expected string
	}{
		{"singleton", "singleton"},
		{"transient", "transient"},
		{"", "transient"}, // Default
		{" singleton ", "singleton"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := parseProviderAnnotation(tt.value)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
