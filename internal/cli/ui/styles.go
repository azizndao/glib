package ui

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
func Success(text string) string {
	return Green + IconSuccess + " " + text + Reset
}

func Error(text string) string {
	return Red + IconError + " " + text + Reset
}

func Warning(text string) string {
	return Yellow + IconWarning + " " + text + Reset
}

func Info(text string) string {
	return Cyan + IconInfo + " " + text + Reset
}

func Muted(text string) string {
	return Gray + text + Reset
}

func Primary(text string) string {
	return Blue + text + Reset
}

func BoldText(text string) string {
	return "\033[1m" + text + Reset
}
