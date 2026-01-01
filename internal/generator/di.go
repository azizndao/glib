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
	// Check if any middleware uses glib-style signature
	needsGlibMiddleware := false
	for _, mw := range g.project.Middleware {
		if mw.Signature == "glib" {
			needsGlibMiddleware = true
			break
		}
	}

	// Add Configs as synthetic providers if they exist
	allProviders := g.project.Providers
	var configProviders []*scanner.Provider
	if len(g.project.Configs) > 0 {
		// Create synthetic providers for each Config
		for _, cfg := range g.project.Configs {
			configProvider := g.createConfigProvider(cfg)
			configProviders = append(configProviders, configProvider)
			// Add to allProviders for dependency graph analysis
			allProviders = append(allProviders, configProvider)
		}
	}

	// NEW: Build dependency graph and analyze initialization order
	depGraph := NewDependencyGraph(&scanner.Project{
		Providers:   allProviders,
		Controllers: g.project.Controllers,
	})
	initPlan := depGraph.AnalyzeUsage()

	// Build config provider data (always first)
	var configProvidersData []ProviderData
	for _, prov := range configProviders {
		configProvidersData = append(configProvidersData, g.buildConfigProviderData(prov))
	}

	// Build provider data in THREE phases
	var criticalTransients []ProviderData
	var singletons []ProviderData
	var nonCriticalTransients []ProviderData

	// Phase 1: Critical Transients
	for _, prov := range initPlan.CriticalTransients {
		criticalTransients = append(criticalTransients, g.buildProviderData(prov))
	}

	// Phase 2: Singletons
	for _, prov := range initPlan.Singletons {
		// Skip config providers (already handled)
		if strings.HasPrefix(prov.Name, "__config_") {
			continue
		}
		singletons = append(singletons, g.buildProviderData(prov))
	}

	// Phase 3: Non-Critical Transients
	for _, prov := range initPlan.NonCriticalTransients {
		nonCriticalTransients = append(nonCriticalTransients, g.buildProviderData(prov))
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
				args = append(args, "c."+providerField)
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

	// Collect all package paths for imports (goimports will remove unused ones)
	var imports []string
	seen := make(map[string]bool)

	// Add config packages
	for _, cfg := range g.project.Configs {
		if cfg.PackagePath != "" && !seen[cfg.PackagePath] && cfg.PackagePath != g.pkgName {
			imports = append(imports, cfg.PackagePath)
			seen[cfg.PackagePath] = true
		}
	}

	// Add controller packages
	for _, ctrl := range g.project.Controllers {
		if ctrl.PackagePath != "" && !seen[ctrl.PackagePath] && ctrl.PackagePath != g.pkgName {
			imports = append(imports, ctrl.PackagePath)
			seen[ctrl.PackagePath] = true
		}
	}

	// Add provider packages
	for _, prov := range g.project.Providers {
		if prov.PackagePath != "" && !seen[prov.PackagePath] && prov.PackagePath != g.pkgName {
			imports = append(imports, prov.PackagePath)
			seen[prov.PackagePath] = true
		}
	}

	// Add middleware packages
	for _, mw := range g.project.Middleware {
		if mw.PackagePath != "" && !seen[mw.PackagePath] && mw.PackagePath != g.pkgName {
			imports = append(imports, mw.PackagePath)
			seen[mw.PackagePath] = true
		}
	}

	data := map[string]any{
		"PackageName":           g.pkgName,
		"NeedsGlibMiddleware":   needsGlibMiddleware,
		"Imports":               imports,
		"ConfigProviders":       configProvidersData,
		"CriticalTransients":    criticalTransients,
		"Singletons":            singletons,
		"NonCriticalTransients": nonCriticalTransients,
		"Controllers":           controllers,
		"Middleware":            middleware,
		"ValidationConfig":      g.validationCfg,
		"ValidationEnabled":     g.validationCfg.Enabled,
	}

	return g.executeTemplate("di.templ", data)
}

// buildProviderData builds ProviderData for a given provider
func (g *Generator) buildProviderData(prov *scanner.Provider) ProviderData {
	// Build dependency arguments
	var args []string
	for _, dep := range prov.Dependencies {
		depFieldName := g.findProviderForType(dep.Type)
		if depFieldName != "" {
			// For transient providers, call the factory function
			if g.isProviderTransient(dep.Type) {
				args = append(args, "c."+depFieldName+"Factory()")
			} else {
				args = append(args, "c."+depFieldName)
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

	return ProviderData{
		FieldName:    g.providerFieldName(prov),
		TypeName:     g.typeString(prov.ReturnType),
		PackageName:  prov.PackageName,
		FunctionName: prov.FunctionName,
		Name:         prov.Name,
		ArgsString:   strings.Join(args, ", "),
		ReturnsError: returnsError,
		Lifecycle:    prov.Lifecycle,
		IsConfig:     false,
	}
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
	fieldName := capitalize(configName) // AppConfig -> AppConfig (exported)

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
	// Convert NewDatabase -> Database (exported)
	name := strings.TrimPrefix(prov.FunctionName, "New")
	return capitalize(name)
}

func (g *Generator) controllerFieldName(ctrl *scanner.Controller) string {
	// Include package name to ensure uniqueness: "CommentController", "PostController" (exported)
	return capitalize(ctrl.PackageName) + capitalize(ctrl.Name)
}

func (g *Generator) middlewareFieldName(mw *scanner.Middleware) string {
	return capitalize(mw.Name) + "Middleware"
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
			// Return field name: AppConfig -> AppConfig (exported)
			return capitalize(cfg.Name)
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
