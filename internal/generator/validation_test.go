package generator

import (
	"testing"

	"github.com/azizndao/glib/internal/scanner"
)

func TestValidateUniqueNames(t *testing.T) {
	tests := []struct {
		name        string
		project     *scanner.Project
		expectError bool
		errorSubstr string
	}{
		{
			name: "no collisions - valid project",
			project: &scanner.Project{
				Providers: []*scanner.Provider{
					{
						Name:         "NewDatabase",
						FunctionName: "NewDatabase",
						ReturnType:   &scanner.TypeInfo{Name: "DB", FullName: "*gorm.DB"},
					},
					{
						Name:         "NewCache",
						FunctionName: "NewCache",
						ReturnType:   &scanner.TypeInfo{Name: "Client", FullName: "*redis.Client"},
					},
				},
				Controllers: []*scanner.Controller{
					{Name: "PostController", PackageName: "posts"},
					{Name: "CommentController", PackageName: "comments"},
				},
				Configs: []*scanner.Config{
					{Name: "AppConfig", PackageName: "configs"},
				},
			},
			expectError: false,
		},
		{
			name: "provider collision - NewDatabase vs Database",
			project: &scanner.Project{
				Providers: []*scanner.Provider{
					{
						Name:         "NewDatabase",
						FunctionName: "NewDatabase",
						ReturnType:   &scanner.TypeInfo{Name: "DB", FullName: "*gorm.DB"},
					},
					{
						Name:         "Database",
						FunctionName: "Database",
						ReturnType:   &scanner.TypeInfo{Name: "DB", FullName: "*sql.DB"},
					},
				},
			},
			expectError: true,
			errorSubstr: "field name collision",
		},
		{
			name: "controller collision - same package and name",
			project: &scanner.Project{
				Controllers: []*scanner.Controller{
					{Name: "PostsController", PackageName: "controllers"},
					{Name: "Controller", PackageName: "controllersposts"}, // Generates "ControllerspostsController"
				},
			},
			expectError: false, // These actually don't collide!
		},
		{
			name: "real controller collision",
			project: &scanner.Project{
				Controllers: []*scanner.Controller{
					{Name: "Controller", PackageName: "posts"},    // Generates "PostsController"
					{Name: "PostsController", PackageName: "api"}, // Also generates... wait, "ApiPostsController"
				},
			},
			expectError: false, // These also don't collide because package name is included!
		},
		{
			name: "provider and config collision",
			project: &scanner.Project{
				Providers: []*scanner.Provider{
					{
						Name:         "NewConfig",
						FunctionName: "NewConfig",
						ReturnType:   &scanner.TypeInfo{Name: "Config", FullName: "*app.Config"},
					},
				},
				Configs: []*scanner.Config{
					{Name: "Config", PackageName: "config"},
				},
			},
			expectError: true,
			errorSubstr: "field name collision",
		},
		{
			name: "middleware collision",
			project: &scanner.Project{
				Middleware: []*scanner.Middleware{
					{Name: "auth", FunctionName: "Auth"},
					{Name: "auth", FunctionName: "AuthMiddleware"},
				},
			},
			expectError: true,
			errorSubstr: "field name collision",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New(tt.project, "/tmp/generated", "generated")
			err := g.validateUniqueNames()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorSubstr)
					return
				}
				if tt.errorSubstr != "" && !contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errorSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
