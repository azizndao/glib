package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/azizndao/glib/internal/cli/ui"
	"github.com/azizndao/glib/internal/scanner"
	"github.com/azizndao/glib/internal/validator"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
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

type validatePhase string

const (
	validatePhaseScanning   validatePhase = "scanning"
	validatePhaseValidating validatePhase = "validating"
	validatePhaseDone       validatePhase = "done"
	validatePhaseError      validatePhase = "error"
)

type validateModel struct {
	spinner  spinner.Model
	phase    validatePhase
	project  *scanner.Project
	errors   []string
	warnings []string
	duration time.Duration
	renderer *ui.Renderer
	verbose  bool
}

type validateScanCompleteMsg struct {
	project  *scanner.Project
	err      error
	duration time.Duration
}

type validateValidationCompleteMsg struct {
	errors   []string
	warnings []string
	duration time.Duration
	err      error
}

func initialValidateModel(verbose bool) validateModel {
	return validateModel{
		spinner:  ui.NewSpinner(),
		phase:    validatePhaseScanning,
		renderer: ui.NewRenderer(),
		verbose:  verbose,
	}
}

func (m validateModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		doValidateScan(),
	)
}

func doValidateScan() tea.Cmd {
	return func() tea.Msg {
		start := time.Now()

		scan, err := scanner.New(".")
		if err != nil {
			return validateScanCompleteMsg{err: fmt.Errorf("failed to create scanner: %w", err), duration: time.Since(start)}
		}

		project, err := scan.Scan()
		if err != nil {
			return validateScanCompleteMsg{err: fmt.Errorf("failed to scan project: %w", err), duration: time.Since(start)}
		}

		return validateScanCompleteMsg{project: project, duration: time.Since(start)}
	}
}

func doValidation(project *scanner.Project) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()

		v := validator.New()
		err := v.Validate(project)

		var errors []string
		for _, verr := range v.Errors() {
			errors = append(errors, verr.Message)
		}

		var warnings []string
		for _, warn := range v.Warnings() {
			warnings = append(warnings, warn.Message)
		}

		return validateValidationCompleteMsg{
			errors:   errors,
			warnings: warnings,
			duration: time.Since(start),
			err:      err,
		}
	}
}

func (m validateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case validateScanCompleteMsg:
		if msg.err != nil {
			m.phase = validatePhaseError
			m.errors = []string{msg.err.Error()}
			return m, tea.Quit
		}
		m.project = msg.project
		m.phase = validatePhaseValidating
		return m, doValidation(msg.project)

	case validateValidationCompleteMsg:
		m.duration = msg.duration
		m.errors = msg.errors
		m.warnings = msg.warnings

		if msg.err != nil {
			m.phase = validatePhaseError
		} else {
			m.phase = validatePhaseDone
		}
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m validateModel) View() string {
	if !m.renderer.IsTTY() {
		return m.plainView()
	}

	switch m.phase {
	case validatePhaseScanning:
		return fmt.Sprintf("%s Scanning project...", m.spinner.View())

	case validatePhaseValidating:
		return fmt.Sprintf("%s Validating...", m.spinner.View())

	case validatePhaseError:
		output := "\n" + ui.Error("Validation failed") + "\n\n"
		for i, err := range m.errors {
			output += fmt.Sprintf("  %d. %s\n", i+1, err)
		}
		return output

	case validatePhaseDone:
		if len(m.errors) > 0 {
			output := ui.Error("Validation failed") + "\n\n"
			for i, err := range m.errors {
				output += fmt.Sprintf("  %d. %s\n", i+1, err)
			}
			return output
		}

		if len(m.warnings) > 0 {
			output := ui.Warning(fmt.Sprintf("Validation passed with %d warnings", len(m.warnings))) + "\n"
			if m.verbose {
				output += "\n"
				for i, warn := range m.warnings {
					output += fmt.Sprintf("  %d. %s\n", i+1, warn)
				}
			}
			return output
		}

		return ui.Success("Validation passed") + "\n"
	}

	return ""
}

func (m validateModel) plainView() string {
	switch m.phase {
	case validatePhaseScanning:
		return "Scanning project..."
	case validatePhaseValidating:
		return "Validating..."
	case validatePhaseError:
		output := "Validation failed\n"
		for i, err := range m.errors {
			output += fmt.Sprintf("  %d. %s\n", i+1, err)
		}
		return output
	case validatePhaseDone:
		if len(m.errors) > 0 {
			output := "Validation failed\n"
			for i, err := range m.errors {
				output += fmt.Sprintf("  %d. %s\n", i+1, err)
			}
			return output
		}

		if len(m.warnings) > 0 {
			return fmt.Sprintf("Validation passed with %d warnings\n", len(m.warnings))
		}

		return "Validation passed\n"
	}
	return ""
}

func runValidate(opts *validateOptions) error {
	// Change to project directory
	if opts.dir != "." {
		if err := os.Chdir(opts.dir); err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}
	}

	// Check TTY and run appropriate UI
	renderer := ui.NewRenderer()
	if !renderer.IsTTY() {
		// Simple non-TTY output for CI/CD
		return runValidateSimple(opts.verbose)
	}

	// Run with Bubble Tea UI
	m := initialValidateModel(opts.verbose)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Check if validation succeeded
	final := finalModel.(validateModel)
	if final.phase == validatePhaseError || len(final.errors) > 0 {
		return fmt.Errorf("validation failed")
	}

	return nil
}

// runValidateSimple is fallback for non-TTY environments (CI/CD)
func runValidateSimple(verbose bool) error {
	fmt.Println("Scanning project...")

	scan, err := scanner.New(".")
	if err != nil {
		return fmt.Errorf("failed to create scanner: %w", err)
	}

	project, err := scan.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan project: %w", err)
	}

	fmt.Println("Validating...")

	v := validator.New()
	if err := v.Validate(project); err != nil {
		for i, verr := range v.Errors() {
			fmt.Printf("  %d. %s\n", i+1, verr.Message)
		}
		return fmt.Errorf("validation failed")
	}

	warnings := v.Warnings()
	if len(warnings) > 0 {
		fmt.Printf("Validation passed with %d warnings\n", len(warnings))
		if verbose {
			for i, warn := range warnings {
				fmt.Printf("  %d. %s\n", i+1, warn.Message)
			}
		}
	} else {
		fmt.Println("Validation passed")
	}

	return nil
}
