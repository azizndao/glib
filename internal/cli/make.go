package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMakeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "make <type> <name>",
		Short: "Generate boilerplate code",
		Long: `Generate boilerplate code for controllers, providers, and middleware.

Types:
  controller  - HTTP controller with CRUD methods
  provider    - Dependency injection provider
  middleware  - HTTP middleware`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("TODO: Implement glib make %s %s\n", args[0], args[1])
			return nil
		},
	}

	return cmd
}
