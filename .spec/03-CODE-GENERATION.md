# 03. Code Generation System

**Status:** Specification (Under Development)  
**Last Updated:** 2025-12-31

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Scanner Phase](#scanner-phase)
4. [Import Resolution](#import-resolution)
5. [Validation Phase](#validation-phase)
6. [Generation Phase](#generation-phase)
7. [Generated Code Structure](#generated-code-structure)
8. [Template System](#template-system)
9. [DI Container with Topological Sort](#di-container-with-topological-sort)
10. [Request Parser Generation](#request-parser-generation)
11. [Route Registration Generation](#route-registration-generation)

---

## Overview

### What Gets Generated

Glib generates **4 files** that wire your entire application together:

```
generated/
├── glib.gen.go      # Main bootstrap function
├── di.gen.go        # DI container with topological sorting
├── routes.gen.go    # Route registration
└── parsers.gen.go   # Handler wrappers (Result[T] and Raw HTTP)
```

**Note:** Error handling is handled by the `Result[T].Write()` method in the framework package (`writer.go`), eliminating the need for a generated `errors.gen.go` file.

### When Generation Happens

```bash
# Explicit generation
glib generate

# Automatic during development (built-in hot reload)
glib dev  # Watches files → generates on change → rebuilds

# Manual integration with go generate
//go:generate glib generate
```

### Generation Flow

```
Source Code
    ↓
[1. SCAN]     ← Parse Go AST, extract annotations, resolve imports
    ↓
[2. VALIDATE] ← Check DI graph, routes, types
    ↓
[3. GENERATE] ← Execute templates, track imports, write files
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
    Tags         []string            // From annotation: ["api", "protected"]
    Fields       []*FieldDeclaration // Struct fields (for DI)
    Handlers     []*HandlerDeclaration
}

type HandlerDeclaration struct {
    Method       string              // HTTP method: "GET"
    Path         string              // Route path: "/{id}"
    FuncName     string              // Go function name: "Show"
    Tags         []string            // From annotation: ["protected"]
    With         []string            // Middleware names: ["auth", "ratelimit"]
    Signature    *SignatureInfo      // Parsed signature
}

type SignatureInfo struct {
    Pattern      string              // "result" or "raw_http"
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
    Target       string              // Target tag: "all", "api", "protected"
    Order        int                 // Execution order: 1, 10, 20
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
        annotation, hasController := annotations["Controller"]
        if !hasController {
            return true
        }

        // Parse annotation: path=/api/v1/posts tags=api,protected
        attrs := parseAnnotation(annotation)

        // Build controller declaration
        ctrl := &ControllerDeclaration{
            Name:       typeSpec.Name.Name,
            Package:    s.currentPackage,
            PathPrefix: attrs["path"],
            Tags:       parseTagList(attrs["tags"]),
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
        annotation, hasRoute := annotations["Route"]
        if !hasRoute {
            return true
        }

        // Parse annotation: method=GET path=/{id} tags=protected with=auth,cache
        attrs := parseAnnotation(annotation)

        // Analyze function signature
        signature := s.analyzeSignature(funcDecl.Type)

        handler := &HandlerDeclaration{
            Method:     attrs["method"],
            Path:       attrs["path"],
            FuncName:   funcDecl.Name.Name,
            Tags:       parseTagList(attrs["tags"]),
            With:       parseTagList(attrs["with"]),
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
            } else if strings.HasPrefix(resultType, "glib.Result[") {
                sig.ResponseType = s.parseType(result.Type)
                sig.Pattern = "result"
            } else {
                sig.ResponseType = s.parseType(result.Type)
            }
        }
    }

    // Determine pattern
    if sig.HasWriter && sig.HasRequest {
        sig.Pattern = "raw_http"
    } else if sig.Pattern == "" {
        sig.Pattern = "result"
    }

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
        annotation, hasProvider := annotations["Provider"]
        if !hasProvider {
            return true
        }

        // Default to singleton if not specified
        lifecycle := annotation
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

## Import Resolution

### Problem

Handlers can return types from other packages (e.g., `models.Post`). The generated code needs to reference these types and import their packages correctly.

**Example:**

```go
// controllers/post/controller.go
package post

import "myapp/models"

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id int) glib.Result[*models.Post] {
    return glib.OK(c.Service.GetPost(id))
}
```

**Generated code must:**

1. Import `myapp/models`
2. Use `*models.Post` correctly in wrapper code

### Solution: parseImports() + trackTypePackage()

#### 1. Scanner: Extract Imports (internal/scanner/controllers.go)

```go
// parseImports extracts import declarations from an AST file
func (s *Scanner) parseImports(file *ast.File) map[string]string {
    imports := make(map[string]string)

    for _, imp := range file.Imports {
        // Remove quotes from import path
        importPath := strings.Trim(imp.Path.Value, `"`)

        // Determine package name
        var pkgName string
        if imp.Name != nil {
            // Aliased import: import foo "github.com/bar"
            pkgName = imp.Name.Name
        } else {
            // Default: use last component of path
            parts := strings.Split(importPath, "/")
            pkgName = parts[len(parts)-1]
        }

        imports[pkgName] = importPath
    }

    return imports
}
```

#### 2. Scanner: Resolve Package Paths (internal/scanner/controllers.go)

```go
// scanHandlers scans all handler methods for a controller
func (s *Scanner) scanHandlers(file *ast.File, ctrl *Controller) error {
    // Parse imports first!
    s.currentImports = s.parseImports(file)

    // Now scan handlers - parseType() can resolve package paths
    for _, decl := range file.Decls {
        funcDecl, ok := decl.(*ast.FuncDecl)
        if !ok {
            continue
        }
        // ... rest of handler scanning
    }

    return nil
}

// parseType analyzes a type expression and returns TypeInfo
func (s *Scanner) parseType(expr ast.Expr) *TypeInfo {
    switch t := expr.(type) {
    case *ast.SelectorExpr:
        // Type from another package: models.Post
        pkgIdent, ok := t.X.(*ast.Ident)
        if !ok {
            return nil
        }

        pkgName := pkgIdent.Name        // "models"
        typeName := t.Sel.Name          // "Post"

        // Look up full import path
        pkgPath := s.currentImports[pkgName]  // "myapp/models"

        return &TypeInfo{
            Name:        typeName,
            PackageName: pkgName,
            PackagePath: pkgPath,  // ← Now set correctly!
            IsPointer:   false,
        }

    // ... other cases
    }
}
```

**Key Fields:**

- `PackageName`: Short name used in code (`models`)
- `PackagePath`: Full import path (`myapp/models`)

#### 3. Generator: Track Required Imports (internal/generator/parsers.go)

```go
// trackTypePackage recursively tracks all packages used by a type
func (g *Generator) trackTypePackage(typeInfo *TypeInfo, imports map[string]string) {
    if typeInfo == nil {
        return
    }

    // Add this type's package
    if typeInfo.PackagePath != "" && typeInfo.PackageName != "" {
        imports[typeInfo.PackageName] = typeInfo.PackagePath
    }

    // Recursively track generic type parameters: Result[*models.Post]
    for _, param := range typeInfo.TypeParams {
        g.trackTypePackage(param, imports)
    }

    // Track map key/value types
    if typeInfo.IsMap {
        g.trackTypePackage(typeInfo.MapKey, imports)
        g.trackTypePackage(typeInfo.MapValue, imports)
    }

    // Track slice element type
    if typeInfo.IsSlice {
        g.trackTypePackage(typeInfo.SliceElem, imports)
    }
}
```

#### 4. Generator: Add Imports to Generated File (internal/generator/parsers.go)

```go
func (g *Generator) GenerateParsers() error {
    imports := make(map[string]string)

    // Always need these
    imports["context"] = "context"
    imports["glib"] = "github.com/azizndao/glib"
    imports["http"] = "net/http"

    // Track imports from all handler response types
    for _, ctrl := range g.model.Controllers {
        for _, handler := range ctrl.Handlers {
            // Track response type packages
            if handler.Signature.ResponseType != nil {
                g.trackTypePackage(handler.Signature.ResponseType, imports)
            }

            // Track request type packages
            if handler.Signature.RequestType != nil {
                g.trackTypePackage(handler.Signature.RequestType, imports)
            }

            // Track path parameter types
            for _, param := range handler.Signature.PathParams {
                g.trackTypePackage(param.Type, imports)
            }
        }
    }

    // Pass imports to template
    data := map[string]any{
        "Imports":     imports,
        "Controllers": g.model.Controllers,
    }

    return g.executeTemplate("parsers.tmpl", data)
}
```

#### 5. Template: Generate Import Statements

```go
// Template: parsers.tmpl
package {{.PackageName}}

import (
{{- range $name, $path := .Imports}}
    {{if ne $name (base $path)}}{{$name}} {{end}}"{{$path}}"
{{- end}}
)

// Now can use models.Post correctly!
func wrapController_Show(...) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        result := ctrl.Show(r.Context(), id)
        writeResult[*models.Post](w, result)  // ← Correct type reference
    })
}
```

### Complete Flow Example

**Input:**

```go
// controllers/post/controller.go
package post

import (
    "context"
    "myapp/models"
    "github.com/azizndao/glib"
)

// @Controller path=/api/v1/posts tags=api
type Controller struct {}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id int) glib.Result[*models.Post] {
    return glib.OK(&models.Post{ID: id})
}
```

**`generated/parsers.gen.go`** - Handler wrappers with optimized patterns

```go
// Code generated by glib. DO NOT EDIT.
package generated

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/azizndao/glib"
    "github.com/google/uuid"

    "myapp/controllers/post"
    "myapp/models"
)

// Result[T] handler - with middleware
func handlePostControllerCreate(container *container) http.HandlerFunc {
    handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // Parse request body
        var req post.CreatePostRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            glib.BadRequest[*models.Post](
                fmt.Sprintf("invalid request body: %v", err)).Write(w)
            return
        }

        result := container.controllers.postController.Create(ctx, req)
        result.Write(w)
    }))

    handler = container.middleware.authMiddleware(handler)
    handler = container.middleware.loggerMiddleware(handler)

    return handler.ServeHTTP
}

