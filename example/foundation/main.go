// Package main demonstrates the foundation layer of the glib framework.
// This example shows how to use the service container, configuration system,
// and service providers.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/azizndao/glib/config"
	"github.com/azizndao/glib/container"
	"github.com/azizndao/glib/foundation"
)

// Example services

// Database represents a database connection
type Database struct {
	Host string
	Port int
}

// Connect simulates connecting to a database
func (db *Database) Connect() error {
	fmt.Printf("Connected to database at %s:%d\n", db.Host, db.Port)
	return nil
}

// Cache represents a cache service
type Cache struct {
	Driver string
}

// Set simulates caching a value
func (c *Cache) Set(key string, value any) {
	fmt.Printf("Cache[%s]: Set %s = %v\n", c.Driver, key, value)
}

// UserRepository demonstrates dependency injection
type UserRepository struct {
	DB    *Database
	Cache *Cache
}

// FindUser simulates finding a user
func (ur *UserRepository) FindUser(id int) string {
	ur.Cache.Set(fmt.Sprintf("user:%d", id), "cached")
	return fmt.Sprintf("User %d", id)
}

// Example service providers

// DatabaseProvider registers the database service
type DatabaseProvider struct {
	foundation.BaseServiceProvider
}

func (p *DatabaseProvider) Register(app *foundation.Application) error {
	return container.Singleton(app.Container(), func(c *container.Container) (*Database, error) {
		cfg := app.Config()
		return &Database{
			Host: cfg.GetString("database.host", "localhost"),
			Port: cfg.GetInt("database.port", 5432),
		}, nil
	})
}

func (p *DatabaseProvider) Boot(app *foundation.Application) error {
	db := container.MustResolve[*Database](app.Container())
	return db.Connect()
}

// CacheProvider registers the cache service
type CacheProvider struct {
	foundation.BaseServiceProvider
}

func (p *CacheProvider) Register(app *foundation.Application) error {
	return container.Singleton(app.Container(), func(c *container.Container) (*Cache, error) {
		cfg := app.Config()
		return &Cache{
			Driver: cfg.GetString("cache.driver", "memory"),
		}, nil
	})
}

// RepositoryProvider registers application repositories
type RepositoryProvider struct {
	foundation.BaseServiceProvider
}

func (p *RepositoryProvider) Register(app *foundation.Application) error {
	// Register the factory - dependencies will be resolved when the service is requested
	return container.Singleton(app.Container(), func(c *container.Container) (*UserRepository, error) {
		// Resolve dependencies when UserRepository is needed, not during registration
		db := container.MustResolve[*Database](c)
		cache := container.MustResolve[*Cache](c)
		return &UserRepository{
			DB:    db,
			Cache: cache,
		}, nil
	})
}

func main() {
	fmt.Println("=== glib Foundation Layer Example ===\n")

	// 1. Create Application
	fmt.Println("1. Creating application...")
	app := foundation.New("/app")

	// Initialize config
	cfg := config.New()
	app.SetConfig(cfg)

	fmt.Printf("   Environment: %s\n", app.Env())
	fmt.Printf("   Base Path: %s\n", app.BasePath())
	fmt.Printf("   Is Development: %v\n", app.IsDevelopment())
	fmt.Println()

	// 2. Configure Application
	fmt.Println("2. Configuring application...")
	cfg.Set("app.name", "Foundation Example")
	cfg.Set("app.debug", true)
	cfg.Set("database.host", "localhost")
	cfg.Set("database.port", 5432)
	cfg.Set("cache.driver", "redis")

	fmt.Printf("   App Name: %s\n", cfg.GetString("app.name"))
	fmt.Printf("   Debug Mode: %v\n", cfg.GetBool("app.debug"))
	fmt.Println()

	// 3. Demonstrate Configuration Caching
	fmt.Println("3. Demonstrating configuration caching...")
	tempDir := os.TempDir()
	cachePath := tempDir + "/glib-example-config.json"

	if err := cfg.Cache(cachePath); err != nil {
		log.Fatalf("Failed to cache config: %v", err)
	}
	fmt.Printf("   Cached config to: %s\n", cachePath)
	fmt.Printf("   Is Cached: %v\n", cfg.IsCached())

	// Load cache into new config
	cfg2 := config.New()
	if err := cfg2.LoadCache(cachePath); err != nil {
		log.Fatalf("Failed to load cache: %v", err)
	}
	fmt.Printf("   Loaded from cache: %s\n", cfg2.GetString("app.name"))
	fmt.Println()

	// 4. Register Service Providers
	fmt.Println("4. Registering service providers...")
	app.Register(&DatabaseProvider{})
	app.Register(&CacheProvider{})
	fmt.Printf("   Registered %d providers\n", len(app.Providers()))
	fmt.Println()

	// 5. Bootstrap Application
	fmt.Println("5. Bootstrapping application...")
	if err := app.Bootstrap(); err != nil {
		log.Fatalf("Failed to bootstrap: %v", err)
	}
	fmt.Println("   All providers registered and booted")
	fmt.Println()

	// 6. Resolve Services from Container
	fmt.Println("6. Resolving services from container...")

	db, err := container.Resolve[*Database](app.Container())
	if err != nil {
		log.Fatalf("Failed to resolve database: %v", err)
	}
	fmt.Printf("   Resolved Database: %s:%d\n", db.Host, db.Port)

	cache, err := container.Resolve[*Cache](app.Container())
	if err != nil {
		log.Fatalf("Failed to resolve cache: %v", err)
	}
	fmt.Printf("   Resolved Cache: %s\n", cache.Driver)

	// Create repository manually to avoid nested resolution deadlock
	repo := &UserRepository{
		DB:    db,
		Cache: cache,
	}
	fmt.Printf("   Created UserRepository (with injected dependencies)\n")
	fmt.Println()

	// 7. Use Services
	fmt.Println("7. Using resolved services...")
	user := repo.FindUser(42)
	fmt.Printf("   Found: %s\n", user)
	fmt.Println()

	// 8. Demonstrate Container Features
	fmt.Println("8. Demonstrating container features...")

	// Singleton behavior
	db2 := container.MustResolve[*Database](app.Container())
	if db == db2 {
		fmt.Println("   ✓ Singleton: Same instance returned")
	}

	// Container.Has
	hasDB := container.Has[*Database](app.Container())
	hasInvalid := container.Has[*slog.Logger](app.Container())
	fmt.Printf("   Has Database: %v\n", hasDB)
	fmt.Printf("   Has Logger: %v\n", hasInvalid)
	fmt.Println()

	// 9. Demonstrate Application Call
	fmt.Println("9. Demonstrating automatic dependency injection...")
	err = app.Call(func(db *Database, cache *Cache) error {
		fmt.Println("   ✓ All dependencies automatically injected!")
		fmt.Printf("   - Database: %s:%d\n", db.Host, db.Port)
		fmt.Printf("   - Cache: %s\n", cache.Driver)
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to call function: %v", err)
	}
	fmt.Println()

	// 10. Graceful Shutdown
	fmt.Println("10. Shutting down application...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown: %v", err)
	}
	fmt.Println("   ✓ Application shutdown complete")
	fmt.Println()

	fmt.Println("=== Foundation Layer Example Complete ===")
}
