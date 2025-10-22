// Package validation provides request validation with multi-language error messages.
package validation

import (
	"os"
	"reflect"
	"strings"

	"github.com/azizndao/glib/errors"
	"github.com/azizndao/glib/slog"
	"github.com/go-playground/locales"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

// Validator wraps go-playground validator with translator support
type Validator struct {
	validate *validator.Validate
	uni      *ut.UniversalTranslator
}

// ValidatorConfig holds configuration for the validator
type ValidatorConfig struct {
	Logger *slog.Logger
	// DefaultLocale is the default language for validation messages (e.g., "en")
	DefaultLocale string
	// UseJSONFieldNames determines if JSON tag names should be used in error messages
	UseJSONFieldNames bool
	// Locales is a list of additional locales to register with the validator
	Locales []LocaleConfig
}

// TranslationRegistrar is a function that registers translations for a locale
type TranslationRegistrar func(v *validator.Validate, trans ut.Translator) error

// LocaleConfig holds configuration for a locale
type LocaleConfig struct {
	Locale    locales.Translator
	Registrar TranslationRegistrar
}

// Locale creates a new locale configuration
func Locale(locale locales.Translator, registrar TranslationRegistrar) LocaleConfig {
	return LocaleConfig{
		Locale:    locale,
		Registrar: registrar,
	}
}

// DefaultValidatorConfig returns default validator configuration
func DefaultValidatorConfig() ValidatorConfig {
	return ValidatorConfig{
		DefaultLocale:     "en",
		UseJSONFieldNames: true,
	}
}

// NewValidator creates a new validator instance with the given configuration
func NewValidator(cfg ValidatorConfig) *Validator {

	v := validator.New()

	// Setup universal translator with English as default
	english := en.New()
	uni := ut.New(english, english)

	for _, locale := range cfg.Locales {
		uni.AddTranslator(locale.Locale, true)
		trans, _ok := uni.GetTranslator(locale.Locale.Locale())
		if !_ok {
			cfg.Logger.Error(errors.New("failed to get translator"), "locale", locale.Locale.Locale())
			os.Exit(0)
		}
		locale.Registrar(v, trans)
	}

	// Register JSON tag names if configured
	if cfg.UseJSONFieldNames {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name == "" {
				return fld.Name
			}
			return name
		})
	}

	validator := &Validator{
		validate: v,
		uni:      uni,
	}

	// Register default English translations
	if trans, ok := uni.GetTranslator("en"); ok {
		_ = en_translations.RegisterDefaultTranslations(v, trans)
	}

	return validator
}

// Validate validates a struct and returns formatted errors
func (v *Validator) Validate(data any, locale ...string) error {
	if err := v.validate.Struct(data); err != nil {
		lang := "en"
		if len(locale) > 0 {
			lang = locale[0]
		}
		return v.formatValidationErrors(err, lang)
	}
	return nil
}

// formatValidationErrors formats validation errors using the translator
func (v *Validator) formatValidationErrors(err error, locale string) error {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return errors.BadRequest("Validation failed", err)
	}

	trans, ok := v.uni.GetTranslator(locale)
	if !ok {
		// Fallback to English if locale not found
		trans, _ = v.uni.GetTranslator("en")
	}

	errs := make(map[string]string)
	for _, fieldError := range validationErrors {
		// Use the translator for user-friendly messages
		errs[fieldError.Field()] = fieldError.Translate(trans)
	}

	return errors.UnprocessableEntity(errs, err)
}
