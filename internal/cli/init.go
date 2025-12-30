package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var (
		module  string
		example bool
		minimal bool
	)

	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a new Glib project",
		Long: `Initialize a new Glib project with scaffolding.

Creates a new project with:
  - main.go (application entry point)
  - config.go (configuration struct)
  - .glibrc (Glib configuration)
  - .gitignore (Git ignore file)

Optional:
  --example  Include example health check controller
  --minimal  Minimal setup (no examples, no comments)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			return initProject(dir, module, example, minimal)
		},
	}

	cmd.Flags().StringVar(&module, "module", "", "Go module name (auto-detected if not specified)")
	cmd.Flags().BoolVar(&example, "example", false, "Include example health check controller")
	cmd.Flags().BoolVar(&minimal, "minimal", false, "Minimal setup (no examples, no comments)")

	return cmd
}

func initProject(dir, module string, example, minimal bool) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Get absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Infer module name if not provided
	if module == "" {
		module = inferModuleName(absDir)
	}

	// Check if directory is empty
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}
	if len(entries) > 0 {
		// Allow if only .git directory exists
		if len(entries) > 1 || entries[0].Name() != ".git" {
			return fmt.Errorf("directory %s is not empty", dir)
		}
	}

	fmt.Printf("🚀 Initializing Glib project in %s\n", absDir)
	fmt.Printf("📦 Module: %s\n\n", module)

	// Create files
	files := buildProjectFiles(module, example, minimal)

	for path, content := range files {
		fullPath := filepath.Join(absDir, path)

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", path, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}

		fmt.Printf("   ✓ Created %s\n", path)
	}

	// Initialize go.mod if it doesn't exist
	goModPath := filepath.Join(absDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		if err := createGoMod(absDir, module); err != nil {
			return fmt.Errorf("failed to initialize go.mod: %w", err)
		}
		fmt.Printf("   ✓ Created go.mod\n")
	}

	fmt.Println("\n✅ Project initialized successfully!")
	fmt.Println("\n📝 Next steps:")
	if dir != "." {
		fmt.Printf("   cd %s\n", dir)
	}
	fmt.Println("   go mod tidy")
	if example {
		fmt.Println("   glib generate")
		fmt.Println("   go run .")
	} else {
		fmt.Println("   glib make controller health")
		fmt.Println("   glib generate")
		fmt.Println("   go run .")
	}

	return nil
}

func inferModuleName(absPath string) string {
	// Try to get from git remote
	// For now, just use the directory name
	base := filepath.Base(absPath)
	return strings.ToLower(base)
}

func createGoMod(dir, module string) error {
	content := fmt.Sprintf(`module %s

go 1.23

require (
	github.com/azizndao/glib v2.0.0-dev
)
`, module)

	return os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0644)
}

func buildProjectFiles(module string, example, minimal bool) map[string]string {
	files := make(map[string]string)

	// main.go
	files["main.go"] = renderMainGo(module, minimal)

	// config.go
	files["config.go"] = renderConfigGo(minimal)

	// .glibrc
	files[".glibrc"] = renderGlibRC()

	// .gitignore
	files[".gitignore"] = renderGitignore()

	// README.md (unless minimal)
	if !minimal {
		files["README.md"] = renderReadme(module)
	}

	// Example controller
	if example {
		files["health/controller.go"] = renderHealthController(module)
	}

	return files
}

func renderMainGo(module string, minimal bool) string {
	if minimal {
		return fmt.Sprintf(`package main

import (
	"context"
	"log"
	"net/http"
	
	"%s/generated"
)

func main() {
	ctx := context.Background()
	
	handler, err := generated.Bootstrap(ctx)
	if err != nil {
		log.Fatalf("bootstrap failed: %%v", err)
	}
	
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("server failed: %%v", err)
	}
}
`, module)
	}

	return fmt.Sprintf(`package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"%s/generated"
)

