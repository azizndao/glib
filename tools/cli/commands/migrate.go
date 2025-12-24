package commands

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

// NewMigrateCommand creates the "glib migrate" command and subcommands
func NewMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		Long: `Run all pending database migrations.

Example:
  glib migrate
  glib migrate --dir=database/migrations`,
		RunE: runMigrate,
	}

	cmd.Flags().String("dir", "database/migrations", "Directory containing migration files")

	// Add subcommands
	cmd.AddCommand(newMigrateUpCommand())
	cmd.AddCommand(newMigrateUpByOneCommand())
	cmd.AddCommand(newMigrateDownCommand())
	cmd.AddCommand(newMigrateDownToCommand())
	cmd.AddCommand(newMigrateRedoCommand())
	cmd.AddCommand(newMigrateResetCommand())
	cmd.AddCommand(newMigrateStatusCommand())
	cmd.AddCommand(newMigrateVersionCommand())
	cmd.AddCommand(newMigrateCreateCommand())
	cmd.AddCommand(newMigrateFixCommand())

	return cmd
}

// getDatabaseConnection loads database configuration from environment and returns a connection
func getDatabaseConnection() (*sql.DB, string, error) {
	driver := os.Getenv("DB_CONNECTION")
	if driver == "" {
		driver = "sqlite"
	}

	var dsn string
	switch driver {
	case "sqlite", "sqlite3":
		database := os.Getenv("DB_DATABASE")
		if database == "" {
			database = "database.sqlite"
		}
		dsn = database
		driver = "sqlite3"

	case "postgres", "postgresql":
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}
		database := os.Getenv("DB_DATABASE")
		username := os.Getenv("DB_USERNAME")
		password := os.Getenv("DB_PASSWORD")
		sslmode := os.Getenv("DB_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}

		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, username, password, database, sslmode)
		driver = "postgres"

	case "mysql":
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "3306"
		}
		database := os.Getenv("DB_DATABASE")
		username := os.Getenv("DB_USERNAME")
		password := os.Getenv("DB_PASSWORD")

		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
			username, password, host, port, database)

	default:
		return nil, "", fmt.Errorf("unsupported database driver: %s (supported: sqlite, postgres, mysql)", driver)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, "", fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, "", fmt.Errorf("failed to ping database: %w", err)
	}

	return db, driver, nil
}

// getMigrationsDir returns the migrations directory path
func getMigrationsDir(cmd *cobra.Command) (string, error) {
	dir, _ := cmd.Flags().GetString("dir")
	if dir == "" {
		dir = "database/migrations"
	}

	// Get absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		return "", fmt.Errorf("migrations directory does not exist: %s", absDir)
	}

	return absDir, nil
}

// runMigrate runs all pending migrations (alias for up)
func runMigrate(cmd *cobra.Command, args []string) error {
	return runMigrateUp(cmd, args)
}

// newMigrateUpCommand creates the migrate up command
func newMigrateUpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		Long: `Run all pending database migrations.

Example:
  glib migrate up
  glib migrate up --dir=database/migrations`,
		RunE: runMigrateUp,
	}

	cmd.Flags().String("dir", "database/migrations", "Directory containing migration files")

	return cmd
}

func runMigrateUp(cmd *cobra.Command, args []string) error {
	db, driver, err := getDatabaseConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	migrationsDir, err := getMigrationsDir(cmd)
	if err != nil {
		return err
	}

	goose.SetDialect(driver)

	cmd.Printf("Running migrations from %s...\n", migrationsDir)

	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	cmd.Println("\n✓ Migrations completed successfully!")

	return nil
}

// newMigrateUpByOneCommand creates the migrate up-by-one command
func newMigrateUpByOneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up-by-one",
		Short: "Run the next pending migration",
		Long: `Run only the next pending migration.

Example:
  glib migrate up-by-one`,
		RunE: runMigrateUpByOne,
	}

	cmd.Flags().String("dir", "database/migrations", "Directory containing migration files")

	return cmd
}

func runMigrateUpByOne(cmd *cobra.Command, args []string) error {
	db, driver, err := getDatabaseConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	migrationsDir, err := getMigrationsDir(cmd)
	if err != nil {
		return err
	}

	goose.SetDialect(driver)

	cmd.Println("Running next migration...")

	if err := goose.UpByOne(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to run migration: %w", err)
	}

	cmd.Println("\n✓ Migration completed successfully!")

	return nil
}

// newMigrateDownCommand creates the migrate down command
func newMigrateDownCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Rollback the last migration",
		Long: `Rollback the last migration that was run.

Example:
  glib migrate down`,
		RunE: runMigrateDown,
	}

	cmd.Flags().String("dir", "database/migrations", "Directory containing migration files")

	return cmd
}

func runMigrateDown(cmd *cobra.Command, args []string) error {
	db, driver, err := getDatabaseConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	migrationsDir, err := getMigrationsDir(cmd)
	if err != nil {
		return err
	}

	goose.SetDialect(driver)

	cmd.Println("Rolling back last migration...")

	if err := goose.Down(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	cmd.Println("\n✓ Rollback completed successfully!")

	return nil
}

// newMigrateDownToCommand creates the migrate down-to command
func newMigrateDownToCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down-to VERSION",
		Short: "Rollback to a specific version",
		Long: `Rollback migrations down to a specific version.

Example:
  glib migrate down-to 20240101000000`,
		Args: cobra.ExactArgs(1),
		RunE: runMigrateDownTo,
	}

	cmd.Flags().String("dir", "database/migrations", "Directory containing migration files")

	return cmd
}

