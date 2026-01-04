package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/azizndao/glib/internal/cli/ui"
	"github.com/azizndao/glib/internal/generator"
	"github.com/azizndao/glib/internal/scanner"
	"github.com/azizndao/glib/internal/validator"
)

// CodegenOptions holds all options for code generation pipeline
type CodegenOptions struct {
	ProjectDir   string   // Project root directory
	OutputDir    string   // Output directory for generated code
	PackageName  string   // Package name for generated code
	Workers      int      // Number of parallel workers (0 = disable parallel)
	NoCache      bool     // Disable all caching
	Verbose      bool     // Show detailed statistics
	ClearCache   bool     // Clear cache before generation
	ChangedFiles []string // Files that changed (nil = full scan, non-nil = incremental)
	ShowProgress bool     // Show streaming progress (for verbose generate)
}

// PerformCodeGeneration runs the complete code generation pipeline
// Consolidates logic from dev.go performGeneration() and generate.go runGenerateSimple()
func PerformCodeGeneration(cfg *glibConfig, opts *CodegenOptions) error {
	start := time.Now()

	// Clear cache if requested
	if opts.ClearCache {
		cacheDir := filepath.Join(opts.ProjectDir, ".glib", "cache")
		if err := os.RemoveAll(cacheDir); err != nil && !os.IsNotExist(err) {
			if opts.Verbose {
				fmt.Println(ui.Warningf("Failed to clear cache: %v", err))
			}
		} else if opts.Verbose {
			fmt.Println(ui.Infof("Cache cleared"))
		}
	}

	// Determine package name
	pkgName := opts.PackageName
	if pkgName == "" {
		pkgName = cfg.Generate.Package
	}
	if pkgName == "" {
		pkgName = "generated"
	}

	// Configure scanner options
	var scanOpts []scanner.ScannerOption

	// Enable caching unless explicitly disabled
	if !opts.NoCache {
		cacheDir := filepath.Join(opts.ProjectDir, ".glib", "cache")
		scanOpts = append(scanOpts, scanner.WithCache(cacheDir))
		if opts.Verbose && opts.ShowProgress {
			fmt.Println(ui.Infof("File caching enabled"))
		}
	}

	// Enable parallel scanning if workers > 0
	if opts.Workers > 0 {
		scanOpts = append(scanOpts, scanner.WithParallel(opts.Workers))
		if opts.Verbose && opts.ShowProgress {
			fmt.Printf("  %s Parallel scanning with %d workers\n", ui.IconBullet, opts.Workers)
		}
	}

	// Apply file filtering from watch config
	if len(cfg.Watch.ExcludeDirs) > 0 {
		scanOpts = append(scanOpts, scanner.WithExcludeDirs(cfg.Watch.ExcludeDirs))
	}
	// Note: We don't pass include_files to scanner because it always scans *.go files
	// The watch config's include_files is for file watching (e.g., *.toml for locale changes)
	if len(cfg.Watch.ExcludeFiles) > 0 {
		scanOpts = append(scanOpts, scanner.WithExcludeFiles(cfg.Watch.ExcludeFiles))
	}

	// Enable i18n scanning if configured
	if cfg.I18n.Enabled && cfg.I18n.LocalesDir != "" {
		scanOpts = append(scanOpts, scanner.WithI18n(cfg.I18n.LocalesDir))
		if opts.Verbose && opts.ShowProgress {
			fmt.Printf("  %s I18n scanning enabled (locales: %s)\n", ui.IconBullet, cfg.I18n.LocalesDir)
		}
	}

	// Create scanner
	scan, err := scanner.New(opts.ProjectDir, scanOpts...)
	if err != nil {
		return fmt.Errorf("failed to create scanner: %w", err)
	}

	// Scan (incremental or full)
	var project *scanner.Project
	scanStart := time.Now()

	isIncremental := len(opts.ChangedFiles) > 0 && !opts.NoCache

	if isIncremental {
		// Incremental scan
		if opts.ShowProgress {
			fmt.Println(ui.Infof("Incremental scan (%d files)...", len(opts.ChangedFiles)))
		} else {
			fmt.Println(ui.Infof("Incremental scan (%d files)...", len(opts.ChangedFiles)))
		}
		project, err = scan.ScanIncremental(opts.ChangedFiles)
	} else {
		// Full scan
		if opts.ShowProgress {
			fmt.Println(ui.Infof("Scanning project..."))
		} else {
			fmt.Println(ui.Infof("Scanning..."))
		}
		project, err = scan.Scan()
	}

	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	scanDuration := time.Since(scanStart)

	// Show scan statistics
	stats := scan.Stats()
	if opts.Verbose {
		if opts.ShowProgress {
			// Detailed stats for generate command - show full breakdown
			fmt.Printf("  %s Files scanned: %d\n", ui.IconBullet, stats.FilesScanned)
			fmt.Printf("  %s Providers: %d\n", ui.IconBullet, stats.Providers)
			fmt.Printf("  %s Controllers: %d\n", ui.IconBullet, stats.Controllers)
			fmt.Printf("  %s Middleware: %d\n", ui.IconBullet, stats.Middleware)
			fmt.Printf("  %s Duration: %dms\n", ui.IconBullet, scanDuration.Milliseconds())
		} else {
			// Compact stats for dev mode
			printDevScanStats(stats, scanDuration)
		}
	} else if !isIncremental && opts.ShowProgress {
		// Compact output for non-incremental non-verbose generate
		fmt.Printf("  %s Scanned: %d providers, %d controllers, %d middleware (%dms)\n",
			ui.IconCheck,
			stats.Providers,
			stats.Controllers,
			stats.Middleware,
			scanDuration.Milliseconds())
	} else if !isIncremental {
		// Compact output for dev mode
		fmt.Printf("  %s Scanned: %d providers, %d controllers, %d middleware (%dms)\n",
			ui.IconCheck,
			stats.Providers,
			stats.Controllers,
			stats.Middleware,
			scanDuration.Milliseconds())
	}

	// Validate with incremental validation if caching enabled
	validationStart := time.Now()
	var valStats *validator.ValidationStats

	if !opts.NoCache {
		cacheDir := filepath.Join(opts.ProjectDir, ".glib", "cache")
		incVal := validator.NewIncrementalValidator(cacheDir)
		if err := incVal.ValidateIncremental(project); err != nil {
			if validationErr, ok := err.(*validator.ValidationErrors); ok {
				for _, verr := range validationErr.Errors {
					fmt.Printf("  %s %s\n", ui.IconBullet, verr.Message)
				}
			}
			return fmt.Errorf("validation failed: %w", err)
		}
		vstats := incVal.Stats()
		valStats = &vstats
		validationDuration := time.Since(validationStart)

		if opts.Verbose {
			fmt.Printf("  %s Validation: %d components (%dms)\n",
				ui.IconCheck,
				valStats.ComponentsValidated,
				validationDuration.Milliseconds())
		} else if opts.ShowProgress {
			fmt.Printf("  %s Validation passed (%dms)\n", ui.IconCheck, validationDuration.Milliseconds())
		} else {
			fmt.Printf("  %s Validation passed (%dms)\n", ui.IconCheck, validationDuration.Milliseconds())
		}
	} else {
		// Regular validation
		val := validator.New()
		if err := val.Validate(project); err != nil {
			for _, verr := range val.Errors() {
				fmt.Printf("  %s %s\n", ui.IconBullet, verr.Message)
			}
			return fmt.Errorf("validation failed: %w", err)
		}
		validationDuration := time.Since(validationStart)
		if opts.ShowProgress {
			fmt.Printf("  %s Validation passed (%dms)\n", ui.IconCheck, validationDuration.Milliseconds())
		} else {
			fmt.Printf("  %s Validation passed (%dms)\n", ui.IconCheck, validationDuration.Milliseconds())
		}
	}

	// Build validation config from environment/CLI settings
	validationEnabled := cfg.Validation.Enabled || len(cfg.Validation.Languages) > 0
	validationCfg := generator.ValidationConfig{
		Enabled:         validationEnabled,
		Languages:       cfg.Validation.Languages,
		DefaultLanguage: cfg.Validation.DefaultLanguage,
	}

	// Build i18n config from environment/CLI settings
	i18nCfg := generator.I18nConfig{
		Enabled:          cfg.I18n.Enabled,
		LocalesDir:       cfg.I18n.LocalesDir,
		DefaultLocale:    cfg.I18n.DefaultLocale,
		SupportedLocales: cfg.I18n.SupportedLocales,
		DetectFrom:       cfg.I18n.DetectFrom,
		QueryParam:       cfg.I18n.QueryParam,
	}

	// Generate code
	genStart := time.Now()
	gen := generator.NewWithValidationAndI18n(project, opts.OutputDir, pkgName, validationCfg, i18nCfg)
	if err := gen.Generate(); err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}
	genDuration := time.Since(genStart)

	// Format generated code
	if err := FormatGeneratedCode(opts.OutputDir, opts.Verbose && opts.ShowProgress); err != nil {
		// Non-fatal - just warn
		if opts.Verbose {
			fmt.Println(ui.Warningf("Failed to format code: %v", err))
		}
	}

	if opts.ShowProgress {
		duration := time.Since(start)
		fmt.Println(ui.Successf("Generation complete (%dms)", duration.Milliseconds()))
	} else {
		fmt.Printf("  %s Generation complete (%dms)\n", ui.IconCheck, genDuration.Milliseconds())
	}

	return nil
}

// FormatGeneratedCode runs goimports and gofmt on generated files
// Moved from generate.go to be shared between commands
func FormatGeneratedCode(outputDir string, verbose bool) error {
	// Find all .gen.go files in output directory
	pattern := filepath.Join(outputDir, "*.gen.go")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find generated files: %w", err)
	}

	if len(files) == 0 {
		return nil
	}

	if verbose {
		fmt.Println(ui.Infof("Formatting generated code..."))
	}

	// Try goimports first (removes unused imports and formats)
	if err := runGoImports(files, verbose); err != nil {
		// Fall back to gofmt if goimports is not available
		if verbose {
			fmt.Println(ui.Warningf("goimports not found, using gofmt only"))
		}
		return runGoFmt(files, verbose)
	}

	return nil
}

// runGoImports runs goimports on the specified files
func runGoImports(files []string, verbose bool) error {
	args := append([]string{"-w"}, files...)
	cmd := exec.Command("goimports", args...)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// runGoFmt runs gofmt on the specified files
func runGoFmt(files []string, verbose bool) error {
	args := append([]string{"-w"}, files...)
	cmd := exec.Command("gofmt", args...)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}
