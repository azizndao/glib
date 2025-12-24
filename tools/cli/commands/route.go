package commands

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// NewRouteListCommand creates the "glib route:list" command
func NewRouteListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route:list",
		Short: "List all registered routes",
		Long: `Display a list of all registered routes in the application.

Example:
  glib route:list
  glib route:list --method=GET
  glib route:list --path=/api`,
		RunE: runRouteList,
	}

	cmd.Flags().String("method", "", "Filter by HTTP method")
	cmd.Flags().String("path", "", "Filter by path pattern")

	return cmd
}

type route struct {
	Method  string
	Path    string
	Handler string
}

func runRouteList(cmd *cobra.Command, args []string) error {
	methodFilter, _ := cmd.Flags().GetString("method")
	pathFilter, _ := cmd.Flags().GetString("path")

	cmd.Println("Analyzing routes...")

	// Find routes directory
	routesDir := "routes"
	if _, err := os.Stat(routesDir); os.IsNotExist(err) {
		return fmt.Errorf("routes directory not found. Make sure you're in a glib project directory")
	}

	routes, err := extractRoutes(routesDir)
	if err != nil {
		return fmt.Errorf("failed to extract routes: %w", err)
	}

	if len(routes) == 0 {
		cmd.Println("\nNo routes found.")
		return nil
	}

	// Apply filters
	filtered := filterRoutes(routes, methodFilter, pathFilter)

	if len(filtered) == 0 {
		cmd.Println("\nNo routes match the given filters.")
		return nil
	}

	// Display routes
	displayRoutes(cmd, filtered)

	return nil
}

func extractRoutes(routesDir string) ([]route, error) {
	var routes []route

	// Walk through routes directory
	err := filepath.Walk(routesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Parse the file
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}

		// Extract route definitions
		ast.Inspect(node, func(n ast.Node) bool {
			// Look for method calls like r.Get("/path", handler)
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					method := sel.Sel.Name
					if isHTTPMethod(method) {
						if len(call.Args) >= 2 {
							// Extract path
							if lit, ok := call.Args[0].(*ast.BasicLit); ok {
								if lit.Kind == token.STRING {
									path := strings.Trim(lit.Value, `"`)
									
									// Extract handler name
									handler := extractHandlerName(call.Args[1])
									
									routes = append(routes, route{
										Method:  strings.ToUpper(method),
										Path:    path,
										Handler: handler,
									})
								}
							}
						}
					}
				}
			}
			return true
		})

		return nil
	})

	return routes, err
}

func isHTTPMethod(method string) bool {
	httpMethods := []string{"Get", "Post", "Put", "Patch", "Delete", "Head", "Options", "Route"}
	for _, m := range httpMethods {
		if method == m {
			return true
		}
	}
	return false
}

func extractHandlerName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return fmt.Sprintf("%v.%s", e.X, e.Sel.Name)
	case *ast.FuncLit:
		return "func()"
	default:
		return "anonymous"
	}
}

func filterRoutes(routes []route, method, path string) []route {
	var filtered []route

	for _, r := range routes {
		if method != "" && !strings.EqualFold(r.Method, method) {
			continue
		}
		if path != "" && !strings.Contains(r.Path, path) {
			continue
		}
		filtered = append(filtered, r)
	}

	return filtered
}

func displayRoutes(cmd *cobra.Command, routes []route) {
	cmd.Println()
	cmd.Println("╔════════════════════════════════════════════════════════════════════════╗")
	cmd.Println("║                            Application Routes                          ║")
	cmd.Println("╠════════════╦═══════════════════════════════╦═══════════════════════════╣")
	cmd.Println("║   Method   ║             Path              ║          Handler          ║")
	cmd.Println("╠════════════╬═══════════════════════════════╬═══════════════════════════╣")

	for _, route := range routes {
		method := padRight(route.Method, 10)
		path := padRight(route.Path, 29)
		handler := padRight(route.Handler, 25)
		
		cmd.Printf("║ %s ║ %s ║ %s ║\n", method, path, handler)
	}

	cmd.Println("╚════════════╩═══════════════════════════════╩═══════════════════════════╝")
	cmd.Printf("\nTotal: %d route(s)\n", len(routes))
}

func padRight(str string, length int) string {
	if len(str) > length {
		return str[:length-3] + "..."
	}
	return str + strings.Repeat(" ", length-len(str))
}
