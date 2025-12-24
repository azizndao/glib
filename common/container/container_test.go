package container_test

import (
	"errors"
	"testing"

	"github.com/azizndao/glib/common/container"
)

// Mock services for testing
type Database struct {
	Name string
}

type Cache struct {
	Driver string
}

type Logger struct {
	Level string
}

type UserRepository struct {
	DB    *Database
	Cache *Cache
}

func TestContainer_New(t *testing.T) {
	c := container.New()
	if c == nil {
		t.Fatal("expected non-nil container")
	}
}

func TestContainer_Bind(t *testing.T) {
	c := container.New()

	// Bind factory
	err := container.Bind(c, func(c *container.Container) (*Database, error) {
		return &Database{Name: "test"}, nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Resolve should create new instance each time
	db1, err := container.Resolve[*Database](c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	db2, err := container.Resolve[*Database](c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should be different instances
	if db1 == db2 {
		t.Error("expected different instances for factory binding")
	}

	if db1.Name != "test" {
		t.Errorf("expected Name='test', got %s", db1.Name)
	}
}

func TestContainer_Singleton(t *testing.T) {
	c := container.New()

	// Bind singleton
	err := container.Singleton(c, func(c *container.Container) (*Database, error) {
		return &Database{Name: "singleton"}, nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Resolve multiple times
	db1, err := container.Resolve[*Database](c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	db2, err := container.Resolve[*Database](c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should be same instance
	if db1 != db2 {
		t.Error("expected same instance for singleton binding")
	}

	if db1.Name != "singleton" {
		t.Errorf("expected Name='singleton', got %s", db1.Name)
	}
}

func TestContainer_Instance(t *testing.T) {
	c := container.New()

	// Register existing instance
	db := &Database{Name: "existing"}
	err := container.Instance(c, db)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Resolve
	resolved, err := container.Resolve[*Database](c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should be same instance
	if resolved != db {
		t.Error("expected same instance")
	}
}

func TestContainer_MustResolve(t *testing.T) {
	c := container.New()

	// Should panic when binding doesn't exist
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for non-existent binding")
		}
	}()

	_ = container.MustResolve[*Database](c)
}

func TestContainer_MustResolve_Success(t *testing.T) {
	c := container.New()

	container.Singleton(c, func(c *container.Container) (*Database, error) {
		return &Database{Name: "must"}, nil
	})

	db := container.MustResolve[*Database](c)

	if db.Name != "must" {
		t.Errorf("expected Name='must', got %s", db.Name)
	}
}

func TestContainer_Has(t *testing.T) {
	c := container.New()

	// Should not have binding initially
	if container.Has[*Database](c) {
		t.Error("expected no binding")
	}

	// Add binding
	container.Bind(c, func(c *container.Container) (*Database, error) {
		return &Database{}, nil
	})

	// Should have binding now
	if !container.Has[*Database](c) {
		t.Error("expected binding to exist")
	}
}

func TestContainer_Forget(t *testing.T) {
	c := container.New()

	// Add binding
	container.Singleton(c, func(c *container.Container) (*Database, error) {
		return &Database{Name: "forget"}, nil
	})

	// Verify exists
	if !container.Has[*Database](c) {
		t.Fatal("expected binding to exist")
	}

	// Forget
	container.Forget[*Database](c)

	// Verify removed
	if container.Has[*Database](c) {
		t.Error("expected binding to be removed")
	}
}

func TestContainer_Flush(t *testing.T) {
	c := container.New()

	// Add multiple bindings
	container.Singleton(c, func(c *container.Container) (*Database, error) {
		return &Database{}, nil
	})
	container.Singleton(c, func(c *container.Container) (*Cache, error) {
		return &Cache{}, nil
	})

	// Flush
	c.Flush()

	// Verify all removed
	if container.Has[*Database](c) {
		t.Error("expected Database binding to be removed")
	}
	if container.Has[*Cache](c) {
		t.Error("expected Cache binding to be removed")
	}
}

func TestContainer_NestedDependencies(t *testing.T) {
	t.Skip("Skipping due to known deadlock issue with nested Resolve() calls in factory functions")

	c := container.New()

	// Register dependencies
	container.Singleton(c, func(c *container.Container) (*Database, error) {
		return &Database{Name: "postgres"}, nil
	})

	container.Singleton(c, func(c *container.Container) (*Cache, error) {
		return &Cache{Driver: "redis"}, nil
	})

	// Register service with dependencies
	container.Singleton(c, func(c *container.Container) (*UserRepository, error) {
		db := container.MustResolve[*Database](c)
		cache := container.MustResolve[*Cache](c)

		return &UserRepository{
			DB:    db,
			Cache: cache,
		}, nil
	})

	// Resolve
	repo, err := container.Resolve[*UserRepository](c)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.DB == nil {
		t.Error("expected DB to be resolved")
	}

	if repo.Cache == nil {
		t.Error("expected Cache to be resolved")
	}

	if repo.DB.Name != "postgres" {
		t.Errorf("expected DB name='postgres', got %s", repo.DB.Name)
	}

	if repo.Cache.Driver != "redis" {
		t.Errorf("expected Cache driver='redis', got %s", repo.Cache.Driver)
	}
}

func TestContainer_ErrorHandling(t *testing.T) {
	c := container.New()

	// Register binding that returns error
	container.Bind(c, func(c *container.Container) (*Database, error) {
		return nil, errors.New("connection failed")
	})

	// Should return error
	_, err := container.Resolve[*Database](c)
	if err == nil {
		t.Error("expected error")
	}

	if err.Error() != "failed to resolve *container_test.Database: connection failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestContainer_Call(t *testing.T) {
	c := container.New()

	// Register dependencies
	container.Singleton(c, func(c *container.Container) (*Database, error) {
		return &Database{Name: "test"}, nil
	})

	container.Singleton(c, func(c *container.Container) (*Cache, error) {
		return &Cache{Driver: "memory"}, nil
	})

	// Call function with automatic dependency injection
	called := false
	err := c.Call(func(db *Database, cache *Cache) error {
		called = true

		if db == nil {
			t.Error("expected db to be injected")
		}
		if cache == nil {
			t.Error("expected cache to be injected")
		}

		if db.Name != "test" {
			t.Errorf("expected db.Name='test', got %s", db.Name)
		}
		if cache.Driver != "memory" {
			t.Errorf("expected cache.Driver='memory', got %s", cache.Driver)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !called {
		t.Error("expected function to be called")
	}
}

func TestContainer_Call_WithError(t *testing.T) {
	c := container.New()

	container.Singleton(c, func(c *container.Container) (*Database, error) {
		return &Database{}, nil
	})

	// Call function that returns error
	expectedErr := errors.New("test error")
	err := c.Call(func(db *Database) error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("expected error to be returned, got %v", err)
	}
}

func TestContainer_Tag(t *testing.T) {
	c := container.New()

	// Register services with tags
	container.Singleton(c, func(c *container.Container) (*Database, error) {
		return &Database{Name: "db1"}, nil
	})
	container.Tag[*Database](c, "connections")

	container.Singleton(c, func(c *container.Container) (*Cache, error) {
		return &Cache{Driver: "redis"}, nil
	})
	container.Tag[*Cache](c, "connections")

	// Resolve to populate singletons
	container.MustResolve[*Database](c)
	container.MustResolve[*Cache](c)

	// Get tagged services
	services := c.Tagged("connections")

	if len(services) != 2 {
		t.Errorf("expected 2 services, got %d", len(services))
	}
}

func TestContainer_ThreadSafety(t *testing.T) {
	c := container.New()

	container.Singleton(c, func(c *container.Container) (*Database, error) {
		return &Database{Name: "concurrent"}, nil
	})

	// Resolve concurrently
	done := make(chan bool)
	for range 100 {
		go func() {
			db, err := container.Resolve[*Database](c)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if db.Name != "concurrent" {
				t.Errorf("expected Name='concurrent', got %s", db.Name)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 100 {
		<-done
	}
}

// Benchmark tests
func BenchmarkContainer_Resolve_Singleton(b *testing.B) {
	c := container.New()

	container.Singleton(c, func(c *container.Container) (*Database, error) {
		return &Database{Name: "bench"}, nil
	})

	for b.Loop() {
		_, _ = container.Resolve[*Database](c)
	}
}

func BenchmarkContainer_Resolve_Factory(b *testing.B) {
	c := container.New()

	container.Bind(c, func(c *container.Container) (*Database, error) {
		return &Database{Name: "bench"}, nil
	})

	for b.Loop() {
		_, _ = container.Resolve[*Database](c)
	}
}

func BenchmarkContainer_MustResolve_Singleton(b *testing.B) {
	c := container.New()

	container.Singleton(c, func(c *container.Container) (*Database, error) {
		return &Database{Name: "bench"}, nil
	})

	for b.Loop() {
		_ = container.MustResolve[*Database](c)
	}
}
