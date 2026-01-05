package parsers

import (
	"strings"
)

// DetectLanguage extracts the primary language code from an Accept-Language header.
// Returns the language code (e.g., "en", "fr", "es") or an empty string if not found.
//
// Example:
//
//	DetectLanguage("en-US,en;q=0.9,fr;q=0.8") // returns "en"
//	DetectLanguage("fr-FR") // returns "fr"
//	DetectLanguage("") // returns ""
func DetectLanguage(acceptLangHeader string, defaultLang string) string {
	if acceptLangHeader == "" {
		return defaultLang
	}

	// Split by comma to get individual language tags
	langs := strings.Split(acceptLangHeader, ",")
	if len(langs) == 0 {
		return defaultLang
	}

	// Take the first language (highest priority)
	firstLang := strings.TrimSpace(langs[0])

	// Remove quality value if present (e.g., "en;q=0.9" -> "en")
	if idx := strings.Index(firstLang, ";"); idx != -1 {
		firstLang = firstLang[:idx]
	}

	// Extract language code (e.g., "en-US" -> "en")
	if before, _, ok := strings.Cut(firstLang, "-"); ok {
		return before
	}

	return firstLang
}
