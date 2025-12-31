package generator

import (
	"strings"

	"github.com/azizndao/glib/internal/scanner"
)

// ProviderData represents data for a provider in the template
type ProviderData struct {
	FieldName    string
	TypeName     string
	PackageName  string
	FunctionName string
	Name         string
	ArgsString   string
	ReturnsError bool
	Lifecycle    string // "singleton" or "transient"
	IsConfig     bool   // true if this is a config provider
}

// ControllerData represents data for a controller in the template
type ControllerData struct {
	FieldName   string
	PackageName string
	Name        string
	Fields      []ControllerFieldData
}

// ControllerFieldData represents a field to inject into a controller
type ControllerFieldData struct {
	Name          string
	ProviderField string
	IsTransient   bool // Whether this field's provider is transient
}

// MiddlewareData represents data for middleware in the template
type MiddlewareData struct {
	FieldName    string
	PackageName  string
	FunctionName string
	ArgsString   string
	Signature    string // "old" or "new"
}

// generateDI generates the DI container (di.gen.go)
func (g *Generator) generateDI() (string, error) {
	// Determine what's needed
	needsFmt := false
	needsHTTP := len(g.project.Middleware) > 0
	needsGlibMiddleware := false

	// Check if any provider returns error (needs fmt for error wrapping)
	for _, prov := range g.project.Providers {
		if prov.FuncDecl != nil && prov.FuncDecl.Type.Results != nil {
			returnCount := 0
			for _, field := range prov.FuncDecl.Type.Results.List {
				if len(field.Names) == 0 {
					returnCount++
				} else {
					returnCount += len(field.Names)
				}
			}
			if returnCount > 1 {
				needsFmt = true
				break
			}
		}
	}

	// Check if any middleware uses new-style signature
	for _, mw := range g.project.Middleware {
		if mw.Signature == "new" {
			needsGlibMiddleware = true
			break
		}
	}

	// Add Configs as synthetic providers if they exist
	allProviders := g.project.Providers
	if len(g.project.Configs) > 0 {
		// Create synthetic providers for each Config
		for _, cfg := range g.project.Configs {
			configProvider := g.createConfigProvider(cfg)
			// Prepend config providers (they should be initialized first)
			allProviders = append([]*scanner.Provider{configProvider}, allProviders...)
		}
		needsFmt = true // Config loading needs fmt for error handling
	}

	// Sort providers in dependency order (topological sort)
	sortedProviders := g.sortProvidersByDependencies(allProviders)

	// Build provider data
	var providers []ProviderData
	for _, prov := range sortedProviders {
		// Skip the synthetic config providers - handle them specially
		if strings.HasPrefix(prov.Name, "__config_") {
			providers = append(providers, g.buildConfigProviderData(prov))
			continue
		}

		// Build dependency arguments
		var args []string
		for _, dep := range prov.Dependencies {
			depFieldName := g.findProviderForType(dep.Type)
			if depFieldName != "" {
				// For transient providers, call the factory function
				if g.isProviderTransient(dep.Type) {
					args = append(args, "c.providers."+depFieldName+"Factory()")
				} else {
					args = append(args, "c.providers."+depFieldName)
				}
			}
		}

		// Check if provider returns error (second return value)
		returnsError := false
		if prov.FuncDecl != nil && prov.FuncDecl.Type.Results != nil {
			returnCount := 0
			for _, field := range prov.FuncDecl.Type.Results.List {
				if len(field.Names) == 0 {
					returnCount++
				} else {
					returnCount += len(field.Names)
				}
			}
			returnsError = returnCount > 1
		}

		providers = append(providers, ProviderData{
			FieldName:    g.providerFieldName(prov),
			TypeName:     g.typeString(prov.ReturnType),
			PackageName:  prov.PackageName,
			FunctionName: prov.FunctionName,
			Name:         prov.Name,
			ArgsString:   strings.Join(args, ", "),
			ReturnsError: returnsError,
			Lifecycle:    prov.Lifecycle,
		})
	}

	// Build controller data
	var controllers []ControllerData
	for _, ctrl := range g.project.Controllers {
		var fields []ControllerFieldData
		for _, field := range ctrl.Fields {
			providerField := g.findProviderForType(field.Type)
			if providerField != "" {
				fields = append(fields, ControllerFieldData{
					Name:          field.Name,
					ProviderField: providerField,
					IsTransient:   g.isProviderTransient(field.Type),
				})
			}
		}

		controllers = append(controllers, ControllerData{
			FieldName:   g.controllerFieldName(ctrl),
			PackageName: ctrl.PackageName,
			Name:        ctrl.Name,
			Fields:      fields,
		})
	}

	// Build middleware data
	var middleware []MiddlewareData
	for _, mw := range g.project.Middleware {
		var args []string
		for _, dep := range mw.Dependencies {
			providerField := g.findProviderForType(dep.Type)
			if providerField != "" {
				args = append(args, "c.providers."+providerField)
			}
		}

		middleware = append(middleware, MiddlewareData{
			FieldName:    g.middlewareFieldName(mw),
			PackageName:  mw.PackageName,
			FunctionName: mw.FunctionName,
			ArgsString:   strings.Join(args, ", "),
			Signature:    mw.Signature,
		})
	}

	data := map[string]any{
		"PackageName":          g.pkgName,
		"NeedsFmt":             needsFmt,
		"NeedsHTTP":            needsHTTP,
		"NeedsGlibMiddleware":  needsGlibMiddleware,
		"ConfigNeedsParameter": false, // Always false - config loaded internally
		"ConfigParameterType":  "",
		"Imports":              g.collectImports(),
		"Providers":            providers,
		"Controllers":          controllers,
		"Middleware":           middleware,
	}

	return g.executeTemplate("di.tmpl", data)
}