// Raw HTTP handler - no middleware (optimized)
func handlePostControllerHealth(container *container) http.HandlerFunc {
    return container.controllers.postController.Health
}

// Raw HTTP handler - with middleware
func handlePostControllerStream(container *container) http.HandlerFunc {
    handler := http.Handler(http.HandlerFunc(container.controllers.postController.Stream))

    handler = container.middleware.ratelimitMiddleware(handler)
    handler = container.middleware.loggerMiddleware(handler)

    return handler.ServeHTTP
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

## DI Container with Topological Sort

### Problem: Dependency Initialization Order

Providers must be initialized in the correct order to avoid nil pointer errors:

```go
// ❌ WRONG ORDER - PostService initialized before UserService
container.postService = services.NewPostService(container.userService) // userService is nil!
container.userService = services.NewUserService()

// ✅ CORRECT ORDER - Dependencies initialized first
container.userService = services.NewUserService()
container.postService = services.NewPostService(container.userService) // userService is ready!
```

### Solution: Topological Sort

The generator uses topological sort to determine the correct initialization order based on the dependency graph.

#### Implementation (internal/generator/di.go)

```go
// sortProvidersByDependencies performs topological sort on providers
func sortProvidersByDependencies(providers []ProviderData) ([]ProviderData, error) {
    // Build dependency graph
    graph := make(map[string][]string)  // provider -> dependencies
    inDegree := make(map[string]int)    // provider -> number of dependencies

    for _, provider := range providers {
        providerKey := provider.FieldName
        inDegree[providerKey] = 0
        graph[providerKey] = []string{}
    }

    // Build edges
    for _, provider := range providers {
        providerKey := provider.FieldName

        for _, dep := range provider.Dependencies {
            // Find which provider provides this dependency
            for _, other := range providers {
                if dep.FullName == other.Type.FullName {
                    graph[other.FieldName] = append(graph[other.FieldName], providerKey)
                    inDegree[providerKey]++
                    break
                }
            }
        }
    }

    // Kahn's algorithm for topological sort
    var queue []string
    for key, degree := range inDegree {
        if degree == 0 {
            queue = append(queue, key)
        }
    }

    var sorted []string
    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]
        sorted = append(sorted, current)

        for _, neighbor := range graph[current] {
            inDegree[neighbor]--
            if inDegree[neighbor] == 0 {
                queue = append(queue, neighbor)
            }
        }
    }

    // Check for cycles
    if len(sorted) != len(providers) {
        return nil, fmt.Errorf("circular dependency detected in providers")
    }

    // Reorder providers
    providerMap := make(map[string]ProviderData)
    for _, provider := range providers {
        providerMap[provider.FieldName] = provider
    }

    result := make([]ProviderData, 0, len(sorted))
    for _, key := range sorted {
        result = append(result, providerMap[key])
    }

    return result, nil
}
```

#### Generated Code Example

**Input:**

```go
// @Provider singleton
func NewUserService() *UserService {
    return &UserService{}
}

// @Provider singleton
func NewPostService(userService *UserService) *PostService {
    return &PostService{UserService: userService}
}
```

**Generated (generated/di.gen.go):**

```go
type container struct {
    userSerivce *services.UserSerivce
    postSerivce *services.PostSerivce
}

func newContainer(ctx context.Context) (*container, error) {
    container := &container{}

    // Topologically sorted - UserService first!
    var err error

    container.userSerivce, err = services.NewUserSerivce()
    if err != nil {
        return nil, fmt.Errorf("failed to create UserSerivce: %w", err)
    }

    container.postSerivce, err = services.NewPostSerivce(container.userSerivce)
    if err != nil {
        return nil, fmt.Errorf("failed to create PostSerivce: %w", err)
    }

    return container, nil
}
```

### Topological Sort Algorithm (Detailed)

```go
// Kahn's algorithm for topological sorting
func TopologicalSort(providers []*Provider) ([]*Provider, error) {
    // Step 1: Build adjacency list and calculate in-degrees
    graph := make(map[string][]string)
    inDegree := make(map[string]int)

    for _, p := range providers {
        inDegree[p.ID] = 0
        graph[p.ID] = []string{}
    }

    for _, p := range providers {
        for _, dep := range p.Dependencies {
            graph[dep.ID] = append(graph[dep.ID], p.ID)
            inDegree[p.ID]++
        }
    }

    // Step 2: Find all nodes with in-degree 0
    var queue []string
    for id, degree := range inDegree {
        if degree == 0 {
            queue = append(queue, id)
        }
    }

    // Step 3: Process queue
    var sorted []string
    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]
        sorted = append(sorted, current)

        // Reduce in-degree for neighbors
        for _, neighbor := range graph[current] {
            inDegree[neighbor]--
            if inDegree[neighbor] == 0 {
                queue = append(queue, neighbor)
            }
        }
    }

    // Step 4: Check for cycles
    if len(sorted) != len(providers) {
        return nil, errors.New("circular dependency detected")
    }

    // Step 5: Return sorted providers
    providerMap := make(map[string]*Provider)
    for _, p := range providers {
        providerMap[p.ID] = p
    }

    result := make([]*Provider, len(sorted))
    for i, id := range sorted {
        result[i] = providerMap[id]
    }

    return result, nil
}
```

### Example: Complex Dependency Graph

**Providers:**

```go
// @Provider singleton
func NewConfig() *Config { return &Config{} }

// @Provider singleton
func NewDatabase(cfg *Config) (*Database, error) {
    return Connect(cfg.DBUrl)
}

// @Provider singleton
func NewCache(cfg *Config) (*Cache, error) {
    return ConnectRedis(cfg.RedisUrl)
}

// @Provider singleton
func NewUserService(db *Database, cache *Cache) *UserService {
    return &UserService{DB: db, Cache: cache}
}

// @Provider singleton
func NewPostService(db *Database, userSvc *UserService) *PostService {
    return &PostService{DB: db, Users: userSvc}
}
```

**Dependency Graph:**

```
Config (no dependencies)
   ↓
   ├─→ Database
   │      ↓
   │      ├─→ UserService
   │      │      ↓
   │      │      └─→ PostService
   │      └─→ PostService
   │
   └─→ Cache
          ↓
          └─→ UserService
```

**Topological Sort Result:**

1. Config
2. Database
3. Cache
4. UserService
5. PostService

**Generated Initialization:**

```go
func newContainer(ctx context.Context) (*container, error) {
    container := &container{}
    var err error

    // 1. Config (no dependencies)
    container.config = NewConfig()

    // 2. Database (needs Config)
    container.database, err = NewDatabase(container.config)
    if err != nil {
        return nil, err
    }

    // 3. Cache (needs Config)
    container.cache, err = NewCache(container.config)
    if err != nil {
        return nil, err
    }

    // 4. UserService (needs Database + Cache)
    container.userService = NewUserService(container.database, container.cache)

    // 5. PostService (needs Database + UserService)
    container.postService = NewPostService(container.database, container.userService)

    return container, nil
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

### Built-in File Watcher

The `glib dev` command includes a built-in file watcher that automatically regenerates code and rebuilds the application on file changes.

**Configuration via CLI flags or environment variables:**

- `--debounce` or `GLIB_WATCH_DEBOUNCE` - Debounce delay in milliseconds (default: 300)
- `--exclude-dirs` or `GLIB_WATCH_EXCLUDE_DIRS` - Comma-separated excluded directories (default: `vendor,node_modules,.git,.glib,tmp`)
- `--include-files` or `GLIB_WATCH_INCLUDE_FILES` - File patterns to watch (default: `*.go`)
- `--exclude-files` or `GLIB_WATCH_EXCLUDE_FILES` - File patterns to ignore (default: `*_test.go,*.gen.go`)

### Watch Flow

```
File Change (*.go)
    ↓
Glib watcher detects change
    ↓
Run: glib generate
    ↓
Generate new code
    ↓
Run: go build
    ↓
Restart process
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

// @Controller path=/api/v1/posts tags=api,protected
// Note: tags are used for middleware targeting
type PostsController struct {
    DB    *gorm.DB
    Cache *redis.Client
}

// @Route method=GET path=/{id} tags=protected with=cache
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) glib.Result[*Post] {
    // Handler implementation
    return glib.OK(&Post{})
}

