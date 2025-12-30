package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate project structure and annotations",
		Long: `Validate project structure and annotations without generating code.

Checks:
  - DI graph (circular dependencies, missing providers)
  - Routes (conflicts, parameter mismatches)
  - Types (signature validation)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("TODO: Implement glib validate")
			return nil
		},
	}
}
