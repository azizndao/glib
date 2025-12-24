// Package foundation provides the core application framework.
package foundation

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/azizndao/glib/config"
	"github.com/azizndao/glib/container"
)

// Application is the core application instance.
type Application struct {
	container *container.Container
	config    config.Config
	providers *ProviderRepository
	booted    bool
	basePath  string
	env       string
	logger    *slog.Logger
	mu        sync.RWMutex
}

// New creates a new Application instance.
func New(basePath string) *Application {
	app := &Application{
		container: container.New(),
		providers: NewProviderRepository(),
		basePath:  basePath,
		env:       os.Getenv("APP_ENV"),
		logger:    slog.Default(),
	}

	// Set default environment
	if app.env == "" {
		app.env = "development"
	}

	// Register core bindings
	app.registerCoreBindings()

	return app
}

// registerCoreBindings registers the core framework bindings.
func (app *Application) registerCoreBindings() {
	// Register application instance
	container.Instance(app.container, app)

	// Register logger
	container.Singleton(app.container, func(c *container.Container) (*slog.Logger, error) {
		return app.logger, nil
	})
}

// Container returns the service container.
func (app *Application) Container() *container.Container {
	return app.container
}

// Config returns the configuration repository.
func (app *Application) Config() config.Config {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.config
}

// SetConfig sets the configuration repository.
func (app *Application) SetConfig(cfg config.Config) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.config = cfg
}

// Logger returns the application logger.
func (app *Application) Logger() *slog.Logger {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.logger
}

// SetLogger sets the application logger.
func (app *Application) SetLogger(logger *slog.Logger) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.logger = logger
}

// BasePath returns the base path of the application.
func (app *Application) BasePath(paths ...string) string {
	if len(paths) == 0 {
		return app.basePath
	}
	return app.basePath + "/" + paths[0]
}

// Env returns the current environment.
func (app *Application) Env() string {
	return app.env
}

// IsProduction checks if the application is running in production.
func (app *Application) IsProduction() bool {
	return app.env == "production"
}

// IsDevelopment checks if the application is running in development.
func (app *Application) IsDevelopment() bool {
	return app.env == "development" || app.env == "dev"
}

// IsTesting checks if the application is running in testing.
func (app *Application) IsTesting() bool {
	return app.env == "testing" || app.env == "test"
}

// IsDebug checks if debug mode is enabled.
func (app *Application) IsDebug() bool {
	if app.config != nil {
		return app.config.IsDebug()
	}
	return false
}

// Register registers a service provider.
func (app *Application) Register(provider ServiceProvider) {
	app.providers.Register(provider)
}

// RegisterAll registers all service providers.
func (app *Application) RegisterAll() error {
	return app.providers.RegisterAll(app)
}

// Boot boots all service providers.
func (app *Application) Boot() error {
	if app.booted {
		return nil
	}

	if err := app.providers.BootAll(app); err != nil {
		return fmt.Errorf("failed to boot providers: %w", err)
	}

	app.booted = true
	return nil
}

// Bootstrap performs the application bootstrap sequence.
func (app *Application) Bootstrap() error {
	app.logger.Info("Bootstrapping application", "env", app.env)

	// Load configuration
	if err := app.loadConfiguration(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Register all providers
	if err := app.RegisterAll(); err != nil {
		return fmt.Errorf("failed to register providers: %w", err)
	}

	// Boot all providers
	if err := app.Boot(); err != nil {
		return fmt.Errorf("failed to boot providers: %w", err)
	}

	app.logger.Info("Application bootstrapped successfully")
	return nil
}

// loadConfiguration loads the application configuration.
func (app *Application) loadConfiguration() error {
	// Only create new config if one doesn't exist
	if app.config == nil {
		// Create config repository
		cfg := config.New()

		// Load from environment variables
		cfg.LoadFromEnv("")

		// Set configuration
		app.SetConfig(cfg)
	}

	// Register in container (as concrete type since Config is an interface)
	return container.Instance(app.container, app.config)
}

// Shutdown gracefully shuts down the application.
func (app *Application) Shutdown(ctx context.Context) error {
	app.logger.Info("Shutting down application")

	// Call shutdown hooks for providers
	// This will be expanded as we add more providers

	app.logger.Info("Application shutdown complete")
	return nil
}

// Run runs the application with graceful shutdown handling.
func (app *Application) Run(startFn func() error) error {
	// Bootstrap the application
	if err := app.Bootstrap(); err != nil {
		return err
	}

	// Start the application in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- startFn()
	}()

	// Set up signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal or error
	select {
	case err := <-errChan:
		if err != nil {
			app.logger.Error("Application error", "error", err)
			return err
		}
	case sig := <-quit:
		app.logger.Info("Received shutdown signal", "signal", sig)

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), app.Config().GetDuration("app.shutdown_timeout", 30))
		defer cancel()

		// Shutdown gracefully
		if err := app.Shutdown(ctx); err != nil {
			app.logger.Error("Shutdown error", "error", err)
			return err
		}
	}

	return nil
}

// Resolve resolves a service from the container.
func (app *Application) Resolve(abstract any) (any, error) {
	// This is a generic wrapper around container.Resolve
	// The actual type-safe resolution happens at the container level
	return nil, fmt.Errorf("use container.Resolve[T] for type-safe resolution")
}

// Make is an alias for Resolve (Laravel compatibility).
func (app *Application) Make(abstract any) (any, error) {
	return app.Resolve(abstract)
}

// Call invokes a function with dependency injection.
func (app *Application) Call(fn any) error {
	return app.container.Call(fn)
}

// Providers returns all registered service providers.
func (app *Application) Providers() []ServiceProvider {
	return app.providers.GetProviders()
}

// Version returns the framework version.
func (app *Application) Version() string {
	return "1.0.0-alpha"
}
