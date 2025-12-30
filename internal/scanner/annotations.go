package scanner

import (
	"go/ast"
	"regexp"
	"strings"
)

var (
	// Annotation patterns
	controllerPattern     = regexp.MustCompile(`^//\s*@Controller\s+(.+)$`)
	routePattern          = regexp.MustCompile(`^//\s*@Route\s+(\w+)\s+(.+)$`)
	providerPattern       = regexp.MustCompile(`^//\s*@Provider\s+(\w+)$`)
	middlewarePattern     = regexp.MustCompile(`^//\s*@Middleware\s+(.+)$`)
	middlewareListPattern = regexp.MustCompile(`^//\s*@Middleware\s+(.+)$`)
)

// extractAnnotations extracts all annotations from a comment group
func extractAnnotations(commentGroup *ast.CommentGroup) []*Annotation {
	if commentGroup == nil {
		return nil
	}

	var annotations []*Annotation

	for _, comment := range commentGroup.List {
		text := comment.Text

		// Controller annotation: @Controller /api/v1/posts
		if match := controllerPattern.FindStringSubmatch(text); match != nil {
			annotations = append(annotations, &Annotation{
				Type:  "Controller",
				Value: strings.TrimSpace(match[1]),
				Line:  int(comment.Pos()),
			})
			continue
		}

		// Route annotation: @Route GET /{id}
		if match := routePattern.FindStringSubmatch(text); match != nil {
			annotations = append(annotations, &Annotation{
				Type:  "Route",
				Value: strings.TrimSpace(match[1]) + " " + strings.TrimSpace(match[2]),
				Line:  int(comment.Pos()),
			})
			continue
		}

		// Provider annotation: @Provider singleton
		if match := providerPattern.FindStringSubmatch(text); match != nil {
			annotations = append(annotations, &Annotation{
				Type:  "Provider",
				Value: strings.TrimSpace(match[1]),
				Line:  int(comment.Pos()),
			})
			continue
		}

		// Middleware annotation: @Middleware auth,ratelimit
		if match := middlewarePattern.FindStringSubmatch(text); match != nil {
			annotations = append(annotations, &Annotation{
				Type:  "Middleware",
				Value: strings.TrimSpace(match[1]),
				Line:  int(comment.Pos()),
			})
			continue
		}
	}

	return annotations
}

// parseControllerAnnotation parses @Controller annotation
// Example: "@Controller /api/v1/posts" returns "/api/v1/posts"
func parseControllerAnnotation(value string) string {
	return strings.TrimSpace(value)
}

// parseRouteAnnotation parses @Route annotation
// Example: "@Route GET /{id}" returns ("GET", "/{id}")
func parseRouteAnnotation(value string) (method, path string) {
	parts := strings.Fields(value)
	if len(parts) < 2 {
		return "", ""
	}
	return strings.ToUpper(parts[0]), strings.Join(parts[1:], " ")
}

// parseMiddlewareAnnotation parses @Middleware annotation
// Example: "@Middleware auth,ratelimit" returns ["auth", "ratelimit"]
func parseMiddlewareAnnotation(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// parseProviderAnnotation parses @Provider annotation
// Example: "@Provider singleton" returns "singleton"
func parseProviderAnnotation(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "transient" // Default
	}
	return value
}

// findAnnotation finds an annotation by type in a list
func findAnnotation(annotations []*Annotation, annotationType string) *Annotation {
	for _, ann := range annotations {
		if ann.Type == annotationType {
			return ann
		}
	}
	return nil
}

// findAnnotations finds all annotations by type in a list
func findAnnotations(annotations []*Annotation, annotationType string) []*Annotation {
	var result []*Annotation
	for _, ann := range annotations {
		if ann.Type == annotationType {
			result = append(result, ann)
		}
	}
	return result
}
