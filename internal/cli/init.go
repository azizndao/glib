package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a new Glib project",
		Long: `Initialize a new Glib project with scaffolding.

Creates a new project with:
  - main.go (application entry point)
  - config.go (configuration struct)
  - .glibrc (Glib configuration)
  - .gitignore (Git ignore file)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("TODO: Implement glib init")
			return nil
		},
	}
}
