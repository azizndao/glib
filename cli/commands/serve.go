package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

// NewServeCommand creates the "glib serve" command for running development server
func NewServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the development server",
		Long: `Start the glib development server with hot reload support.

The serve command will:
  - Load environment variables from .env
  - Build and run your application
  - Watch for file changes and automatically reload
  - Show application logs in real-time

Example:
  glib serve
  glib serve --port=3000
  glib serve --host=0.0.0.0
  glib serve --no-reload`,
		RunE: runServe,
	}

	cmd.Flags().IntP("port", "p", 0, "Port to listen on (overrides .env)")
	cmd.Flags().String("host", "", "Host to bind to (overrides .env)")
	cmd.Flags().Bool("no-reload", false, "Disable hot reload")
	cmd.Flags().String("main", "main.go", "Path to main.go file")

	return cmd
}

type devServer struct {
	cmd        *exec.Cmd
	mainFile   string
	port       int
	host       string
	noReload   bool
	watcher    *fsnotify.Watcher
	restartCh  chan bool
	stopCh     chan bool
	cobraCmd   *cobra.Command
	binaryPath string
}

func runServe(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	host, _ := cmd.Flags().GetString("host")
	noReload, _ := cmd.Flags().GetBool("no-reload")
	mainFile, _ := cmd.Flags().GetString("main")

	// Check if main.go exists
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		return fmt.Errorf("main file not found: %s\nMake sure you're in a glib project directory", mainFile)
	}

	// Check if go.mod exists
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return fmt.Errorf("go.mod not found. Make sure you're in a Go project directory")
	}

	// Clean up old temp binaries from previous runs
	cleanupOldTempBinaries()

	server := &devServer{
		mainFile:  mainFile,
		port:      port,
		host:      host,
		noReload:  noReload,
		restartCh: make(chan bool),
		stopCh:    make(chan bool),
		cobraCmd:  cmd,
	}

	// Ensure cleanup on exit
	defer server.cleanup()

	cmd.Println("╔═══════════════════════════════════════════════════════╗")
	cmd.Println("║         Glib Development Server                       ║")
	cmd.Println("╚═══════════════════════════════════════════════════════╝")
	cmd.Println()

	// Setup file watcher if hot reload is enabled
	if !noReload {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("failed to create file watcher: %w", err)
		}
		defer watcher.Close()
		server.watcher = watcher

		// Watch directories
		if err := server.setupWatcher(); err != nil {
			return err
		}

		// Start watching for file changes
		go server.watchFiles()
	}

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start the server
	if err := server.start(); err != nil {
		return err
	}

	// Main loop
	for {
		select {
		case <-sigCh:
			cmd.Println("\n\n⏹  Shutting down server...")
			return nil

		case <-server.restartCh:
			cmd.Println("\n🔄  Restarting server...")
			server.stop()
			time.Sleep(500 * time.Millisecond)
			if err := server.start(); err != nil {
				cmd.Printf("❌  Failed to restart: %v\n", err)
			}

		case <-server.stopCh:
			return nil
		}
	}
}

// cleanupOldTempBinaries removes old glib-serve-* binaries from /tmp
func cleanupOldTempBinaries() {
	tmpDir := os.TempDir()
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "glib-serve-") {
			path := filepath.Join(tmpDir, entry.Name())
			os.Remove(path)
		}
	}
}

func (s *devServer) setupWatcher() error {
	// Watch common directories
	dirsToWatch := []string{
		".",
		"app",
		"app/controllers",
		"app/models",
		"app/middleware",
		"config",
		"routes",
	}

	for _, dir := range dirsToWatch {
		if _, err := os.Stat(dir); err == nil {
			if err := s.watcher.Add(dir); err != nil {
				s.cobraCmd.Printf("⚠  Warning: Could not watch %s: %v\n", dir, err)
			}
		}
	}

	return nil
}

func (s *devServer) watchFiles() {
	debounce := time.NewTimer(0)
	<-debounce.C

	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			// Only restart on write or create events for .go files
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				if strings.HasSuffix(event.Name, ".go") {
					// Debounce - wait for 500ms of no changes
					debounce.Reset(500 * time.Millisecond)
					go func() {
						<-debounce.C
						s.restartCh <- true
					}()
				}
			}

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			s.cobraCmd.Printf("⚠  Watch error: %v\n", err)
		}
	}
}

func (s *devServer) start() error {
	s.cobraCmd.Println("🔨  Building application...")

	// Create temp binary path
	s.binaryPath = filepath.Join(os.TempDir(), fmt.Sprintf("glib-serve-%d", time.Now().Unix()))

	// Build the application
	buildCmd := exec.Command("go", "build", "-o", s.binaryPath, s.mainFile)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	s.cobraCmd.Println("✅  Build successful")

	// Load .env for display purposes
	envVars := s.loadEnvFile()

	// Prepare environment variables
	env := os.Environ()

	// Override port and host if provided
	if s.port > 0 {
		env = append(env, fmt.Sprintf("PORT=%d", s.port))
		envVars["PORT"] = fmt.Sprintf("%d", s.port)
	}
	if s.host != "" {
		env = append(env, fmt.Sprintf("HOST=%s", s.host))
		envVars["HOST"] = s.host
	}

	// Display server info
	port := envVars["PORT"]
	if port == "" {
		port = "8080"
	}
	host := envVars["HOST"]
	if host == "" {
		host = "localhost"
	}

	s.cobraCmd.Println()
	s.cobraCmd.Printf("🚀  Server starting at http://%s:%s\n", host, port)
	if !s.noReload {
		s.cobraCmd.Println("👀  Watching for file changes...")
	}
	s.cobraCmd.Println("📝  Server logs:\n")

	// Run the binary
	s.cmd = exec.Command(s.binaryPath)
	s.cmd.Env = env

	// Stream output
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := s.cmd.StderrPipe()
	if err != nil {
		return err
	}

	// Start the process
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Stream logs
	go s.streamLogs(stdout, "")
	go s.streamLogs(stderr, "ERROR: ")

	return nil
}

func (s *devServer) stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		// Try graceful shutdown first
		s.cmd.Process.Signal(syscall.SIGTERM)

		// Wait for up to 5 seconds
		done := make(chan error)
		go func() {
			done <- s.cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited
		case <-time.After(5 * time.Second):
			// Force kill
			s.cmd.Process.Kill()
		}
	}

	// Clean up binary
	s.cleanup()
}

func (s *devServer) cleanup() {
	if s.binaryPath != "" {
		os.Remove(s.binaryPath)
		s.binaryPath = ""
	}
}

func (s *devServer) streamLogs(pipe io.ReadCloser, prefix string) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		s.cobraCmd.Printf("%s%s\n", prefix, scanner.Text())
	}
}

func (s *devServer) loadEnvFile() map[string]string {
	envVars := make(map[string]string)

	file, err := os.Open(".env")
	if err != nil {
		return envVars
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			envVars[key] = value
		}
	}

	return envVars
}
