package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/azizndao/glib/tools/cli/generators"
	"github.com/spf13/cobra"
)

// NewMakeCommand creates the "glib make" parent command with all generators
func NewMakeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "make",
		Short: "Generate boilerplate code",
		Long:  `Generate models, controllers, migrations, and other boilerplate code.`,
	}

	// Add all make subcommands
	cmd.AddCommand(newMakeModelCommand())
	cmd.AddCommand(newMakeControllerCommand())
	cmd.AddCommand(newMakeMigrationCommand())
	cmd.AddCommand(newMakeMiddlewareCommand())

	return cmd
}

func newMakeModelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model [name]",
		Short: "Generate a new model",
		Long: `Generate a new model class.

Example:
  glib make:model User
  glib make:model User --migration
  glib make:model Post --migration --controller`,
		Args: cobra.ExactArgs(1),
		RunE: runMakeModel,
	}

	cmd.Flags().BoolP("migration", "m", false, "Also create a migration file")
	cmd.Flags().BoolP("controller", "c", false, "Also create a controller")

	return cmd
}

func runMakeModel(cmd *cobra.Command, args []string) error {
	name := args[0]
	migration, _ := cmd.Flags().GetBool("migration")
	controller, _ := cmd.Flags().GetBool("controller")

	cmd.Printf("Creating model: %s\n", name)

	// Initialize generator
	gen, err := generators.NewGenerator()
	if err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}

	// Prepare model data
	modelData := generators.ModelData{
		Package:   "models",
		Name:      name,
		TableName: toSnakeCase(name) + "s",
		Imports:   []string{},
		Comment:   fmt.Sprintf("%s represents a %s record", name, strings.ToLower(name)),
	}

	// Generate model file
	modelPath := filepath.Join("app", "models", toSnakeCase(name)+".go")
	if err := gen.Generate("model.tmpl", modelPath, modelData); err != nil {
		return fmt.Errorf("failed to generate model: %w", err)
	}

	cmd.Printf("  Created: %s\n", modelPath)

	// Generate migration if requested
	if migration {
		migrationData := generators.MigrationData{
			Name:      fmt.Sprintf("create_%s_table", toSnakeCase(name)+"s"),
			TableName: toSnakeCase(name) + "s",
			Timestamp: generators.Timestamp(),
			Type:      "sql",
		}

		migrationPath := filepath.Join("database", "migrations",
			fmt.Sprintf("%s_create_%s_table.sql", migrationData.Timestamp, toSnakeCase(name)+"s"))

		if err := gen.Generate("migration_sql.tmpl", migrationPath, migrationData); err != nil {
			return fmt.Errorf("failed to generate migration: %w", err)
		}

		cmd.Printf("  Created: %s\n", migrationPath)
	}

	// Generate controller if requested
	if controller {
		controllerData := generators.ControllerData{
			Package:  "controllers",
			Name:     name + "Controller",
			Model:    name,
			Resource: true,
			Imports:  []string{},
			Comment:  fmt.Sprintf("%sController handles HTTP requests for %s resources", name, strings.ToLower(name)),
		}

		controllerPath := filepath.Join("app", "controllers", toSnakeCase(name)+"_controller.go")
		if err := gen.Generate("controller.tmpl", controllerPath, controllerData); err != nil {
			return fmt.Errorf("failed to generate controller: %w", err)
		}

		cmd.Printf("  Created: %s\n", controllerPath)
	}

	cmd.Println("\n✓ Model created successfully!")

	return nil
}

func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r+('a'-'A'))
		if r < 'A' || r > 'Z' {
			result[len(result)-1] = r
		}
	}
	return string(result)
}

func newMakeControllerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "controller [name]",
		Short: "Generate a new controller",
		Long: `Generate a new controller class.

Example:
  glib make:controller UserController
  glib make:controller UserController --resource
  glib make:controller Api/UserController`,
		Args: cobra.ExactArgs(1),
		RunE: runMakeController,
	}

	cmd.Flags().BoolP("resource", "r", false, "Generate a resource controller with CRUD methods")

	return cmd
}

