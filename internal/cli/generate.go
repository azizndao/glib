package cli

import (
	"fmt"
	"os"

	"github.com/azizndao/glib/internal/cli/ui"
	"github.com/azizndao/glib/internal/scanner"
	"github.com/azizndao/glib/internal/validator"
	"github.com/spf13/cobra"
)

type generateOptions struct {
	dir        string
	output     string
	verbose    bool
	watch      bool
	workers    int
	noCache    bool
	clearCache bool
}

func newGenerateCmd() *cobra.Command {
	opts := &generateOptions{}

	cmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen"},
		Short:   "Generate code from annotations",
		Long: `Scan Go source code for annotations and generate code.

Generates:
  - DI container (generated/di.gen.go)
  - Route registration (generated/routes.gen.go)  
  - Request parsers (generated/parsers.gen.go)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(opts)
		},
	}

	cmd.Flags().StringVar(&opts.dir, "dir", ".", "Project root directory")
	cmd.Flags().StringVar(&opts.output, "output", "", "Output directory (default: from .glib.toml or 'generated')")
	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "Verbose output (default: from .glib.toml or false)")
	cmd.Flags().BoolVar(&opts.watch, "watch", false, "Watch mode")
	cmd.Flags().IntVar(&opts.workers, "workers", 0, "Number of parallel workers (default: from .glib.toml or 4)")
	cmd.Flags().BoolVar(&opts.noCache, "no-cache", false, "Disable file caching (default: cache enabled from .glib.toml)")
	cmd.Flags().BoolVar(&opts.clearCache, "clear-cache", false, "Clear cache before scanning")

	return cmd
}

func runGenerate(opts *generateOptions) error {
	if opts.watch {
		return fmt.Errorf("watch mode not implemented yet")
	}

	// Change to project directory
	if opts.dir != "." {
		if err := os.Chdir(opts.dir); err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}
	}

	// Load config from environment variables and defaults
	cfg, err := loadConfigs()
	if err != nil {
		return err
	}

	// Priority resolution: CLI args > environment variables > defaults

	// Verbose: CLI flag OR config value
	if !opts.verbose {
		opts.verbose = cfg.Verbose
	}

	// Workers: CLI flag (if not default/0) OR config value
	if opts.workers == 0 {
		opts.workers = cfg.Generate.Workers
	}

	// Cache: CLI flag --no-cache OR config value
	if !opts.noCache {
		opts.noCache = !cfg.Generate.Cache
	}

	// Determine output directory
	outputDir := opts.output
	if outputDir == "" {
		outputDir = cfg.Generate.Output
		if outputDir == "" {
			outputDir = "generated"
		}
	}

	// Determine package name
	pkgName := cfg.Generate.Package
	if pkgName == "" {
		pkgName = "generated"
	}

	return runGenerateSimple(outputDir, pkgName, cfg, opts)
}

// runGenerateSimple performs code generation with simple output
func runGenerateSimple(outputDir, pkgName string, cfg *glibConfig, opts *generateOptions) error {
	codegenOpts := &CodegenOptions{
		ProjectDir:   ".",
		OutputDir:    outputDir,
		PackageName:  pkgName,
		Workers:      opts.workers,
		NoCache:      opts.noCache,
		Verbose:      opts.verbose,
		ClearCache:   opts.clearCache,
		ChangedFiles: nil,  // Always full scan for generate command
		ShowProgress: true, // Generate command shows detailed output
	}

	return PerformCodeGeneration(cfg, codegenOpts)
}

// scanWithProgress runs scanner with streaming progress updates
func scanWithProgress(scan *scanner.Scanner, opts *generateOptions) (*scanner.Project, error) {
	events := make(chan scanner.StreamEvent, 100)

	// Start goroutine to consume events
	done := make(chan struct{})
	go func() {
		defer close(done)
		for event := range events {
			switch event.Type {
			case scanner.EventProvider:
				fmt.Printf("  %s %s %s\n",
					ui.Cyan+ui.IconProvider+ui.Reset,
					ui.Mutedf("Provider:"),
					ui.Primaryf("%s.%s", event.Provider.PackageName, event.Provider.Name))
			case scanner.EventController:
				fmt.Printf("  %s %s %s\n",
					ui.Blue+ui.IconController+ui.Reset,
					ui.Mutedf("Controller:"),
					ui.Primaryf("%s.%s", event.Controller.PackageName, event.Controller.Name))
			case scanner.EventMiddleware:
				fmt.Printf("  %s %s %s\n",
					ui.Yellow+ui.IconMiddleware+ui.Reset,
					ui.Mutedf("Middleware:"),
					ui.Primaryf("%s.%s", event.Middleware.PackageName, event.Middleware.Name))
			case scanner.EventConfig:
				fmt.Printf("  %s %s %s\n",
					ui.Gray+ui.IconConfig+ui.Reset,
					ui.Mutedf("Config:"),
					ui.Primaryf("%s", event.Config.PackageName))
			case scanner.EventProgress:
				// Could add progress bar here
			case scanner.EventError:
				fmt.Println(ui.Warningf("Warning: %v", event.Error))
			}
		}
	}()

	// Run scan with streaming
	project, err := scan.ScanWithStream(events)
	<-done // Wait for event processing to complete

	return project, err
}

// printScanSummary prints a formatted summary table of scan statistics
func printScanSummary(scanStats scanner.ScanStats, valStats *validator.ValidationStats, cacheEnabled bool) {
	fmt.Println(ui.BoldTextf("  Scan Summary:"))
	fmt.Println(ui.Mutedf("  ┌────────────────────────┬──────────────┐"))

	// Components found
	fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" %12s "+ui.Mutedf("│")+"\n",
		"Providers", ui.Cyan+fmt.Sprintf("%d", scanStats.Providers)+ui.Reset)
	fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" %12s "+ui.Mutedf("│")+"\n",
		"Controllers", ui.Blue+fmt.Sprintf("%d", scanStats.Controllers)+ui.Reset)
	fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" %12s "+ui.Mutedf("│")+"\n",
		"Middleware", ui.Yellow+fmt.Sprintf("%d", scanStats.Middleware)+ui.Reset)
	fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" %12s "+ui.Mutedf("│")+"\n",
		"Handlers", ui.Green+fmt.Sprintf("%d", scanStats.Handlers)+ui.Reset)

	if cacheEnabled && scanStats.FilesScanned > 0 {
		fmt.Println(ui.Mutedf("  ├────────────────────────┼──────────────┤"))

		hitRate := float64(scanStats.CacheHits) * 100 / float64(scanStats.FilesScanned)
		fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" %12d "+ui.Mutedf("│")+"\n",
			"Files Scanned", scanStats.FilesScanned)

		cacheHitStr := fmt.Sprintf("%d", scanStats.CacheHits)
		fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" "+ui.Green+"%-12s"+ui.Reset+" "+ui.Mutedf("│")+"\n",
			"Cache Hits", fmt.Sprintf("%s (%.1f%%)", cacheHitStr, hitRate))

		fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" "+ui.Yellow+"%-12d"+ui.Reset+" "+ui.Mutedf("│")+"\n",
			"Cache Misses", scanStats.CacheMisses)
	}

	if valStats != nil && cacheEnabled {
		fmt.Println(ui.Mutedf("  ├────────────────────────┼──────────────┤"))

		valHitRate := float64(valStats.CacheHits) * 100 / float64(valStats.ComponentsValidated)
		fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" %12d "+ui.Mutedf("│")+"\n",
			"Components Validated", valStats.ComponentsValidated)

		valCacheStr := fmt.Sprintf("%d", valStats.CacheHits)
		fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" "+ui.Green+"%-12s"+ui.Reset+" "+ui.Mutedf("│")+"\n",
			"Validation Cached", fmt.Sprintf("%s (%.1f%%)", valCacheStr, valHitRate))
	}

	fmt.Println(ui.Mutedf("  └────────────────────────┴──────────────┘"))
}
