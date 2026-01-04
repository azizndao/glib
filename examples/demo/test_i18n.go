package main

import (
	"context"
	"fmt"
	"glib/demo/generated/i18n"
)

func testI18n() {
	translator, err := i18n.NewTranslator("locales", "en", []string{"en", "fr"})
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// Test English (default)
	fmt.Println("=== English ===")
	fmt.Println(translator.Errors.Posts.NotFound(ctx, "abc123"))
	fmt.Println(translator.Success.PostCreated(ctx, "My First Post"))
	fmt.Println(translator.Validation.Required(ctx, "username"))

	// Test French
	ctxFr := i18n.WithLocale(ctx, "fr")
	fmt.Println("\n=== French ===")
	fmt.Println(translator.Errors.Posts.NotFound(ctxFr, "abc123"))
	fmt.Println(translator.Success.PostCreated(ctxFr, "Mon Premier Article"))
	fmt.Println(translator.Validation.Required(ctxFr, "nom d'utilisateur"))
}