// createConfigProvider creates a synthetic provider for Config
func (g *Generator) createConfigProvider(cfg *scanner.Config) *scanner.Provider {
	configType := g.getConfigType(cfg)
	funcName := "load" + cfg.Name // loadAppConfig, loadDatabaseConfig, loadConfig

	return &scanner.Provider{
		Name:         "__config_" + cfg.Name + "__", // __config_AppConfig__, __config_Config__
		FunctionName: funcName,                      // Private function in generated package
		PackageName:  g.pkgName,                     // generated package (where loadXxxConfig lives)
		PackagePath:  "",                            // Empty = same package, no import needed
		ReturnType:   configType,
		Dependencies: nil,
		Lifecycle:    "singleton",
	}
}

// buildConfigProviderData builds ProviderData for the synthetic config provider
func (g *Generator) buildConfigProviderData(prov *scanner.Provider) ProviderData {
	// Extract config name from provider name: __config_AppConfig__ -> AppConfig
	configName := strings.TrimSuffix(strings.TrimPrefix(prov.Name, "__config_"), "__")
	fieldName := strings.ToLower(configName[:1]) + configName[1:] // AppConfig -> appConfig

	// loadXxxConfig is in the generated package (same package as DI container)
	return ProviderData{
		FieldName:    fieldName,
		TypeName:     g.typeString(prov.ReturnType),
		PackageName:  "",                // Empty = same package, no prefix needed
		FunctionName: prov.FunctionName, // loadAppConfig, loadConfig, etc.
		Name:         configName,
		ArgsString:   "",
		ReturnsError: true, // loadXxxConfig() always returns (*XxxConfig, error)
		Lifecycle:    "singleton",
		IsConfig:     true, // Mark as config provider
	}
}

// getConfigType returns the TypeInfo for a Config
func (g *Generator) getConfigType(cfg *scanner.Config) *scanner.TypeInfo {
	// Config is always in an importable package (e.g., configs)
	return &scanner.TypeInfo{
		Name:        cfg.Name,
		PackagePath: cfg.PackagePath,
		PackageName: cfg.PackageName,
		FullName:    "*" + cfg.PackageName + "." + cfg.Name,
		IsPointer:   true,
		IsPrimitive: false,
	}
}

// Helper functions

func (g *Generator) providerFieldName(prov *scanner.Provider) string {
	// Convert NewDatabase -> database
	name := strings.TrimPrefix(prov.FunctionName, "New")
	return strings.ToLower(name[:1]) + name[1:]
}

func (g *Generator) controllerFieldName(ctrl *scanner.Controller) string {
	// Include package name to ensure uniqueness: "commentController", "postController"
	return ctrl.PackageName + capitalize(ctrl.Name)
}

