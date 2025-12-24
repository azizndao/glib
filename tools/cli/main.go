package main

import (
	"fmt"
	"os"

	"github.com/azizndao/glib/tools/cli/commands"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "glib",
		Short: "Glib - A Laravel-inspired Go backend framework",
		Long: `Glib is a comprehensive backend framework for Go that provides
an elegant, expressive syntax inspired by Laravel.

The glib CLI tool helps you build amazing applications with:
- Project scaffolding
- Code generators
- Database migrations
- Development server
- And much more!`,
		Version: Version,
	}

	// Set version template
	rootCmd.SetVersionTemplate(fmt.Sprintf("glib version %s (built %s)\n", Version, BuildDate))

	// Global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "quiet output")

	// Add all commands
	// Project commands
	rootCmd.AddCommand(commands.NewNewCommand())
	rootCmd.AddCommand(commands.NewServeCommand())

	// Make commands (code generators) - using colon syntax like Laravel Artisan
	rootCmd.AddCommand(commands.NewMakeModelCommand())
	rootCmd.AddCommand(commands.NewMakeControllerCommand())
	rootCmd.AddCommand(commands.NewMakeMigrationCommand())
	rootCmd.AddCommand(commands.NewMakeMiddlewareCommand())
	rootCmd.AddCommand(commands.NewMakeSeederCommand())
	rootCmd.AddCommand(commands.NewMakePolicyCommand())

	// Migration commands - using colon syntax
	rootCmd.AddCommand(commands.NewMigrateCommand())
	rootCmd.AddCommand(commands.NewMigrateRollbackCommand())
	rootCmd.AddCommand(commands.NewMigrateFreshCommand())
	rootCmd.AddCommand(commands.NewMigrateResetCommand())
	rootCmd.AddCommand(commands.NewMigrateStatusCommand())
	rootCmd.AddCommand(commands.NewMigrateRefreshCommand())

	// Route commands
	rootCmd.AddCommand(commands.NewRouteListCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