func main() {
	ctx := context.Background()
	
	// Load configuration
	cfg := LoadConfig()
	
	// Bootstrap application
	handler, err := generated.Bootstrap(ctx)
	if err != nil {
		log.Fatalf("bootstrap failed: %%v", err)
	}
	
	// Create server
	addr := fmt.Sprintf(":%%d", cfg.App.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	// Start server in goroutine
	go func() {
		log.Printf("🚀 Server starting on %%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %%v", err)
		}
	}()
	
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("🛑 Shutting down server...")
	
	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %%v", err)
	}
	
	log.Println("✅ Server stopped")
}
`, module)
}

func renderConfigGo(minimal bool) string {
	if minimal {
		return `package main

type Config struct {
	App struct {
		Port int
	}
}

func LoadConfig() *Config {
	cfg := &Config{}
	cfg.App.Port = 8080
	return cfg
}
`
	}

	return `package main

import (
	"os"
	"strconv"
)

// Config holds application configuration.
type Config struct {
	App struct {
		Port int
		Env  string
	}
	Database struct {
		Host     string
		Port     int
		Name     string
		User     string
		Password string
	}
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() *Config {
	cfg := &Config{}
	
	// App configuration
	cfg.App.Port = getEnvInt("APP_PORT", 8080)
	cfg.App.Env = getEnv("APP_ENV", "development")
	
	// Database configuration (if needed)
	cfg.Database.Host = getEnv("DB_HOST", "localhost")
	cfg.Database.Port = getEnvInt("DB_PORT", 5432)
	cfg.Database.Name = getEnv("DB_NAME", "app")
	cfg.Database.User = getEnv("DB_USER", "postgres")
	cfg.Database.Password = getEnv("DB_PASSWORD", "")
	
	return cfg
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}
`
}

func renderGlibRC() string {
	return `{
  "version": "2",
  "generate": {
    "output": "generated",
    "package": "generated"
  },
  "make": {
    "controllers": "controllers",
    "providers": "providers",
    "middleware": "middleware"
  },
  "dev": {
    "port": 8080
  }
}
`
}

func renderGitignore() string {
	return `# Binaries
bin/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of the go coverage tool
*.out

# Go workspace file
go.work.sum

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Environment
.env
.env.local

# Generated code
generated/

# Temporary files
tmp/
`
}

func renderReadme(module string) string {
	return fmt.Sprintf(`# %s

Glib 2.0 application.

## Getting Started

### Prerequisites

- Go 1.23 or higher
- Glib CLI: `+"`"+`go install github.com/azizndao/glib/cmd/glib@latest`+"`"+`

### Development

Install dependencies:
`+"```"+`bash
go mod tidy
`+"```"+`

Generate code:
`+"```"+`bash
glib generate
`+"```"+`

Run development server with hot reload:
`+"```"+`bash
glib dev
`+"```"+`

Run without hot reload:
`+"```"+`bash
go run .
`+"```"+`

### Project Structure

- `+"`main.go`"+` - Application entry point
- `+"`config.go`"+` - Configuration loading
- `+"`generated/`"+` - Auto-generated code (do not edit)
- `+"`controllers/`"+` - HTTP controllers
- `+"`providers/`"+` - Dependency injection providers
- `+"`middleware/`"+` - HTTP middleware

### Creating Components

Create a controller:
`+"```"+`bash
glib make controller posts
`+"```"+`

Create a provider:
`+"```"+`bash
glib make provider database
`+"```"+`

Create middleware:
`+"```"+`bash
glib make middleware auth
`+"```"+`

### Building for Production

Build:
`+"```"+`bash
glib generate
go build -o bin/app .
`+"```"+`

Run:
`+"```"+`bash
./bin/app
`+"```"+`

## Documentation

- [Glib Documentation](https://github.com/azizndao/glib)
- [Annotations Reference](https://github.com/azizndao/glib/blob/main/.spec/01-ANNOTATIONS.md)
- [Handler Patterns](https://github.com/azizndao/glib/blob/main/.spec/02-HANDLERS.md)

## License

MIT
`, module)
}

func renderHealthController(module string) string {
	return `package health

import (
	"context"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status string ` + "`json:\"status\"`" + `
}

// @Controller /health
type HealthController struct{}

// @Route GET /
func (c *HealthController) Check(ctx context.Context) (*HealthResponse, error) {
	return &HealthResponse{
		Status: "ok",
	}, nil
}
`
}
