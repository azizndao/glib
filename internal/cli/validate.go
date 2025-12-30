package cli

import (
	"fmt"
	"os"

	"github.com/goyave/glib/v2/internal/scanner"
	"github.com/goyave/glib/v2/internal/validator"
	"github.com/spf13/cobra"
)

type validateOptions struct {
	dir     string
	verbose bool
}

func newValidateCmd() *cobra.Command {
	opts := &validateOptions{}

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate project structure and annotations",
		Long: `Validate project structure and annotations without generating code.

Checks:
  - DI graph (circular dependencies, missing providers)
  - Routes (conflicts, parameter mismatches)
  - Types (signature validation)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(opts)
		},
	}

	cmd.Flags().StringVar(&opts.dir, "dir", ".", "Project root directory")
	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "Verbose output")

	return cmd
}

func runValidate(opts *validateOptions) error {
	// Change to project directory
	if opts.dir != "." {
		if err := os.Chdir(opts.dir); err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}
	}

	fmt.Println("🔍 Scanning project...")

	// Create scanner
	scan, err := scanner.New(".")
	if err != nil {
		return fmt.Errorf("failed to create scanner: %w", err)
	}

	// Scan project
	project, err := scan.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan project: %w", err)
	}

	// Count handlers
	totalHandlers := 0
	for _, ctrl := range project.Controllers {
		totalHandlers += len(ctrl.Handlers)
	}

	fmt.Printf("   Found %d controllers\n", len(project.Controllers))
	fmt.Printf("   Found %d handlers\n", totalHandlers)
	fmt.Printf("   Found %d providers\n", len(project.Providers))
	fmt.Printf("   Found %d middleware\n", len(project.Middleware))

	// Validate
	fmt.Println("\n🔍 Validating...")
	v := validator.New()
	if err := v.Validate(project); err != nil {
		fmt.Println()

		// Print all errors
		errors := v.Errors()
		for i, verr := range errors {
			fmt.Printf("   %d. ❌ %s\n", i+1, verr.Message)
			if opts.verbose {
				fmt.Printf("      Location: %s\n", verr.Location)
			}
		}

		fmt.Printf("\n❌ Validation failed with %d errors\n", len(errors))
		return fmt.Errorf("validation failed")
	}

	// Print warnings
	warnings := v.Warnings()
	if len(warnings) > 0 {
		fmt.Println()
		for i, warn := range warnings {
			fmt.Printf("   %d. ⚠️  %s\n", i+1, warn.Message)
			if opts.verbose {
				fmt.Printf("      Location: %s\n", warn.Location)
			}
		}
		fmt.Printf("\n⚠️  Validation passed with %d warnings\n", len(warnings))
	} else {
		fmt.Println("\n✅ Validation passed with no errors or warnings")
	}

	return nil
}
