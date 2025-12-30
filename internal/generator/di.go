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
}

// MiddlewareData represents data for middleware in the template
type MiddlewareData struct {
	FieldName    string
	PackageName  string
	FunctionName string
	ArgsString   string
}

// generateDI generates the DI container (di.gen.go)
func (g *Generator) generateDI() (string, error) {
	// Determine what's needed
	needsFmt := len(g.project.Providers) > 0
	needsHTTP := len(g.project.Middleware) > 0

	// Build provider data
	var providers []ProviderData
	for _, prov := range g.project.Providers {
		// Build dependency arguments
		var args []string
		for _, dep := range prov.Dependencies {
			depFieldName := g.findProviderForType(dep.Type)
			if depFieldName != "" {
				args = append(args, "c."+depFieldName)
			}
		}

		providers = append(providers, ProviderData{
			FieldName:    g.providerFieldName(prov),
			TypeName:     g.typeString(prov.ReturnType),
			PackageName:  prov.PackageName,
			FunctionName: prov.FunctionName,
			Name:         prov.Name,
			ArgsString:   strings.Join(args, ", "),
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
		})
	}

	data := map[string]any{
		"PackageName": g.pkgName,
		"NeedsFmt":    needsFmt,
		"NeedsHTTP":   needsHTTP,
		"Imports":     g.collectImports(),
		"Providers":   providers,
		"Controllers": controllers,
		"Middleware":  middleware,
	}

	return g.executeTemplate("di.tmpl", data)
}

// Helper functions

func (g *Generator) providerFieldName(prov *scanner.Provider) string {
	// Convert NewDatabase -> database
	name := prov.FunctionName
	if strings.HasPrefix(name, "New") {
		name = name[3:]
	}
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

	// Find a provider that returns this type
	for _, prov := range g.project.Providers {
		if prov.ReturnType != nil && prov.ReturnType.FullName == typeInfo.FullName {
			return g.providerFieldName(prov)
		}
	}

	return ""
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

	// Add imports from controllers
	for _, ctrl := range g.project.Controllers {
		if !seen[ctrl.PackagePath] && ctrl.PackagePath != g.pkgName {
			seen[ctrl.PackagePath] = true
			imports = append(imports, ctrl.PackagePath)
		}
	}

	// Add imports from providers
	for _, prov := range g.project.Providers {
		if !seen[prov.PackagePath] && prov.PackagePath != g.pkgName {
			seen[prov.PackagePath] = true
			imports = append(imports, prov.PackagePath)
		}
	}

	// Add imports from middleware
	for _, mw := range g.project.Middleware {
		if !seen[mw.PackagePath] && mw.PackagePath != g.pkgName {
			seen[mw.PackagePath] = true
			imports = append(imports, mw.PackagePath)
		}
	}

	return imports
}