// @Route method=POST path=/
func (c *PostsController) Create(ctx context.Context, req CreatePostRequest) glib.Result[*Post] {
    post := &Post{
        Title:   req.Title,
        Content: req.Content,
    }
    if err := c.DB.Create(post).Error; err != nil {
        return glib.Fail[*Post](err)
    }
    return glib.Created(post)
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
        handler := handlePostsControllerShow(container)
        mux.Handle("GET /api/v1/posts/{id}", handler)
    }

    {
        handler := handlePostsControllerCreate(container)
        mux.Handle("POST /api/v1/posts", handler)
    }

    return nil
}
```

---

## Summary

### What Gets Generated

1. **Bootstrap code** - Initializes everything (`glib.gen.go`)
2. **DI container** - Dependency injection with topological sort (`di.gen.go`)
3. **Route registration** - Maps HTTP routes to handlers (`routes.gen.go`)
4. **Handler wrappers** - Adapts handlers to `http.Handler` (`parsers.gen.go`)
    - **Result[T] handlers**: Parse request → call handler → `result.Write(w)`
    - **Raw HTTP handlers (no middleware)**: Direct method reference (optimized, 2 lines)
    - **Raw HTTP handlers (with middleware)**: Method wrapped in middleware chain (6 lines)
5. **Middleware chains** - Applied by handler wrappers based on tags and `with` directive

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
