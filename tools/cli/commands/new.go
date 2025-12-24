package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// NewNewCommand creates the "glib new" command for scaffolding new projects
func NewNewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "Create a new glib project",
		Long: `Create a new glib project with the standard directory structure.

Example:
  glib new blog
  glib new blog --template=api
  glib new blog --database=postgres`,
		Args: cobra.ExactArgs(1),
		RunE: runNew,
	}

	cmd.Flags().String("template", "full", "Project template (full, api, minimal)")
	cmd.Flags().String("database", "sqlite", "Database driver (sqlite, postgres, mysql)")
	cmd.Flags().Bool("no-git", false, "Skip git initialization")
	cmd.Flags().String("module", "", "Go module path (default: project-name)")

	return cmd
}

type projectConfig struct {
	Name       string
	ModulePath string
	Template   string
	Database   string
	NoGit      bool
	ProjectDir string
}

func runNew(cmd *cobra.Command, args []string) error {
	projectName := args[0]
	template, _ := cmd.Flags().GetString("template")
	database, _ := cmd.Flags().GetString("database")
	noGit, _ := cmd.Flags().GetBool("no-git")
	modulePath, _ := cmd.Flags().GetString("module")

	// Validate template
	if !isValidTemplate(template) {
		return fmt.Errorf("invalid template: %s (valid options: full, api, minimal)", template)
	}

	// Validate database
	if !isValidDatabase(database) {
		return fmt.Errorf("invalid database: %s (valid options: sqlite, postgres, mysql)", database)
	}

	// Use project name as module path if not specified
	if modulePath == "" {
		modulePath = projectName
	}

	cfg := &projectConfig{
		Name:       projectName,
		ModulePath: modulePath,
		Template:   template,
		Database:   database,
		NoGit:      noGit,
		ProjectDir: projectName,
	}

	cmd.Printf("Creating new glib project: %s\n", projectName)
	cmd.Printf("  Template: %s\n", template)
	cmd.Printf("  Module: %s\n", modulePath)
	cmd.Printf("  Database: %s\n", database)
	cmd.Println()

	// Create project
	if err := createProject(cmd, cfg); err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	cmd.Printf("\n✓ Project created successfully!\n")
	cmd.Printf("\nNext steps:\n")
	cmd.Printf("  cd %s\n", projectName)
	cmd.Printf("  go mod tidy\n")
	cmd.Printf("  go run main.go\n")
	cmd.Printf("\nOr use glib serve for development:\n")
	cmd.Printf("  cd %s\n", projectName)
	cmd.Printf("  glib serve\n")

	return nil
}

func isValidTemplate(template string) bool {
	validTemplates := []string{"full", "api", "minimal"}
	for _, t := range validTemplates {
		if t == template {
			return true
		}
	}
	return false
}

func isValidDatabase(db string) bool {
	validDatabases := []string{"sqlite", "postgres", "mysql"}
	for _, d := range validDatabases {
		if d == db {
			return true
		}
	}
	return false
}

func createProject(cmd *cobra.Command, cfg *projectConfig) error {
	// Check if directory already exists
	if _, err := os.Stat(cfg.ProjectDir); err == nil {
		return fmt.Errorf("directory already exists: %s", cfg.ProjectDir)
	}

	// Create project directory
	cmd.Printf("Creating directory structure...\n")
	if err := createDirectoryStructure(cfg); err != nil {
		return err
	}

	// Generate files based on template
	cmd.Printf("Generating project files...\n")
	if err := generateProjectFiles(cfg); err != nil {
		return err
	}

	// Initialize git repository
	if !cfg.NoGit {
		cmd.Printf("Initializing git repository...\n")
		if err := initGitRepository(cfg.ProjectDir); err != nil {
			cmd.Printf("Warning: Failed to initialize git: %v\n", err)
		}
	}

	return nil
}

