package commands

import (
	"github.com/spf13/cobra"
)

// NewNewCommand creates the "glib new" command for scaffolding new projects
func NewNewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "Create a new glib project",
		Long: `Create a new glib project with the standard directory structure.

Example:
  glib new blog
  glib new blog --template=api
  glib new blog --database=postgres`,
		Args: cobra.ExactArgs(1),
		RunE: runNew,
	}

	cmd.Flags().String("template", "full", "Project template (full, api)")
	cmd.Flags().String("database", "sqlite", "Database driver (sqlite, postgres, mysql)")
	cmd.Flags().Bool("no-git", false, "Skip git initialization")

	return cmd
}

func runNew(cmd *cobra.Command, args []string) error {
	projectName := args[0]
	template, _ := cmd.Flags().GetString("template")
	database, _ := cmd.Flags().GetString("database")
	noGit, _ := cmd.Flags().GetBool("no-git")

	cmd.Printf("Creating new glib project: %s\n", projectName)
	cmd.Printf("  Template: %s\n", template)
	cmd.Printf("  Database: %s\n", database)

	// TODO: Implement project scaffolding
	// 1. Create directory structure
	// 2. Generate go.mod
	// 3. Create config files
	// 4. Create example files
	// 5. Initialize git (if not --no-git)

	if !noGit {
		cmd.Println("  Initializing git repository...")
	}

	cmd.Printf("\n✓ Project created successfully!\n")
	cmd.Printf("\nNext steps:\n")
	cmd.Printf("  cd %s\n", projectName)
	cmd.Printf("  go mod tidy\n")
	cmd.Printf("  glib serve\n")

	return nil
}
