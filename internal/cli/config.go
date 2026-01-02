package cli

import (
	"github.com/azizndao/glib/utils"
)

// glibConfig represents the CLI configuration
// Configuration is loaded from: CLI args > Environment variables > Defaults
type glibConfig struct {
	Version  string
	Generate struct {
		Output  string // Where to output generated code
		Package string // Package name for generated code
		Workers int    // Number of parallel workers (0=auto, -1=disable)
		Cache   bool   // Enable file caching (default: true)
	}
	Make struct {
		Controllers string // Directory for new controllers
		Providers   string // Directory for new providers
		Middleware  string // Directory for new middleware
	}
	Watch struct {
		Debounce     int      // File watch debounce in milliseconds
		ExcludeDirs  []string // Directories to exclude from watching
		IncludeFiles []string // File patterns to include (e.g., "*.go")
		ExcludeFiles []string // File patterns to exclude (e.g., "*_test.go", "*.gen.go")
	}
	Validation struct {
		Enabled         bool     // Explicitly enable validation
		Languages       []string // Supported languages
		DefaultLanguage string   // Default language
	}
	Verbose bool // Global verbose flag (can be overridden by CLI)
}

// getDefaultConfig returns a glibConfig with sensible defaults
// Defaults can be overridden by environment variables
func getDefaultConfig() *glibConfig {
	cfg := &glibConfig{
		Version: "2",
		Verbose: false,
	}

	// Generate defaults
	cfg.Generate.Output = utils.GetEnvOr("GLIB_OUTPUT", "generated")
	cfg.Generate.Package = utils.GetEnvOr("GLIB_PACKAGE", "generated")
	cfg.Generate.Workers, _ = utils.GetEnvInt("GLIB_WORKERS", 4)
	cfg.Generate.Workers = max(1, cfg.Generate.Workers)
	cfg.Generate.Cache, _ = utils.GetEnvBool("GLIB_CACHE", true)

	// Make defaults
	cfg.Make.Controllers = utils.GetEnvOr("GLIB_MAKE_CONTROLLERS", "controllers")
	cfg.Make.Providers = utils.GetEnvOr("GLIB_MAKE_PROVIDERS", "providers")
	cfg.Make.Middleware = utils.GetEnvOr("GLIB_MAKE_MIDDLEWARE", "middleware")

	// Watch defaults
	cfg.Watch.Debounce, _ = utils.GetEnvInt("GLIB_WATCH_DEBOUNCE", 300)
	cfg.Watch.Debounce = max(100, cfg.Watch.Debounce)
	cfg.Watch.ExcludeDirs = utils.GetEnvSlice("GLIB_WATCH_EXCLUDE_DIRS", "vendor,node_modules,.git,.glib,tmp")
	cfg.Watch.IncludeFiles = utils.GetEnvSlice("GLIB_WATCH_INCLUDE_FILES", "*.go")
	cfg.Watch.ExcludeFiles = utils.GetEnvSlice("GLIB_WATCH_EXCLUDE_FILES", "*_test.go,*.gen.go")

	// Validation defaults
	cfg.Validation.Enabled, _ = utils.GetEnvBool("GLIB_VALIDATION_ENABLED", false)
	cfg.Validation.Languages = utils.GetEnvSlice("GLIB_VALIDATION_LANGUAGES", "")
	cfg.Validation.DefaultLanguage = utils.GetEnvOr("GLIB_VALIDATION_DEFAULT_LANGUAGE", "")

	return cfg
}
