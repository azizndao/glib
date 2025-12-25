package cache

import (
	"github.com/azizndao/glib/common/container"
)

// ServiceProvider provides cache services to the application.
type ServiceProvider struct {
	manager *Manager
}

// NewServiceProvider creates a new cache service provider.
func NewServiceProvider() *ServiceProvider {
	return &ServiceProvider{
		manager: NewManager(),
	}
}

// Register registers the cache manager in the container.
func (p *ServiceProvider) Register(c *container.Container) error {
	// Register cache manager as singleton
	return container.Singleton(c, func(c *container.Container) (*Manager, error) {
		if p.manager == nil {
			p.manager = NewManager()
		}
		return p.manager, nil
	})
}

// Boot configures the cache manager with default drivers.
// Note: Drivers must be registered by the application.
func (p *ServiceProvider) Boot(c *container.Container) error {
	// Get the manager from container
	_, err := container.Resolve[*Manager](c)
	if err != nil {
		return err
	}

	// Note: Applications should register drivers like:
	// manager.RegisterDriver("memory", func() cache.Store {
	//     return memory.New()
	// })

	return nil
}

// Provides returns the services provided by this provider.
func (p *ServiceProvider) Provides() []string {
	return []string{"*cache.Manager"}
}
