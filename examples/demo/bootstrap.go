package main

import (
	"context"
	"glib/demo/generated"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Bootstrap initializes the application and returns a configured HTTP handler.
func Bootstrap(ctx context.Context) (http.Handler, error) {
	// Initialize DI container
	container, err := generated.InitContainer(ctx)
	if err != nil {
		return nil, err
	}

	// Create chi router
	router := chi.NewRouter()

	// Apply default middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	// Register routes
	if err := generated.RegisterRoutes(router, container); err != nil {
		return nil, err
	}

	return router, nil
}