func createDirectoryStructure(cfg *projectConfig) error {
	dirs := []string{
		cfg.ProjectDir,
		filepath.Join(cfg.ProjectDir, "app"),
		filepath.Join(cfg.ProjectDir, "app", "controllers"),
		filepath.Join(cfg.ProjectDir, "app", "models"),
		filepath.Join(cfg.ProjectDir, "app", "middleware"),
		filepath.Join(cfg.ProjectDir, "config"),
		filepath.Join(cfg.ProjectDir, "database"),
		filepath.Join(cfg.ProjectDir, "database", "migrations"),
		filepath.Join(cfg.ProjectDir, "routes"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func generateProjectFiles(cfg *projectConfig) error {
	// Generate go.mod
	if err := generateGoMod(cfg); err != nil {
		return err
	}

	// Generate .env.example
	if err := generateEnvExample(cfg); err != nil {
		return err
	}

	// Generate .gitignore
	if err := generateGitignore(cfg); err != nil {
		return err
	}

	// Generate main.go based on template
	if err := generateMainGo(cfg); err != nil {
		return err
	}

	// Generate routes
	if err := generateRoutes(cfg); err != nil {
		return err
	}

	// Generate config files
	if err := generateConfig(cfg); err != nil {
		return err
	}

	// Generate README
	if err := generateReadme(cfg); err != nil {
		return err
	}

	return nil
}

func generateGoMod(cfg *projectConfig) error {
	content := fmt.Sprintf(`module %s

go 1.25.1

require (
	github.com/azizndao/glib v0.0.0
	github.com/joho/godotenv v1.5.1
)
`, cfg.ModulePath)

	path := filepath.Join(cfg.ProjectDir, "go.mod")
	return os.WriteFile(path, []byte(content), 0644)
}

func generateEnvExample(cfg *projectConfig) error {
	content := `# Server Configuration
HOST=localhost
PORT=8080

# Timeout Settings
READ_TIMEOUT=10s
WRITE_TIMEOUT=10s
IDLE_TIMEOUT=120s
SHUTDOWN_TIMEOUT=30s

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json
IS_DEBUG=false

# Database Configuration
DB_CONNECTION=` + cfg.Database + `
DB_HOST=localhost
DB_PORT=` + getDatabasePort(cfg.Database) + `
DB_DATABASE=` + cfg.Name + `
DB_USERNAME=
DB_PASSWORD=

# Middleware Configuration
ENABLE_CORS=true
ENABLE_COMPRESS=true
ENABLE_RECOVERER=true

# CORS Configuration
CORS_ALLOWED_ORIGINS=*
CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,HEAD,OPTIONS
CORS_ALLOWED_HEADERS=*
CORS_ALLOW_CREDENTIALS=false
CORS_MAX_AGE=86400
`

	path := filepath.Join(cfg.ProjectDir, ".env.example")
	return os.WriteFile(path, []byte(content), 0644)
}

func generateGitignore(cfg *projectConfig) error {
	content := `# Binary
` + cfg.Name + `

# Database files
*.db
*.sqlite
*.sqlite3

# Environment
.env

# Temp files
tmp/
`

	path := filepath.Join(cfg.ProjectDir, ".gitignore")
	return os.WriteFile(path, []byte(content), 0644)
}

func generateMainGo(cfg *projectConfig) error {
	var content string

	switch cfg.Template {
	case "minimal":
		content = generateMinimalMain(cfg)
	case "api":
		content = generateAPIMain(cfg)
	default: // "full"
		content = generateFullMain(cfg)
	}

	path := filepath.Join(cfg.ProjectDir, "main.go")
	return os.WriteFile(path, []byte(content), 0644)
}

func generateMinimalMain(cfg *projectConfig) string {
	return fmt.Sprintf(`package main

import (
	"log"

	"github.com/azizndao/glib"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	godotenv.Load()

	// Create server
	server := glib.New(glib.Config{})
	r := server.Router()

	// Define routes
	r.Get("/", func(c *glib.Ctx) error {
		return c.JSON(map[string]string{
			"message": "Welcome to %s",
		})
	})

	r.Get("/health", func(c *glib.Ctx) error {
		return c.JSON(map[string]string{
			"status": "healthy",
		})
	})

	// Start server
	server.Logger().Info("Server starting on " + server.Address())
	if err := server.ListenWithGracefulShutdown(); err != nil {
		log.Fatal(err)
	}
}
`, cfg.Name)
}

func generateAPIMain(cfg *projectConfig) string {
	return fmt.Sprintf(`package main

import (
	"log"

	"github.com/azizndao/glib"
	"%s/routes"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	godotenv.Load()

	// Create server
	server := glib.New(glib.Config{})
	r := server.Router()

	// Setup routes
	routes.SetupAPIRoutes(r)

	// Start server
	server.Logger().Info("API server starting on " + server.Address())
	if err := server.ListenWithGracefulShutdown(); err != nil {
		log.Fatal(err)
	}
}
`, cfg.ModulePath)
}

func generateFullMain(cfg *projectConfig) string {
	return fmt.Sprintf(`package main

import (
	"log"

	"github.com/azizndao/glib"
	"%s/routes"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	godotenv.Load()

	// Create server with configuration
	server := glib.New(glib.Config{})
	r := server.Router()

	// Setup routes
	routes.SetupRoutes(r)

	// Custom 404 handler
	r.NotFound(func(c *glib.Ctx) error {
		return c.Status(404).JSON(map[string]string{
			"error":   "Not Found",
			"message": "The requested resource was not found",
			"path":    c.Path(),
		})
	})

	// Start server
	server.Logger().Info("Server starting on " + server.Address())
	if err := server.ListenWithGracefulShutdown(); err != nil {
		log.Fatal(err)
	}
}
`, cfg.ModulePath)
}

func generateRoutes(cfg *projectConfig) error {
	var content string

	switch cfg.Template {
	case "minimal":
		// No routes file for minimal template
		return nil
	case "api":
		content = generateAPIRoutes(cfg)
	default: // "full"
		content = generateFullRoutes(cfg)
	}

	path := filepath.Join(cfg.ProjectDir, "routes", "routes.go")
	return os.WriteFile(path, []byte(content), 0644)
}

func generateAPIRoutes(cfg *projectConfig) string {
	return fmt.Sprintf(`package routes

import (
	"github.com/azizndao/glib"
)

// SetupAPIRoutes configures all API routes
func SetupAPIRoutes(r glib.Router) {
	// Health check
	r.Get("/health", healthHandler)

	// API v1 routes
	r.Route("/api/v1", func(api glib.Router) {
		api.Get("/", func(c *glib.Ctx) error {
			return c.JSON(map[string]string{
				"version": "1.0.0",
				"status":  "active",
			})
		})

		// Add your API routes here
		// Example:
		// api.Route("/users", setupUserRoutes)
		// api.Route("/posts", setupPostRoutes)
	})
}

func healthHandler(c *glib.Ctx) error {
	return c.JSON(map[string]interface{}{
		"status": "healthy",
		"service": "%s",
	})
}
`, cfg.Name)
}

func generateFullRoutes(cfg *projectConfig) string {
	return fmt.Sprintf(`package routes

import (
	"github.com/azizndao/glib"
)

// SetupRoutes configures all application routes
func SetupRoutes(r glib.Router) {
	// Public routes
	r.Get("/", homeHandler)
	r.Get("/health", healthHandler)

	// API routes
	r.Route("/api", func(api glib.Router) {
		api.Get("/", apiIndexHandler)
		
		// Add your API routes here
		// Example:
		// api.Route("/users", setupUserRoutes)
		// api.Route("/posts", setupPostRoutes)
	})
}

func homeHandler(c *glib.Ctx) error {
	return c.JSON(map[string]interface{}{
		"message": "Welcome to %s",
		"version": "1.0.0",
		"endpoints": map[string]string{
			"health": "/health",
			"api":    "/api",
		},
	})
}

func healthHandler(c *glib.Ctx) error {
	return c.JSON(map[string]interface{}{
		"status": "healthy",
		"service": "%s",
	})
}

func apiIndexHandler(c *glib.Ctx) error {
	return c.JSON(map[string]string{
		"message": "API is running",
		"version": "1.0.0",
	})
}
`, cfg.Name, cfg.Name)
}

func generateConfig(cfg *projectConfig) error {
	content := `package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Log      LogConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host            string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Connection string
	Host       string
	Port       string
	Database   string
	Username   string
	Password   string
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level   string
	Format  string
	IsDebug bool
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            getEnv("HOST", "localhost"),
			Port:            getEnv("PORT", "8080"),
			ReadTimeout:     getDuration("READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getDuration("WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:     getDuration("IDLE_TIMEOUT", 120*time.Second),
			ShutdownTimeout: getDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Connection: getEnv("DB_CONNECTION", "sqlite"),
			Host:       getEnv("DB_HOST", "localhost"),
			Port:       getEnv("DB_PORT", "5432"),
			Database:   getEnv("DB_DATABASE", ""),
			Username:   getEnv("DB_USERNAME", ""),
			Password:   getEnv("DB_PASSWORD", ""),
		},
		Log: LogConfig{
			Level:   getEnv("LOG_LEVEL", "info"),
			Format:  getEnv("LOG_FORMAT", "json"),
			IsDebug: getBool("IS_DEBUG", false),
		},
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}
`

	path := filepath.Join(cfg.ProjectDir, "config", "config.go")
	return os.WriteFile(path, []byte(content), 0644)
}

func generateReadme(cfg *projectConfig) error {
	content := fmt.Sprintf(`# %s

A web application built with [glib](https://github.com/azizndao/glib), a Laravel-inspired Go web framework.

## Getting Started

### Prerequisites

- Go 1.25.1 or higher
- %s database (if using database features)

### Installation

1. Clone the repository:
   \'\'\'bash
   cd %s
   \'\'\'

2. Install dependencies:
   \'\'\'bash
   go mod tidy
   \'\'\'

3. Copy the example environment file:
   \'\'\'bash
   cp .env.example .env
   \'\'\'

4. Update the .env file with your configuration

### Running the Application

Run the development server:
\'\'\'bash
go run main.go
\'\'\'

Or use the glib CLI for development with hot reload:
\'\'\'bash
glib serve
\'\'\'

The application will be available at http://localhost:8080

## Project Structure

\'\'\'
%s/
├── app/
│   ├── controllers/    # HTTP request handlers
│   ├── models/        # Database models
│   └── middleware/    # Custom middleware
├── config/           # Configuration files
│   └── config.go     # Application configuration
├── database/
│   └── migrations/   # Database migrations
├── routes/          # Route definitions
│   └── routes.go    # Application routes
├── .env.example     # Example environment variables
├── .gitignore       # Git ignore rules
├── go.mod           # Go module definition
├── main.go          # Application entry point
└── README.md        # This file
\'\'\'

## Development

### Generate Code

Use the glib CLI to generate boilerplate code:

\'\'\'bash
# Generate a model
glib make model User

# Generate a controller
glib make controller UserController --resource

# Generate a migration
glib make migration create_users_table

# Generate middleware
glib make middleware Auth
\'\'\'

### Database Migrations

This project uses [goose](https://github.com/pressly/goose/v3) for database migrations.

Install goose:
\'\'\'bash
go install github.com/pressly/goose/v3/cmd/goose@latest
\'\'\'

Run migrations:
\'\'\'bash
goose -dir database/migrations %s up
\'\'\'

Create a new migration:
\'\'\'bash
glib make migration create_posts_table
\'\'\'

## API Endpoints

- \'GET /\' - Home page
- \'GET /health\' - Health check
- \'GET /api\' - API index

## License

MIT
`, cfg.Name, strings.Title(cfg.Database), cfg.Name, cfg.Name, cfg.Database)

	// Replace triple backticks properly (avoiding Go raw string limitation)
	content = strings.ReplaceAll(content, "'''", "```")

	path := filepath.Join(cfg.ProjectDir, "README.md")
	return os.WriteFile(path, []byte(content), 0644)
}

func getDatabasePort(database string) string {
	switch database {
	case "postgres":
		return "5432"
	case "mysql":
		return "3306"
	case "sqlite":
		return ""
	default:
		return "5432"
	}
}

func initGitRepository(projectDir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		return err
	}

	// Create initial commit
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = projectDir
	return cmd.Run()
}
