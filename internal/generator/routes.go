package generator

import (
	"fmt"

	"github.com/azizndao/glib/internal/scanner"
)

// RouteData represents data for a single route
type RouteData struct {
	Pattern     string
	HandlerName string
}

// generateRoutes generates the route registration code (routes.gen.go)
func (g *Generator) generateRoutes() (string, error) {
	// Build route data
	var routes []RouteData
	for _, ctrl := range g.project.Controllers {
		for _, handler := range ctrl.Handlers {
			routes = append(routes, RouteData{
				Pattern:     fmt.Sprintf("%s %s", handler.Method, handler.FullPath),
				HandlerName: g.handlerWrapperName(ctrl, handler),
			})
		}
	}

	data := map[string]any{
		"PackageName": g.pkgName,
		"Routes":      routes,
	}
	return g.executeTemplate("routes.tmpl", data)
}

// handlerWrapperName generates a unique name for handler wrapper
func (g *Generator) handlerWrapperName(ctrl *scanner.Controller, handler *scanner.Handler) string {
	// Use PackageName + Name to ensure uniqueness across multiple controllers
	return fmt.Sprintf("handle%s%s%s",
		capitalize(ctrl.PackageName),
		ctrl.Name,
		handler.Name)
}
