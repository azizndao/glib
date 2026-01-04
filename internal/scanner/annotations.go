package scanner

import (
	"go/ast"
	"maps"
	"regexp"
	"strings"
)

var (
	// Annotation patterns
	controllerPattern = regexp.MustCompile(`^//\s*@Controller\s+(.+)$`)
	routePattern      = regexp.MustCompile(`^//\s*@Route\s+(.+)$`)
	providerPattern   = regexp.MustCompile(`^//\s*@Provider\s+(\w+)$`)
	middlewarePattern = regexp.MustCompile(`^//\s*@Middleware\s+(.+)$`)
	configPattern     = regexp.MustCompile(`^//\s*@Config\s*$`)
)

// extractAnnotations extracts all annotations from a comment group
func extractAnnotations(commentGroup *ast.CommentGroup) []*Annotation {
	if commentGroup == nil {
		return nil
	}

	var annotations []*Annotation

	for _, comment := range commentGroup.List {
		text := comment.Text

		// Controller annotation: @Controller path=/api/v1/posts tags=api,public
		if match := controllerPattern.FindStringSubmatch(text); match != nil {
			annotations = append(annotations, &Annotation{
				Type:  AnnotationController,
				Value: strings.TrimSpace(match[1]),
				Line:  int(comment.Pos()),
			})
			continue
		}

		// Route annotation: @Route method=GET path=/{id} tags=protected with=auth,admin
		if match := routePattern.FindStringSubmatch(text); match != nil {
			annotations = append(annotations, &Annotation{
				Type:  AnnotationRoute,
				Value: strings.TrimSpace(match[1]),
				Line:  int(comment.Pos()),
			})
			continue
		}

		// Provider annotation: @Provider singleton
		if match := providerPattern.FindStringSubmatch(text); match != nil {
			annotations = append(annotations, &Annotation{
				Type:  AnnotationProvider,
				Value: strings.TrimSpace(match[1]),
				Line:  int(comment.Pos()),
			})
			continue
		}

		// Middleware annotation: @Middleware name=auth target=protected order=10
		if match := middlewarePattern.FindStringSubmatch(text); match != nil {
			annotations = append(annotations, &Annotation{
				Type:  AnnotationMiddleware,
				Value: strings.TrimSpace(match[1]),
				Line:  int(comment.Pos()),
			})
			continue
		}

		// Config annotation: @Config
		if configPattern.MatchString(text) {
			annotations = append(annotations, &Annotation{
				Type:  AnnotationConfig,
				Value: "", // No value for @Config
				Line:  int(comment.Pos()),
			})
			continue
		}
	}

	return annotations
}

// parseControllerAnnotation parses @Controller annotation
// New style: "path=/api/v1/posts tags=api,public" returns map with parsed values
// Example: "@Controller path=/api/v1/posts tags=api,public"
func parseControllerAnnotation(value string) map[string]string {
	return parseKeyValuePairs(value, map[string]string{
		"tags": "", // default: no tags
	})
}

// parseRouteAnnotation parses @Route annotation
// New style: "method=GET path=/{id} tags=protected with=auth,admin"
// Returns map with parsed values
func parseRouteAnnotation(value string) map[string]string {
	return parseKeyValuePairs(value, map[string]string{
		"tags": "", // default: no tags
		"with": "", // default: no explicit override
	})
}

// parseKeyValuePairs parses key=value pairs from annotation value
// Applies defaults for missing keys
func parseKeyValuePairs(value string, defaults map[string]string) map[string]string {
	result := make(map[string]string)

	// Copy defaults
	maps.Copy(result, defaults)

	// Parse key=value pairs
	pairs := strings.FieldsSeq(value)
	for pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			result[key] = val
		}
	}

	return result
}

// parseMiddlewareAnnotation parses comma-separated middleware list (legacy)
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

// parseCommaSeparated parses comma-separated values
// Example: "protected,admin" returns ["protected", "admin"]
// Example: "none" returns ["none"]
func parseCommaSeparated(value string) []string {
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

// parseMiddlewareDefinition parses middleware definition annotations
// Example: "name=auth target=protected order=10" returns map with parsed values
func parseMiddlewareDefinition(value string) map[string]string {
	return parseKeyValuePairs(value, map[string]string{
		"target": TargetAll,
		"order":  "100",
	})
}

// parseProviderAnnotation parses @Provider annotation
// Example: "@Provider singleton" returns "singleton"
func parseProviderAnnotation(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return LifecycleSingleton.String() // Default
	}
	return value
}

// findAnnotation finds an annotation by type in a list
func findAnnotation(annotations []*Annotation, annotationType AnnotationType) *Annotation {
	for _, ann := range annotations {
		if ann.Type == annotationType {
			return ann
		}
	}
	return nil
}

// findAnnotations finds all annotations by type in a list
func findAnnotations(annotations []*Annotation, annotationType AnnotationType) []*Annotation {
	var result []*Annotation
	for _, ann := range annotations {
		if ann.Type == annotationType {
			result = append(result, ann)
		}
	}
	return result
}
