package ui

import "fmt"

// ANSI color codes
const (
	Reset = "\033[0m"
	Bold  = "\033[1m"

	// Colors
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Gray    = "\033[90m"
	White   = "\033[97m"

	// Bright colors
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
)

// Helper functions for colored output
func Successf(text string, args ...any) string {
	return Green + IconSuccess + " " + fmt.Sprintf(text, args...) + Reset
}

func Success(text string) string {
	return Green + IconSuccess + " " + text + Reset
}

func Errorf(text string, args ...any) string {
	return Red + IconError + " " + fmt.Sprintf(text, args...) + Reset
}

func Error(text string) string {
	return Red + IconError + " " + text + Reset
}

func Warningf(text string, args ...any) string {
	return Yellow + IconWarning + " " + fmt.Sprintf(text, args...) + Reset
}

func Warning(text string) string {
	return Yellow + IconWarning + " " + text + Reset
}

func Infof(text string, args ...any) string {
	return Cyan + IconInfo + " " + fmt.Sprintf(text, args...) + Reset
}

func Info(text string) string {
	return Cyan + IconInfo + " " + text + Reset
}

func Mutedf(text string, args ...any) string {
	return Gray + fmt.Sprintf(text, args...) + Reset
}

func Muted(text string) string {
	return Gray + text + Reset
}

func Primaryf(text string, args ...any) string {
	return Blue + fmt.Sprintf(text, args...) + Reset
}

func Primary(text string) string {
	return Blue + text + Reset
}

func BoldTextf(text string, args ...any) string {
	return Bold + fmt.Sprintf(text, args...) + Reset
}

func BoldText(text string) string {
	return Bold + text + Reset
}
