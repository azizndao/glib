package main

import (
	"context"
	"glib/demo/generated"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Bootstrap initializes the application and returns a configured HTTP handler.
func Bootstrap(ctx context.Context) (*http.Server, error) {
	// Initialize DI container with router
	app, err := generated.InitContainer(ctx)
	if err != nil {
		return nil, err
	}

	// Apply default middleware to app router
	app.Router.Use(middleware.RequestID)
	app.Router.Use(middleware.RealIP)
	app.Router.Use(middleware.Logger)
	app.Router.Use(middleware.Recoverer)
	app.Router.Use(middleware.Timeout(60 * time.Second))

	// Register routes
	if err := app.RegisterRoutes(); err != nil {
		return nil, err
	}

	server := &http.Server{
		Addr:         app.Config.Addr(),
		Handler:      app.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server, nil
}
