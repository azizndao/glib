// Package main demonstrates using grouter.Server with environment variable configuration
package main

import (
	"fmt"

	"github.com/azizndao/grouter"
	"github.com/azizndao/grouter/router"
)

// This example shows how to use environment variables for server configuration.
//
// To run this example:
// 1. Copy .env.example to .env
// 2. Customize the values in .env as needed
// 3. Load the .env file using a tool like godotenv or export variables manually:
//    export GROUTER_PORT=3000
//    export GROUTER_ENABLE_RATE_LIMIT=true
// 4. Run: go run env_server.go
//
// All configuration will be loaded from environment variables.

func mainEnv() {
	// Create server from environment variables
	// All configuration is automatically loaded from environment variables
	server := grouter.New()

	// Get the router from the server to register routes
	r := server.Router()

	// Register routes
	r.Get("/", func(c *router.Ctx) error {
		return c.JSON(map[string]string{
			"message": "Hello from env-configured server!",
			"address": server.Address(),
		})
	})

	r.Get("/health", func(c *router.Ctx) error {
		return c.JSON(map[string]string{
			"status": "healthy",
		})
	})

	r.Get("/hello/{name}", func(c *router.Ctx) error {
		return c.JSON(map[string]string{
			"message": fmt.Sprintf("Hello %s", c.PathValue("name")),
		})
	})

	// Log startup message
	server.Logger().Info("Starting server with configuration from environment variables")
	server.Logger().Info("Address: " + server.Address())

	// Start server with automatic graceful shutdown
	if err := server.ListenWithGracefulShutdown(); err != nil {
		server.Logger().Error(err)
	}
}
