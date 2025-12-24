package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/azizndao/glib/cli/generators"
	"github.com/spf13/cobra"
)

// NewMakeModelCommand creates the "glib make:model" command
func NewMakeModelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "make:model [name]",
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

// NewMakeControllerCommand creates the "glib make:controller" command
func NewMakeControllerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "make:controller [name]",
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

// NewMakeMigrationCommand creates the "glib make:migration" command
func NewMakeMigrationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "make:migration [name]",
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

// NewMakeMiddlewareCommand creates the "glib make:middleware" command
func NewMakeMiddlewareCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "make:middleware [name]",
		Short: "Generate a new middleware",
		Long: `Generate a new middleware.

Example:
  glib make:middleware AdminOnly
  glib make:middleware Auth/RequireRole`,
		Args: cobra.ExactArgs(1),
		RunE: runMakeMiddleware,
	}
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

// NewMakeSeederCommand creates the "glib make:seeder" command
func NewMakeSeederCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "make:seeder [name]",
		Short: "Generate a new database seeder",
		Long: `Generate a new database seeder.

Example:
  glib make:seeder UserSeeder
  glib make:seeder DatabaseSeeder`,
		Args: cobra.ExactArgs(1),
		RunE: runMakeSeeder,
	}
}

func runMakeSeeder(cmd *cobra.Command, args []string) error {
	name := args[0]

	cmd.Printf("Creating seeder: %s\n", name)

	// Ensure name ends with "Seeder"
	if !strings.HasSuffix(name, "Seeder") {
		name = name + "Seeder"
	}

	// Create seeder template data (we'll create a simple struct)
	content := fmt.Sprintf(`package seeders

import (
	"gorm.io/gorm"
)

// %s seeds the database
type %s struct{}

// Run executes the seeder
func (s *%s) Run(db *gorm.DB) error {
	// TODO: Implement seeding logic
	// Example:
	// users := []models.User{
	//     {Name: "John Doe", Email: "john@example.com"},
	//     {Name: "Jane Smith", Email: "jane@example.com"},
	// }
	// return db.Create(&users).Error
	
	return nil
}
`, name, name, name)

	// Generate seeder file
	seederPath := filepath.Join("database", "seeders", toSnakeCase(name)+".go")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(seederPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(seederPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write seeder: %w", err)
	}

	cmd.Printf("  Created: %s\n", seederPath)
	cmd.Println("\n✓ Seeder created successfully!")

	return nil
}

// NewMakePolicyCommand creates the "glib make:policy" command
func NewMakePolicyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "make:policy [name]",
		Short: "Generate a new policy class",
		Long: `Generate a new policy class for authorization.

Example:
  glib make:policy PostPolicy
  glib make:policy UserPolicy`,
		Args: cobra.ExactArgs(1),
		RunE: runMakePolicy,
	}
}

func runMakePolicy(cmd *cobra.Command, args []string) error {
	name := args[0]

	cmd.Printf("Creating policy: %s\n", name)

	// Ensure name ends with "Policy"
	if !strings.HasSuffix(name, "Policy") {
		name = name + "Policy"
	}

	// Extract model name
	modelName := strings.TrimSuffix(name, "Policy")

	content := fmt.Sprintf(`package policies

import (
	"github.com/azizndao/glib"
	"app/models"
)

// %s handles authorization for %s resources
type %s struct{}

// ViewAny determines if the user can view any %s
func (p *%s) ViewAny(c *glib.Ctx, user *models.User) bool {
	// TODO: Implement authorization logic
	return true
}

// View determines if the user can view the %s
func (p *%s) View(c *glib.Ctx, user *models.User, %s *models.%s) bool {
	// TODO: Implement authorization logic
	return true
}

// Create determines if the user can create %s
func (p *%s) Create(c *glib.Ctx, user *models.User) bool {
	// TODO: Implement authorization logic
	return true
}

// Update determines if the user can update the %s
func (p *%s) Update(c *glib.Ctx, user *models.User, %s *models.%s) bool {
	// TODO: Implement authorization logic
	return true
}

// Delete determines if the user can delete the %s
func (p *%s) Delete(c *glib.Ctx, user *models.User, %s *models.%s) bool {
	// TODO: Implement authorization logic
	return true
}
`, name, modelName, name,
		modelName, name,
		modelName, name, toSnakeCase(modelName), modelName,
		modelName, name,
		modelName, name, toSnakeCase(modelName), modelName,
		modelName, name, toSnakeCase(modelName), modelName)

	// Generate policy file
	policyPath := filepath.Join("app", "policies", toSnakeCase(name)+".go")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(policyPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(policyPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write policy: %w", err)
	}

	cmd.Printf("  Created: %s\n", policyPath)
	cmd.Println("\n✓ Policy created successfully!")

	return nil
}
