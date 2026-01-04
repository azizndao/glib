package scanner

import "fmt"

// AnnotationType represents the type of annotation
type AnnotationType string

const (
	AnnotationController AnnotationType = "Controller" // @Controller annotation
	AnnotationRoute      AnnotationType = "Route"      // @Route annotation
	AnnotationProvider   AnnotationType = "Provider"   // @Provider annotation
	AnnotationMiddleware AnnotationType = "Middleware" // @Middleware annotation
	AnnotationConfig     AnnotationType = "Config"     // @Config annotation
)

// String returns the string representation of the annotation type
func (a AnnotationType) String() string {
	return string(a)
}

// IsValid checks if the annotation type is valid
func (a AnnotationType) IsValid() bool {
	switch a {
	case AnnotationController, AnnotationRoute, AnnotationProvider, AnnotationMiddleware, AnnotationConfig:
		return true
	}
	return false
}

// HTTPMethod represents an HTTP method
type HTTPMethod string

const (
	MethodGet     HTTPMethod = "GET"
	MethodPost    HTTPMethod = "POST"
	MethodPut     HTTPMethod = "PUT"
	MethodPatch   HTTPMethod = "PATCH"
	MethodDelete  HTTPMethod = "DELETE"
	MethodHead    HTTPMethod = "HEAD"
	MethodOptions HTTPMethod = "OPTIONS"
)

// String returns the string representation of the HTTP method
func (m HTTPMethod) String() string {
	return string(m)
}

// IsValid checks if the HTTP method is valid
func (m HTTPMethod) IsValid() bool {
	switch m {
	case MethodGet, MethodPost, MethodPut, MethodPatch, MethodDelete, MethodHead, MethodOptions:
		return true
	}
	return false
}

// ParseHTTPMethod parses a string into an HTTPMethod
func ParseHTTPMethod(s string) (HTTPMethod, error) {
	method := HTTPMethod(s)
	if !method.IsValid() {
		return "", fmt.Errorf("invalid HTTP method: %s", s)
	}
	return method, nil
}

// HandlerPattern represents the pattern of a handler signature
type HandlerPattern string

const (
	PatternDataError HandlerPattern = "data_error" // (T, error) - Returns data and error
	PatternErrorOnly HandlerPattern = "error_only" // error - Returns error only (no content)
	PatternRawHTTP   HandlerPattern = "raw_http"   // Raw HTTP - Direct http.ResponseWriter and *http.Request handlers
)

// String returns the string representation of the handler pattern
func (p HandlerPattern) String() string {
	return string(p)
}

// IsValid checks if the handler pattern is valid
func (p HandlerPattern) IsValid() bool {
	switch p {
	case PatternDataError, PatternErrorOnly, PatternRawHTTP:
		return true
	}
	return false
}

// MiddlewareSignature represents the signature type of middleware
type MiddlewareSignature string

const (
	MiddlewareSignatureStandard MiddlewareSignature = "standard" // func(http.Handler) http.Handler
	MiddlewareSignatureGlib     MiddlewareSignature = "glib"     // func(glib.Request, glib.Next) glib.Response
)

// String returns the string representation of the middleware signature
func (s MiddlewareSignature) String() string {
	return string(s)
}

// IsValid checks if the middleware signature is valid
func (s MiddlewareSignature) IsValid() bool {
	switch s {
	case MiddlewareSignatureStandard, MiddlewareSignatureGlib:
		return true
	}
	return false
}

// Lifecycle represents the lifecycle of a provider
type Lifecycle string

const (
	LifecycleSingleton Lifecycle = "singleton" // Single instance shared across the application
	LifecycleTransient Lifecycle = "transient" // New instance created for each request
)

// String returns the string representation of the lifecycle
func (l Lifecycle) String() string {
	return string(l)
}

// IsValid checks if the lifecycle is valid
func (l Lifecycle) IsValid() bool {
	switch l {
	case LifecycleSingleton, LifecycleTransient:
		return true
	}
	return false
}

// ParseLifecycle parses a string into a Lifecycle
func ParseLifecycle(s string) (Lifecycle, error) {
	lifecycle := Lifecycle(s)
	if !lifecycle.IsValid() {
		return "", fmt.Errorf("invalid lifecycle: %s (must be '%s' or '%s')", s, LifecycleSingleton, LifecycleTransient)
	}
	return lifecycle, nil
}

