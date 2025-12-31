package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/azizndao/glib/internal/cli/ui"
	"github.com/azizndao/glib/internal/generator"
	"github.com/azizndao/glib/internal/scanner"
	"github.com/azizndao/glib/internal/validator"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type generateOptions struct {
	dir     string
	output  string
	config  string
	verbose bool
	watch   bool
}

func newGenerateCmd() *cobra.Command {
	opts := &generateOptions{}

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate code from annotations",
		Long: `Scan Go source code for annotations and generate code.

Generates:
  - DI container (generated/di.gen.go)
  - Route registration (generated/routes.gen.go)  
  - Request parsers (generated/parsers.gen.go)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(opts)
		},
	}

	cmd.Flags().StringVar(&opts.dir, "dir", ".", "Project root directory")
	cmd.Flags().StringVar(&opts.output, "output", "", "Output directory (from glib.json)")
	cmd.Flags().StringVar(&opts.config, "config", "glib.json", "Config file")
	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "Verbose output")
	cmd.Flags().BoolVar(&opts.watch, "watch", false, "Watch mode")

	return cmd
}

type generatePhase string

const (
	phaseScanning   generatePhase = "scanning"
	phaseValidating generatePhase = "validating"
	phaseGenerating generatePhase = "generating"
	phaseDone       generatePhase = "done"
	phaseError      generatePhase = "error"
)

type generateModel struct {
	spinner     spinner.Model
	phase       generatePhase
	project     *scanner.Project
	outputDir   string
	pkgName     string
	scanStart   time.Time
	scanDur     time.Duration
	validateDur time.Duration
	generateDur time.Duration
	errors      []string
	warnings    []string
	renderer    *ui.Renderer
	verbose     bool
}

type scanCompleteMsg struct {
	project  *scanner.Project
	duration time.Duration
	err      error
}

type validateCompleteMsg struct {
	errors   []string
	warnings []string
	duration time.Duration
	err      error
}

type generateCompleteMsg struct {
	duration time.Duration
	err      error
}

func initialGenerateModel(outputDir, pkgName string, verbose bool) generateModel {
	return generateModel{
		spinner:   ui.NewSpinner(),
		phase:     phaseScanning,
		outputDir: outputDir,
		pkgName:   pkgName,
		scanStart: time.Now(),
		renderer:  ui.NewRenderer(),
		verbose:   verbose,
	}
}

func (m generateModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		doScan(),
	)
}

func doScan() tea.Cmd {
	return func() tea.Msg {
		start := time.Now()

		scan, err := scanner.New(".")
		if err != nil {
			return scanCompleteMsg{err: fmt.Errorf("failed to create scanner: %w", err), duration: time.Since(start)}
		}

		project, err := scan.Scan()
		if err != nil {
			return scanCompleteMsg{err: fmt.Errorf("failed to scan project: %w", err), duration: time.Since(start)}
		}

		return scanCompleteMsg{project: project, duration: time.Since(start)}
	}
}

func doValidate(project *scanner.Project) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()

		val := validator.New()
		err := val.Validate(project)

		var errors []string
		for _, verr := range val.Errors() {
			errors = append(errors, verr.Message)
		}

		var warnings []string
		for _, warn := range val.Warnings() {
			warnings = append(warnings, warn.Message)
		}

		return validateCompleteMsg{
			errors:   errors,
			warnings: warnings,
			duration: time.Since(start),
			err:      err,
		}
	}
}

func doGenerate(project *scanner.Project, outputDir, pkgName string) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()

		gen := generator.New(project, outputDir, pkgName)
		err := gen.Generate()

		return generateCompleteMsg{
			duration: time.Since(start),
			err:      err,
		}
	}
}

func (m generateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case scanCompleteMsg:
		if msg.err != nil {
			m.phase = phaseError
			m.errors = []string{msg.err.Error()}
			return m, tea.Quit
		}
		m.project = msg.project
		m.scanDur = msg.duration
		m.phase = phaseValidating
		return m, doValidate(msg.project)

	case validateCompleteMsg:
		m.validateDur = msg.duration
		m.errors = msg.errors
		m.warnings = msg.warnings

		if msg.err != nil {
			m.phase = phaseError
			return m, tea.Quit
		}

		m.phase = phaseGenerating
		return m, doGenerate(m.project, m.outputDir, m.pkgName)

	case generateCompleteMsg:
		m.generateDur = msg.duration
		if msg.err != nil {
			m.phase = phaseError
			m.errors = []string{msg.err.Error()}
			return m, tea.Quit
		}
		m.phase = phaseDone
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m generateModel) View() string {
	if !m.renderer.IsTTY() {
		return m.plainView()
	}

	switch m.phase {
	case phaseScanning:
		return fmt.Sprintf("%s Scanning project...", m.spinner.View())

	case phaseValidating:
		return fmt.Sprintf("%s Validating...", m.spinner.View())

	case phaseGenerating:
		return fmt.Sprintf("%s Generating code...", m.spinner.View())

	case phaseError:
		var output strings.Builder
		output.WriteString("\n" + ui.Error("Generation failed") + "\n\n")
		for _, err := range m.errors {
			output.WriteString("  " + ui.IconBullet + " " + err + "\n")
		}
		return output.String()

	case phaseDone:
		output := ui.Success(fmt.Sprintf("Generation complete (%dms)", m.totalDuration().Milliseconds())) + "\n"

		if m.verbose && m.project != nil {
			output += "\n  " + ui.MutedStyle.Render(fmt.Sprintf(
				"%s %d controllers, %s %d providers, %s %d middleware",
				ui.IconController, len(m.project.Controllers),
				ui.IconProvider, len(m.project.Providers),
				ui.IconMiddleware, len(m.project.Middleware),
			)) + "\n"
		}

		return output
	}

	return ""
}

func (m generateModel) plainView() string {
	switch m.phase {
	case phaseScanning:
		return "Scanning project..."
	case phaseValidating:
		return "Validating..."
	case phaseGenerating:
		return "Generating code..."
	case phaseError:
		var output strings.Builder
		output.WriteString("Generation failed\n")
		for _, err := range m.errors {
			output.WriteString("  " + err + "\n")
		}
		return output.String()
	case phaseDone:
		return fmt.Sprintf("Generation complete (%dms)\n", m.totalDuration().Milliseconds())
	}
	return ""
}

func (m generateModel) totalDuration() time.Duration {
	return m.scanDur + m.validateDur + m.generateDur
}

func runGenerate(opts *generateOptions) error {
	if opts.watch {
		return fmt.Errorf("watch mode not implemented yet")
	}

	// Change to project directory
	if opts.dir != "." {
		if err := os.Chdir(opts.dir); err != nil {
			return fmt.Errorf("failed to change directory: %w", err)
		}
	}

	// Load config
	cfg, err := loadGlibrc()
	if err != nil {
		return fmt.Errorf("failed to load glib.json: %w", err)
	}

	// Determine output directory
	outputDir := opts.output
	if outputDir == "" {
		outputDir = cfg.Generate.Output
		if outputDir == "" {
			outputDir = "generated"
		}
	}

	// Determine package name
	pkgName := cfg.Generate.Package
	if pkgName == "" {
		pkgName = "generated"
	}

	// Check TTY and run appropriate UI
	renderer := ui.NewRenderer()
	if !renderer.IsTTY() {
		// Simple non-TTY output for CI/CD
		return runGenerateSimple(outputDir, pkgName)
	}

	// Run with Bubble Tea UI
	m := initialGenerateModel(outputDir, pkgName, opts.verbose)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Check if generation succeeded
	final := finalModel.(generateModel)
	if final.phase == phaseError {
		return fmt.Errorf("generation failed")
	}

	return nil
}

// runGenerateSimple is fallback for non-TTY environments (CI/CD)
func runGenerateSimple(outputDir, pkgName string) error {
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

	val := validator.New()
	if err := val.Validate(project); err != nil {
		for _, verr := range val.Errors() {
			fmt.Printf("  %s\n", verr.Message)
		}
		return err
	}

	fmt.Println("Generating code...")

	gen := generator.New(project, outputDir, pkgName)
	if err := gen.Generate(); err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	fmt.Println("Generation complete")

	return nil
}
