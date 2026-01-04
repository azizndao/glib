package scanner

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// LocaleFile represents a parsed locale TOML file
type LocaleFile struct {
	Code         string                  // e.g., "en", "fr"
	FilePath     string                  // e.g., "locales/en.toml"
	Translations map[string]*Translation // Key -> Translation
}

// Translation represents a single translation entry
type Translation struct {
	Key        string              // e.g., "errors.posts.not_found"
	Template   string              // e.g., "Post with ID %s not found"
	Section    []string            // e.g., ["errors", "posts"]
	Name       string              // e.g., "not_found"
	Parameters []*TranslationParam // Inferred parameters
	HasParams  bool                // Whether it has format verbs
}

// TranslationParam represents an inferred parameter from format verb
type TranslationParam struct {
	Name  string // e.g., "id", "count", "arg1"
	Type  string // e.g., "string", "int", "float64"
	Verb  string // e.g., "%s", "%d", "%.2f"
	Index int    // Position in parameter list
}

// FormatVerbRegex matches Go format verbs
// Supports: %v, %T, %t, %b, %c, %d, %o, %O, %q, %x, %X, %U, %e, %E, %f, %F, %g, %G, %s, %p
// With optional: [flags] [width] [.precision]
var FormatVerbRegex = regexp.MustCompile(`%(?:\[\d+\])?(?:[-+# 0])?(?:\*|\d+)?(?:\.(?:\*|\d+))?[vTtbcdoOqxXUeEfFgGsp]`)

// ScanLocales scans all locale TOML files in a directory
func ScanLocales(localesDir string) ([]*LocaleFile, error) {
	// Check if directory exists
	if _, err := os.Stat(localesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("locales directory does not exist: %s", localesDir)
	}

	// Find all .toml files
	pattern := filepath.Join(localesDir, "*.toml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob locale files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no locale files found in: %s", localesDir)
	}

	// Parse each locale file
	var localeFiles []*LocaleFile
	for _, file := range files {
		localeFile, err := parseLocaleFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file, err)
		}
		localeFiles = append(localeFiles, localeFile)
	}

	return localeFiles, nil
}

// parseLocaleFile parses a single locale TOML file
func parseLocaleFile(filePath string) (*LocaleFile, error) {
	// Extract locale code from filename (e.g., "en.toml" -> "en")
	filename := filepath.Base(filePath)
	code := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Parse TOML
	var data map[string]any
	if _, err := toml.DecodeFile(filePath, &data); err != nil {
		return nil, fmt.Errorf("failed to decode TOML: %w", err)
	}

	// Parse translations recursively
	translations := parseTranslations(data, nil)

	return &LocaleFile{
		Code:         code,
		FilePath:     filePath,
		Translations: translations,
	}, nil
}

// parseTranslations recursively parses TOML structure into flat translation map
func parseTranslations(data map[string]any, prefix []string) map[string]*Translation {
	translations := make(map[string]*Translation)

	for key, value := range data {
		currentPath := append(prefix, key)

		switch v := value.(type) {
		case string:
			// Leaf node - this is a translation
			fullKey := strings.Join(currentPath, ".")
			translation := &Translation{
				Key:      fullKey,
				Template: v,
				Section:  prefix,
				Name:     key,
			}

			// Parse format verbs and infer parameters
			translation.Parameters = inferParameters(v, key)
			translation.HasParams = len(translation.Parameters) > 0

			translations[fullKey] = translation

		case map[string]any:
			// Nested section - recurse
			nested := parseTranslations(v, currentPath)
			maps.Copy(translations, nested)
		}
	}

	return translations
}

// inferParameters extracts and types parameters from format string
func inferParameters(template string, key string) []*TranslationParam {
	verbs := FormatVerbRegex.FindAllString(template, -1)
	if len(verbs) == 0 {
		return nil
	}

	params := make([]*TranslationParam, len(verbs))

	for i, verb := range verbs {
		paramType := verbToGoType(verb)
		paramName := inferParamName(template, key, i, paramType)

		params[i] = &TranslationParam{
			Name:  paramName,
			Type:  paramType,
			Verb:  verb,
			Index: i,
		}
	}

	return params
}

// verbToGoType maps format verb to Go type
func verbToGoType(verb string) string {
	// Extract base verb character (last character)
	if len(verb) == 0 {
		return "any"
	}

	baseVerb := verb[len(verb)-1]

	switch baseVerb {
	case 's', 'q': // string, quoted string
		return "string"
	case 'd', 'i', 'b', 'o', 'O', 'x', 'X', 'U': // integer formats
		return "int"
	case 'f', 'e', 'E', 'g', 'G', 'F': // float formats
		return "float64"
	case 't': // bool
		return "bool"
	case 'c': // rune/character
		return "rune"
	case 'p': // pointer
		return "uintptr"
	case 'v', 'T': // any value, type
		return "any"
	default:
		return "any"
	}
}

