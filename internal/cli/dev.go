package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/azizndao/glib/internal/cli/ui"
	"github.com/azizndao/glib/internal/generator"
	"github.com/azizndao/glib/internal/scanner"
	"github.com/azizndao/glib/internal/validator"
	"github.com/spf13/cobra"
)

// ProcessManager manages the running server process
type ProcessManager struct {
	cmd     *exec.Cmd
	mu      sync.Mutex
	running bool
	done    chan struct{}
	quiet   bool // If true, don't forward stdout/stderr (for tests)
}

// Start launches the server process
func (pm *ProcessManager) Start(binaryPath string, port int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.running {
		return fmt.Errorf("process already running")
	}

	pm.cmd = exec.Command(binaryPath)
	pm.cmd.Env = append(os.Environ(), fmt.Sprintf("APP_PORT=%d", port))

	if !pm.quiet {
		pm.cmd.Stdout = os.Stdout
		pm.cmd.Stderr = os.Stderr
		pm.cmd.Stdin = os.Stdin
	}

	pm.done = make(chan struct{})

	if err := pm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	pm.running = true

	// Monitor process in background
	go func() {
		_ = pm.cmd.Wait()
		pm.mu.Lock()
		pm.running = false
		if pm.done != nil {
			close(pm.done)
		}
		pm.mu.Unlock()
	}()

	return nil
}

// Stop gracefully terminates the server
func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()

	if !pm.running || pm.cmd == nil || pm.cmd.Process == nil {
		pm.mu.Unlock()
		return nil
	}

	// Mark as not running immediately to prevent double-stop
	pm.running = false
	process := pm.cmd.Process
	done := pm.done
	pm.mu.Unlock()

	// Try graceful shutdown with SIGTERM
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, force kill
		_ = process.Kill()
		// Wait for process to finish
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	// Wait for process to exit (with timeout)
	select {
	case <-done:
		// Process exited cleanly
	case <-time.After(3 * time.Second):
		// Timeout - force kill
		_ = process.Kill()
		// Give it a moment to die
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// Restart stops the old process and starts a new one
func (pm *ProcessManager) Restart(binaryPath string, port int) error {
	if err := pm.Stop(); err != nil {
		return fmt.Errorf("failed to stop old process: %w", err)
	}

	// Give the OS a moment to release the port
	time.Sleep(200 * time.Millisecond)

	return pm.Start(binaryPath, port)
}

// IsRunning returns whether the server is running
func (pm *ProcessManager) IsRunning() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.running
}

func newDevCmd() *cobra.Command {
	var port int
	var verbose bool
	var workers int
	var noCache bool
	var debounce int

	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Start development server with hot reload",
		Long: `Start development server with automatic code generation and hot reload.

Features:
  - Incremental code generation (fast!)
  - Automatic rebuild on file changes
  - Native file watching (no external dependencies)
  - Process management with graceful restart
  - Press Ctrl+C to stop`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDev(port, verbose, workers, noCache, time.Duration(debounce)*time.Millisecond)
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "Server port (default: from glib.json or 8080)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed statistics")
	cmd.Flags().IntVar(&workers, "workers", 4, "Number of parallel workers (0=auto, -1=disable)")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Disable incremental caching")
	cmd.Flags().IntVar(&debounce, "debounce", 300, "Debounce duration in milliseconds")

	return cmd
}

