package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate code from annotations",
		Long: `Scan Go source code for annotations and generate code.

Generates:
  - Bootstrap code (generated/glib.gen.go)
  - DI container (generated/di.gen.go)
  - Route registration (generated/routes.gen.go)  
  - Request parsers (generated/parsers.gen.go)
  - Error handlers (generated/errors.gen.go)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("TODO: Implement glib generate")
			return nil
		},
	}
}
