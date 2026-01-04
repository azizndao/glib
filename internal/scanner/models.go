package scanner

// Handler pattern types
const (
	PatternDataError = "data_error" // (T, error) - Returns data and error
	PatternErrorOnly = "error_only" // error - Returns error only (no content)
	PatternRawHTTP   = "raw_http"   // Raw HTTP - Direct http.ResponseWriter and *http.Request handlers
)

// Middleware signature types
const (
	MiddlewareSignatureStandard = "standard" // func(http.Handler) http.Handler
	MiddlewareSignatureGlib     = "glib"     // func(glib.Request, glib.Next) glib.Response
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
	Method     string
	Path       string
	FullPath   string
	SourceLine int
	Tags       []string
	With       []string          // e.g., ["auth", "cache"] or ["none"] - explicit middleware override
	Signature  *HandlerSignature // Parsed signature
}

// HandlerSignature represents a parsed handler signature
type HandlerSignature struct {
	Pattern               string
	Receiver              *Field
	PathParams            []*PathParam
	QueryParams           []*QueryParam
	HeaderParams          []*HeaderParam
	ParamsStructType      *TypeInfo
	RequestType           *TypeInfo
	ResponseType          *TypeInfo
	ResponseMetadata      *ResponseMetadata // Metadata from response struct tags
	ReturnsError          bool
	HasContext            bool
	HasRawHTTP            bool
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
	Lifecycle    string // "singleton" or "transient"
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
	Signature    string // "standard" or "glib" - see MiddlewareSignature constants
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
	Type  string            // e.g., "Controller", "Route", "Provider", "Middleware"
	Value string            // Primary value (e.g., "/api/v1/posts", "GET /", "singleton")
	Args  map[string]string // Additional arguments (future use)
	Line  int
}
