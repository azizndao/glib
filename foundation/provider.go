// Package foundation provides the core application framework including
// service providers, application lifecycle, and bootstrap functionality.
package foundation

import (
	"fmt"

	"github.com/azizndao/glib/common/container"
)

// ServiceProvider is the interface for service providers.
// Service providers are the central place to configure services in the container.
type ServiceProvider interface {
	// Register registers services in the container.
	// This method is called first for all providers.
	Register(app *Application) error

	// Boot performs any initialization after all providers have registered.
	// This method is called after all Register methods have been called.
	Boot(app *Application) error
}

// DeferredProvider is a service provider that should only be registered
// when one of its services is requested.
type DeferredProvider interface {
	ServiceProvider

	// Provides returns the service types provided by this provider.
	Provides() []string
}

// BootableProvider is called during the boot phase.
type BootableProvider interface {
	// Boot is called after all providers have been registered.
	Boot(app *Application) error
}

// BaseServiceProvider provides a base implementation for service providers.
// Providers can embed this to only implement the methods they need.
type BaseServiceProvider struct{}

// Register is a no-op implementation.
func (p *BaseServiceProvider) Register(app *Application) error {
	return nil
}

// Boot is a no-op implementation.
func (p *BaseServiceProvider) Boot(app *Application) error {
	return nil
}

// ProviderRepository manages service providers.
type ProviderRepository struct {
	providers         []ServiceProvider
	deferredProviders map[string]DeferredProvider
	registered        map[string]bool
	booted            bool
}

// NewProviderRepository creates a new provider repository.
func NewProviderRepository() *ProviderRepository {
	return &ProviderRepository{
		providers:         make([]ServiceProvider, 0),
		deferredProviders: make(map[string]DeferredProvider),
		registered:        make(map[string]bool),
		booted:            false,
	}
}

// Register registers a service provider.
func (r *ProviderRepository) Register(provider ServiceProvider) {
	// Check if this is a deferred provider
	if deferred, ok := provider.(DeferredProvider); ok {
		for _, service := range deferred.Provides() {
			r.deferredProviders[service] = deferred
		}
		return
	}

	r.providers = append(r.providers, provider)
}

// RegisterDeferred checks if a service has a deferred provider and registers it.
func (r *ProviderRepository) RegisterDeferred(service string, app *Application) error {
	if provider, exists := r.deferredProviders[service]; exists {
		// Check if already registered
		providerName := getProviderName(provider)
		if r.registered[providerName] {
			return nil
		}

		// Register the provider
		if err := provider.Register(app); err != nil {
			return err
		}

		r.registered[providerName] = true
		r.providers = append(r.providers, provider)

		// If already booted, boot this provider immediately
		if r.booted {
			return provider.Boot(app)
		}
	}

	return nil
}

// RegisterAll registers all providers.
func (r *ProviderRepository) RegisterAll(app *Application) error {
	for _, provider := range r.providers {
		providerName := getProviderName(provider)
		if r.registered[providerName] {
			continue
		}

		if err := provider.Register(app); err != nil {
			return err
		}

		r.registered[providerName] = true
	}

	return nil
}

// BootAll boots all registered providers.
func (r *ProviderRepository) BootAll(app *Application) error {
	if r.booted {
		return nil
	}

	for _, provider := range r.providers {
		// All ServiceProviders have a Boot method
		if err := provider.Boot(app); err != nil {
			return err
		}
	}

	r.booted = true
	return nil
}

// GetProviders returns all registered providers.
func (r *ProviderRepository) GetProviders() []ServiceProvider {
	return r.providers
}

// getProviderName returns a unique name for a provider.
// It uses the pointer address to ensure each provider instance is unique.
func getProviderName(provider ServiceProvider) string {
	return container.TypeName(provider) + "@" + getProviderAddress(provider)
}

// getProviderAddress returns the memory address of a provider as a string.
func getProviderAddress(provider ServiceProvider) string {
	return fmt.Sprintf("%p", provider)
}

// Common service provider implementations

// ConfigServiceProvider registers configuration services.
type ConfigServiceProvider struct {
	BaseServiceProvider
}

// Register registers the configuration service.
func (p *ConfigServiceProvider) Register(app *Application) error {
	return container.Singleton(app.container, func(c *container.Container) (*Application, error) {
		return app, nil
	})
}

// LogServiceProvider registers logging services.
type LogServiceProvider struct {
	BaseServiceProvider
}

// Register registers the logger service.
func (p *LogServiceProvider) Register(app *Application) error {
	// Logger registration will be implemented later
	return nil
}

// Boot configures the logger.
func (p *LogServiceProvider) Boot(app *Application) error {
	// Logger boot logic will be implemented later
	return nil
}
