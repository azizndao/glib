package cli

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/azizndao/glib/internal/cli/ui"
)

const configFile = ".config.toml"

// glibConfig represents the CLI configuration
// Configuration is loaded from: CLI args > .config.toml > default
type glibConfig struct {
	Version  string `toml:"version"`
	Verbose  bool   `toml:"verbose"`
	Generate struct {
		Output  string `toml:"output"`
		Package string `toml:"package"`
		Workers int    `toml:"workers"`
		Cache   bool   `toml:"cache"`
	} `toml:"generate"`
	Make struct {
		Controllers string `toml:"controllers"`
		Providers   string `toml:"providers"`
		Middleware  string `toml:"middleware"`
	} `toml:"make"`
	Watch struct {
		Debounce     int      `toml:"debounce"`
		ExcludeDirs  []string `toml:"exclude_dirs"`
		IncludeFiles []string `toml:"include_files"`
		ExcludeFiles []string `toml:"exclude_files"`
	} `toml:"watch"`
	Validation struct {
		Enabled         bool     `toml:"enabled"`
		Languages       []string `toml:"languages"`
		DefaultLanguage string   `toml:"default_language"`
	} `toml:"validation"`
	I18n struct {
		Enabled          bool     `toml:"enabled"`
		LocalesDir       string   `toml:"locales_dir"`
		DefaultLocale    string   `toml:"default_locale"`
		SupportedLocales []string `toml:"supported_locales"`
		DetectFrom       []string `toml:"detect_from"`
		QueryParam       string   `toml:"query_param"`
	} `toml:"i18n"`
}

// getDefaultConfig returns a glibConfig with sensible defaults
// Defaults can be overridden by environment variables
func getDefaultConfig() *glibConfig {
	cfg := &glibConfig{Version: "2", Verbose: false}

	// Generate defaults
	cfg.Generate.Output = "generated"
	cfg.Generate.Package = "generated"
	cfg.Generate.Workers = 4
	cfg.Generate.Cache = true

	// Make defaults
	cfg.Make.Controllers = "controllers"
	cfg.Make.Providers = "services"
	cfg.Make.Middleware = "middleware"

	// Watch defaults (also used by scanner for file filtering)
	cfg.Watch.Debounce = 300
	cfg.Watch.ExcludeDirs = []string{"vendor", "node_modules", ".git", ".glib", "tmp"}
	cfg.Watch.IncludeFiles = []string{"*.go"}
	cfg.Watch.ExcludeFiles = []string{"*_test.go", "*.gen.go"}

	// Validation defaults
	cfg.Validation.Enabled = false
	cfg.Validation.Languages = []string{"en"}
	cfg.Validation.DefaultLanguage = "en"

	// I18n defaults
	cfg.I18n.Enabled = false
	cfg.I18n.LocalesDir = "locales"
	cfg.I18n.DefaultLocale = "en"
	cfg.I18n.SupportedLocales = []string{"en"}
	cfg.I18n.DetectFrom = []string{"header", "query"}
	cfg.I18n.QueryParam = "lang"

	return cfg
}

func loadConfigs() (*glibConfig, error) {
	cfg := getDefaultConfig()
	file, err := os.ReadFile(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file doesn't exist, create it with defaults
		fmt.Println(ui.Infof("Writing default config to %s", configFile))
		jsonStr, err := toml.Marshal(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal default config: %w", err)
		}
		if err := os.WriteFile(configFile, jsonStr, 0o600); err != nil {
			return nil, fmt.Errorf("failed to write config file: %w", err)
		}
		// Return the default config (no need to unmarshal)
		return cfg, nil
	}

	// Parse existing config file
	if err := toml.Unmarshal(file, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate config values
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// validateConfig validates configuration values
func validateConfig(cfg *glibConfig) error {
	if cfg.Watch.Debounce < 0 {
		return fmt.Errorf("watch.debounce cannot be negative: %d", cfg.Watch.Debounce)
	}
	if cfg.Generate.Workers < 0 {
		return fmt.Errorf("generate.workers cannot be negative: %d", cfg.Generate.Workers)
	}
	return nil
}
