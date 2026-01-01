package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/azizndao/glib/internal/cli/ui"
	"github.com/azizndao/glib/internal/scanner"
	"github.com/azizndao/glib/internal/validator"
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

	return runValidateSimple(opts.verbose)
}

// runValidateSimple performs validation with simple output
func runValidateSimple(verbose bool) error {
	start := time.Now()

	fmt.Println(ui.Info("Scanning project..."))

	scan, err := scanner.New(".")
	if err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to create scanner: %v", err)))
		return err
	}

	project, err := scan.Scan()
	if err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to scan project: %v", err)))
		return err
	}

	v := validator.New()
	if err := v.Validate(project); err != nil {
		fmt.Println(ui.Error("Validation failed"))
		for i, verr := range v.Errors() {
			fmt.Printf("  %d. %s\n", i+1, verr.Message)
		}
		return err
	}

	duration := time.Since(start)
	warnings := v.Warnings()

	if len(warnings) > 0 {
		fmt.Println(ui.Success(fmt.Sprintf("Validation passed (%dms)", duration.Milliseconds())))
		fmt.Println(ui.Warning(fmt.Sprintf("%d warnings", len(warnings))))
		if verbose {
			for i, warn := range warnings {
				fmt.Printf("  %d. %s\n", i+1, warn.Message)
			}
		}
	} else {
		fmt.Println(ui.Success(fmt.Sprintf("Validation passed (%dms)", duration.Milliseconds())))
	}

	if verbose {
		fmt.Printf("  %s\n", ui.Muted(fmt.Sprintf(
			"%d controllers, %d providers, %d middleware",
			len(project.Controllers),
			len(project.Providers),
			len(project.Middleware),
		)))
	}

	return nil
}
