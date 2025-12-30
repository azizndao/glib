# 03. Code Generation System

**Status:** Specification v1.0  
**Last Updated:** 2025-12-30

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Scanner Phase](#scanner-phase)
4. [Validation Phase](#validation-phase)
5. [Generation Phase](#generation-phase)
6. [Generated Code Structure](#generated-code-structure)
7. [Template System](#template-system)
8. [DI Graph Resolution](#di-graph-resolution)
9. [Request Parser Generation](#request-parser-generation)
10. [Route Registration Generation](#route-registration-generation)
11. [Hot Reload Integration](#hot-reload-integration)

---

## Overview

### What Gets Generated

Glib 2.0 generates a **single cohesive bootstrapping package** that wires your entire application together:

```
generated/
├── glib.gen.go              # Main bootstrap file
├── di.gen.go                # Dependency injection container
├── routes.gen.go            # Route registration
├── parsers.gen.go           # Request parsers
└── middleware.gen.go        # Middleware chain builders
```

### When Generation Happens

```bash
# Explicit generation
glib generate

# Automatic during development (Air integration)
glib dev  # Watches files → generates on change → rebuilds

# Manual integration with go generate
//go:generate glib generate
```

### Generation Flow

```
Source Code
    ↓
[1. SCAN]     ← Parse Go AST, extract annotations
    ↓
[2. VALIDATE] ← Check DI graph, routes, types
    ↓
[3. GENERATE] ← Execute templates, write files
    ↓
Generated Code
```

---

## Architecture

### Three-Phase System

```go
// Internal scanner architecture
type CodeGenerator struct {
    scanner   *Scanner      // Phase 1: AST parsing
    validator *Validator    // Phase 2: Validation
    generator *Generator    // Phase 3: Code generation
}

func (cg *CodeGenerator) Generate() error {
    // Phase 1: Scan source files
    model, err := cg.scanner.Scan()
    if err != nil {
        return fmt.Errorf("scan failed: %w", err)
    }
    
    // Phase 2: Validate the model
    if err := cg.validator.Validate(model); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // Phase 3: Generate code
    if err := cg.generator.Generate(model); err != nil {
        return fmt.Errorf("generation failed: %w", err)
    }
    
    return nil
}
```

### Internal Model (IR)

The scanner produces an **Intermediate Representation (IR)** that validation and generation use:

```go
// Internal representation of the entire application
type ApplicationModel struct {
    Module       string                    // Go module path
    Config       *ConfigDeclaration        // type Config struct {...}
    Controllers  []*ControllerDeclaration  // @Controller
    Providers    []*ProviderDeclaration    // @Provider
    Middleware   []*MiddlewareDeclaration  // @Middleware
}

type ControllerDeclaration struct {
    Name         string              // Type name: "PostsController"
    Package      string              // Import path: "github.com/user/app/posts"
    PathPrefix   string              // From annotation: "/api/v1/posts"
    Middleware   []string            // Names: ["auth", "ratelimit"]
    Fields       []*FieldDeclaration // Struct fields (for DI)
    Handlers     []*HandlerDeclaration
}

type HandlerDeclaration struct {
    Method       string              // HTTP method: "GET"
    Path         string              // Route path: "/{id}"
    FuncName     string              // Go function name: "Show"
    Middleware   []string            // Handler-level middleware
    Signature    *SignatureInfo      // Parsed signature
}

type SignatureInfo struct {
    Pattern      HandlerPattern      // One of 9 patterns
    Params       []*ParamInfo        // Path parameters
    RequestType  *TypeInfo           // Request struct type
    ResponseType *TypeInfo           // Response type
    HasContext   bool
    HasWriter    bool
    HasRequest   bool
    ReturnsError bool
}

type ProviderDeclaration struct {
    FuncName     string              // Function name: "NewDatabase"
    Package      string              // Import path
    Lifecycle    string              // "singleton" | "transient"
    Provides     *TypeInfo           // Return type (what it provides)
    Dependencies []*TypeInfo         // Parameter types (what it needs)
}

type MiddlewareDeclaration struct {
    Name         string              // Middleware name: "auth"
    FuncName     string              // Function name: "AuthMiddleware"
    Package      string              // Import path
}
```

---

## Scanner Phase

### Step 1: Discover Go Files

```go
// Walk project directory, find all .go files
func (s *Scanner) DiscoverFiles(root string) ([]string, error) {
    var files []string
    
    err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        // Skip vendor, node_modules, generated/, etc.
        if shouldSkip(d.Name()) {
            return filepath.SkipDir
        }
        
        if !d.IsDir() && strings.HasSuffix(path, ".go") {
            // Skip test files and generated files
            if !strings.HasSuffix(path, "_test.go") && 
               !strings.HasSuffix(path, ".gen.go") {
                files = append(files, path)
            }
        }
        
        return nil
    })
    
    return files, err
}
```

### Step 2: Parse AST

```go
// Parse each file into Go AST
func (s *Scanner) ParseFile(path string) (*ast.File, error) {
    fset := token.NewFileSet()
    
    file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
    if err != nil {
        return nil, fmt.Errorf("parse %s: %w", path, err)
    }
    
    return file, nil
}
```

### Step 3: Extract Annotations

```go
// Extract annotations from doc comments
func (s *Scanner) ExtractAnnotations(doc *ast.CommentGroup) map[string]string {
    if doc == nil {
        return nil
    }
    
    annotations := make(map[string]string)
    
    for _, comment := range doc.List {
        text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
        
        // Match annotation pattern: @Name value
        if match := annotationRegex.FindStringSubmatch(text); match != nil {
            name := match[1]   // @Controller
            value := match[2]  // /api/v1/posts
            annotations[name] = value
        }
    }
    
    return annotations
}

var annotationRegex = regexp.MustCompile(`^@(\w+)\s*(.*)$`)
```

### Step 4: Scan Controllers

```go
// Find all @Controller declarations
func (s *Scanner) ScanControllers(file *ast.File) []*ControllerDeclaration {
    var controllers []*ControllerDeclaration
    
    ast.Inspect(file, func(n ast.Node) bool {
        // Look for type declarations
        typeSpec, ok := n.(*ast.TypeSpec)
        if !ok {
            return true
        }
        
        // Check if it's a struct
        structType, ok := typeSpec.Type.(*ast.StructType)
        if !ok {
            return true
        }
        
        // Extract annotations from doc comments
        annotations := s.ExtractAnnotations(typeSpec.Doc)
        
        // Check for @Controller annotation
        pathPrefix, hasController := annotations["Controller"]
        if !hasController {
            return true
        }
        
        // Build controller declaration
        ctrl := &ControllerDeclaration{
            Name:       typeSpec.Name.Name,
            Package:    s.currentPackage,
            PathPrefix: pathPrefix,
            Middleware: parseMiddlewareList(annotations["Middleware"]),
            Fields:     s.scanFields(structType),
        }
        
        controllers = append(controllers, ctrl)
        return true
    })
    
    return controllers
}
```

### Step 5: Scan Handlers

```go
// Find all @Route declarations (methods on controllers)
func (s *Scanner) ScanHandlers(file *ast.File, ctrl *ControllerDeclaration) {
    ast.Inspect(file, func(n ast.Node) bool {
        // Look for function declarations
        funcDecl, ok := n.(*ast.FuncDecl)
        if !ok {
            return true
        }
        
        // Check if it's a method on this controller
        if funcDecl.Recv == nil || !s.isReceiverType(funcDecl.Recv, ctrl.Name) {
            return true
        }
        
        // Extract annotations
        annotations := s.ExtractAnnotations(funcDecl.Doc)
        
        // Check for @Route annotation
        routeValue, hasRoute := annotations["Route"]
        if !hasRoute {
            return true
        }
        
        // Parse route: "GET /{id}"
        method, path := s.parseRoute(routeValue)
        
        // Analyze function signature
        signature := s.analyzeSignature(funcDecl.Type)
        
        handler := &HandlerDeclaration{
            Method:     method,
            Path:       path,
            FuncName:   funcDecl.Name.Name,
            Middleware: parseMiddlewareList(annotations["Middleware"]),
            Signature:  signature,
        }
        
        ctrl.Handlers = append(ctrl.Handlers, handler)
        return true
    })
}
```

### Step 6: Analyze Signatures

```go
// Analyze handler function signature to determine pattern
func (s *Scanner) analyzeSignature(funcType *ast.FuncType) *SignatureInfo {
    sig := &SignatureInfo{}
    
    // Analyze parameters
    for _, param := range funcType.Params.List {
        paramType := s.typeToString(param.Type)
        
        switch paramType {
        case "context.Context":
            sig.HasContext = true
        case "http.ResponseWriter":
            sig.HasWriter = true
        case "*http.Request":
            sig.HasRequest = true
        default:
            // Could be path param or request struct
            if s.isPathParam(param) {
                sig.Params = append(sig.Params, s.parsePathParam(param))
            } else {
                sig.RequestType = s.parseType(param.Type)
            }
        }
    }
    
    // Analyze return values
    if funcType.Results != nil {
        for _, result := range funcType.Results.List {
            resultType := s.typeToString(result.Type)
            
            if resultType == "error" {
                sig.ReturnsError = true
            } else {
                sig.ResponseType = s.parseType(result.Type)
            }
        }
    }
    
    // Determine pattern (1-9)
    sig.Pattern = s.determinePattern(sig)
    
    return sig
}
```

### Step 7: Scan Providers

```go
// Find all @Provider declarations
func (s *Scanner) ScanProviders(file *ast.File) []*ProviderDeclaration {
    var providers []*ProviderDeclaration
    
    ast.Inspect(file, func(n ast.Node) bool {
        // Look for function declarations
        funcDecl, ok := n.(*ast.FuncDecl)
        if !ok {
            return true
        }
        
        // Extract annotations
        annotations := s.ExtractAnnotations(funcDecl.Doc)
        
        // Check for @Provider annotation
        lifecycle, hasProvider := annotations["Provider"]
        if !hasProvider {
            return true
        }
        
        // Default to singleton if not specified
        if lifecycle == "" {
            lifecycle = "singleton"
        }
        
        // Analyze function signature
        provides := s.getProviderReturnType(funcDecl.Type)
        deps := s.getProviderDependencies(funcDecl.Type)
        
        provider := &ProviderDeclaration{
            FuncName:     funcDecl.Name.Name,
            Package:      s.currentPackage,
            Lifecycle:    lifecycle,
            Provides:     provides,
            Dependencies: deps,
        }
        
        providers = append(providers, provider)
        return true
    })
    
    return providers
}
```

### Step 8: Scan Config

```go
// Find type Config struct declaration
func (s *Scanner) ScanConfig(file *ast.File) *ConfigDeclaration {
    var config *ConfigDeclaration
    
    ast.Inspect(file, func(n ast.Node) bool {
        typeSpec, ok := n.(*ast.TypeSpec)
        if !ok {
            return true
        }
        
        // Look for "Config" type
        if typeSpec.Name.Name != "Config" {
            return true
        }
        
        structType, ok := typeSpec.Type.(*ast.StructType)
        if !ok {
            return true
        }
        
        config = &ConfigDeclaration{
            Name:    "Config",
            Package: s.currentPackage,
            Fields:  s.scanConfigFields(structType),
        }
        
        return false // Found it, stop searching
    })
    
    return config
}
```

---

## Validation Phase

### DI Graph Validation

```go
// Ensure all dependencies can be satisfied
func (v *Validator) ValidateDI(model *ApplicationModel) error {
    // Build dependency graph
    graph := NewDependencyGraph()
    
    // Add all providers to graph
    for _, provider := range model.Providers {
        graph.AddProvider(provider)
    }
    
    // Add Config as a provider (always available)
    graph.AddConfig(model.Config)
    
    // Check each controller's dependencies
    for _, ctrl := range model.Controllers {
        for _, field := range ctrl.Fields {
            if !graph.CanProvide(field.Type) {
                return fmt.Errorf(
                    "controller %s.%s: no provider for type %s",
                    ctrl.Name, field.Name, field.Type,
                )
            }
        }
    }
    
    // Check for circular dependencies
    if cycles := graph.FindCycles(); len(cycles) > 0 {
        return fmt.Errorf("circular dependencies detected: %v", cycles)
    }
    
    return nil
}
```

### Route Validation

```go
// Ensure routes don't conflict
func (v *Validator) ValidateRoutes(model *ApplicationModel) error {
    routes := make(map[string]string) // "GET /posts/{id}" -> controller
    
    for _, ctrl := range model.Controllers {
        for _, handler := range ctrl.Handlers {
            // Build full route path
            fullPath := ctrl.PathPrefix + handler.Path
            key := handler.Method + " " + fullPath
            
            // Check for conflicts
            if existing, exists := routes[key]; exists {
                return fmt.Errorf(
                    "route conflict: %s defined in both %s and %s",
                    key, existing, ctrl.Name,
                )
            }
            
            routes[key] = ctrl.Name
        }
    }
    
    return nil
}
```

### Type Validation

```go
// Ensure request/response types are valid
func (v *Validator) ValidateTypes(model *ApplicationModel) error {
    for _, ctrl := range model.Controllers {
        for _, handler := range ctrl.Handlers {
            sig := handler.Signature
            
            // Validate request type (if present)
            if sig.RequestType != nil {
                if err := v.validateRequestType(sig.RequestType); err != nil {
                    return fmt.Errorf(
                        "%s.%s: %w",
                        ctrl.Name, handler.FuncName, err,
                    )
                }
            }
            
            // Validate path params match signature
            pathParams := extractPathParams(handler.Path)
            if len(pathParams) != len(sig.Params) {
                return fmt.Errorf(
                    "%s.%s: path has %d params but signature has %d",
                    ctrl.Name, handler.FuncName,
                    len(pathParams), len(sig.Params),
                )
            }
        }
    }
    
    return nil
}
```

---

## Generation Phase

### Main Bootstrap File

**Template: `templates/glib.gen.go.tmpl`**

```go
// Code generated by glib. DO NOT EDIT.
package generated

import (
    "context"
    "net/http"
    
    {{- range .Imports }}
    {{ .Alias }} "{{ .Path }}"
    {{- end }}
)

func Bootstrap(ctx context.Context) (http.Handler, error) {
    cfg, err := loadConfig()
    if err != nil {
        return nil, fmt.Errorf("load config: %w", err)
    }
    
    container, err := buildContainer(ctx, cfg)
    if err != nil {
        return nil, fmt.Errorf("build container: %w", err)
    }
    
    mux := http.NewServeMux()
    if err := registerRoutes(mux, container); err != nil {
        return nil, fmt.Errorf("register routes: %w", err)
    }
    
    return mux, nil
}
```

### DI Container Generation

**Template: `templates/di.gen.go.tmpl`**

```go
// Code generated by glib. DO NOT EDIT.
package generated

type Container struct {
    {{- range .Providers }}
    {{ .FieldName }} {{ .Type }}
    {{- end }}
    
    {{- range .Controllers }}
    {{ .FieldName }} *{{ .Package }}.{{ .Name }}
    {{- end }}
}

func buildContainer(ctx context.Context, cfg *config.Config) (*Container, error) {
    c := &Container{}
    
    {{- range .Providers }}
    {
        {{- if .Dependencies }}
        val, err := {{ .Package }}.{{ .FuncName }}(
            {{- range .Dependencies }}
            c.{{ .FieldName }},
            {{- end }}
        )
        {{- else }}
        val, err := {{ .Package }}.{{ .FuncName }}()
        {{- end }}
        if err != nil {
            return nil, fmt.Errorf("provider {{ .FuncName }}: %w", err)
        }
        c.{{ .FieldName }} = val
    }
    {{- end }}
    
    {{- range .Controllers }}
    c.{{ .FieldName }} = &{{ .Package }}.{{ .Name }}{
        {{- range .Fields }}
        {{ .Name }}: c.{{ .DependencyFieldName }},
        {{- end }}
    }
    {{- end }}
    
    return c, nil
}
```

### Route Registration Generation

**Template: `templates/routes.gen.go.tmpl`**

```go
// Code generated by glib. DO NOT EDIT.
package generated

func registerRoutes(mux *http.ServeMux, container *Container) error {
    {{- range .Controllers }}
    {{- range .Handlers }}
    {
        handler := wrap{{ $.Name }}_{{ .FuncName }}(container.{{ $.FieldName }})
        {{- range .Middleware }}
        handler = {{ .Package }}.{{ .FuncName }}()(handler)
        {{- end }}
        mux.Handle("{{ .Method }} {{ .FullPath }}", handler)
    }
    {{- end }}
    {{- end }}
    
    return nil
}
```

### Request Parser Generation

**Template: `templates/parsers.gen.go.tmpl`**

```go
// Code generated by glib. DO NOT EDIT.
package generated

{{- range .Handlers }}
{{- if .NeedsParser }}

func parse{{ .ControllerName }}_{{ .FuncName }}(r *http.Request) ({{ .RequestType }}, error) {
    var req {{ .RequestType }}
    
    {{- if .HasPathParams }}
    {{- range .PathParams }}
    {
        val := r.PathValue("{{ .Name }}")
        parsed, err := parse{{ .Type }}(val)
        if err != nil {
            return req, fmt.Errorf("path param {{ .Name }}: %w", err)
        }
        req.{{ .FieldName }} = parsed
    }
    {{- end }}
    {{- end }}
    
    {{- if .HasQueryParams }}
    query := r.URL.Query()
    {{- range .QueryParams }}
    if val := query.Get("{{ .QueryName }}"); val != "" {
        parsed, err := parse{{ .Type }}(val)
        if err != nil {
            return req, fmt.Errorf("query param {{ .QueryName }}: %w", err)
        }
        req.{{ .FieldName }} = parsed
    } {{- if .HasDefault }} else {
        req.{{ .FieldName }} = {{ .DefaultValue }}
    }
    {{- end }}
    {{- end }}
    {{- end }}
    
    {{- if .HasHeaders }}
    {{- range .Headers }}
    if val := r.Header.Get("{{ .HeaderName }}"); val != "" {
        req.{{ .FieldName }} = val
    }
    {{- end }}
    {{- end }}
    
    {{- if .HasBody }}
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        return req, fmt.Errorf("decode body: %w", err)
    }
    {{- end }}
    
    {{- if .HasValidation }}
    if err := validate.Struct(&req); err != nil {
        return req, fmt.Errorf("validation: %w", err)
    }
    {{- end }}
    
    return req, nil
}
{{- end }}
{{- end }}
```

### Handler Wrapper Generation

**Template: `templates/wrappers.gen.go.tmpl`**

```go
// Code generated by glib. DO NOT EDIT.
package generated

{{- range .Handlers }}

func wrap{{ .ControllerName }}_{{ .FuncName }}(ctrl *{{ .ControllerPackage }}.{{ .ControllerName }}) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        {{- if eq .Pattern "raw" }}
        ctrl.{{ .FuncName }}(w, r)
        
        {{- else if eq .Pattern "context_only" }}
        if err := ctrl.{{ .FuncName }}(r.Context()); err != nil {
            handleError(w, err)
            return
        }
        w.WriteHeader(http.StatusNoContent)
        
        {{- else if eq .Pattern "context_response" }}
        resp, err := ctrl.{{ .FuncName }}(r.Context())
        if err != nil {
            handleError(w, err)
            return
        }
        writeJSON(w, http.StatusOK, resp)
        
        {{- else if eq .Pattern "context_request_response" }}
        req, err := parse{{ .ControllerName }}_{{ .FuncName }}(r)
        if err != nil {
            handleError(w, err)
            return
        }
        
        resp, err := ctrl.{{ .FuncName }}(r.Context(), req)
        if err != nil {
            handleError(w, err)
            return
        }
        
        writeJSON(w, http.StatusOK, resp)
        
        {{- else if eq .Pattern "params_request_response" }}
        {{- range .PathParams }}
        {{ .Name }}, err := parse{{ .Type }}(r.PathValue("{{ .Name }}"))
        if err != nil {
            handleError(w, fmt.Errorf("path param {{ .Name }}: %w", err))
            return
        }
        {{- end }}
        
        req, err := parse{{ .ControllerName }}_{{ .FuncName }}(r)
        if err != nil {
            handleError(w, err)
            return
        }
        
        resp, err := ctrl.{{ .FuncName }}(r.Context(), {{ range .PathParams }}{{ .Name }}, {{ end }}req)
        if err != nil {
            handleError(w, err)
            return
        }
        
        writeJSON(w, http.StatusOK, resp)
        {{- end }}
    })
}
{{- end }}
```

---

## DI Graph Resolution

### Topological Sort Algorithm

```go
// Sort providers in dependency order
func (g *DependencyGraph) TopologicalSort() ([]*ProviderDeclaration, error) {
    var sorted []*ProviderDeclaration
    visited := make(map[string]bool)
    visiting := make(map[string]bool)
    
    var visit func(string) error
    visit = func(name string) error {
        if visited[name] {
            return nil
        }
        if visiting[name] {
            return fmt.Errorf("circular dependency: %s", name)
        }
        
        visiting[name] = true
        
        provider := g.providers[name]
        for _, dep := range provider.Dependencies {
            if err := visit(dep.TypeName); err != nil {
                return err
            }
        }
        
        visiting[name] = false
        visited[name] = true
        sorted = append(sorted, provider)
        
        return nil
    }
    
    for name := range g.providers {
        if err := visit(name); err != nil {
            return nil, err
        }
    }
    
    return sorted, nil
}
```

---

## Request Parser Generation

### Field Tag Analysis

```go
// Analyze struct tags to determine parsing strategy
func (g *Generator) analyzeRequestType(typeInfo *TypeInfo) *ParserInfo {
    parser := &ParserInfo{
        TypeName: typeInfo.Name,
    }
    
    for _, field := range typeInfo.Fields {
        // Check for param tag
        if tag := field.Tag.Get("param"); tag != "" {
            parser.PathParams = append(parser.PathParams, &ParamInfo{
                Name:      tag,
                FieldName: field.Name,
                Type:      field.Type,
            })
        }
        
        // Check for query tag
        if tag := field.Tag.Get("query"); tag != "" {
            param := &ParamInfo{
                Name:      tag,
                FieldName: field.Name,
                Type:      field.Type,
            }
            
            // Check for default value
            if defaultVal := field.Tag.Get("default"); defaultVal != "" {
                param.DefaultValue = defaultVal
            }
            
            parser.QueryParams = append(parser.QueryParams, param)
        }
        
        // Check for header tag
        if tag := field.Tag.Get("header"); tag != "" {
            parser.Headers = append(parser.Headers, &ParamInfo{
                Name:      tag,
                FieldName: field.Name,
                Type:      field.Type,
            })
        }
        
        // No special tag = JSON body field
        if field.Tag.Get("param") == "" && 
           field.Tag.Get("query") == "" && 
           field.Tag.Get("header") == "" {
            parser.HasBody = true
        }
        
        // Check for validation tags
        if field.Tag.Get("validate") != "" {
            parser.HasValidation = true
        }
    }
    
    return parser
}
```

---

## Hot Reload Integration

### Air Configuration

When user runs `glib dev`, we generate `.air.toml`:

```toml
# Auto-generated by glib dev
root = "."
tmp_dir = "tmp"

[build]
  # Run glib generate before building
  pre_cmd = ["glib generate"]
  
  cmd = "go build -o ./tmp/main ."
  bin = "tmp/main"
  
  # Watch Go files
  include_ext = ["go"]
  exclude_dir = ["tmp", "vendor", "node_modules"]
  
  # Trigger rebuild on generation
  include_file = ["generated/glib.gen.go"]
  
  delay = 1000

[log]
  time = true

[color]
  main = "magenta"
  watcher = "cyan"
  build = "yellow"
  runner = "green"
```

### Watch Flow

```
File Change (*.go)
    ↓
Air detects change
    ↓
Run pre_cmd: glib generate
    ↓
Generate new code
    ↓
Air rebuilds binary
    ↓
Air restarts process
```

---

## Complete Generation Example

### Input Source

```go
// posts/controller.go
package posts

import (
    "context"
    "github.com/google/uuid"
)

// @Controller /api/v1/posts
// @Middleware auth
type PostsController struct {
    DB    *gorm.DB
    Cache *redis.Client
}

// @Route GET /{id}
// @Middleware cache
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) (*Post, error) {
    // Flexible handler signature - 9 patterns supported
}
    return &post, nil
}

// @Route POST /
func (c *PostsController) Create(ctx context.Context, req CreatePostRequest) (*Post, error) {
    post := &Post{
        Title:   req.Title,
        Content: req.Content,
    }
    if err := c.DB.Create(post).Error; err != nil {
        return nil, err
    }
    return post, nil
}
```

### Generated Output

**`generated/glib.gen.go`**

```go
// Code generated by glib. DO NOT EDIT.
package generated

func Bootstrap(ctx context.Context) (http.Handler, error) {
    cfg, err := loadConfig()
    if err != nil {
        return nil, err
    }
    
    container, err := buildContainer(ctx, cfg)
    if err != nil {
        return nil, err
    }
    
    mux := http.NewServeMux()
    if err := registerRoutes(mux, container); err != nil {
        return nil, err
    }
    
    return mux, nil
}
```

**`generated/di.gen.go`**

```go
// Code generated by glib. DO NOT EDIT.
package generated

type Container struct {
    db         *gorm.DB
    cache      *redis.Client
    postsCtrl  *posts.PostsController
}

func buildContainer(ctx context.Context, cfg *config.Config) (*Container, error) {
    c := &Container{}
    
    db, err := providers.NewDatabase(cfg)
    if err != nil {
        return nil, fmt.Errorf("provider NewDatabase: %w", err)
    }
    c.db = db
    
    cache, err := providers.NewRedis(cfg)
    if err != nil {
        return nil, fmt.Errorf("provider NewRedis: %w", err)
    }
    c.cache = cache
    
    c.postsCtrl = &posts.PostsController{
        DB:    c.db,
        Cache: c.cache,
    }
    
    return c, nil
}
```

**`generated/routes.gen.go`**

```go
// Code generated by glib. DO NOT EDIT.
package generated

func registerRoutes(mux *http.ServeMux, container *Container) error {
    {
        handler := wrapPostsController_Show(container.postsCtrl)
        handler = middleware.Auth()(handler)
        mux.Handle("GET /api/v1/posts/{id}", handler)
    }
    
    {
        handler := wrapPostsController_Create(container.postsCtrl)
        handler = middleware.Auth()(handler)
        mux.Handle("POST /api/v1/posts", handler)
    }
    
    return nil
}

func wrapPostsController_Show(ctrl *posts.PostsController) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        idStr := r.PathValue("id")
        id, err := uuid.Parse(idStr)
        if err != nil {
            handleError(w, fmt.Errorf("invalid uuid: %w", err))
            return
        }
        
        resp, err := ctrl.Show(r.Context(), id)
        if err != nil {
            handleError(w, err)
            return
        }
        
        writeJSON(w, http.StatusOK, resp)
    })
}

func wrapPostsController_Create(ctrl *posts.PostsController) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req posts.CreatePostRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            handleError(w, fmt.Errorf("decode body: %w", err))
            return
        }
        
        if err := validate.Struct(&req); err != nil {
            handleError(w, fmt.Errorf("validation: %w", err))
            return
        }
        
        resp, err := ctrl.Create(r.Context(), req)
        if err != nil {
            handleError(w, err)
            return
        }
        
        writeJSON(w, http.StatusCreated, resp)
    })
}
```

---

## Summary

### What Gets Generated

1. **Bootstrap code** - Initializes everything
2. **DI container** - Dependency injection wiring
3. **Route registration** - Maps HTTP routes to handlers
4. **Request parsers** - Parses path/query/header/body
5. **Handler wrappers** - Adapts handlers to `http.Handler`
6. **Middleware chains** - Applies middleware in order

### Key Principles

- **Single source of truth** - User code drives generation
- **Type-safe** - All types checked at compile time
- **Zero reflection** - Generated code = hand-written code
- **Debuggable** - Generated code is readable
- **Incremental** - Only regenerate on changes
- **Fast** - Sub-second generation for most projects

### Performance Characteristics

- **Scan:** ~100ms for 1000 files
- **Validate:** ~50ms for complex DI graphs
- **Generate:** ~200ms for 100 controllers
- **Total:** < 500ms for typical projects

---

**Next Steps:**
- Define CLI commands (`04-CLI.md`)
- Write complete examples (`05-EXAMPLES.md`)
- Document migration path (`06-MIGRATION.md`)
- Create implementation roadmap (`07-IMPLEMENTATION.md`)
