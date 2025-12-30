package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDevCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dev",
		Short: "Start development server with hot reload",
		Long: `Start development server with automatic code generation and hot reload.

Uses Air for hot reload if available, falls back to basic file watcher.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("TODO: Implement glib dev")
			return nil
		},
	}
}
