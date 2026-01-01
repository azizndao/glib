package cli

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
)

// glibConfig represents the config.toml configuration file structure
// This is for CLI/build-time configuration only
type glibConfig struct {
	Version  string `toml:"version"`
	Generate struct {
		Output  string `toml:"output"`  // Where to output generated code
		Package string `toml:"package"` // Package name for generated code
		Workers int    `toml:"workers"` // Number of parallel workers (0=auto, -1=disable)
		Cache   bool   `toml:"cache"`   // Enable file caching (default: true)
	} `toml:"generate"`
	Make struct {
		Controllers string `toml:"controllers"` // Directory for new controllers
		Providers   string `toml:"providers"`   // Directory for new providers
		Middleware  string `toml:"middleware"`  // Directory for new middleware
	} `toml:"make"`
	Watch struct {
		Debounce     int      `toml:"debounce"`     // File watch debounce in milliseconds
		ExcludeDirs  []string `toml:"excludeDirs"`  // Directories to exclude from watching
		IncludeFiles []string `toml:"includeFiles"` // File patterns to include (e.g., "*.go", "config.toml")
		ExcludeFiles []string `toml:"excludeFiles"` // File patterns to exclude (e.g., "*_test.go", "*.gen.go")
	} `toml:"watch"`
	Validation struct {
		Enabled         bool     `toml:"enabled"`         // Explicitly enable validation
		Languages       []string `toml:"languages"`       // Supported languages
		DefaultLanguage string   `toml:"defaultLanguage"` // Default language
	} `toml:"validation"`
	Verbose bool `toml:"verbose"` // Global verbose flag (can be overridden by CLI)
}

// envVarPattern matches ${VAR_NAME} or ${VAR_NAME:-default}
var envVarPattern = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)(:-([^}]+))?\}`)

// expandEnvVars recursively expands environment variables in config struct
// Supports: ${VAR} and ${VAR:-default}
func expandEnvVars(v reflect.Value) error {
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.String:
		original := v.String()
		expanded := envVarPattern.ReplaceAllStringFunc(original, func(match string) string {
			parts := envVarPattern.FindStringSubmatch(match)
			if len(parts) < 2 {
				return match
			}

			varName := parts[1]
			defaultValue := ""
			if len(parts) >= 4 {
				defaultValue = parts[3]
			}

			if val := os.Getenv(varName); val != "" {
				return val
			}

			if defaultValue != "" {
				return defaultValue
			}

			// If no default and var not set, return empty (will be caught by validation)
			return ""
		})
		v.SetString(expanded)

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if field.CanSet() {
				if err := expandEnvVars(field); err != nil {
					return err
				}
			}
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := expandEnvVars(v.Index(i)); err != nil {
				return err
			}
		}

	case reflect.Pointer:
		if !v.IsNil() {
			if err := expandEnvVars(v.Elem()); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateRequiredEnvVars checks if any required env vars are missing
func validateRequiredEnvVars(v reflect.Value, path string) error {
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.String:
		str := v.String()
		// Check if string contains unexpanded ${VAR} without default
		if strings.Contains(str, "${") && !strings.Contains(str, ":-") {
			matches := envVarPattern.FindAllStringSubmatch(str, -1)
			for _, match := range matches {
				if len(match) >= 2 && len(match) < 4 {
					varName := match[1]
					if os.Getenv(varName) == "" {
						return fmt.Errorf("required environment variable %s not set (referenced in %s)", varName, path)
					}
				}
			}
		}

	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldName := t.Field(i).Name
			fieldPath := path
			if fieldPath != "" {
				fieldPath += "."
			}
			fieldPath += fieldName

			if err := validateRequiredEnvVars(field, fieldPath); err != nil {
				return err
			}
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := validateRequiredEnvVars(v.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}

	case reflect.Pointer:
		if !v.IsNil() {
			if err := validateRequiredEnvVars(v.Elem(), path); err != nil {
				return err
			}
		}
	}

	return nil
}

// getDefaultConfig returns a glibConfig with sensible defaults
func getDefaultConfig() *glibConfig {
	cfg := &glibConfig{
		Version: "2",
		Verbose: false,
	}

	// Generate defaults
	cfg.Generate.Output = "generated"
	cfg.Generate.Package = "generated"
	cfg.Generate.Workers = 4
	cfg.Generate.Cache = true

	// Make defaults
	cfg.Make.Controllers = "controllers"
	cfg.Make.Providers = "providers"
	cfg.Make.Middleware = "middleware"

	// Watch defaults
	cfg.Watch.Debounce = 300
	cfg.Watch.ExcludeDirs = []string{"vendor", "node_modules", ".git", ".glib", "tmp"}
	cfg.Watch.IncludeFiles = []string{"*.go", "config.toml"}
	cfg.Watch.ExcludeFiles = []string{"*_test.go", "*.gen.go"}

	// Validation defaults
	cfg.Validation.Enabled = false
	cfg.Validation.Languages = []string{}
	cfg.Validation.DefaultLanguage = ""

	return cfg
}

// mergeWithDefaults merges user config with defaults
func mergeWithDefaults(userCfg *glibConfig) *glibConfig {
	defaults := getDefaultConfig()

	// If user didn't specify a field, use default
	if userCfg.Version == "" {
		userCfg.Version = defaults.Version
	}

	// Generate
	if userCfg.Generate.Output == "" {
		userCfg.Generate.Output = defaults.Generate.Output
	}
	if userCfg.Generate.Package == "" {
		userCfg.Generate.Package = defaults.Generate.Package
	}
	if userCfg.Generate.Workers == 0 {
		userCfg.Generate.Workers = defaults.Generate.Workers
	}
	// Cache defaults to true if not explicitly set to false

	// Make
	if userCfg.Make.Controllers == "" {
		userCfg.Make.Controllers = defaults.Make.Controllers
	}
	if userCfg.Make.Providers == "" {
		userCfg.Make.Providers = defaults.Make.Providers
	}
	if userCfg.Make.Middleware == "" {
		userCfg.Make.Middleware = defaults.Make.Middleware
	}

	// Watch
	if userCfg.Watch.Debounce == 0 {
		userCfg.Watch.Debounce = defaults.Watch.Debounce
	}
	if len(userCfg.Watch.ExcludeDirs) == 0 {
		userCfg.Watch.ExcludeDirs = defaults.Watch.ExcludeDirs
	}
	if len(userCfg.Watch.IncludeFiles) == 0 {
		userCfg.Watch.IncludeFiles = defaults.Watch.IncludeFiles
	}
	if len(userCfg.Watch.ExcludeFiles) == 0 {
		userCfg.Watch.ExcludeFiles = defaults.Watch.ExcludeFiles
	}

	return userCfg
}