func runMigrateDownTo(cmd *cobra.Command, args []string) error {
	db, driver, err := getDatabaseConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	migrationsDir, err := getMigrationsDir(cmd)
	if err != nil {
		return err
	}

	version := args[0]
	var versionNum int64
	if _, err := fmt.Sscanf(version, "%d", &versionNum); err != nil {
		return fmt.Errorf("invalid version number: %s", version)
	}

	goose.SetDialect(driver)

	cmd.Printf("Rolling back to version %s...\n", version)

	if err := goose.DownTo(db, migrationsDir, versionNum); err != nil {
		return fmt.Errorf("failed to rollback to version: %w", err)
	}

	cmd.Println("\n✓ Rollback completed successfully!")

	return nil
}

// newMigrateRedoCommand creates the migrate redo command
func newMigrateRedoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "redo",
		Short: "Rollback and re-run the last migration",
		Long: `Rollback the last migration and re-run it.

Example:
  glib migrate redo`,
		RunE: runMigrateRedo,
	}

	cmd.Flags().String("dir", "database/migrations", "Directory containing migration files")

	return cmd
}

func runMigrateRedo(cmd *cobra.Command, args []string) error {
	db, driver, err := getDatabaseConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	migrationsDir, err := getMigrationsDir(cmd)
	if err != nil {
		return err
	}

	goose.SetDialect(driver)

	cmd.Println("Re-running last migration...")

	if err := goose.Redo(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to redo migration: %w", err)
	}

	cmd.Println("\n✓ Migration re-run completed successfully!")

	return nil
}

// newMigrateResetCommand creates the migrate reset command
func newMigrateResetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Rollback all migrations",
		Long: `Rollback all migrations.

WARNING: This will rollback all migrations!

Example:
  glib migrate reset`,
		RunE: runMigrateReset,
	}

	cmd.Flags().String("dir", "database/migrations", "Directory containing migration files")

	return cmd
}

func runMigrateReset(cmd *cobra.Command, args []string) error {
	db, driver, err := getDatabaseConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	migrationsDir, err := getMigrationsDir(cmd)
	if err != nil {
		return err
	}

	goose.SetDialect(driver)

	cmd.Println("Rolling back all migrations...")

	if err := goose.Reset(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to reset migrations: %w", err)
	}

	cmd.Println("\n✓ All migrations rolled back successfully!")

	return nil
}

// newMigrateStatusCommand creates the migrate status command
func newMigrateStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the status of each migration",
		Long: `Display a list of all migrations and their status.

Example:
  glib migrate status`,
		RunE: runMigrateStatus,
	}

	cmd.Flags().String("dir", "database/migrations", "Directory containing migration files")

	return cmd
}

func runMigrateStatus(cmd *cobra.Command, args []string) error {
	db, driver, err := getDatabaseConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	migrationsDir, err := getMigrationsDir(cmd)
	if err != nil {
		return err
	}

	goose.SetDialect(driver)

	cmd.Println("Migration Status:\n")

	if err := goose.Status(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to get migration status: %w", err)
	}

	return nil
}

// newMigrateVersionCommand creates the migrate version command
func newMigrateVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show the current migration version",
		Long: `Display the current migration version.

Example:
  glib migrate version`,
		RunE: runMigrateVersion,
	}

	cmd.Flags().String("dir", "database/migrations", "Directory containing migration files")

	return cmd
}

func runMigrateVersion(cmd *cobra.Command, args []string) error {
	db, driver, err := getDatabaseConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = getMigrationsDir(cmd)
	if err != nil {
		return err
	}

	goose.SetDialect(driver)

	version, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	cmd.Printf("Current version: %d\n", version)

	return nil
}

// newMigrateCreateCommand creates the migrate create command
func newMigrateCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new migration file",
		Long: `Create a new migration file with the given name.

Example:
  glib migrate create add_users_table
  glib migrate create add_users_table --type=go`,
		Args: cobra.ExactArgs(1),
		RunE: runMigrateCreate,
	}

	cmd.Flags().String("dir", "database/migrations", "Directory to create migration in")
	cmd.Flags().String("type", "sql", "Migration type (sql or go)")

	return cmd
}

func runMigrateCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	migrationsDir, err := getMigrationsDir(cmd)
	if err != nil {
		// If directory doesn't exist, create it
		dir, _ := cmd.Flags().GetString("dir")
		if dir == "" {
			dir = "database/migrations"
		}
		absDir, err := filepath.Abs(dir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		if err := os.MkdirAll(absDir, 0755); err != nil {
			return fmt.Errorf("failed to create migrations directory: %w", err)
		}
		migrationsDir = absDir
	}

	migType, _ := cmd.Flags().GetString("type")

	if err := goose.Create(nil, migrationsDir, name, migType); err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	cmd.Printf("\n✓ Migration created successfully in %s\n", migrationsDir)

	return nil
}

// newMigrateFixCommand creates the migrate fix command
func newMigrateFixCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Fix migration file names to sequential ordering",
		Long: `Fix migration file names by applying sequential ordering.

Example:
  glib migrate fix`,
		RunE: runMigrateFix,
	}

	cmd.Flags().String("dir", "database/migrations", "Directory containing migration files")

	return cmd
}

func runMigrateFix(cmd *cobra.Command, args []string) error {
	migrationsDir, err := getMigrationsDir(cmd)
	if err != nil {
		return err
	}

	if err := goose.Fix(migrationsDir); err != nil {
		return fmt.Errorf("failed to fix migrations: %w", err)
	}

	cmd.Println("\n✓ Migrations fixed successfully!")

	return nil
}
