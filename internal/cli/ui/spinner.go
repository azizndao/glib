package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
)

// NewSpinner creates a styled spinner with braille dots
func NewSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot // Braille dots: ⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏
	s.Style = SpinnerStyle
	return s
}
