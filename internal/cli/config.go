package cli

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/azizndao/glib/internal/cli/ui"
)

const glibConfigFile = ".glib.toml"

// glibConfig represents the CLI configuration
// Configuration is loaded from: CLI args > .glib.toml > default
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
func _getDefaultConfig() *glibConfig {
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
	cfg := _getDefaultConfig()
	file, err := os.ReadFile(glibConfigFile)
	if err != nil {
		fmt.Println(ui.Infof("Writting default config to %s", glibConfigFile))
		jsonStr, err := toml.Marshal(cfg)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(glibConfigFile, jsonStr, 0o644); err != nil {
			return nil, err
		}
	}

	if err := toml.Unmarshal(file, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
