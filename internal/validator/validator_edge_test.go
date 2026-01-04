package validator

import (
	"testing"

	"github.com/azizndao/glib/internal/scanner"
)

// TestValidator_EdgeCases tests various edge cases for the validator
func TestValidator_EdgeCases(t *testing.T) {
	t.Run("empty project", func(t *testing.T) {
		v := New()
		project := &scanner.Project{
			Controllers: []*scanner.Controller{},
			Providers:   []*scanner.Provider{},
			Middleware:  []*scanner.Middleware{},
		}

		err := v.Validate(project)
		if err != nil {
			t.Errorf("empty project should not produce validation errors, got: %v", err)
		}
	})

	t.Run("nil signature in handler", func(t *testing.T) {
		v := New()
		ctrl := &scanner.Controller{
			Name:        "TestController",
			RoutePrefix: "/test",
			FilePath:    "test.go",
			Handlers: []*scanner.Handler{
				{
					Name:      "Test",
					Method:    "GET",
					Path:      "/",
					Signature: nil, // nil signature
				},
			},
		}

		v.validateController(ctrl)
		if len(v.errors) == 0 {
			t.Error("expected error for nil signature")
		}
	})

	t.Run("path parameters with special characters", func(t *testing.T) {
		v := New()
		handler := &scanner.Handler{
			Name:   "Show",
			Method: "GET",
			Path:   "/{user-id}/{post_id}/{commentID}",
			Signature: &scanner.HandlerSignature{
				Pattern: scanner.PatternResult,
				PathParams: []*scanner.PathParam{
					{Name: "user-id", Type: &scanner.TypeInfo{Name: "string", IsPrimitive: true, FullName: "string"}},
					{Name: "post_id", Type: &scanner.TypeInfo{Name: "string", IsPrimitive: true, FullName: "string"}},
					{Name: "commentID", Type: &scanner.TypeInfo{Name: "string", IsPrimitive: true, FullName: "string"}},
				},
			},
		}
		ctrl := &scanner.Controller{FilePath: "test.go"}

		v.validateHandler(handler, ctrl)
		if len(v.errors) > 0 {
			t.Errorf("valid path parameters with special chars should not produce errors, got: %v", v.errors)
		}
	})

	t.Run("nested path parameters", func(t *testing.T) {
		params := extractPathParams("/{orgId}/projects/{projectId}/issues/{issueId}/comments/{commentId}")
		expected := []string{"orgId", "projectId", "issueId", "commentId"}

		if len(params) != len(expected) {
			t.Errorf("expected %d params, got %d", len(expected), len(params))
		}
		for i, p := range params {
			if p != expected[i] {
				t.Errorf("param %d: expected %s, got %s", i, expected[i], p)
			}
		}
	})

	t.Run("path parameters with empty braces", func(t *testing.T) {
		params := extractPathParams("/api/{}/test")
		if len(params) != 1 || params[0] != "" {
			t.Errorf("expected one empty param, got: %v", params)
		}
	})

	t.Run("path with malformed parameters", func(t *testing.T) {
		params := extractPathParams("/api/{unclosed/test/{valid}")
		// Should extract "valid" but not "unclosed"
		if len(params) != 1 || params[0] != "valid" {
			t.Errorf("expected ['valid'], got: %v", params)
		}
	})

	t.Run("UUID path parameter type", func(t *testing.T) {
		valid := isValidPathParamType(&scanner.TypeInfo{
			Name:        "UUID",
			PackageName: "uuid",
			FullName:    "github.com/google/uuid.UUID",
		})
		if !valid {
			t.Error("UUID should be valid path parameter type")
		}
	})

	t.Run("unsupported path parameter types", func(t *testing.T) {
		testCases := []struct {
			name     string
			typeInfo *scanner.TypeInfo
			valid    bool
		}{
			{
				name:     "struct type",
				typeInfo: &scanner.TypeInfo{Name: "User", FullName: "models.User"},
				valid:    false,
			},
			{
				name:     "slice type",
				typeInfo: &scanner.TypeInfo{Name: "string", IsSlice: true, FullName: "[]string"},
				valid:    false,
			},
			{
				name:     "int32",
				typeInfo: &scanner.TypeInfo{Name: "int32", IsPrimitive: true, FullName: "int32"},
				valid:    true,
			},
			{
				name:     "uint32",
				typeInfo: &scanner.TypeInfo{Name: "uint32", IsPrimitive: true, FullName: "uint32"},
				valid:    true,
			},
			{
				name:     "float32",
				typeInfo: &scanner.TypeInfo{Name: "float32", IsPrimitive: true, FullName: "float32"},
				valid:    true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				valid := isValidPathParamType(tc.typeInfo)
				if valid != tc.valid {
					t.Errorf("expected %v for %s, got %v", tc.valid, tc.name, valid)
				}
			})
		}
	})

	t.Run("query parameter types", func(t *testing.T) {
		testCases := []struct {
			name     string
			typeInfo *scanner.TypeInfo
			valid    bool
		}{
			{
				name:     "string pointer",
				typeInfo: &scanner.TypeInfo{Name: "string", IsPointer: true, IsPrimitive: true, FullName: "*string"},
				valid:    true,
			},
			{
				name:     "int pointer",
				typeInfo: &scanner.TypeInfo{Name: "int", IsPointer: true, IsPrimitive: true, FullName: "*int"},
				valid:    true,
			},
			{
				name:     "string slice",
				typeInfo: &scanner.TypeInfo{Name: "string", IsSlice: true, FullName: "[]string"},
				valid:    true,
			},
			{
				name:     "int slice",
				typeInfo: &scanner.TypeInfo{Name: "int", IsSlice: true, FullName: "[]int"},
				valid:    false,
			},
			{
				name:     "time.Time",
				typeInfo: &scanner.TypeInfo{Name: "Time", PackageName: "time", FullName: "time.Time"},
				valid:    true,
			},
			{
				name:     "UUID pointer",
				typeInfo: &scanner.TypeInfo{Name: "UUID", PackageName: "uuid", IsPointer: true, FullName: "*uuid.UUID"},
				valid:    true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				valid := isValidQueryParamType(tc.typeInfo)
				if valid != tc.valid {
					t.Errorf("expected %v for %s, got %v", tc.valid, tc.name, valid)
				}
			})
		}
	})

	t.Run("header parameter types", func(t *testing.T) {
		testCases := []struct {
			name     string
			typeInfo *scanner.TypeInfo
			valid    bool
		}{
			{
				name:     "string",
				typeInfo: &scanner.TypeInfo{Name: "string", IsPrimitive: true, FullName: "string"},
				valid:    true,
			},
			{
				name:     "string pointer",
				typeInfo: &scanner.TypeInfo{Name: "string", IsPointer: true, FullName: "*string"},
				valid:    true,
			},
			{
				name:     "int",
				typeInfo: &scanner.TypeInfo{Name: "int", IsPrimitive: true, FullName: "int"},
				valid:    false,
			},
			{
				name:     "int pointer",
				typeInfo: &scanner.TypeInfo{Name: "int", IsPointer: true, FullName: "*int"},
				valid:    false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				valid := isValidHeaderParamType(tc.typeInfo)
				if valid != tc.valid {
					t.Errorf("expected %v for %s, got %v", tc.valid, tc.name, valid)
				}
			})
		}
	})

	t.Run("middleware with empty name", func(t *testing.T) {
		v := New()
		mw := &scanner.Middleware{
			Name:     "",
			Target:   "all",
			Order:    0,
			FilePath: "test.go",
		}

		v.validateMiddleware(mw)
		if len(v.errors) == 0 {
			t.Error("expected error for empty middleware name")
		}
	})

	t.Run("middleware with empty target", func(t *testing.T) {
		v := New()
		mw := &scanner.Middleware{
			Name:     "auth",
			Target:   "",
			Order:    0,
			FilePath: "test.go",
		}

		v.validateMiddleware(mw)
		if len(v.errors) == 0 {
			t.Error("expected error for empty middleware target")
		}
	})

	t.Run("middleware with negative order", func(t *testing.T) {
		v := New()
		mw := &scanner.Middleware{
			Name:     "auth",
			Target:   "all",
			Order:    -1,
			FilePath: "test.go",
		}

		v.validateMiddleware(mw)
		if len(v.errors) == 0 {
			t.Error("expected error for negative middleware order")
		}
	})

	t.Run("middleware with invalid signature", func(t *testing.T) {
		v := New()
		mw := &scanner.Middleware{
			Name:      "auth",
			Target:    "all",
			Order:     0,
			Signature: "invalid",
			FilePath:  "test.go",
		}

		v.validateMiddleware(mw)
		if len(v.errors) == 0 {
			t.Error("expected error for invalid middleware signature")
		}
	})

	t.Run("middleware with valid chi signature", func(t *testing.T) {
		v := New()
		mw := &scanner.Middleware{
			Name:      "auth",
			Target:    "all",
			Order:     0,
			Signature: "chi",
			FilePath:  "test.go",
		}

		v.validateMiddleware(mw)
		if len(v.errors) > 0 {
			t.Errorf("chi signature should be valid, got errors: %v", v.errors)
		}
	})

	t.Run("middleware with valid glib signature", func(t *testing.T) {
		v := New()
		mw := &scanner.Middleware{
			Name:      "auth",
			Target:    "all",
			Order:     0,
			Signature: "glib",
			FilePath:  "test.go",
		}

		v.validateMiddleware(mw)
		if len(v.errors) > 0 {
			t.Errorf("glib signature should be valid, got errors: %v", v.errors)
		}
	})

	t.Run("handler with @With none", func(t *testing.T) {
		v := New()
		project := &scanner.Project{
			Middleware: []*scanner.Middleware{
				{Name: "auth"},
			},
			Controllers: []*scanner.Controller{
				{
					FilePath: "test.go",
					Handlers: []*scanner.Handler{
						{With: []string{"none"}},
					},
				},
			},
		}

		v.validateMiddlewareReferences(project)
		if len(v.errors) > 0 {
			t.Errorf("@With none should be valid, got errors: %v", v.errors)
		}
	})

	t.Run("middleware targeting non-existent tag", func(t *testing.T) {
		v := New()
		project := &scanner.Project{
			Middleware: []*scanner.Middleware{
				{
					Name:     "auth",
					Target:   "admin,public",
					FilePath: "test.go",
				},
			},
			Controllers: []*scanner.Controller{
				{
					FilePath: "test.go",
					Tags:     []string{"public"}, // "admin" doesn't exist
				},
			},
		}

		v.validateMiddlewareReferences(project)
		if len(v.warnings) == 0 {
			t.Error("expected warning for middleware targeting non-existent tag")
		}
	})

	t.Run("three-way circular dependency", func(t *testing.T) {
		v := New()
		project := &scanner.Project{
			Providers: []*scanner.Provider{
				{
					Name:     "NewA",
					FilePath: "test.go",
					ReturnType: &scanner.TypeInfo{
						FullName: "*A",
					},
					Dependencies: []*scanner.Field{
						{Type: &scanner.TypeInfo{FullName: "*B"}},
					},
				},
				{
					Name:     "NewB",
					FilePath: "test.go",
					ReturnType: &scanner.TypeInfo{
						FullName: "*B",
					},
					Dependencies: []*scanner.Field{
						{Type: &scanner.TypeInfo{FullName: "*C"}},
					},
				},
				{
					Name:     "NewC",
					FilePath: "test.go",
					ReturnType: &scanner.TypeInfo{
						FullName: "*C",
					},
					Dependencies: []*scanner.Field{
						{Type: &scanner.TypeInfo{FullName: "*A"}},
					},
				},
			},
		}

		v.validateDependencies(project)
		if len(v.errors) == 0 {
			t.Error("expected error for three-way circular dependency")
		}
	})

	t.Run("self-referencing provider", func(t *testing.T) {
		v := New()
		project := &scanner.Project{
			Providers: []*scanner.Provider{
				{
					Name:     "NewA",
					FilePath: "test.go",
					ReturnType: &scanner.TypeInfo{
						FullName: "*A",
					},
					Dependencies: []*scanner.Field{
						{Type: &scanner.TypeInfo{FullName: "*A"}},
					},
				},
			},
		}

		v.validateDependencies(project)
		if len(v.errors) == 0 {
			t.Error("expected error for self-referencing provider")
		}
	})

	t.Run("provider with nil return type in dependencies", func(t *testing.T) {
		v := New()
		project := &scanner.Project{
			Providers: []*scanner.Provider{
				{
					Name:       "NewA",
					FilePath:   "test.go",
					ReturnType: nil, // nil return type
					Dependencies: []*scanner.Field{
						{Type: &scanner.TypeInfo{FullName: "*B"}},
					},
				},
			},
		}

		v.validateDependencies(project)
		// Should not crash
	})

	t.Run("controller with missing provider dependency", func(t *testing.T) {
		v := New()
		project := &scanner.Project{
			Controllers: []*scanner.Controller{
				{
					Name:     "UserController",
					FilePath: "test.go",
					Fields: []*scanner.Field{
						{
							Name: "userService",
							Type: &scanner.TypeInfo{
								FullName:    "*services.UserService",
								IsPrimitive: false,
							},
						},
					},
				},
			},
			Providers: []*scanner.Provider{}, // No providers
		}

		v.validateDependencies(project)
		if len(v.warnings) == 0 {
			t.Error("expected warning for missing provider dependency")
		}
	})

	t.Run("handler with mismatched path param count", func(t *testing.T) {
		v := New()
		handler := &scanner.Handler{
			Name:   "Show",
			Method: "GET",
			Path:   "/{id}/{name}",
			Signature: &scanner.HandlerSignature{
				Pattern: scanner.PatternResult,
				PathParams: []*scanner.PathParam{
					{Name: "id", Type: &scanner.TypeInfo{Name: "int", IsPrimitive: true}},
					// Missing "name" parameter
				},
			},
		}
		ctrl := &scanner.Controller{FilePath: "test.go"}

		v.validateHandler(handler, ctrl)
		if len(v.errors) == 0 {
			t.Error("expected error for mismatched path param count")
		}
	})

	t.Run("handler with mismatched path param names", func(t *testing.T) {
		v := New()
		handler := &scanner.Handler{
			Name:   "Show",
			Method: "GET",
			Path:   "/{id}",
			Signature: &scanner.HandlerSignature{
				Pattern: scanner.PatternResult,
				PathParams: []*scanner.PathParam{
					{Name: "userId", Type: &scanner.TypeInfo{Name: "int", IsPrimitive: true}}, // Different name
				},
			},
		}
		ctrl := &scanner.Controller{FilePath: "test.go"}

		v.validateHandler(handler, ctrl)
		if len(v.warnings) == 0 {
			t.Error("expected warning for mismatched path param names")
		}
	})

	t.Run("validate full project", func(t *testing.T) {
		v := New()
		project := &scanner.Project{
			Controllers: []*scanner.Controller{
				{
					Name:        "UserController",
					RoutePrefix: "/users",
					FilePath:    "controllers/user.go",
					Handlers: []*scanner.Handler{
						{
							Name:     "List",
							Method:   "GET",
							Path:     "/",
							FullPath: "/users",
							Signature: &scanner.HandlerSignature{
								Pattern: scanner.PatternResult,
							},
						},
					},
				},
			},
			Providers: []*scanner.Provider{
				{
					Name:       "NewDB",
					Lifecycle:  "singleton",
					FilePath:   "providers/db.go",
					ReturnType: &scanner.TypeInfo{FullName: "*gorm.DB"},
				},
			},
			Middleware: []*scanner.Middleware{
				{
					Name:      "auth",
					Target:    "all",
					Order:     0,
					Signature: "chi",
					FilePath:  "middleware/auth.go",
				},
			},
		}

		err := v.Validate(project)
		if err != nil {
			t.Errorf("valid project should not produce errors, got: %v", err)
		}
	})

	t.Run("ValidationError Error method", func(t *testing.T) {
		err := &ValidationError{
			Type:     "error",
			Location: "test.go:10",
			Message:  "test error message",
		}

		expected := "[error] test.go:10: test error message"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("Errors and Warnings accessors", func(t *testing.T) {
		v := New()
		v.addError("test.go:10", "error message")
		v.addWarning("test.go:20", "warning message")

		if len(v.Errors()) != 1 {
			t.Errorf("expected 1 error, got %d", len(v.Errors()))
		}
		if len(v.Warnings()) != 1 {
			t.Errorf("expected 1 warning, got %d", len(v.Warnings()))
		}
	})
}

// TestValidator_QueryAndHeaderValidation tests query and header parameter validation
func TestValidator_QueryAndHeaderValidation(t *testing.T) {
	t.Run("handler with invalid query param type", func(t *testing.T) {
		v := New()
		handler := &scanner.Handler{
			Name:   "Search",
			Method: "GET",
			Path:   "/search",
			Signature: &scanner.HandlerSignature{
				Pattern: scanner.PatternResult,
				QueryParams: []*scanner.QueryParam{
					{
						FieldName: "Query",
						ParamName: "q",
						Type: &scanner.TypeInfo{
							Name:     "ComplexType",
							FullName: "models.ComplexType",
						},
					},
				},
			},
		}
		ctrl := &scanner.Controller{FilePath: "test.go"}

		v.validateHandler(handler, ctrl)
		if len(v.errors) == 0 {
			t.Error("expected error for invalid query param type")
		}
	})

	t.Run("handler with valid query param types", func(t *testing.T) {
		v := New()
		handler := &scanner.Handler{
			Name:   "Search",
			Method: "GET",
			Path:   "/search",
			Signature: &scanner.HandlerSignature{
				Pattern: scanner.PatternResult,
				QueryParams: []*scanner.QueryParam{
					{FieldName: "Q", ParamName: "q", Type: &scanner.TypeInfo{Name: "string", IsPrimitive: true}},
					{FieldName: "Page", ParamName: "page", Type: &scanner.TypeInfo{Name: "int", IsPrimitive: true}},
					{FieldName: "Limit", ParamName: "limit", Type: &scanner.TypeInfo{Name: "int64", IsPrimitive: true}},
					{FieldName: "Active", ParamName: "active", Type: &scanner.TypeInfo{Name: "bool", IsPrimitive: true}},
					{FieldName: "Price", ParamName: "price", Type: &scanner.TypeInfo{Name: "float64", IsPrimitive: true}},
					{FieldName: "Tags", ParamName: "tags", Type: &scanner.TypeInfo{Name: "string", IsSlice: true}},
				},
			},
		}
		ctrl := &scanner.Controller{FilePath: "test.go"}

		v.validateHandler(handler, ctrl)
		if len(v.errors) > 0 {
			t.Errorf("valid query params should not produce errors, got: %v", v.errors)
		}
	})

	t.Run("handler with invalid header param type", func(t *testing.T) {
		v := New()
		handler := &scanner.Handler{
			Name:   "Get",
			Method: "GET",
			Path:   "/",
			Signature: &scanner.HandlerSignature{
				Pattern: scanner.PatternResult,
				HeaderParams: []*scanner.HeaderParam{
					{
						FieldName:  "ContentType",
						HeaderName: "Content-Type",
						Type: &scanner.TypeInfo{
							Name:        "int",
							IsPrimitive: true,
							FullName:    "int",
						},
					},
				},
			},
		}
		ctrl := &scanner.Controller{FilePath: "test.go"}

		v.validateHandler(handler, ctrl)
		if len(v.errors) == 0 {
			t.Error("expected error for invalid header param type")
		}
	})

	t.Run("handler with valid header param types", func(t *testing.T) {
		v := New()
		handler := &scanner.Handler{
			Name:   "Get",
			Method: "GET",
			Path:   "/",
			Signature: &scanner.HandlerSignature{
				Pattern: scanner.PatternResult,
				HeaderParams: []*scanner.HeaderParam{
					{FieldName: "Auth", HeaderName: "Authorization", Type: &scanner.TypeInfo{Name: "string", IsPrimitive: true}},
					{FieldName: "OptionalHeader", HeaderName: "X-Custom", Type: &scanner.TypeInfo{Name: "string", IsPointer: true}},
				},
			},
		}
		ctrl := &scanner.Controller{FilePath: "test.go"}

		v.validateHandler(handler, ctrl)
		if len(v.errors) > 0 {
			t.Errorf("valid header params should not produce errors, got: %v", v.errors)
		}
	})

	t.Run("raw HTTP handler skips param validation", func(t *testing.T) {
		v := New()
		handler := &scanner.Handler{
			Name:   "RawHandler",
			Method: "GET",
			Path:   "/{id}",
			Signature: &scanner.HandlerSignature{
				Pattern:    scanner.PatternRawHTTP,
				HasRawHTTP: true,
				// No params defined
			},
		}
		ctrl := &scanner.Controller{FilePath: "test.go"}

		v.validateHandler(handler, ctrl)
		// Should not validate path params for raw HTTP
		if len(v.errors) > 0 {
			t.Errorf("raw HTTP handler should skip param validation, got: %v", v.errors)
		}
	})
}

// TestValidator_ComplexScenarios tests complex validation scenarios
func TestValidator_ComplexScenarios(t *testing.T) {
	t.Run("complex dependency graph", func(t *testing.T) {
		v := New()
		project := &scanner.Project{
			Providers: []*scanner.Provider{
				{
					Name:       "NewDB",
					FilePath:   "db.go",
					ReturnType: &scanner.TypeInfo{FullName: "*gorm.DB"},
				},
				{
					Name:     "NewCache",
					FilePath: "cache.go",
					ReturnType: &scanner.TypeInfo{
						FullName: "*redis.Client",
					},
					Dependencies: []*scanner.Field{
						{Type: &scanner.TypeInfo{FullName: "*gorm.DB"}},
					},
				},
				{
					Name:     "NewUserService",
					FilePath: "user.go",
					ReturnType: &scanner.TypeInfo{
						FullName: "*UserService",
					},
					Dependencies: []*scanner.Field{
						{Type: &scanner.TypeInfo{FullName: "*gorm.DB"}},
						{Type: &scanner.TypeInfo{FullName: "*redis.Client"}},
					},
				},
				{
					Name:     "NewPostService",
					FilePath: "post.go",
					ReturnType: &scanner.TypeInfo{
						FullName: "*PostService",
					},
					Dependencies: []*scanner.Field{
						{Type: &scanner.TypeInfo{FullName: "*UserService"}},
						{Type: &scanner.TypeInfo{FullName: "*redis.Client"}},
					},
				},
			},
		}

		v.validateDependencies(project)
		if len(v.errors) > 0 {
			t.Errorf("complex but valid dependency graph should not produce errors, got: %v", v.errors)
		}
	})

	t.Run("multiple controllers with mixed routes", func(t *testing.T) {
		v := New()
		project := &scanner.Project{
			Controllers: []*scanner.Controller{
				{
					Name:        "UserController",
					RoutePrefix: "/api/users",
					FilePath:    "user.go",
					Tags:        []string{"api", "users"},
					Handlers: []*scanner.Handler{
						{Method: "GET", Path: "/", FullPath: "/api/users", With: []string{"auth"}},
						{Method: "POST", Path: "/", FullPath: "/api/users", With: []string{"auth", "ratelimit"}},
						{Method: "GET", Path: "/{id}", FullPath: "/api/users/{id}"},
					},
				},
				{
					Name:        "PostController",
					RoutePrefix: "/api/posts",
					FilePath:    "post.go",
					Tags:        []string{"api", "posts"},
					Handlers: []*scanner.Handler{
						{Method: "GET", Path: "/", FullPath: "/api/posts"},
						{Method: "POST", Path: "/", FullPath: "/api/posts", With: []string{"auth"}},
					},
				},
			},
			Middleware: []*scanner.Middleware{
				{Name: "auth", Target: "api", Order: 0, Signature: "chi", FilePath: "auth.go"},
				{Name: "ratelimit", Target: "all", Order: 1, Signature: "chi", FilePath: "ratelimit.go"},
			},
		}

		v.validateMiddlewareReferences(project)
		v.validateUniqueRoutes(project.Controllers)

		if len(v.errors) > 0 {
			t.Errorf("valid multi-controller setup should not produce errors, got: %v", v.errors)
		}
	})

	t.Run("middleware with comma-separated targets and empty spaces", func(t *testing.T) {
		v := New()
		project := &scanner.Project{
			Middleware: []*scanner.Middleware{
				{
					Name:     "auth",
					Target:   "api, admin,  public",
					FilePath: "auth.go",
				},
			},
			Controllers: []*scanner.Controller{
				{
					FilePath: "test.go",
					Tags:     []string{"api", "admin", "public"},
				},
			},
		}

		v.validateMiddlewareReferences(project)
		if len(v.warnings) > 0 {
			t.Errorf("valid comma-separated targets should not produce warnings, got: %v", v.warnings)
		}
	})

	t.Run("provider with primitive dependencies", func(t *testing.T) {
		v := New()
		project := &scanner.Project{
			Providers: []*scanner.Provider{
				{
					Name:     "NewConfig",
					FilePath: "config.go",
					ReturnType: &scanner.TypeInfo{
						FullName: "*Config",
					},
					Dependencies: []*scanner.Field{
						{Type: &scanner.TypeInfo{Name: "string", IsPrimitive: true}},
						{Type: &scanner.TypeInfo{Name: "int", IsPrimitive: true}},
					},
				},
			},
		}

		v.validateDependencies(project)
		// Primitive dependencies should be ignored
		if len(v.errors) > 0 {
			t.Errorf("primitive dependencies should not produce errors, got: %v", v.errors)
		}
	})
}
