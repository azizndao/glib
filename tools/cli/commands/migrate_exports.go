package commands

import "github.com/spf13/cobra"

// NewMigrateRollbackCommand creates the "glib migrate:rollback" command
func NewMigrateRollbackCommand() *cobra.Command {
	return newMigrateDownCommand()
}

// NewMigrateFreshCommand creates the "glib migrate:fresh" command  
func NewMigrateFreshCommand() *cobra.Command {
	return newMigrateResetUpCommand()
}

// NewMigrateResetCommand creates the "glib migrate:reset" command
func NewMigrateResetCommand() *cobra.Command {
	return newMigrateResetCommand()
}

// NewMigrateStatusCommand creates the "glib migrate:status" command
func NewMigrateStatusCommand() *cobra.Command {
	cmd := newMigrateStatusCommand()
	cmd.Use = "migrate:status"
	return cmd
}

// NewMigrateRefreshCommand creates the "glib migrate:refresh" command
func NewMigrateRefreshCommand() *cobra.Command {
	return newMigrateRedoAllCommand()
}

// Helper to create migrate:fresh command (reset + up)
func newMigrateResetUpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate:fresh",
		Short: "Drop all tables and re-run all migrations",
		Long: `Drop all tables and re-run all migrations.

WARNING: This will delete all data in your database!

Example:
  glib migrate:fresh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// First reset (down all)
			if err := runMigrateReset(cmd, args); err != nil {
				return err
			}
			// Then up all
			return runMigrateUp(cmd, args)
		},
	}

	cmd.Flags().String("dir", "database/migrations", "Directory containing migration files")

	return cmd
}

// Helper to create migrate:refresh command (down + up recent)
func newMigrateRedoAllCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate:refresh",
		Short: "Reset and re-run all migrations",
		Long: `Reset and re-run all migrations.

Example:
  glib migrate:refresh`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Reset all
			if err := runMigrateReset(cmd, args); err != nil {
				return err
			}
			// Re-run all
			return runMigrateUp(cmd, args)
		},
	}

	cmd.Flags().String("dir", "database/migrations", "Directory containing migration files")

	return cmd
}