// inferParamName generates meaningful parameter names based on context
func inferParamName(template string, key string, index int, paramType string) string {
	lower := strings.ToLower(template)
	lowerKey := strings.ToLower(key)

	// Common patterns based on template content and key
	if paramType == "string" {
		if strings.Contains(lower, "id") || strings.Contains(lowerKey, "id") {
			if index == 0 {
				return "id"
			}
			return fmt.Sprintf("id%d", index)
		}
		if strings.Contains(lower, "name") || strings.Contains(lowerKey, "name") {
			if index == 0 {
				return "name"
			}
			return fmt.Sprintf("name%d", index)
		}
		if strings.Contains(lower, "email") {
			return "email"
		}
		if strings.Contains(lower, "username") {
			return "username"
		}
		if strings.Contains(lower, "field") {
			return "fieldName"
		}
		if strings.Contains(lower, "status") {
			return "status"
		}
		if strings.Contains(lower, "title") {
			return "title"
		}
	}

	if paramType == "int" {
		if strings.Contains(lower, "count") {
			return "count"
		}
		if strings.Contains(lower, "minutes") {
			return "minutes"
		}
		if strings.Contains(lower, "hours") {
			return "hours"
		}
		if strings.Contains(lower, "days") {
			return "days"
		}
		if strings.Contains(lower, "attempt") {
			return "attempts"
		}
		if strings.Contains(lower, "min") && strings.Contains(lower, "max") {
			if index == 0 {
				return "min"
			}
			if index == 1 {
				return "max"
			}
		}
		if strings.Contains(lower, "length") {
			if strings.Contains(lower, "current") && index == 1 {
				return "currentLength"
			}
			if strings.Contains(lower, "min") {
				return "minLength"
			}
			if strings.Contains(lower, "max") {
				return "maxLength"
			}
		}
	}

	if paramType == "float64" {
		if strings.Contains(lower, "price") {
			if index == 0 {
				return "oldPrice"
			}
			if index == 1 {
				return "newPrice"
			}
			return "price"
		}
		if strings.Contains(lower, "percent") || strings.Contains(lower, "%") {
			return "percentage"
		}
		if strings.Contains(lower, "amount") {
			return "amount"
		}
	}

	// Fallback to generic names
	return fmt.Sprintf("arg%d", index+1)
}

// ValidateTranslationCompleteness checks all locales have the same keys
func ValidateTranslationCompleteness(locales []*LocaleFile, defaultLocale string) error {
	if len(locales) == 0 {
		return fmt.Errorf("no locales to validate")
	}

	// Find base locale
	var baseLocale *LocaleFile
	for _, locale := range locales {
		if locale.Code == defaultLocale {
			baseLocale = locale
			break
		}
	}

	if baseLocale == nil {
		return fmt.Errorf("default locale %s not found", defaultLocale)
	}

	// Check each locale against base
	for _, locale := range locales {
		if locale.Code == defaultLocale {
			continue
		}

		// Check for missing keys
		for key := range baseLocale.Translations {
			if _, exists := locale.Translations[key]; !exists {
				return fmt.Errorf(
					"locale %s missing translation for key: %s",
					locale.Code,
					key,
				)
			}
		}

		// Check for extra keys (warning, not error)
		for key := range locale.Translations {
			if _, exists := baseLocale.Translations[key]; !exists {
				fmt.Printf(
					"Warning: locale %s has unexpected key: %s (not in default locale)\n",
					locale.Code,
					key,
				)
			}
		}
	}

	return nil
}

// ValidateFormatVerbs checks all locales have matching format verbs for each key
func ValidateFormatVerbs(locales []*LocaleFile) error {
	if len(locales) <= 1 {
		return nil // Nothing to compare
	}

	baseLocale := locales[0]

	for key, baseTranslation := range baseLocale.Translations {
		baseVerbs := FormatVerbRegex.FindAllString(baseTranslation.Template, -1)

		for _, locale := range locales[1:] {
			translation, exists := locale.Translations[key]
			if !exists {
				continue // Will be caught by completeness check
			}

			verbs := FormatVerbRegex.FindAllString(translation.Template, -1)

			// Check count matches
			if len(baseVerbs) != len(verbs) {
				return fmt.Errorf(
					"format verb count mismatch for key '%s': %s has %d verbs, %s has %d verbs",
					key,
					baseLocale.Code,
					len(baseVerbs),
					locale.Code,
					len(verbs),
				)
			}

			// Check types match
			for i := range baseVerbs {
				baseType := verbToGoType(baseVerbs[i])
				localeType := verbToGoType(verbs[i])

				if baseType != localeType {
					return fmt.Errorf(
						"format verb type mismatch for key '%s' at position %d: %s has %s (%s), %s has %s (%s)",
						key,
						i,
						baseLocale.Code,
						baseVerbs[i],
						baseType,
						locale.Code,
						verbs[i],
						localeType,
					)
				}
			}
		}
	}

	return nil
}
