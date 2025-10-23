// Package main demonstrates the glib.Server with comprehensive production-ready configuration
package main

import (
	"fmt"
	"time"

	"github.com/azizndao/glib"
	"github.com/azizndao/glib/errors"
	"github.com/azizndao/glib/router"
	"github.com/go-playground/locales/es"
	"github.com/go-playground/locales/fr"
	est "github.com/go-playground/validator/v10/translations/es"
	frt "github.com/go-playground/validator/v10/translations/fr"
	"github.com/joho/godotenv"
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
	godotenv.Load()
	// Create server - all configuration loaded from environment variables
	// See .env.example for available configuration options
	// Set environment variables to customize the server behavior
	options := glib.Config{
		Locales: []glib.LocaleConfig{
			glib.Locale(fr.New(), frt.RegisterDefaultTranslations),
			glib.Locale(es.New(), est.RegisterDefaultTranslations),
		},
	}

	server := glib.New(options)

	// Get the router from the server to register routes
	rf := server.Router()

	// Create a sub-router for /api routes using Route()
	r := rf.Route("/api", func(api router.Router) {})

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
		// Request ID can be available from Chi middleware if configured
		requestID := c.Get("X-Request-Id")
		if requestID == "" {
			requestID = "not-configured"
		}
		return c.JSON(map[string]string{
			"request_id": requestID,
			"message":    "Request ID can be added by Chi middleware",
		})
	})

	// Demonstrate slow endpoint using route group
	r.Route("/slow", func(slow router.Router) {
		slow.Get("/endpoint", func(c *router.Ctx) error {
			// Simulate slow processing
			time.Sleep(3 * time.Second)
			return c.JSON(map[string]string{"message": "Slow response completed"})
		})
	})

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