func runMakeController(cmd *cobra.Command, args []string) error {
	name := args[0]
	resource, _ := cmd.Flags().GetBool("resource")

	cmd.Printf("Creating controller: %s\n", name)

	// Initialize generator
	gen, err := generators.NewGenerator()
	if err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}

	// Extract model name from controller name (e.g., UserController -> User)
	modelName := strings.TrimSuffix(name, "Controller")
	if modelName == name {
		modelName = name
		name = name + "Controller"
	}

	// Prepare controller data
	controllerData := generators.ControllerData{
		Package:  "controllers",
		Name:     name,
		Model:    modelName,
		Resource: resource,
		Imports:  []string{},
		Comment:  fmt.Sprintf("%s handles HTTP requests", name),
	}

	// Generate controller file
	controllerPath := filepath.Join("app", "controllers", toSnakeCase(name)+".go")
	if err := gen.Generate("controller.tmpl", controllerPath, controllerData); err != nil {
		return fmt.Errorf("failed to generate controller: %w", err)
	}

	cmd.Printf("  Created: %s\n", controllerPath)

	if resource {
		cmd.Println("  Methods: Index, Show, Store, Update, Destroy")
	}

	cmd.Println("\n✓ Controller created successfully!")

	return nil
}

func newMakeMigrationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migration [name]",
		Short: "Generate a new migration file",
		Long: `Generate a new database migration file.

Example:
  glib make:migration create_users_table
  glib make:migration add_published_to_posts`,
		Args: cobra.ExactArgs(1),
		RunE: runMakeMigration,
	}

	cmd.Flags().String("type", "sql", "Migration type (sql, go)")

	return cmd
}

func runMakeMigration(cmd *cobra.Command, args []string) error {
	name := args[0]
	migrationType, _ := cmd.Flags().GetString("type")

	cmd.Printf("Creating migration: %s\n", name)

	// Initialize generator
	gen, err := generators.NewGenerator()
	if err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}

	// Extract table name from migration name
	tableName := extractTableName(name)

	// Prepare migration data
	migrationData := generators.MigrationData{
		Name:      name,
		TableName: tableName,
		Timestamp: generators.Timestamp(),
		Type:      migrationType,
	}

	// Generate migration file
	extension := ".sql"
	templateName := "migration_sql.tmpl"
	if migrationType == "go" {
		extension = ".go"
		templateName = "migration_go.tmpl"
	}

	migrationPath := filepath.Join("database", "migrations",
		fmt.Sprintf("%s_%s%s", migrationData.Timestamp, name, extension))

	if err := gen.Generate(templateName, migrationPath, migrationData); err != nil {
		return fmt.Errorf("failed to generate migration: %w", err)
	}

	cmd.Printf("  Created: %s\n", migrationPath)

	cmd.Println("\n✓ Migration created successfully!")

	return nil
}

func extractTableName(migrationName string) string {
	// Extract table name from patterns like:
	// - create_users_table -> users
	// - add_email_to_users -> users
	// - drop_posts_table -> posts

	parts := strings.Split(migrationName, "_")

	// Look for "create_X_table" pattern
	if len(parts) >= 3 && parts[0] == "create" && parts[len(parts)-1] == "table" {
		return strings.Join(parts[1:len(parts)-1], "_")
	}

	// Look for "add_X_to_Y" pattern
	for i, part := range parts {
		if part == "to" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	// Default: use the last meaningful part
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return migrationName
}

func newMakeMiddlewareCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "middleware [name]",
		Short: "Generate a new middleware",
		Long: `Generate a new middleware.

Example:
  glib make:middleware AdminOnly
  glib make:middleware Auth/RequireRole`,
		Args: cobra.ExactArgs(1),
		RunE: runMakeMiddleware,
	}
}

func runMakeMiddleware(cmd *cobra.Command, args []string) error {
	name := args[0]

	cmd.Printf("Creating middleware: %s\n", name)

	// Initialize generator
	gen, err := generators.NewGenerator()
	if err != nil {
		return fmt.Errorf("failed to initialize generator: %w", err)
	}

	// Prepare middleware data
	middlewareData := generators.MiddlewareData{
		Package: "middleware",
		Name:    name,
		Imports: []string{},
		Comment: fmt.Sprintf("%s is a middleware function", name),
	}

	// Generate middleware file
	middlewarePath := filepath.Join("app", "middleware", toSnakeCase(name)+".go")
	if err := gen.Generate("middleware.tmpl", middlewarePath, middlewareData); err != nil {
		return fmt.Errorf("failed to generate middleware: %w", err)
	}

	cmd.Printf("  Created: %s\n", middlewarePath)

	cmd.Println("\n✓ Middleware created successfully!")

	return nil
}
