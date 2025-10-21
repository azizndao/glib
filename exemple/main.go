// Package main demonstrates the grouter.Server with comprehensive production-ready configuration
package main

import (
	"fmt"
	"time"

	"github.com/azizndao/grouter"
	"github.com/azizndao/grouter/errors"
	"github.com/azizndao/grouter/middleware"
	"github.com/azizndao/grouter/router"
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
	// Create server - all configuration loaded from environment variables
	// See .env.example for available configuration options
	// Set environment variables to customize the server behavior
	//
	// Optionally add validation locales for multi-language validation error messages
	server := grouter.New(
		validation.Locale(fr.New(), fr_translations.RegisterDefaultTranslations),
		validation.Locale(es.New(), es_translations.RegisterDefaultTranslations),
	)

	// Get the router from the server to register routes
	r := server.Router()

	// Register routes using the router
	// Simple hello endpoint
	r.Get("/hello", func(c *router.Ctx) error {
		return c.JSON(map[string]string{"message": "Hello World"})
	})

	// Hello with name parameter
	r.Get("/hello/{name}", func(c *router.Ctx) error {
		return c.JSON(map[string]string{
			"message": fmt.Sprintf("Hello %s", c.PathValue("name")),
			"query":   c.Query("q"),
		})
	})

	// Error example
	r.Get("/error", func(c *router.Ctx) error {
		return errors.BadRequest("Bad request example", nil)
	})

	// User registration with validation
	// Validates based on Accept-Language header (French/Spanish/English)
	r.Post("/register", registerUser)

	// Product creation with validation
	r.Post("/products", createProduct)

	// Request ID demonstration
	r.Get("/request-id", func(c *router.Ctx) error {
		requestID := middleware.GetRequestID(c)
		return c.JSON(map[string]string{
			"request_id": requestID,
			"message":    "Request ID is automatically added to all requests",
		})
	})

	// Demonstrate timeout with a slow endpoint using route group
	slowGroup := r.Group("/slow", middleware.Timeout(middleware.TimeoutConfig{
		Timeout: 2 * time.Second,
	}))
	slowGroup.Get("/endpoint", func(c *router.Ctx) error {
		// Simulate slow processing (will timeout after 2 seconds)
		time.Sleep(3 * time.Second)
		return c.JSON(map[string]string{"message": "This should timeout"})
	})

	// Start server with automatic graceful shutdown on SIGINT/SIGTERM
	// This handles Ctrl+C gracefully, completing in-flight requests
	server.Logger().Info("Starting server on http://localhost:8080")
	server.Logger().Info("Press Ctrl+C to gracefully shutdown")

	if err := server.ListenWithGracefulShutdown(); err != nil {
		server.Logger().Error(err)
	}
}

func registerUser(c *router.Ctx) error {
	var req RegisterRequest

	// ValidateBody automatically uses Accept-Language header
	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	return c.Status(201).JSON(req)
}

func createProduct(c *router.Ctx) error {
	var req CreateProductRequest

	// Parse and validate in one call
	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	return c.Status(201).JSON(req)
}
