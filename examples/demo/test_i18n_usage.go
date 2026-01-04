package main

import (
	"context"
	"fmt"
	"glib/demo/generated/i18n"
)

// This file demonstrates the improved i18n usage

func demonstrateI18nFeatures() {
	// 1. Using locale constants (type-safe)
	fmt.Println("=== Locale Constants ===")
	fmt.Printf("English: %s\n", i18n.LocaleEN)
	fmt.Printf("French: %s\n", i18n.LocaleFR)
	fmt.Printf("Supported: %v\n", i18n.SupportedLocales)

	// 2. Create translator - values come from config
	translator, err := i18n.NewTranslator(
		"locales",
		i18n.LocaleEN,         // Use constant instead of hardcoded string
		i18n.SupportedLocales, // Use constant array from generated code
	)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// 3. Test English (default)
	fmt.Println("\n=== English ===")
	fmt.Println(translator.Errors.Posts.NotFound(ctx, "abc123"))
	fmt.Println(translator.Success.PostCreated(ctx, "My First Post"))

	// 4. Test French using WithLocale
	ctxFr := i18n.WithLocale(ctx, i18n.LocaleFR) // Use constant
	fmt.Println("\n=== French ===")
	fmt.Println(translator.Errors.Posts.NotFound(ctxFr, "abc123"))
	fmt.Println(translator.Success.PostCreated(ctxFr, "Mon Premier Article"))

	// 5. Middleware usage - no need to pass config!
	// Before: i18n.LocaleDetectorMiddleware("en", []string{"en", "fr"})
	// After:  translator.Middleware()
	fmt.Println("\n=== Middleware ===")
	fmt.Println("Simply use: app.Router.Use(app.Translator.Middleware())")
	fmt.Println("No need to pass default locale or supported locales!")
}
