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
	"github.com/spf13/cobra"
)

const (
	ProcessGracefulTimeout = 3 * time.Second
	ProcessCleanupDelay    = 100 * time.Millisecond
	PortReleaseDelay       = 200 * time.Millisecond
	ShutdownTimeout        = 5 * time.Second
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
		time.Sleep(ProcessCleanupDelay)
		return nil
	}

	// Wait for process to exit (with timeout)
	select {
	case <-done:
		// Process exited cleanly
	case <-time.After(ProcessGracefulTimeout):
		// Timeout - force kill
		_ = process.Kill()
		// Give it a moment to die
		time.Sleep(ProcessCleanupDelay)
	}

	return nil
}

// Restart stops the old process and starts a new one
func (pm *ProcessManager) Restart(binaryPath string, port int) error {
	if err := pm.Stop(); err != nil {
		return fmt.Errorf("failed to stop old process: %w", err)
	}

	// Give the OS a moment to release the port
	time.Sleep(PortReleaseDelay)

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
		Use:     "dev",
		Aliases: []string{"serve"},
		Short:   "Start development server with hot reload",
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

	cmd.Flags().IntVar(&port, "port", 0, "Server port (default: from .env or 8080)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed statistics (default: from .glib.toml or false)")
	cmd.Flags().IntVar(&workers, "workers", 0, "Number of parallel workers (default: from .glib.toml or 4)")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Disable incremental caching (default: cache enabled from .glib.toml)")
	cmd.Flags().IntVar(&debounce, "debounce", 0, "Debounce duration in milliseconds (default: from .glib.toml or 300)")

	return cmd
}

func runDev(port int, verbose bool, workers int, noCache bool, debounce time.Duration) error {
	// Load config from environment variables and defaults
	cfg, err := loadConfigs()
	if err != nil {
		return err
	}

	// Priority resolution: CLI args > environment variables > defaults

	// Verbose: CLI flag OR config value
	if !verbose {
		verbose = cfg.Verbose
	}

	// Workers: CLI flag (if not default/0) OR config value
	if workers == 0 {
		workers = cfg.Generate.Workers
	}

	// Cache: CLI flag --no-cache OR config value
	cache := !noCache
	if !noCache {
		cache = cfg.Generate.Cache
	}
	noCache = !cache

	// Debounce: CLI flag (if not default/0) OR config value
	if debounce == 0 {
		debounce = time.Duration(cfg.Watch.Debounce) * time.Millisecond
	}

	// Port: CLI flag OR default (port is managed via .env, not config)
	if port == 0 {
		port = 8080
	}

	// Determine output directory
	outputDir := cfg.Generate.Output
	if outputDir == "" {
		outputDir = "generated"
	}

	// Ensure tmp directory exists
	if err := os.MkdirAll("tmp", 0o755); err != nil {
		return fmt.Errorf("failed to create tmp directory: %w", err)
	}

	// Ensure cache directory exists
	cacheDir := filepath.Join(".glib", "cache")
	if err := ensureCacheDir(cacheDir); err != nil {
		return err
	}

	binaryPath := "./tmp/main"

	fmt.Println(ui.Infof("Starting Glib dev server on port %d", port))
	fmt.Println()

	// Initial generation
	fmt.Println(ui.Infof("Initial generation..."))
	if err := performGeneration(".", outputDir, cfg, workers, noCache, verbose, nil); err != nil {
		fmt.Println(ui.Errorf("Initial generation failed: %v", err))
		return err
	}

	// Initial build
	fmt.Println()
	fmt.Println(ui.Infof("Building application..."))
	if err := buildApp(binaryPath, verbose); err != nil {
		fmt.Println(ui.Errorf("Build failed: %v", err))
		return err
	}
	fmt.Println(ui.Successf("Build complete"))

	// Start server
	fmt.Println()
	pm := &ProcessManager{}
	if err := pm.Start(binaryPath, port); err != nil {
		fmt.Println(ui.Errorf("Failed to start server: %v", err))
		return err
	}
	fmt.Println(ui.Successf("Server started on http://localhost:%d", port))

	// Create file watcher
	watchCfg := &WatchConfig{
		Debounce:     debounce,
		ExcludeDirs:  cfg.Watch.ExcludeDirs,
		IncludeFiles: cfg.Watch.IncludeFiles,
		ExcludeFiles: cfg.Watch.ExcludeFiles,
	}
	watcher, err := NewFileWatcher(".", outputDir, watchCfg)
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
		ui.Mutedf("(Press Ctrl+C to stop)"))

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Main event loop
	for {
		select {
		case <-sigChan:
			fmt.Println()
			fmt.Println(ui.Infof("Shutting down..."))

			// Shutdown with timeout to prevent hanging
			done := make(chan struct{})
			go func() {
				watcher.Stop()
				pm.Stop()
				close(done)
			}()

			select {
			case <-done:
				fmt.Println(ui.Successf("Goodbye!"))
			case <-time.After(ShutdownTimeout):
				fmt.Println(ui.Warningf("Shutdown timeout - forcing exit"))
			}
			return nil

		case changedFiles := <-changes:
			fmt.Println()
			printSeparator()
			fmt.Printf("%s %s Changes detected:\n",
				ui.Yellow+ui.IconGenerate+ui.Reset,
				ui.Mutedf("%s", time.Now().Format("15:04:05")))
			for _, file := range changedFiles {
				relPath, _ := filepath.Rel(".", file)
				fmt.Printf("  %s %s\n", ui.IconBullet, ui.Mutedf("%s", relPath))
			}
			fmt.Println()

			if err := handleReload(pm, binaryPath, port, ".", outputDir, cfg, workers, noCache, verbose, changedFiles); err != nil {
				fmt.Println(ui.Errorf("Reload failed: %v", err))
				if pm.IsRunning() {
					fmt.Println(ui.Warningf("Previous server still running"))
				}
			}

		case err := <-watchErrors:
			fmt.Println(ui.Warningf("Watch error: %v", err))
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
	fmt.Println(ui.Infof("Building..."))
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
	fmt.Println(ui.Successf("Server restarted (%dms total)", totalDuration.Milliseconds()))
	fmt.Println()
	fmt.Printf("%s %s\n",
		ui.Cyan+ui.IconWatch+ui.Reset,
		ui.Mutedf("Watching for changes..."))

	return nil
}

// performGeneration runs code generation (incremental or full)
func performGeneration(projectDir, outputDir string, cfg *glibConfig, workers int, noCache bool, verbose bool, changedFiles []string) error {
	opts := &CodegenOptions{
		ProjectDir:   projectDir,
		OutputDir:    outputDir,
		PackageName:  "", // Will use cfg.Generate.Package
		Workers:      workers,
		NoCache:      noCache,
		Verbose:      verbose,
		ClearCache:   false, // Dev mode doesn't clear cache
		ChangedFiles: changedFiles,
		ShowProgress: false, // Dev mode uses compact output
	}

	return PerformCodeGeneration(cfg, opts)
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
