package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Color palette
	ColorPrimary   = lipgloss.Color("39")  // Blue
	ColorSuccess   = lipgloss.Color("42")  // Green
	ColorError     = lipgloss.Color("196") // Red
	ColorWarning   = lipgloss.Color("214") // Orange
	ColorInfo      = lipgloss.Color("86")  // Cyan
	ColorMuted     = lipgloss.Color("241") // Gray
	ColorHighlight = lipgloss.Color("212") // Pink

	// Base styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			MarginBottom(1)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	InfoStyle = lipgloss.NewStyle().
			Foreground(ColorInfo)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Component styles
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(3)

	SubItemStyle = lipgloss.NewStyle().
			PaddingLeft(5).
			Foreground(ColorMuted)
)

func Success(text string) string {
	return SuccessStyle.Render(IconSuccess) + " " + text
}

func Error(text string) string {
	return ErrorStyle.Render(IconError) + " " + text
}

func Warning(text string) string {
	return WarningStyle.Render(IconWarning) + " " + text
}

func Info(text string) string {
	return InfoStyle.Render(IconInfo + " " + text)
}

func Muted(text string) string {
	return MutedStyle.Render(text)
}

func Primary(text string) string {
	return lipgloss.NewStyle().Foreground(ColorPrimary).Render(text)
}
