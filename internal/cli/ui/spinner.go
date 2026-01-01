package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
)

func NewSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle
	return s
}
