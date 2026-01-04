package validator

import (
	"testing"

	"github.com/azizndao/glib/internal/scanner"
)

func TestValidateController(t *testing.T) {
	tests := []struct {
		name           string
		controller     *scanner.Controller
		expectErrors   int
		expectWarnings int
	}{
		{
			name: "valid controller with handlers",
			controller: &scanner.Controller{
				Name:        "PostController",
				RoutePrefix: "/api/posts",
				FilePath:    "test.go",
				Handlers: []*scanner.Handler{
					{
						Name:   "Index",
						Method: "GET",
						Path:   "/",
						Signature: &scanner.HandlerSignature{
							Pattern: scanner.PatternDataError,
						},
					},
				},
			},
			expectErrors:   0,
			expectWarnings: 0,
		},
		{
			name: "invalid route prefix",
			controller: &scanner.Controller{
				Name:        "PostController",
				RoutePrefix: "api/posts",
				FilePath:    "test.go",
				Handlers:    []*scanner.Handler{},
			},
			expectErrors:   1,
			expectWarnings: 1, // No handlers warning
		},
		{
			name: "controller with no handlers",
			controller: &scanner.Controller{
				Name:        "PostController",
				RoutePrefix: "/api/posts",
				FilePath:    "test.go",
				Handlers:    []*scanner.Handler{},
			},
			expectErrors:   0,
			expectWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.validateController(tt.controller)

			if len(v.errors) != tt.expectErrors {
				t.Errorf("expected %d errors, got %d", tt.expectErrors, len(v.errors))
			}
			if len(v.warnings) != tt.expectWarnings {
				t.Errorf("expected %d warnings, got %d", tt.expectWarnings, len(v.warnings))
			}
		})
	}
}

func TestValidateHandler(t *testing.T) {
	tests := []struct {
		name         string
		handler      *scanner.Handler
		expectErrors int
	}{
		{
			name: "valid handler - Result pattern",
			handler: &scanner.Handler{
				Name:   "Show",
				Method: "GET",
				Path:   "/{id}",
				Signature: &scanner.HandlerSignature{
					Pattern: scanner.PatternDataError,
					PathParams: []*scanner.PathParam{
						{
							Name: "id",
							Type: &scanner.TypeInfo{
								Name:        "int",
								IsPrimitive: true,
								FullName:    "int",
							},
						},
					},
				},
			},
			expectErrors: 0,
		},
		{
			name: "valid handler - Raw HTTP pattern",
			handler: &scanner.Handler{
				Name:   "Export",
				Method: "GET",
				Path:   "/export",
				Signature: &scanner.HandlerSignature{
					Pattern:    scanner.PatternRawHTTP,
					HasRawHTTP: true,
				},
			},
			expectErrors: 0,
		},
		{
			name: "invalid HTTP method",
			handler: &scanner.Handler{
				Name:   "Show",
				Method: "INVALID",
				Path:   "/{id}",
				Signature: &scanner.HandlerSignature{
					Pattern: scanner.PatternDataError,
					PathParams: []*scanner.PathParam{
						{
							Name: "id",
							Type: &scanner.TypeInfo{
								Name:        "int",
								IsPrimitive: true,
								FullName:    "int",
							},
						},
					},
				},
			},
			expectErrors: 1,
		},
		{
			name: "invalid path",
			handler: &scanner.Handler{
				Name:   "Show",
				Method: "GET",
				Path:   "users",
				Signature: &scanner.HandlerSignature{
					Pattern: scanner.PatternDataError,
				},
			},
			expectErrors: 1,
		},
		{
			name: "invalid signature pattern",
			handler: &scanner.Handler{
				Name:   "Show",
				Method: "GET",
				Path:   "/{id}",
				Signature: &scanner.HandlerSignature{
					Pattern: "invalid_pattern",
					PathParams: []*scanner.PathParam{
						{
							Name: "id",
							Type: &scanner.TypeInfo{
								Name:        "int",
								IsPrimitive: true,
								FullName:    "int",
							},
						},
					},
				},
			},
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			ctrl := &scanner.Controller{
				FilePath: "test.go",
			}
			v.validateHandler(tt.handler, ctrl)

			if len(v.errors) != tt.expectErrors {
				t.Errorf("expected %d errors, got %d", tt.expectErrors, len(v.errors))
			}
		})
	}
}

func TestValidateProvider(t *testing.T) {
	tests := []struct {
		name         string
		provider     *scanner.Provider
		expectErrors int
	}{
		{
			name: "valid singleton provider",
			provider: &scanner.Provider{
				Name:       "NewDatabase",
				Lifecycle:  "singleton",
				FilePath:   "test.go",
				ReturnType: &scanner.TypeInfo{FullName: "*gorm.DB"},
			},
			expectErrors: 0,
		},
		{
			name: "valid transient provider",
			provider: &scanner.Provider{
				Name:       "NewLogger",
				Lifecycle:  "transient",
				FilePath:   "test.go",
				ReturnType: &scanner.TypeInfo{FullName: "*logger.Logger"},
			},
			expectErrors: 0,
		},
		{
			name: "invalid lifecycle",
			provider: &scanner.Provider{
				Name:       "NewDatabase",
				Lifecycle:  "invalid",
				FilePath:   "test.go",
				ReturnType: &scanner.TypeInfo{FullName: "*gorm.DB"},
			},
			expectErrors: 1,
		},
		{
			name: "missing return type",
			provider: &scanner.Provider{
				Name:       "NewDatabase",
				Lifecycle:  "singleton",
				FilePath:   "test.go",
				ReturnType: nil,
			},
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.validateProvider(tt.provider)

			if len(v.errors) != tt.expectErrors {
				t.Errorf("expected %d errors, got %d", tt.expectErrors, len(v.errors))
			}
		})
	}
}

