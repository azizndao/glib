package validation

import (
	"log/slog"

	"github.com/azizndao/grouter"
)

// Middleware creates a middleware that injects the validator into the request context
// Accepts optional locale configurations to register additional languages
// Example: Middleware(Locale(fr.New(), fr_translations.RegisterDefaultTranslations))
func Middleware(locales ...LocaleConfig) grouter.Middleware {
	// Create validator with default configuration
	validator := NewValidator()

	// Register additional locales
	for _, locale := range locales {
		if err := validator.AddLocale(locale.Locale, locale.Registrar); err != nil {
			// Log error but don't fail - continue with other locales
			// The validator will fall back to English for this locale
			slog.Warn("failed to register locale",
				"locale", locale.Locale.Locale(),
				"error", err,
			)
			continue
		}
	}

	return func(next grouter.Handler) grouter.Handler {
		return func(c *grouter.Ctx) error {
			// Store validator in context
			c.Request = c.SetValue("validator", validator)
			return next(c)
		}
	}
}