func runDev(port int, verbose bool, workers int, noCache bool, debounce time.Duration) error {
	// Check if config exists
	if _, err := os.Stat("glib.json"); os.IsNotExist(err) {
		if _, err := os.Stat(".glibrc"); os.IsNotExist(err) {
			fmt.Println(ui.Error("No glib.json or .glibrc found - run 'glib init' first"))
			return fmt.Errorf("no glib.json or .glibrc found")
		}
	}

	// Load config
	cfg, err := loadGlibrc()
	if err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to load config: %v", err)))
		return err
	}

	// Determine port
	if port == 0 {
		if cfg.Dev.Port != 0 {
			port = cfg.Dev.Port
		} else {
			port = 8080
		}
	}

	// Determine output directory
	outputDir := cfg.Generate.Output
	if outputDir == "" {
		outputDir = "generated"
	}

	// Ensure tmp directory exists
	if err := os.MkdirAll("tmp", 0755); err != nil {
		return fmt.Errorf("failed to create tmp directory: %w", err)
	}

	// Ensure cache directory exists
	cacheDir := filepath.Join(".glib", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	binaryPath := "./tmp/main"

	fmt.Println(ui.Info(fmt.Sprintf("Starting Glib dev server on port %d", port)))
	fmt.Println()

	// Initial generation
	fmt.Println(ui.Info("Initial generation..."))
	if err := performGeneration(".", outputDir, cfg, workers, noCache, verbose, nil); err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Initial generation failed: %v", err)))
		return err
	}

	// Initial build
	fmt.Println()
	fmt.Println(ui.Info("Building application..."))
	if err := buildApp(binaryPath, verbose); err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Build failed: %v", err)))
		return err
	}
	fmt.Println(ui.Success("Build complete"))

	// Start server
	fmt.Println()
	pm := &ProcessManager{}
	if err := pm.Start(binaryPath, port); err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to start server: %v", err)))
		return err
	}
	fmt.Println(ui.Success(fmt.Sprintf("Server started on http://localhost:%d", port)))

	// Create file watcher
	watcher, err := NewFileWatcher(".", outputDir, debounce)
	if err != nil {
		pm.Stop()
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	changes, watchErrors, err := watcher.Start()
	if err != nil {
		pm.Stop()
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	// Count watched files
	fileCount, _ := watcher.CountWatchedFiles()
	fmt.Println()
	fmt.Printf("%s Watching %d files... %s\n",
		ui.Cyan+ui.IconWatch+ui.Reset,
		fileCount,
		ui.Muted("(Press Ctrl+C to stop)"))

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Main event loop
	for {
		select {
		case <-sigChan:
			fmt.Println()
			fmt.Println(ui.Info("Shutting down..."))
			watcher.Stop()
			pm.Stop()
			fmt.Println(ui.Success("Goodbye!"))
			return nil

		case changedFiles := <-changes:
			fmt.Println()
			printSeparator()
			fmt.Printf("%s %s Changes detected:\n",
				ui.Yellow+ui.IconGenerate+ui.Reset,
				ui.Muted(time.Now().Format("15:04:05")))
			for _, file := range changedFiles {
				relPath, _ := filepath.Rel(".", file)
				fmt.Printf("  %s %s\n", ui.IconBullet, ui.Muted(relPath))
			}
			fmt.Println()

			if err := handleReload(pm, binaryPath, port, ".", outputDir, cfg, workers, noCache, verbose, changedFiles); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Reload failed: %v", err)))
				if pm.IsRunning() {
					fmt.Println(ui.Warning("Previous server still running"))
				}
			}

		case err := <-watchErrors:
			fmt.Println(ui.Warning(fmt.Sprintf("Watch error: %v", err)))
		}
	}
}

// handleReload performs incremental generation, build, and restart
func handleReload(pm *ProcessManager, binaryPath string, port int, projectDir, outputDir string, cfg *glibConfig, workers int, noCache bool, verbose bool, changedFiles []string) error {
	start := time.Now()

	// Generate code
	if err := performGeneration(projectDir, outputDir, cfg, workers, noCache, verbose, changedFiles); err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	// Build
	fmt.Println(ui.Info("Building..."))
	buildStart := time.Now()
	if err := buildApp(binaryPath, verbose); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	buildDuration := time.Since(buildStart)
	fmt.Printf("  %s Build complete (%dms)\n", ui.IconCheck, buildDuration.Milliseconds())

	// Restart server
	if err := pm.Restart(binaryPath, port); err != nil {
		return fmt.Errorf("failed to restart server: %w", err)
	}

	totalDuration := time.Since(start)
	fmt.Println(ui.Success(fmt.Sprintf("Server restarted (%dms total)", totalDuration.Milliseconds())))
	fmt.Println()
	fmt.Printf("%s %s\n",
		ui.Cyan+ui.IconWatch+ui.Reset,
		ui.Muted("Watching for changes..."))

	return nil
}

