package scanner

import (
	"go/ast"
	"go/token"
)

// Project represents the complete scanned project
type Project struct {
	Module      string
	Controllers []*Controller
	Providers   []*Provider
	Middleware  []*Middleware
}

// Controller represents a scanned controller
type Controller struct {
	Name        string         // e.g., "PostsController"
	PackageName string         // e.g., "posts"
	PackagePath string         // e.g., "myapp/controllers/posts"
	FilePath    string         // e.g., "/path/to/controllers/posts/controller.go"
	RoutePrefix string         // e.g., "/api/v1/posts"
	Middlewares []string       // e.g., ["auth", "ratelimit"]
	Handlers    []*Handler     // Handler methods
	Fields      []*Field       // Injected dependencies
	TypeSpec    *ast.TypeSpec  // Original AST node
	Position    token.Position // Source position
}

// Handler represents a controller method annotated with @Route
type Handler struct {
	Name        string            // e.g., "Show"
	Method      string            // e.g., "GET"
	Path        string            // e.g., "/{id}"
	FullPath    string            // e.g., "/api/v1/posts/{id}"
	Middlewares []string          // e.g., ["cache"]
	Signature   *HandlerSignature // Parsed signature
	FuncDecl    *ast.FuncDecl     // Original AST node
	Position    token.Position    // Source position
}

// HandlerSignature represents a parsed handler signature
type HandlerSignature struct {
	Pattern      int          // 1-9 (see 02-HANDLERS.md)
	Receiver     *Field       // Controller receiver
	PathParams   []*PathParam // Path parameters (e.g., id uuid.UUID)
	RequestType  *TypeInfo    // Request body type (if any)
	ResponseType *TypeInfo    // Response type (if any)
	ReturnsError bool         // Whether it returns error
	HasContext   bool         // Whether it has context.Context param
	HasRawHTTP   bool         // Whether it uses http.ResponseWriter/Request
}

// PathParam represents a path parameter in handler signature
type PathParam struct {
	Name     string    // e.g., "id"
	Type     *TypeInfo // e.g., uuid.UUID
	Position int       // Parameter position in function signature
}

// Provider represents a function annotated with @Provider
type Provider struct {
	Name         string         // e.g., "NewDatabase"
	FunctionName string         // e.g., "NewDatabase"
	PackageName  string         // e.g., "providers"
	PackagePath  string         // e.g., "myapp/providers"
	FilePath     string         // e.g., "/path/to/providers/database.go"
	Lifecycle    string         // "singleton" or "transient"
	ReturnType   *TypeInfo      // What it provides
	Dependencies []*Field       // What it depends on
	FuncDecl     *ast.FuncDecl  // Original AST node
	Position     token.Position // Source position
}

// Middleware represents a function annotated with @Middleware
type Middleware struct {
	Name         string         // e.g., "auth"
	FunctionName string         // e.g., "Auth"
	PackageName  string         // e.g., "middleware"
	PackagePath  string         // e.g., "myapp/middleware"
	FilePath     string         // e.g., "/path/to/middleware/auth.go"
	Dependencies []*Field       // What it depends on
	FuncDecl     *ast.FuncDecl  // Original AST node
	Position     token.Position // Source position
}

// Field represents a struct field (for DI) or function parameter
type Field struct {
	Name string    // e.g., "DB"
	Type *TypeInfo // e.g., *gorm.DB
}

// TypeInfo represents a Go type with package information
type TypeInfo struct {
	Name        string // e.g., "DB", "UUID", "string", "int"
	PackagePath string // e.g., "gorm.io/gorm", "github.com/google/uuid"
	PackageName string // e.g., "gorm", "uuid"
	IsPointer   bool   // e.g., true for *gorm.DB
	IsSlice     bool   // e.g., true for []Post
	IsError     bool   // Special case for error type
	IsContext   bool   // Special case for context.Context
	IsPrimitive bool   // true for string, int, etc.
	FullName    string // e.g., "*gorm.DB", "[]Post", "uuid.UUID"
}

// Annotation represents a parsed annotation from comments
type Annotation struct {
	Type  string            // e.g., "Controller", "Route", "Provider", "Middleware"
	Value string            // Primary value (e.g., "/api/v1/posts", "GET /", "singleton")
	Args  map[string]string // Additional arguments (future use)
	Line  int               // Line number in source
}