// Middleware target constants
const (
	TargetAll = "all" // Apply to all routes
)

// Project represents the complete scanned project
type Project struct {
	Module      string
	Configs     []*Config
	Controllers []*Controller
	Providers   []*Provider
	Middleware  []*Middleware
	LocaleFiles []*LocaleFile
}

// Controller represents a scanned controller
type Controller struct {
	Name        string
	PackageName string
	PackagePath string
	FilePath    string
	SourceLine  int
	RoutePrefix string
	Tags        []string
	Handlers    []*Handler
	Fields      []*Field
}

// Handler represents a controller method annotated with @Route
type Handler struct {
	Name       string
	Method     HTTPMethod
	Path       string
	FullPath   string
	SourceLine int
	Tags       []string
	With       []string          // e.g., ["auth", "cache"] or ["none"] - explicit middleware override
	Signature  *HandlerSignature // Parsed signature
}

// HandlerSignature represents a parsed handler signature
type HandlerSignature struct {
	Pattern               HandlerPattern
	Receiver              *Field
	PathParams            []*PathParam
	QueryParams           []*QueryParam
	HeaderParams          []*HeaderParam
	ParamsStructType      *TypeInfo
	RequestType           *TypeInfo
	ResponseMetadata      *ResponseMetadata // Metadata from response struct tags
	ReturnsError          bool
	NeedsValidation       bool
	NeedsParamsValidation bool
}

// ResponseMetadata represents metadata extracted from response struct tags
type ResponseMetadata struct {
	HeaderFields    []*HeaderField
	StatusCodeField string // Field name with response:"httpstatus" tag
}

// HeaderField represents a response field with header tag
type HeaderField struct {
	FieldName  string
	HeaderName string
	Type       *TypeInfo
	OmitEmpty  bool
}

// PathParam represents a path parameter in handler signature
type PathParam struct {
	Name     string
	Type     *TypeInfo
	Position int
}

// QueryParam represents a query parameter from struct tag
type QueryParam struct {
	FieldName  string
	ParamName  string
	Type       *TypeInfo
	IsOptional bool
}

// HeaderParam represents a header parameter from struct tag
type HeaderParam struct {
	FieldName  string
	HeaderName string
	Type       *TypeInfo
	IsOptional bool
}

// Provider represents a function annotated with @Provider
type Provider struct {
	Name         string
	FunctionName string
	PackageName  string
	PackagePath  string
	FilePath     string
	SourceLine   int
	Lifecycle    Lifecycle
	ReturnType   *TypeInfo
	Dependencies []*Field
	ReturnsError bool
}

// Middleware represents a function annotated with @Middleware
type Middleware struct {
	Name         string
	FunctionName string
	PackageName  string
	PackagePath  string
	FilePath     string
	SourceLine   int
	Target       string // e.g., "all", "protected", "public,admin" - targeting expression
	Order        int    // Execution order (default: 100)
	Signature    MiddlewareSignature
	Dependencies []*Field
}

// Config represents a scanned configuration struct (annotated with @Config)
type Config struct {
	Name        string // e.g., "Config", "AppConfig", "DatabaseConfig"
	PackageName string
	PackagePath string
	FilePath    string
	SourceLine  int
	Fields      []*ConfigField // Top-level config fields
}

// ConfigField represents a field in the Config struct
type ConfigField struct {
	Name         string
	Type         *TypeInfo
	EnvName      string
	DefaultValue string
	Required     bool
	IsNested     bool
	Fields       []*ConfigField
	SourceLine   int
}

// Field represents a struct field (for DI) or function parameter
type Field struct {
	Name string    // e.g., "DB"
	Type *TypeInfo // e.g., *gorm.DB
}

// TypeInfo represents a Go type with package information
type TypeInfo struct {
	Name        string
	PackagePath string
	PackageName string
	IsPointer   bool
	IsSlice     bool
	IsError     bool
	IsContext   bool
	IsPrimitive bool
	IsGeneric   bool
	TypeParams  []*TypeInfo
	FullName    string
}

// Annotation represents a parsed annotation from comments
type Annotation struct {
	Type  AnnotationType    // e.g., "Controller", "Route", "Provider", "Middleware"
	Value string            // Primary value (e.g., "/api/v1/posts", "GET /", "singleton")
	Args  map[string]string // Additional arguments (future use)
	Line  int
}
