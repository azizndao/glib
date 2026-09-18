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
				{Type: AnnotationController, Value: "path=/api/v1/posts tags=api,public"},
			},
		},
		{
			name:    "route annotation",
			comment: "// @Route method=GET path=/{id}",
			expected: []Annotation{
				{Type: AnnotationRoute, Value: "method=GET path=/{id}"},
			},
		},
		{
			name:    "provider singleton",
			comment: "// @Provider singleton",
			expected: []Annotation{
				{Type: AnnotationProvider, Value: LifecycleSingleton.String()},
			},
		},
		{
			name:    "provider transient",
			comment: "// @Provider transient",
			expected: []Annotation{
				{Type: AnnotationProvider, Value: LifecycleTransient.String()},
			},
		},
		{
			name:    "middleware annotation",
			comment: "// @Middleware name=auth target=protected order=10",
			expected: []Annotation{
				{Type: AnnotationMiddleware, Value: "name=auth target=protected order=10"},
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
					t.Errorf("expected type %s, got %s", expType(tt.expected[i].Type), ann.Type)
				}
				if ann.Value != tt.expected[i].Value {
					t.Errorf("expected value %s, got %s", tt.expected[i].Value, ann.Value)
				}
			}
		})
	}
}

func expType(t AnnotationType) AnnotationType {
	return t
}

func TestParseControllerAnnotation(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected map[string]string
	}{
		{
			name:  "path only",
			value: "path=/api/v1/posts",
			expected: map[string]string{
				"path": "/api/v1/posts",
				"tags": "",
			},
		},
		{
			name:  "path and tags",
			value: "path=/api/v1/posts tags=api,public",
			expected: map[string]string{
				"path": "/api/v1/posts",
				"tags": "api,public",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseControllerAnnotation(tt.value)
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("expected %s=%s, got %s=%s", k, v, k, result[k])
				}
			}
		})
	}
}

func TestParseRouteAnnotation(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected map[string]string
	}{
		{
			name:  "method and path",
			value: "method=GET path=/",
			expected: map[string]string{
				"method": "GET",
				"path":   "/",
				"tags":   "",
				"with":   "",
			},
		},
		{
			name:  "with tags and with",
			value: "method=POST path=/ tags=protected with=auth,admin",
			expected: map[string]string{
				"method": "POST",
				"path":   "/",
				"tags":   "protected",
				"with":   "auth,admin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRouteAnnotation(tt.value)
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("expected %s=%s, got %s=%s", k, v, k, result[k])
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
		{LifecycleSingleton.String(), LifecycleSingleton.String()},
		{LifecycleTransient.String(), LifecycleTransient.String()},
		{"", LifecycleSingleton.String()}, // Default
		{" singleton ", LifecycleSingleton.String()},
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
