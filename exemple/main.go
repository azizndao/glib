// Package main is an example package for grouter
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/azizndao/grouter"
	"github.com/azizndao/grouter/errors"
	"github.com/azizndao/grouter/middleware"
	"github.com/azizndao/grouter/validation"
	"github.com/go-playground/locales/es"
	"github.com/go-playground/locales/fr"
	es_translations "github.com/go-playground/validator/v10/translations/es"
	fr_translations "github.com/go-playground/validator/v10/translations/fr"
)

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required,min=2"`
	Age      int    `json:"age" validate:"required,gte=18"`
}

type CreateProductRequest struct {
	Name        string  `json:"name" validate:"required,min=3,max=100"`
	Description string  `json:"description" validate:"required,min=10"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	Stock       int     `json:"stock" validate:"required,gte=0"`
	SKU         string  `json:"sku" validate:"required,alphanum"`
}

func main() {
	router := grouter.NewRouter()

	// Add comprehensive middleware stack
	router.Use(
		middleware.RequestID(),                           // Generate unique request IDs
		middleware.Recovery(),                            // Panic recovery
		middleware.Logger(),                              // Request logging
		middleware.Compress(),                            // Response compression
		middleware.BodyLimit5MB(),                        // Limit request body size to 5MB
		middleware.RateLimit(),                           // Rate limiting (100 req/min by default)
		middleware.CORS(middleware.DefaultCORSOptions()), // CORS support
		validation.Middleware( // Request validation with i18n
			validation.Locale(fr.New(), fr_translations.RegisterDefaultTranslations),
			validation.Locale(es.New(), es_translations.RegisterDefaultTranslations),
		),
	)

	// Simple hello endpoint
	router.Get("/hello", func(c *grouter.Ctx) error {
		return c.Status(200).JSON(map[string]string{"message": "Hello World"})
	})

	// Hello with name parameter
	router.Get("/hello/{name}", func(c *grouter.Ctx) error {
		return c.Status(200).JSON(map[string]string{
			"message": fmt.Sprintf("Hello %s", c.PathValue("name")),
			"query":   c.Query("q"),
		})
	})

	// Error example
	router.Get("/error", func(c *grouter.Ctx) error {
		return errors.ErrorBadRequest("Bad request example", nil)
	})

	// User registration with validation
	// Validates based on Accept-Language header
	router.Post("/register", registerUser)

	// Product creation with validation
	router.Post("/products", createProduct)

	// Request ID demonstration
	router.Get("/request-id", func(c *grouter.Ctx) error {
		requestID := middleware.GetRequestID(c)
		return c.Status(200).JSON(map[string]string{
			"request_id": requestID,
			"message":    "Request ID is automatically added to all requests",
		})
	})

	// Demonstrate timeout with a slow endpoint
	slowRouter := router.Group("/slow", middleware.Timeout(2*time.Second))
	slowRouter.Get("/endpoint", func(c *grouter.Ctx) error {
		// Simulate slow processing
		time.Sleep(3 * time.Second)
		return c.Status(200).JSON(map[string]string{"message": "This should timeout"})
	})

	slog.Info("Server started on :8080")

	http.ListenAndServe(":8080", router.Handler())
}

func registerUser(c *grouter.Ctx) error {
	var req RegisterRequest

	// ValidateBody automatically uses Accept-Language header
	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	return c.Status(201).JSON(map[string]any{
		"message": "User registered successfully",
		"user": map[string]any{
			"name":  req.Name,
			"email": req.Email,
			"age":   req.Age,
		},
	})
}

func createProduct(c *grouter.Ctx) error {
	var req CreateProductRequest

	// Parse and validate in one call
	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	return c.Status(201).JSON(map[string]any{
		"message": "Product created successfully",
		"product": req,
	})
}
