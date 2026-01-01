package generator

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/azizndao/glib/internal/scanner"
)

type ControllerGroup struct {
	Controller       *scanner.Controller
	Prefix           string
	Middleware       []string
	TagGroups        []*TagGroup
	StandaloneRoutes []*RouteData // Routes that don't fit into tag groups
}

// TagGroup represents routes grouped by common tags with shared middleware
type TagGroup struct {
	Tags       []string // Tags for this group (e.g., ["protected"])
	Middleware []string
	Routes     []*RouteData
}

// RouteData represents data for a single route
type RouteData struct {
	Method      string
	Path        string
	HandlerName string
	Tags        []string
	With        []string // Explicit middleware override
}

func (g *Generator) generateRoutes() (string, error) {
	controllerGroups := g.organizeControllerGroups()

	data := map[string]any{
		"PackageName":      g.pkgName,
		"ControllerGroups": controllerGroups,
		"AllMiddleware":    g.project.Middleware,
	}
	return g.executeTemplate("routes.templ", data)
}

// organizeControllerGroups organizes routes by controller and tag groups
func (g *Generator) organizeControllerGroups() []*ControllerGroup {
	var groups []*ControllerGroup

	for _, ctrl := range g.project.Controllers {
		group := &ControllerGroup{
			Controller: ctrl,
			Prefix:     ctrl.RoutePrefix,
			Middleware: g.getControllerMiddleware(ctrl),
		}

		// Build route data for all handlers
		routes := make([]*RouteData, 0, len(ctrl.Handlers))
		for _, handler := range ctrl.Handlers {
			routes = append(routes, &RouteData{
				Method:      handler.Method,
				Path:        strings.TrimPrefix(handler.Path, ctrl.RoutePrefix),
				HandlerName: g.handlerWrapperName(ctrl, handler),
				Tags:        handler.Tags,
				With:        handler.With,
			})
		}

		// Group routes by tags
		group.TagGroups, group.StandaloneRoutes = g.organizeTagGroups(routes, ctrl)

		groups = append(groups, group)
	}

	return groups
}

func (g *Generator) getControllerMiddleware(ctrl *scanner.Controller) []string {
	var middleware []string

	for _, mw := range g.project.Middleware {
		if g.middlewareAppliesToController(mw, ctrl) {
			middleware = append(middleware, fmt.Sprintf("container.middleware.%sMiddleware", mw.Name))
		}
	}

	// Sort by order
	sort.SliceStable(middleware, func(i, j int) bool {
		return g.getMiddlewareOrder(middleware[i]) < g.getMiddlewareOrder(middleware[j])
	})

	return middleware
}

// middlewareAppliesToController checks if middleware applies to the entire controller
func (g *Generator) middlewareAppliesToController(mw *scanner.Middleware, ctrl *scanner.Controller) bool {
	// "all" target applies to everything
	if mw.Target == "all" {
		return true
	}

	// Check if ALL controller tags match the middleware target
	targets := strings.SplitSeq(mw.Target, ",")
	for target := range targets {
		target = strings.TrimSpace(target)
		if slices.Contains(ctrl.Tags, target) {
			return true
		}
	}

	return false
}

// organizeTagGroups organizes routes into tag-based groups with shared middleware
func (g *Generator) organizeTagGroups(routes []*RouteData, ctrl *scanner.Controller) ([]*TagGroup, []*RouteData) {
	// Find routes with common tags (excluding controller tags)
	tagMap := make(map[string][]*RouteData) // tag signature -> routes

	var standalone []*RouteData

	for _, route := range routes {
		// Check for explicit middleware override
		if len(route.With) > 0 {
			// If "none", this route has no middleware
			if len(route.With) == 1 && route.With[0] == "none" {
				standalone = append(standalone, route)
				continue
			}
			// Routes with explicit middleware go standalone
			standalone = append(standalone, route)
			continue
		}

		// Find handler-specific tags (not in controller tags)
		handlerTags := diffTags(route.Tags, ctrl.Tags)

		if len(handlerTags) == 0 {
			// No extra tags, just use controller middleware
			standalone = append(standalone, route)
		} else {
			// Group by tag signature
			tagSig := strings.Join(handlerTags, ",")
			tagMap[tagSig] = append(tagMap[tagSig], route)
		}
	}

	// Convert tag map to tag groups
	var tagGroups []*TagGroup
	for tagSig, groupRoutes := range tagMap {
		if len(groupRoutes) > 1 {
			// Multiple routes with same tags -> create a group
			tags := strings.Split(tagSig, ",")
			tagGroups = append(tagGroups, &TagGroup{
				Tags:       tags,
				Middleware: g.getTagGroupMiddleware(tags),
				Routes:     groupRoutes,
			})
		} else {
			// Single route -> standalone
			standalone = append(standalone, groupRoutes...)
		}
	}

	return tagGroups, standalone
}

// getTagGroupMiddleware returns middleware for a tag group
func (g *Generator) getTagGroupMiddleware(tags []string) []string {
	var middleware []string

	for _, mw := range g.project.Middleware {
		targets := strings.SplitSeq(mw.Target, ",")
		for target := range targets {
			target = strings.TrimSpace(target)
			if slices.Contains(tags, target) {
				middleware = append(middleware, fmt.Sprintf("container.middleware.%sMiddleware", mw.Name))
				break
			}
		}
	}

	// Sort by order
	sort.SliceStable(middleware, func(i, j int) bool {
		return g.getMiddlewareOrder(middleware[i]) < g.getMiddlewareOrder(middleware[j])
	})

	return middleware
}

// getMiddlewareOrder returns the order of a middleware by name
func (g *Generator) getMiddlewareOrder(middlewareName string) int {
	// Extract middleware name from "container.middleware.authMiddleware"
	parts := strings.Split(middlewareName, ".")
	if len(parts) != 3 {
		return 100
	}
	name := strings.TrimSuffix(parts[2], "Middleware")

	for _, mw := range g.project.Middleware {
		if mw.Name == name {
			return mw.Order
		}
	}
	return 100
}

// handlerWrapperName generates a unique name for handler wrapper
func (g *Generator) handlerWrapperName(ctrl *scanner.Controller, handler *scanner.Handler) string {
	// Use PackageName + Name to ensure uniqueness across multiple controllers
	return fmt.Sprintf("handle%s%s%s",
		capitalize(ctrl.PackageName),
		ctrl.Name,
		handler.Name)
}

// diffTags returns tags in a that are not in b
func diffTags(a, b []string) []string {
	var diff []string
	for _, tag := range a {
		if !slices.Contains(b, tag) {
			diff = append(diff, tag)
		}
	}
	return diff
}
