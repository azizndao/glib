package generator

import (
	"testing"

	"github.com/azizndao/glib/internal/scanner"
)

func TestProviderFieldName(t *testing.T) {
	tests := []struct {
		functionName string
		expected     string
	}{
		{"NewDatabase", "database"},
		{"NewCache", "cache"},
		{"NewAuthService", "authService"},
		{"Database", "database"},
		{"CreateLogger", "createLogger"},
	}

	g := &Generator{}
	for _, tt := range tests {
		t.Run(tt.functionName, func(t *testing.T) {
			prov := &scanner.Provider{
				FunctionName: tt.functionName,
			}
			result := g.providerFieldName(prov)
			if result != tt.expected {
				t.Errorf("providerFieldName(%s) = %s, expected %s", tt.functionName, result, tt.expected)
			}
		})
	}
}

func TestControllerFieldName(t *testing.T) {
	tests := []struct {
		packageName string
		name        string
		expected    string
	}{
		{"post", "PostController", "postPostController"},
		{"comment", "CommentController", "commentCommentController"},
		{"auth", "AuthController", "authAuthController"},
		{"user", "UserService", "userUserService"},
	}

	g := &Generator{}
	for _, tt := range tests {
		t.Run(tt.packageName+"."+tt.name, func(t *testing.T) {
			ctrl := &scanner.Controller{
				PackageName: tt.packageName,
				Name:        tt.name,
			}
			result := g.controllerFieldName(ctrl)
			if result != tt.expected {
				t.Errorf("controllerFieldName(%s.%s) = %s, expected %s", tt.packageName, tt.name, result, tt.expected)
			}
		})
	}
}

func TestMiddlewareFieldName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"auth", "authMiddleware"},
		{"ratelimit", "ratelimitMiddleware"},
		{"cors", "corsMiddleware"},
	}

	g := &Generator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &scanner.Middleware{
				Name: tt.name,
			}
			result := g.middlewareFieldName(mw)
			if result != tt.expected {
				t.Errorf("middlewareFieldName(%s) = %s, expected %s", tt.name, result, tt.expected)
			}
		})
	}
}

func TestFindProviderForType(t *testing.T) {
	g := &Generator{
		project: &scanner.Project{
			Providers: []*scanner.Provider{
				{
					FunctionName: "NewDatabase",
					ReturnType: &scanner.TypeInfo{
						FullName: "*gorm.DB",
					},
				},
				{
					FunctionName: "NewCache",
					ReturnType: &scanner.TypeInfo{
						FullName: "*redis.Client",
					},
				},
			},
		},
	}

	tests := []struct {
		name     string
		typeInfo *scanner.TypeInfo
		expected string
	}{
		{
			name: "find gorm.DB",
			typeInfo: &scanner.TypeInfo{
				FullName: "*gorm.DB",
			},
			expected: "database",
		},
		{
			name: "find redis.Client",
			typeInfo: &scanner.TypeInfo{
				FullName: "*redis.Client",
			},
			expected: "cache",
		},
		{
			name: "not found",
			typeInfo: &scanner.TypeInfo{
				FullName: "*logger.Logger",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.findProviderForType(tt.typeInfo)
			if result != tt.expected {
				t.Errorf("findProviderForType() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestIsProviderTransient(t *testing.T) {
	tests := []struct {
		name      string
		typeInfo  *scanner.TypeInfo
		providers []*scanner.Provider
		expected  bool
	}{
		{
			name: "transient provider",
			typeInfo: &scanner.TypeInfo{
				FullName: "services.Logger",
			},
			providers: []*scanner.Provider{
				{
					Lifecycle: "transient",
					ReturnType: &scanner.TypeInfo{
						FullName: "services.Logger",
					},
				},
			},
			expected: true,
		},
		{
			name: "singleton provider",
			typeInfo: &scanner.TypeInfo{
				FullName: "services.Database",
			},
			providers: []*scanner.Provider{
				{
					Lifecycle: "singleton",
					ReturnType: &scanner.TypeInfo{
						FullName: "services.Database",
					},
				},
			},
			expected: false,
		},
		{
			name: "not found",
			typeInfo: &scanner.TypeInfo{
				FullName: "services.Unknown",
			},
			providers: []*scanner.Provider{
				{
					Lifecycle: "singleton",
					ReturnType: &scanner.TypeInfo{
						FullName: "services.Database",
					},
				},
			},
			expected: false,
		},
		{
			name:      "nil type info",
			typeInfo:  nil,
			providers: []*scanner.Provider{},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Generator{
				project: &scanner.Project{
					Providers: tt.providers,
				},
			}

			result := g.isProviderTransient(tt.typeInfo)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