// performGeneration runs code generation (incremental or full)
func performGeneration(projectDir, outputDir string, cfg *glibConfig, workers int, noCache bool, verbose bool, changedFiles []string) error {
	// Determine package name
	pkgName := cfg.Generate.Package
	if pkgName == "" {
		pkgName = "generated"
	}

	// Configure scanner options
	var scanOpts []scanner.ScannerOption

	// Enable caching unless explicitly disabled
	if !noCache {
		cacheDir := filepath.Join(".glib", "cache")
		scanOpts = append(scanOpts, scanner.WithCache(cacheDir))
	}

	// Enable parallel scanning if workers > 0
	if workers > 0 {
		scanOpts = append(scanOpts, scanner.WithParallel(workers))
	}

	// Create scanner
	scan, err := scanner.New(projectDir, scanOpts...)
	if err != nil {
		return fmt.Errorf("failed to create scanner: %w", err)
	}

	// Scan (incremental or full)
	var project *scanner.Project
	scanStart := time.Now()

	if changedFiles != nil && len(changedFiles) > 0 && !noCache {
		// Incremental scan
		fmt.Println(ui.Info(fmt.Sprintf("Incremental scan (%d files)...", len(changedFiles))))
		project, err = scan.ScanIncremental(changedFiles)
	} else {
		// Full scan
		fmt.Println(ui.Info("Scanning..."))
		project, err = scan.Scan()
	}

	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	scanDuration := time.Since(scanStart)

	// Show statistics if verbose
	stats := scan.Stats()
	if verbose {
		printDevScanStats(stats, scanDuration)
	} else {
		// Compact output
		fmt.Printf("  %s Scanned: %d providers, %d controllers, %d middleware (%dms)\n",
			ui.IconCheck,
			stats.Providers,
			stats.Controllers,
			stats.Middleware,
			scanDuration.Milliseconds())
	}

	// Validate with incremental validation
	validationStart := time.Now()
	if !noCache {
		cacheDir := filepath.Join(".glib", "cache")
		incVal := validator.NewIncrementalValidator(cacheDir)
		if err := incVal.ValidateIncremental(project); err != nil {
			if validationErr, ok := err.(*validator.ValidationErrors); ok {
				for _, verr := range validationErr.Errors {
					fmt.Printf("  %s %s\n", ui.IconBullet, verr.Message)
				}
			}
			return fmt.Errorf("validation failed: %w", err)
		}
		validationDuration := time.Since(validationStart)

		if verbose {
			valStats := incVal.Stats()
			fmt.Printf("  %s Validation: %d components (%dms)\n",
				ui.IconCheck,
				valStats.ComponentsValidated,
				validationDuration.Milliseconds())
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
		fmt.Printf("  %s Validation passed (%dms)\n", ui.IconCheck, validationDuration.Milliseconds())
	}

	// Generate code
	genStart := time.Now()
	validationEnabled := cfg.Validation.Enabled || len(cfg.Validation.Languages) > 0
	validationCfg := generator.ValidationConfig{
		Enabled:         validationEnabled,
		Languages:       cfg.Validation.Languages,
		DefaultLanguage: cfg.Validation.DefaultLanguage,
	}

	gen := generator.NewWithValidation(project, outputDir, pkgName, validationCfg)
	if err := gen.Generate(); err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}
	genDuration := time.Since(genStart)

	// Format generated code
	if err := formatGeneratedCode(outputDir, false); err != nil {
		// Non-fatal - just warn
		if verbose {
			fmt.Println(ui.Warning(fmt.Sprintf("Failed to format code: %v", err)))
		}
	}

	fmt.Printf("  %s Generation complete (%dms)\n", ui.IconCheck, genDuration.Milliseconds())

	return nil
}

// buildApp builds the application
func buildApp(outputPath string, verbose bool) error {
	cmd := exec.Command("go", "build", "-o", outputPath, ".")
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// Capture errors only
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// printDevScanStats prints detailed scan statistics
func printDevScanStats(stats scanner.ScanStats, duration time.Duration) {
	fmt.Printf("  %s Providers: %d, Controllers: %d, Middleware: %d, Handlers: %d\n",
		ui.IconBullet,
		stats.Providers,
		stats.Controllers,
		stats.Middleware,
		stats.Handlers)
	fmt.Printf("  %s Files: %d scanned, %d hits (%.1f%%), %d misses\n",
		ui.IconBullet,
		stats.FilesScanned,
		stats.CacheHits,
		float64(stats.CacheHits)/float64(stats.FilesScanned)*100,
		stats.CacheMisses)
	fmt.Printf("  %s Duration: %dms\n", ui.IconBullet, duration.Milliseconds())
}

// printSeparator prints a visual separator
func printSeparator() {
	fmt.Println(ui.Gray + "─────────────────────────────────────────" + ui.Reset)
}