func (g *Generator) middlewareFieldName(mw *scanner.Middleware) string {
	return mw.Name + "Middleware"
}

func (g *Generator) findProviderForType(typeInfo *scanner.TypeInfo) string {
	if typeInfo == nil {
		return ""
	}

	// Check if this is a Config type request
	for _, cfg := range g.project.Configs {
		configType := g.getConfigType(cfg)

		// Match by name or by full type path
		if typeInfo.Name == cfg.Name || typeInfo.FullName == configType.FullName {
			// Return field name: AppConfig -> appConfig
			return strings.ToLower(cfg.Name[:1]) + cfg.Name[1:]
		}
	}

	// Find a provider that returns this type
	for _, prov := range g.project.Providers {
		if prov.ReturnType != nil && prov.ReturnType.FullName == typeInfo.FullName {
			return g.providerFieldName(prov)
		}
	}

	return ""
}

// isProviderTransient checks if the provider for a type is transient
func (g *Generator) isProviderTransient(typeInfo *scanner.TypeInfo) bool {
	if typeInfo == nil {
		return false
	}

	// Find the provider that returns this type
	for _, prov := range g.project.Providers {
		if prov.ReturnType != nil && prov.ReturnType.FullName == typeInfo.FullName {
			return prov.Lifecycle == "transient"
		}
	}

	return false
}

func (g *Generator) typeString(typeInfo *scanner.TypeInfo) string {
	if typeInfo == nil {
		return "any"
	}
	return typeInfo.FullName
}

func (g *Generator) collectImports() []string {
	seen := make(map[string]bool)
	var imports []string

	// Helper to add an import if valid
	addImport := func(path string) {
		if path == "" || path == g.pkgName || path == "main" || seen[path] {
			return
		}
		seen[path] = true
		imports = append(imports, path)
	}

	// Helper to recursively collect imports from a type
	var collectTypeImports func(*scanner.TypeInfo)
	collectTypeImports = func(typeInfo *scanner.TypeInfo) {
		if typeInfo == nil {
			return
		}
		addImport(typeInfo.PackagePath)
		for _, param := range typeInfo.TypeParams {
			collectTypeImports(param)
		}
	}

	// Add config package imports if configs exist
	for _, cfg := range g.project.Configs {
		addImport(cfg.PackagePath)
	}

	// Add imports from controllers
	for _, ctrl := range g.project.Controllers {
		addImport(ctrl.PackagePath)
	}

	// Add imports from providers
	for _, prov := range g.project.Providers {
		addImport(prov.PackagePath)

		// Add imports from provider return type
		collectTypeImports(prov.ReturnType)

		// Add imports from provider dependencies
		for _, dep := range prov.Dependencies {
			collectTypeImports(dep.Type)
		}
	}

	// Add imports from middleware
	for _, mw := range g.project.Middleware {
		addImport(mw.PackagePath)
	}

	return imports
}

// sortProvidersByDependencies performs topological sort on providers
// to ensure dependencies are initialized before dependents
func (g *Generator) sortProvidersByDependencies(providers []*scanner.Provider) []*scanner.Provider {
	// Build a map of type -> provider for quick lookup
	providerByType := make(map[string]*scanner.Provider)
	for _, prov := range providers {
		if prov.ReturnType != nil {
			providerByType[prov.ReturnType.FullName] = prov
		}
	}

	// Track visited providers and build result
	visited := make(map[string]bool)
	var result []*scanner.Provider

	// Helper function for DFS
	var visit func(*scanner.Provider)
	visit = func(prov *scanner.Provider) {
		if prov.ReturnType == nil {
			return
		}

		typeName := prov.ReturnType.FullName
		if visited[typeName] {
			return
		}

		// Visit dependencies first
		for _, dep := range prov.Dependencies {
			if dep.Type == nil || dep.Type.IsPrimitive {
				continue
			}

			depProvider := providerByType[dep.Type.FullName]
			if depProvider != nil {
				visit(depProvider)
			}
		}

		// Mark as visited and add to result
		visited[typeName] = true
		result = append(result, prov)
	}

	// Visit all providers
	for _, prov := range providers {
		visit(prov)
	}

	return result
}