func TestValidateUniqueRoutes(t *testing.T) {
	tests := []struct {
		name         string
		controllers  []*scanner.Controller
		expectErrors int
	}{
		{
			name: "unique routes",
			controllers: []*scanner.Controller{
				{
					FilePath: "test.go",
					Handlers: []*scanner.Handler{
						{Method: "GET", FullPath: "/api/posts"},
						{Method: "POST", FullPath: "/api/posts"},
					},
				},
			},
			expectErrors: 0,
		},
		{
			name: "duplicate routes",
			controllers: []*scanner.Controller{
				{
					FilePath: "test.go",
					Handlers: []*scanner.Handler{
						{Method: "GET", FullPath: "/api/posts"},
						{Method: "GET", FullPath: "/api/posts"},
					},
				},
			},
			expectErrors: 1,
		},
		{
			name: "duplicate across controllers",
			controllers: []*scanner.Controller{
				{
					FilePath: "test1.go",
					Handlers: []*scanner.Handler{
						{Method: "GET", FullPath: "/api/posts"},
					},
				},
				{
					FilePath: "test2.go",
					Handlers: []*scanner.Handler{
						{Method: "GET", FullPath: "/api/posts"},
					},
				},
			},
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.validateUniqueRoutes(tt.controllers)

			if len(v.errors) != tt.expectErrors {
				t.Errorf("expected %d errors, got %d", tt.expectErrors, len(v.errors))
			}
		})
	}
}

func TestValidateMiddlewareReferences(t *testing.T) {
	tests := []struct {
		name         string
		project      *scanner.Project
		expectErrors int
	}{
		{
			name: "valid middleware references",
			project: &scanner.Project{
				Middleware: []*scanner.Middleware{
					{Name: "auth"},
					{Name: "ratelimit"},
				},
				Controllers: []*scanner.Controller{
					{
						FilePath: "test.go",
						Tags:     []string{}, // No tags
						Handlers: []*scanner.Handler{
							{With: []string{"auth", "ratelimit"}},
						},
					},
				},
			},
			expectErrors: 0,
		},
		{
			name: "undefined middleware in @With",
			project: &scanner.Project{
				Middleware: []*scanner.Middleware{
					{Name: "auth"},
				},
				Controllers: []*scanner.Controller{
					{
						FilePath: "test.go",
						Tags:     []string{},
					},
				},
			},
			expectErrors: 0, // No error since Tags are not validated, only @With
		},
		{
			name: "undefined middleware in handler @With",
			project: &scanner.Project{
				Middleware: []*scanner.Middleware{
					{Name: "auth"},
				},
				Controllers: []*scanner.Controller{
					{
						FilePath: "test.go",
						Tags:     []string{},
						Handlers: []*scanner.Handler{
							{With: []string{"undefined"}},
						},
					},
				},
			},
			expectErrors: 1,
		},
		{
			name: "empty middleware name in @With",
			project: &scanner.Project{
				Middleware: []*scanner.Middleware{},
				Controllers: []*scanner.Controller{
					{
						FilePath: "test.go",
						Tags:     []string{},
						Handlers: []*scanner.Handler{
							{With: []string{""}},
						},
					},
				},
			},
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.validateMiddlewareReferences(tt.project)

			if len(v.errors) != tt.expectErrors {
				t.Errorf("expected %d errors, got %d", tt.expectErrors, len(v.errors))
				for _, err := range v.errors {
					t.Logf("  - %s", err.Message)
				}
			}
		})
	}
}

func TestDetectCircularDeps(t *testing.T) {
	tests := []struct {
		name         string
		project      *scanner.Project
		expectErrors int
	}{
		{
			name: "no circular dependencies",
			project: &scanner.Project{
				Providers: []*scanner.Provider{
					{
						Name:     "NewDatabase",
						FilePath: "test.go",
						ReturnType: &scanner.TypeInfo{
							FullName: "*gorm.DB",
						},
						Dependencies: []*scanner.Field{},
					},
					{
						Name:     "NewCache",
						FilePath: "test.go",
						ReturnType: &scanner.TypeInfo{
							FullName: "*redis.Client",
						},
						Dependencies: []*scanner.Field{
							{
								Type: &scanner.TypeInfo{
									FullName: "*gorm.DB",
								},
							},
						},
					},
				},
			},
			expectErrors: 0,
		},
		{
			name: "circular dependency",
			project: &scanner.Project{
				Providers: []*scanner.Provider{
					{
						Name:     "NewA",
						FilePath: "test.go",
						ReturnType: &scanner.TypeInfo{
							FullName: "*A",
						},
						Dependencies: []*scanner.Field{
							{
								Type: &scanner.TypeInfo{
									FullName: "*B",
								},
							},
						},
					},
					{
						Name:     "NewB",
						FilePath: "test.go",
						ReturnType: &scanner.TypeInfo{
							FullName: "*B",
						},
						Dependencies: []*scanner.Field{
							{
								Type: &scanner.TypeInfo{
									FullName: "*A",
								},
							},
						},
					},
				},
			},
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := New()
			v.validateDependencies(tt.project)

			if len(v.errors) != tt.expectErrors {
				t.Errorf("expected %d errors, got %d", tt.expectErrors, len(v.errors))
				for _, err := range v.errors {
					t.Logf("  - %s", err.Message)
				}
			}
		})
	}
}
