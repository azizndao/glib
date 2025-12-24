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
	rootCmd.AddCommand(
		commands.NewNewCommand(),
		commands.NewServeCommand(),
		commands.NewMigrateCommand(),
		commands.NewMakeCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
