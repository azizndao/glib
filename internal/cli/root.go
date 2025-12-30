// Package cli provides the command-line interface for Glib.
package cli

import (
	"github.com/spf13/cobra"
)

// Execute runs the root command.
func Execute(version string) error {
	rootCmd := newRootCmd(version)
	return rootCmd.Execute()
}

func newRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "glib",
		Short: "Glib - Go web framework with code generation",
		Long: `Glib 2.0 - Code generation framework for building Go web applications.

Glib uses annotations in your Go code to generate type-safe HTTP handlers,
dependency injection, and request/response parsing.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(
		newVersionCmd(version),
		newInitCmd(),
		newGenerateCmd(),
		newDevCmd(),
		newMakeCmd(),
		newValidateCmd(),
	)

	return cmd
}
