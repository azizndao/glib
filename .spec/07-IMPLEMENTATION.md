# 07. Implementation Roadmap

**Status:** Specification v1.0  
**Last Updated:** 2025-12-30

---

## Table of Contents

1. [Overview](#overview)
2. [Timeline](#timeline)
3. [Phase 1: CLI Foundation](#phase-1-cli-foundation)
4. [Phase 2: Scanner Implementation](#phase-2-scanner-implementation)
5. [Phase 3: Validator Implementation](#phase-3-validator-implementation)
6. [Phase 4: Code Generator Implementation](#phase-4-code-generator-implementation)
7. [Phase 5: Hot Reload Integration](#phase-5-hot-reload-integration)
8. [Phase 6: Testing & Documentation](#phase-6-testing--documentation)
9. [Success Criteria](#success-criteria)
10. [Post-Launch Roadmap](#post-launch-roadmap)

---

## Overview

This document provides a **phase-by-phase implementation plan** for building Glib 2.0 from scratch.

### Development Approach

- **Test-Driven Development (TDD)** - Write tests first
- **Incremental delivery** - Each phase produces working software
- **Documentation as code** - Keep specs updated
- **Example-driven** - Build examples alongside framework

### Core Principles

1. **Delete old code first** - Fresh start, no migration burden
2. **Spec-driven implementation** - Follow specifications exactly
3. **Test everything** - High test coverage (>80%)
4. **Dogfood early** - Use Glib to build Glib tools
5. **Keep it simple** - Avoid over-engineering

---

## Timeline

**Total Duration:** 4-6 weeks (1 developer, full-time)

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| **Phase 1** | Week 1 (3-5 days) | CLI foundation + scaffolding |
| **Phase 2** | Week 1-2 (4-7 days) | AST scanner + annotation parser |
| **Phase 3** | Week 2 (2-3 days) | Validation (DI, routes, types) |
| **Phase 4** | Week 2-3 (5-7 days) | Code generators + templates |
| **Phase 5** | Week 3 (2-3 days) | Hot reload + Air integration |
| **Phase 6** | Week 3-4 (5-7 days) | Testing + docs + examples |

### Milestones

- **Week 1:** `glib init` and `glib make` working
- **Week 2:** `glib generate` produces valid code
- **Week 3:** `glib dev` with hot reload working
- **Week 4:** All examples working, documentation complete

---

## Phase 1: CLI Foundation

**Duration:** 3-5 days  
**Goal:** Working CLI with scaffolding commands

### Tasks

#### 1.1 Project Setup

```bash
# Delete all old code
cd glib
rm -rf cache/ database/ foundation/ http/ queue/ storage/ example/

# Create v2 structure
mkdir -p cmd/glib internal/{cli,scanner,validator,generator,templates}
```

**Directory structure:**

```
glib/
├── cmd/
│   └── glib/
│       └── main.go
├── internal/
│   ├── cli/              # CLI commands
│   ├── scanner/          # AST scanner (Phase 2)
│   ├── validator/        # Validators (Phase 3)
│   ├── generator/        # Code generators (Phase 4)
│   └── templates/        # Go templates (Phase 4)
├── pkg/
│   └── errs/             # Error handling package
├── .spec/                # Specifications (already done)
└── examples/             # Example apps (Phase 6)
```

#### 1.2 CLI Framework

Use [cobra](https://github.com/spf13/cobra) for CLI.

**`cmd/glib/main.go`:**

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/goyave/glib/v2/internal/cli"
    "github.com/spf13/cobra"
)

var version = "2.0.0"

func main() {
    rootCmd := &cobra.Command{
        Use:   "glib",
        Short: "Glib - Go web framework with code generation",
        Long:  "Glib 2.0 - Code generation framework for building Go web applications",
    }
    
    rootCmd.AddCommand(
        cli.InitCmd(),
        cli.GenerateCmd(),
        cli.DevCmd(),
        cli.MakeCmd(),
        cli.ValidateCmd(),
        cli.VersionCmd(version),
    )
    
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

#### 1.3 Implement `glib init`

**`internal/cli/init.go`:**

```go
package cli

import (
    "embed"
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/spf13/cobra"
)

//go:embed templates/init/*
var initTemplates embed.FS

func InitCmd() *cobra.Command {
    var (
        module  string
        example bool
        minimal bool
    )
    
    cmd := &cobra.Command{
        Use:   "init [directory]",
        Short: "Initialize a new Glib project",
        Args:  cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            dir := "."
            if len(args) > 0 {
                dir = args[0]
            }
            
            return initProject(dir, module, example, minimal)
        },
    }
    
    cmd.Flags().StringVar(&module, "module", "", "Go module name")
    cmd.Flags().BoolVar(&example, "example", false, "Include example code")
    cmd.Flags().BoolVar(&minimal, "minimal", false, "Minimal setup (no examples)")
    
    return cmd
}

func initProject(dir, module string, example, minimal bool) error {
    // Create directory
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    
    // Infer module name
    if module == "" {
        module = filepath.Base(dir)
    }
    
    // Create files from templates
    files := map[string]string{
        "main.go":    renderMainGo(module),
        "config.go":  renderConfigGo(module),
        ".glibrc":    renderGlibRC(),
        ".gitignore": renderGitignore(),
    }
    
    if example {
        files["controllers/health.go"] = renderHealthController(module)
    }
    
    for path, content := range files {
        fullPath := filepath.Join(dir, path)
        if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
            return err
        }
        if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
            return err
        }
    }
    
    // Initialize go.mod
    // Run: go mod init <module>
    
    fmt.Printf("✅ Project initialized in %s\n", dir)
    fmt.Println("Next steps:")
    fmt.Printf("  cd %s\n", dir)
    fmt.Println("  glib generate")
    fmt.Println("  glib dev")
    
    return nil
}
```

#### 1.4 Implement `glib make`

**`internal/cli/make.go`:**

```go
package cli

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
    
    "github.com/spf13/cobra"
)

func MakeCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "make <type> <name>",
        Short: "Generate boilerplate code",
        Args:  cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            typ := args[0]
            name := args[1]
            
            switch typ {
            case "controller":
                return makeController(name)
            case "provider":
                return makeProvider(name)
            case "middleware":
                return makeMiddleware(name)
            default:
                return fmt.Errorf("unknown type: %s", typ)
            }
        },
    }
    
    return cmd
}

func makeController(name string) error {
    // Create controller file
    dir := inferDirectory(name, "controllers")
    filename := filepath.Join(dir, "controller.go")
    
    content := renderControllerTemplate(name)
    
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }
    if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
        return err
    }
    
    fmt.Printf("✅ Created controller: %s\n", filename)
    return nil
}
```

#### 1.5 Testing

Write tests for CLI commands:

```go
// internal/cli/init_test.go
func TestInitCommand(t *testing.T) {
    tmpDir := t.TempDir()
    
    err := initProject(tmpDir, "test-app", false, true)
    assert.NoError(t, err)
    
    // Check files created
    assert.FileExists(t, filepath.Join(tmpDir, "main.go"))
    assert.FileExists(t, filepath.Join(tmpDir, "config.go"))
    assert.FileExists(t, filepath.Join(tmpDir, ".glibrc"))
}
```

### Deliverables

- ✅ `glib init` creates project scaffold
- ✅ `glib make controller` creates controller
- ✅ `glib make provider` creates provider
- ✅ `glib make middleware` creates middleware
- ✅ Tests for all commands
- ✅ CI/CD pipeline setup

---

## Phase 2: Scanner Implementation

**Duration:** 4-7 days  
**Goal:** Parse Go source files and extract annotations

### Tasks

#### 2.1 AST Parser

**`internal/scanner/scanner.go`:**

```go
package scanner

import (
    "go/ast"
    "go/parser"
    "go/token"
    "path/filepath"
    "strings"
)

type Scanner struct {
    fset   *token.FileSet
    module string
}

func New(module string) *Scanner {
    return &Scanner{
        fset:   token.NewFileSet(),
        module: module,
    }
}

func (s *Scanner) Scan(dir string) (*ApplicationModel, error) {
    model := &ApplicationModel{
        Module: s.module,
    }
    
    // Discover all .go files
    files, err := s.discoverFiles(dir)
    if err != nil {
        return nil, err
    }
    
    // Parse each file
    for _, file := range files {
        if err := s.scanFile(file, model); err != nil {
            return nil, err
        }
    }
    
    return model, nil
}

func (s *Scanner) scanFile(path string, model *ApplicationModel) error {
    file, err := parser.ParseFile(s.fset, path, nil, parser.ParseComments)
    if err != nil {
        return err
    }
    
    // Extract annotations from declarations
    ast.Inspect(file, func(n ast.Node) bool {
        switch decl := n.(type) {
        case *ast.TypeSpec:
            s.scanTypeDecl(decl, file.Name.Name, model)
        case *ast.FuncDecl:
            s.scanFuncDecl(decl, file.Name.Name, model)
        }
        return true
    })
    
    return nil
}
```

#### 2.2 Annotation Parser

**`internal/scanner/annotations.go`:**

```go
package scanner

import (
    "go/ast"
    "regexp"
    "strings"
)

var annotationRegex = regexp.MustCompile(`^@(\w+)\s*(.*)$`)

func extractAnnotations(doc *ast.CommentGroup) map[string]string {
    if doc == nil {
        return nil
    }
    
    annotations := make(map[string]string)
    
    for _, comment := range doc.List {
        text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
        
        if match := annotationRegex.FindStringSubmatch(text); match != nil {
            name := match[1]
            value := strings.TrimSpace(match[2])
            annotations[name] = value
        }
    }
    
    return annotations
}
```

#### 2.3 Controller Scanner

**`internal/scanner/controllers.go`:**

```go
func (s *Scanner) scanTypeDecl(typeSpec *ast.TypeSpec, pkg string, model *ApplicationModel) {
    structType, ok := typeSpec.Type.(*ast.StructType)
    if !ok {
        return
    }
    
    annotations := extractAnnotations(typeSpec.Doc)
    
    pathPrefix, hasController := annotations["Controller"]
    if !hasController {
        return
    }
    
    ctrl := &ControllerDeclaration{
        Name:       typeSpec.Name.Name,
        Package:    pkg,
        PathPrefix: pathPrefix,
        Middleware: parseMiddlewareList(annotations["Middleware"]),
        Fields:     s.scanFields(structType),
    }
    
    model.Controllers = append(model.Controllers, ctrl)
}
```

#### 2.4 Handler Scanner

**`internal/scanner/handlers.go`:**

```go
func (s *Scanner) scanFuncDecl(funcDecl *ast.FuncDecl, pkg string, model *ApplicationModel) {
    if funcDecl.Recv == nil {
        // Not a method, check if it's a provider
        s.scanProvider(funcDecl, pkg, model)
        return
    }
    
    // Find controller
    receiverType := s.getReceiverType(funcDecl.Recv)
    ctrl := model.FindController(receiverType)
    if ctrl == nil {
        return
    }
    
    annotations := extractAnnotations(funcDecl.Doc)
    
    routeValue, hasRoute := annotations["Route"]
    if !hasRoute {
        return
    }
    
    method, path := parseRoute(routeValue)
    signature := s.analyzeSignature(funcDecl.Type)
    
    handler := &HandlerDeclaration{
        Method:     method,
        Path:       path,
        FuncName:   funcDecl.Name.Name,
        Middleware: parseMiddlewareList(annotations["Middleware"]),
        Signature:  signature,
    }
    
    ctrl.Handlers = append(ctrl.Handlers, handler)
}
```

#### 2.5 Signature Analyzer

**`internal/scanner/signature.go`:**

```go
func (s *Scanner) analyzeSignature(funcType *ast.FuncType) *SignatureInfo {
    sig := &SignatureInfo{}
    
    // Analyze parameters
    if funcType.Params != nil {
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
                // Path param or request struct
                if s.isBasicType(paramType) {
                    sig.Params = append(sig.Params, s.parseParam(param))
                } else {
                    sig.RequestType = s.parseTypeInfo(param.Type)
                }
            }
        }
    }
    
    // Analyze returns
    if funcType.Results != nil {
        for _, result := range funcType.Results.List {
            resultType := s.typeToString(result.Type)
            
            if resultType == "error" {
                sig.ReturnsError = true
            } else {
                sig.ResponseType = s.parseTypeInfo(result.Type)
            }
        }
    }
    
    sig.Pattern = determinePattern(sig)
    
    return sig
}
```

#### 2.6 Testing

```go
// internal/scanner/scanner_test.go
func TestScanController(t *testing.T) {
    source := `
package test

// @Controller /api/posts
type PostsController struct {
    DB *gorm.DB
}

// @Route GET /{id}
func (c *PostsController) Show(ctx context.Context, id uuid.UUID) (*Post, error) {
    return nil, nil
}
`
    
    scanner := New("test")
    model, err := scanner.ScanString(source)
    assert.NoError(t, err)
    
    assert.Len(t, model.Controllers, 1)
    ctrl := model.Controllers[0]
    assert.Equal(t, "PostsController", ctrl.Name)
    assert.Equal(t, "/api/posts", ctrl.PathPrefix)
    
    assert.Len(t, ctrl.Handlers, 1)
    handler := ctrl.Handlers[0]
    assert.Equal(t, "GET", handler.Method)
    assert.Equal(t, "/{id}", handler.Path)
}
```

### Deliverables

- ✅ AST parser working
- ✅ Annotation extraction working
- ✅ Controller scanning complete
- ✅ Handler scanning complete
- ✅ Provider scanning complete
- ✅ Middleware scanning complete
- ✅ Signature analysis working
- ✅ Comprehensive test suite

---

## Phase 3: Validator Implementation

**Duration:** 2-3 days  
**Goal:** Validate scanned model for errors

### Tasks

#### 3.1 DI Graph Validator

**`internal/validator/di.go`:**

```go
package validator

import (
    "fmt"
    
    "github.com/goyave/glib/v2/internal/scanner"
)

type DIValidator struct{}

func (v *DIValidator) Validate(model *scanner.ApplicationModel) error {
    graph := NewDependencyGraph()
    
    // Add all providers
    for _, provider := range model.Providers {
        graph.AddProvider(provider)
    }
    
    // Add config as provider
    if model.Config != nil {
        graph.AddConfig(model.Config)
    }
    
    // Validate controller dependencies
    for _, ctrl := range model.Controllers {
        for _, field := range ctrl.Fields {
            if !graph.CanProvide(field.Type) {
                return fmt.Errorf(
                    "controller %s: no provider for type %s (field %s)",
                    ctrl.Name, field.Type, field.Name,
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

#### 3.2 Route Validator

**`internal/validator/routes.go`:**

```go
func (v *RouteValidator) Validate(model *scanner.ApplicationModel) error {
    routes := make(map[string]*scanner.HandlerDeclaration)
    
    for _, ctrl := range model.Controllers {
        for _, handler := range ctrl.Handlers {
            fullPath := ctrl.PathPrefix + handler.Path
            key := handler.Method + " " + fullPath
            
            if existing, exists := routes[key]; exists {
                return fmt.Errorf(
                    "route conflict: %s defined in %s.%s and %s.%s",
                    key,
                    ctrl.Name, handler.FuncName,
                    existing.Controller.Name, existing.FuncName,
                )
            }
            
            routes[key] = handler
        }
    }
    
    return nil
}
```

#### 3.3 Type Validator

**`internal/validator/types.go`:**

```go
func (v *TypeValidator) Validate(model *scanner.ApplicationModel) error {
    for _, ctrl := range model.Controllers {
        for _, handler := range ctrl.Handlers {
            // Validate path params match signature
            pathParams := extractPathParams(handler.Path)
            sigParams := handler.Signature.Params
            
            if len(pathParams) != len(sigParams) {
                return fmt.Errorf(
                    "%s.%s: path has %d params but signature has %d",
                    ctrl.Name, handler.FuncName,
                    len(pathParams), len(sigParams),
                )
            }
            
            // Validate param names match
            for _, pathParam := range pathParams {
                found := false
                for _, sigParam := range sigParams {
                    if sigParam.Name == pathParam {
                        found = true
                        break
                    }
                }
                if !found {
                    return fmt.Errorf(
                        "%s.%s: path param {%s} not found in signature",
                        ctrl.Name, handler.FuncName, pathParam,
                    )
                }
            }
        }
    }
    
    return nil
}
```

#### 3.4 Testing

```go
func TestDIValidation(t *testing.T) {
    model := &scanner.ApplicationModel{
        Controllers: []*scanner.ControllerDeclaration{
            {
                Name: "PostsController",
                Fields: []*scanner.FieldDeclaration{
                    {Name: "DB", Type: "*gorm.DB"},
                },
            },
        },
        Providers: []*scanner.ProviderDeclaration{},
    }
    
    validator := &DIValidator{}
    err := validator.Validate(model)
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "no provider for type *gorm.DB")
}
```

### Deliverables

- ✅ DI graph validation
- ✅ Circular dependency detection
- ✅ Route conflict detection
- ✅ Type mismatch detection
- ✅ Comprehensive test suite

---

## Phase 4: Code Generator Implementation

**Duration:** 5-7 days  
**Goal:** Generate production-ready Go code

### Tasks

#### 4.1 Template System

Use Go's `text/template` with custom functions.

**`internal/generator/generator.go`:**

```go
package generator

import (
    "bytes"
    "embed"
    "go/format"
    "os"
    "path/filepath"
    "text/template"
    
    "github.com/goyave/glib/v2/internal/scanner"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

type Generator struct {
    templates *template.Template
    outputDir string
}

func New(outputDir string) (*Generator, error) {
    tmpl, err := template.ParseFS(templatesFS, "templates/*.tmpl")
    if err != nil {
        return nil, err
    }
    
    return &Generator{
        templates: tmpl,
        outputDir: outputDir,
    }, nil
}

func (g *Generator) Generate(model *scanner.ApplicationModel) error {
    files := map[string]string{
        "glib.gen.go":       "bootstrap.tmpl",
        "di.gen.go":         "di.tmpl",
        "routes.gen.go":     "routes.tmpl",
        "parsers.gen.go":    "parsers.tmpl",
        "errors.gen.go":     "errors.tmpl",
        "middleware.gen.go": "middleware.tmpl",
    }
    
    for filename, tmplName := range files {
        if err := g.generateFile(filename, tmplName, model); err != nil {
            return err
        }
    }
    
    return nil
}

func (g *Generator) generateFile(filename, tmplName string, model interface{}) error {
    var buf bytes.Buffer
    
    if err := g.templates.ExecuteTemplate(&buf, tmplName, model); err != nil {
        return err
    }
    
    // Format generated code
    formatted, err := format.Source(buf.Bytes())
    if err != nil {
        return err
    }
    
    // Write to file
    path := filepath.Join(g.outputDir, filename)
    return os.WriteFile(path, formatted, 0644)
}
```

#### 4.2 Bootstrap Template

**`internal/generator/templates/bootstrap.tmpl`:**

```go
// Code generated by glib. DO NOT EDIT.
package generated

import (
    "context"
    "net/http"
    
    {{- range .Imports }}
    "{{ . }}"
    {{- end }}
)

func Bootstrap(ctx context.Context) (http.Handler, error) {
    cfg := LoadConfig()
    
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

#### 4.3 DI Template

**`internal/generator/templates/di.tmpl`:**

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

func buildContainer(ctx context.Context, cfg *Config) (*Container, error) {
    c := &Container{}
    
    {{- range .SortedProviders }}
    {{ template "provider" . }}
    {{- end }}
    
    {{- range .Controllers }}
    {{ template "controller" . }}
    {{- end }}
    
    return c, nil
}

{{- define "provider" }}
{{ .VarName }}, err := {{ .Package }}.{{ .FuncName }}(
    {{- range .Dependencies }}
    c.{{ .FieldName }},
    {{- end }}
)
if err != nil {
    return nil, fmt.Errorf("provider {{ .FuncName }}: %w", err)
}
c.{{ .FieldName }} = {{ .VarName }}
{{- end }}

{{- define "controller" }}
c.{{ .FieldName }} = &{{ .Package }}.{{ .Name }}{
    {{- range .Fields }}
    {{ .Name }}: c.{{ .DependencyFieldName }},
    {{- end }}
}
{{- end }}
```

#### 4.4 Routes Template

**`internal/generator/templates/routes.tmpl`:**

```go
// Code generated by glib. DO NOT EDIT.
package generated

func registerRoutes(mux *http.ServeMux, container *Container) error {
    {{- range .Controllers }}
    {{- range .Handlers }}
    {
        handler := wrap{{ .ControllerName }}_{{ .FuncName }}(container.{{ .ControllerField }})
        {{- range .Middleware }}
        handler = {{ .Package }}.{{ .FuncName }}()(handler)
        {{- end }}
        mux.Handle("{{ .Method }} {{ .FullPath }}", handler)
    }
    {{- end }}
    {{- end }}
    
    return nil
}

{{- range .Handlers }}
{{ template "wrapper" . }}
{{- end }}

{{- define "wrapper" }}
func wrap{{ .ControllerName }}_{{ .FuncName }}(ctrl *{{ .ControllerPackage }}.{{ .ControllerName }}) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        {{ template "signature_" .Signature.Pattern . }}
    })
}
{{- end }}
```

#### 4.5 Testing

```go
func TestGenerateBootstrap(t *testing.T) {
    model := &scanner.ApplicationModel{
        Module: "test-app",
        Controllers: []*scanner.ControllerDeclaration{
            {Name: "PostsController", Package: "posts"},
        },
    }
    
    gen, _ := New(t.TempDir())
    err := gen.Generate(model)
    assert.NoError(t, err)
    
    // Check file created
    assert.FileExists(t, filepath.Join(gen.outputDir, "glib.gen.go"))
    
    // Check it compiles
    // TODO: Use go/build to verify
}
```

### Deliverables

- ✅ Template system working
- ✅ Bootstrap generation
- ✅ DI container generation
- ✅ Route registration generation
- ✅ Request parser generation
- ✅ Handler wrapper generation
- ✅ Error handling generation
- ✅ All generated code compiles
- ✅ Test suite

---

## Phase 5: Hot Reload Integration

**Duration:** 2-3 days  
**Goal:** `glib dev` with automatic rebuild

### Tasks

#### 5.1 Implement `glib dev`

**`internal/cli/dev.go`:**

```go
func DevCmd() *cobra.Command {
    var port int
    
    cmd := &cobra.Command{
        Use:   "dev",
        Short: "Start development server with hot reload",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runDevServer(port)
        },
    }
    
    cmd.Flags().IntVar(&port, "port", 8080, "Server port")
    
    return cmd
}

func runDevServer(port int) error {
    // Generate initial code
    if err := generate(); err != nil {
        return err
    }
    
    // Check if Air is installed
    if !isAirInstalled() {
        fmt.Println("Air not found, using basic file watcher")
        return runBasicWatcher(port)
    }
    
    // Generate .air.toml
    if err := generateAirConfig(port); err != nil {
        return err
    }
    
    // Run Air
    return runAir()
}
```

#### 5.2 Air Configuration Generator

```go
func generateAirConfig(port int) error {
    config := fmt.Sprintf(`
root = "."
tmp_dir = "tmp"

[build]
  pre_cmd = ["glib generate"]
  cmd = "go build -o ./tmp/main ."
  bin = "tmp/main"
  include_ext = ["go"]
  exclude_dir = ["tmp", "vendor", "node_modules", "generated"]
  include_file = ["generated/glib.gen.go"]
  delay = 1000

[log]
  time = true
`, port)
    
    return os.WriteFile(".air.toml", []byte(config), 0644)
}
```

#### 5.3 Basic File Watcher (Fallback)

```go
func runBasicWatcher(port int) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    defer watcher.Close()
    
    // Watch all .go files
    filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
        if strings.HasSuffix(path, ".go") && !strings.Contains(path, "generated") {
            watcher.Add(path)
        }
        return nil
    })
    
    // Start server
    cmd := exec.Command("go", "run", ".")
    cmd.Start()
    
    for {
        select {
        case event := <-watcher.Events:
            if event.Op&fsnotify.Write == fsnotify.Write {
                // Regenerate
                generate()
                
                // Restart server
                cmd.Process.Kill()
                cmd = exec.Command("go", "run", ".")
                cmd.Start()
            }
        }
    }
}
```

### Deliverables

- ✅ `glib dev` command working
- ✅ Air integration
- ✅ Basic file watcher fallback
- ✅ Auto-regeneration on file changes

---

## Phase 6: Testing & Documentation

**Duration:** 5-7 days  
**Goal:** Production-ready release

### Tasks

#### 6.1 End-to-End Tests

Create test projects and verify everything works:

```go
func TestE2E_BlogAPI(t *testing.T) {
    // Create temporary project
    dir := t.TempDir()
    
    // Run: glib init blog-api
    runCommand(t, "glib", "init", dir)
    
    // Add blog controller
    // Run: glib make controller posts
    
    // Generate code
    runCommand(t, "glib", "generate", "--dir", dir)
    
    // Build project
    runCommand(t, "go", "build", "-C", dir)
    
    // Verify binary works
    // Start server, make HTTP requests, verify responses
}
```

#### 6.2 Example Applications

Build all examples from `06-EXAMPLES.md`:

```
examples/
├── blog-api/           # Simple blog (Example 1)
├── ecommerce-api/      # E-commerce (Example 2)
├── user-service/       # Microservice (Example 3)
└── chat-api/           # Real-time chat (Example 4)
```

Each example should:
- Have complete working code
- Have README with setup instructions
- Have tests
- Demonstrate best practices

#### 6.3 Error Package

Implement `pkg/errs/` package for Encore-style errors:

```go
package errs

type Code int

const (
    InvalidArgument Code = 400
    Unauthenticated Code = 401
    PermissionDenied Code = 403
    NotFound Code = 404
    AlreadyExists Code = 409
    Internal Code = 500
    Unavailable Code = 503
)

type Error struct {
    code    Code
    message string
    meta    map[string]interface{}
    details map[string][]string
    cause   error
}

func B() *Builder {
    return &Builder{err: &Error{}}
}

type Builder struct {
    err *Error
}

func (b *Builder) Code(code Code) *Builder {
    b.err.code = code
    return b
}

func (b *Builder) Msg(msg string) *Builder {
    b.err.message = msg
    return b
}

func (b *Builder) Meta(key string, value interface{}) *Builder {
    if b.err.meta == nil {
        b.err.meta = make(map[string]interface{})
    }
    b.err.meta[key] = value
    return b
}

func (b *Builder) Cause(err error) *Builder {
    b.err.cause = err
    return b
}

func (b *Builder) Err() error {
    return b.err
}
```

#### 6.4 Documentation

1. **README.md** - Project overview
2. **GETTING_STARTED.md** - Quick start guide
3. **API Reference** - Generated from code docs
4. **Migration Guide** - (None needed - fresh start)
5. **FAQ** - Common questions

#### 6.5 GitHub Repository Setup

- CI/CD with GitHub Actions
- Issue templates
- PR template
- Contributing guide
- Code of conduct
- License (MIT)

### Deliverables

- ✅ End-to-end tests passing
- ✅ All examples working
- ✅ Error package complete
- ✅ Documentation complete
- ✅ GitHub repo ready
- ✅ Test coverage >80%

---

## Success Criteria

### Must Have (Launch Blockers)

- [x] All specifications written
- [ ] `glib init` working
- [ ] `glib generate` working
- [ ] `glib dev` working
- [ ] `glib make` working
- [ ] All 4 examples working
- [ ] Test coverage >80%
- [ ] Documentation complete
- [ ] No known bugs

### Should Have (Post-Launch OK)

- [ ] VS Code extension
- [ ] JetBrains plugin
- [ ] Performance benchmarks
- [ ] Video tutorials
- [ ] Community examples

### Nice to Have (Future)

- [ ] Web UI for project visualization
- [ ] OpenAPI/Swagger generation
- [ ] GraphQL support
- [ ] gRPC support

---

## Post-Launch Roadmap

### v2.1 (1-2 months after launch)

- **Queue/Task system** - Background jobs (deferred from v2.0)
- **WebSocket support** - Built-in WebSocket helpers
- **gRPC integration** - First-class gRPC support
- **OpenAPI generation** - Automatic API docs

### v2.2 (3-4 months after launch)

- **GraphQL support** - GraphQL schema from annotations
- **Testing helpers** - HTTP test client, fixtures
- **Deployment tools** - Docker, Kubernetes helpers
- **Observability** - Built-in metrics, tracing

### v2.3+ (6+ months)

- **Admin panel** - Auto-generated admin UI
- **Multi-tenancy** - Built-in tenant isolation
- **Real-time updates** - Server-sent events, WebSockets
- **Plugin system** - Third-party extensions

---

## Risk Mitigation

### Technical Risks

| Risk | Mitigation |
|------|------------|
| AST parsing complexity | Use `go/ast` package, extensive testing |
| Template bugs | Test-driven development, example validation |
| Performance issues | Benchmark early, optimize hot paths |
| Breaking changes | Semantic versioning, deprecation warnings |

### Project Risks

| Risk | Mitigation |
|------|------------|
| Scope creep | Defer non-critical features to v2.1 |
| Timeline slip | Daily progress tracking, adjust scope |
| Adoption low | Focus on docs, examples, community |
| Competition | Differentiate (flexibility, simplicity) |

---

## Daily Progress Tracking

Use this checklist to track daily progress:

### Week 1
- [ ] Day 1: Project setup + CLI framework
- [ ] Day 2: `glib init` command
- [ ] Day 3: `glib make` commands
- [ ] Day 4: AST parser foundation
- [ ] Day 5: Annotation extraction

### Week 2
- [ ] Day 6: Controller scanning
- [ ] Day 7: Handler scanning
- [ ] Day 8: Provider/middleware scanning
- [ ] Day 9: DI validation
- [ ] Day 10: Route/type validation

### Week 3
- [ ] Day 11: Template system
- [ ] Day 12: Bootstrap generation
- [ ] Day 13: DI generation
- [ ] Day 14: Route generation
- [ ] Day 15: Hot reload

### Week 4
- [ ] Day 16-17: Example apps
- [ ] Day 18-19: Documentation
- [ ] Day 20-21: Testing + polish

---

## Summary

### Phases Overview

| Phase | Output | Critical? |
|-------|--------|-----------|
| **1. CLI** | Working commands | ✅ Critical |
| **2. Scanner** | Parse source code | ✅ Critical |
| **3. Validator** | Catch errors early | ✅ Critical |
| **4. Generator** | Generate code | ✅ Critical |
| **5. Hot Reload** | Dev experience | ⚠️ Important |
| **6. Testing** | Confidence | ✅ Critical |

### Success Metrics

- **Compile time:** <500ms for typical project
- **Test coverage:** >80%
- **Example apps:** All 4 working
- **Documentation:** 100% complete
- **Bugs:** Zero known critical bugs

---

**Status:** Ready for implementation  
**Next Step:** Delete old code, start Phase 1

