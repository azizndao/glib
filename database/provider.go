package database

import (
	"github.com/azizndao/glib/common/container"
	"github.com/azizndao/glib/foundation"
)

// ServiceProvider registers database services in the container.
type ServiceProvider struct {
	foundation.BaseServiceProvider
}

// Register registers the database manager in the container.
func (p *ServiceProvider) Register(app *foundation.Application) error {
	return container.Singleton(app.Container(), func(c *container.Container) (*Manager, error) {
		cfg := app.Config()
		logger := app.Logger()
		return NewManager(cfg, logger), nil
	})
}

// Boot initializes database connections.
func (p *ServiceProvider) Boot(app *foundation.Application) error {
	// Database manager is created lazily when first accessed
	// No need to connect eagerly
	return nil
}
