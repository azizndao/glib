package validator

import (
	"strings"

	"github.com/azizndao/glib/errs"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

// Validator is a wrapper around go-playground/validator with i18n support
type Validator struct {
	validate    *validator.Validate
	uni         *ut.UniversalTranslator
	defaultLang string
}

// NewValidator creates a new validator instance with default English language (for testing/backward compatibility)
// In production, the generated code will use NewValidatorWithTranslator
func NewValidator() *Validator {
	v := validator.New()
	return &Validator{
		validate:    v,
		uni:         nil, // No i18n support in simple mode
		defaultLang: "en",
	}
}

// NewValidatorWithTranslator creates a validator with a pre-configured UniversalTranslator
// This is used by generated code to only load configured locales
func NewValidatorWithTranslator(v *validator.Validate, uni *ut.UniversalTranslator, defaultLang string) *Validator {
	if defaultLang == "" {
		defaultLang = "en"
	}

	return &Validator{
		validate:    v,
		uni:         uni,
		defaultLang: defaultLang,
	}
}

// Validate validates a struct using the default language
func (v *Validator) Validate(s any) error {
	return v.ValidateWithLang(s, v.defaultLang)
}

// ValidateWithLang validates a struct with a specific language
func (v *Validator) ValidateWithLang(s any, lang string) error {
	return v.ValidateWithLangAndSection(s, lang, "body")
}

// ValidateWithLangAndSection validates a struct with language and section specification
func (v *Validator) ValidateWithLangAndSection(s any, lang string, section string) error {
	// Check if struct implements Validable interface
	if validable, ok := s.(Validable); ok {
		if !validable.Validate() {
			// Validation is disabled for this struct
			return nil
		}
	}

	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}

	// Convert validator errors to glib errors
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	// If no i18n support (simple mode), use nested errors with English messages
	if v.uni == nil {
		return v.convertToGoyaveErrorsSimple(validationErrs, section)
	}

	// Get translator for the specified language
	trans, found := v.uni.GetTranslator(lang)
	if !found {
		// Fallback to default language
		trans, _ = v.uni.GetTranslator(v.defaultLang)
	}

	return v.convertToGoyaveErrors(validationErrs, trans, section)
}

// convertToGoyaveErrorsSimple creates Goyave-style errors without i18n (for simple mode)
func (v *Validator) convertToGoyaveErrorsSimple(validationErrs validator.ValidationErrors, section string) error {
	goyaveErrs := NewValidationErrors()

	for _, fieldErr := range validationErrs {
		path := buildFieldPathArray(fieldErr)
		// Use simple English message since no i18n available
		message := fieldErr.Error()

		switch section {
		case "body":
			goyaveErrs.AddBodyError(path, message)
		case "query":
			goyaveErrs.AddQueryError(path, message)
		case "headers":
			goyaveErrs.AddHeaderError(path, message)
		}
	}

	return errs.B().
		Code(errs.InvalidArgument).
		Msg("Validation failed").
		Details(goyaveErrs).
		Err()
}

// convertToGoyaveErrors creates Goyave-style error format with i18n
func (v *Validator) convertToGoyaveErrors(validationErrs validator.ValidationErrors, trans ut.Translator, section string) error {
	goyaveErrs := NewValidationErrors()

	for _, fieldErr := range validationErrs {
		path := buildFieldPathArray(fieldErr)
		message := fieldErr.Translate(trans)

		switch section {
		case "body":
			goyaveErrs.AddBodyError(path, message)
		case "query":
			goyaveErrs.AddQueryError(path, message)
		case "headers":
			goyaveErrs.AddHeaderError(path, message)
		}
	}

	return errs.B().
		Code(errs.InvalidArgument).
		Msg("Validation failed").
		Details(goyaveErrs).
		Err()
}

// buildFieldPathArray creates an array of field names for nested paths
// Example: "CreatePostRequest.User.Address.City" -> ["user", "address", "city"]
func buildFieldPathArray(fe validator.FieldError) []string {
	namespace := fe.Namespace()

	// Remove the struct name prefix (e.g., "CreatePostRequest.Title" -> "Title")
	parts := strings.Split(namespace, ".")
	if len(parts) > 1 {
		parts = parts[1:] // Skip struct name
	}

	// Convert field names to snake_case
	for i, part := range parts {
		// Check if it's an array index [0], [1], etc
		if len(part) > 0 && part[0] >= '0' && part[0] <= '9' {
			// Keep array indices as-is
			parts[i] = part
		} else {
			parts[i] = toSnakeCase(part)
		}
	}

	return parts
}

// toSnakeCase converts PascalCase to snake_case
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// DetectLanguage parses Accept-Language header and returns the best match
func DetectLanguage(acceptLanguage string) string {
	if acceptLanguage == "" {
		return "en"
	}

	// Parse Accept-Language header (e.g., "es-MX,es;q=0.9,en;q=0.8")
	languages := strings.SplitSeq(acceptLanguage, ",")
	for lang := range languages {
		// Extract language code (before '-' or ';')
		langCode := strings.TrimSpace(lang)
		if idx := strings.IndexAny(langCode, "-;"); idx != -1 {
			langCode = langCode[:idx]
		}

		// Return first supported language
		switch langCode {
		case "en", "es", "fr":
			return langCode
		}
	}

	return "en" // Default fallback
}
