package cache_test

import (
	"context"
	"fmt"
	"time"

	"github.com/azizndao/glib/cache"
	"github.com/azizndao/glib/cache/memory"
)

func ExampleManager() {
	// Create cache manager
	manager := cache.NewManager()

	// Register memory driver
	manager.RegisterDriver("memory", func() cache.Store {
		return memory.New()
	})

	// Get default store
	store, _ := manager.Default()

	ctx := context.Background()

	// Put a value
	_ = store.Put(ctx, "greeting", "Hello, World!", 5*time.Minute)

	// Get the value back
	var greeting string
	_ = store.Get(ctx, "greeting", &greeting)

	fmt.Println(greeting)
	// Output: Hello, World!
}

func ExampleStore_Remember() {
	manager := cache.NewManager()
	manager.RegisterDriver("memory", func() cache.Store {
		return memory.New()
	})

	store, _ := manager.Default()
	ctx := context.Background()

	callCount := 0

	// Remember will cache the result of the callback
	var result string
	_ = store.Remember(ctx, "expensive-key", &result, 1*time.Hour, func() (any, error) {
		callCount++
		return "computed value", nil
	})

	fmt.Printf("First call count: %d, result: %s\n", callCount, result)

	// Second call will use cached value
	var result2 string
	_ = store.Remember(ctx, "expensive-key", &result2, 1*time.Hour, func() (any, error) {
		callCount++
		return "computed value", nil
	})

	fmt.Printf("Second call count: %d, result: %s\n", callCount, result2)

	// Output:
	// First call count: 1, result: computed value
	// Second call count: 1, result: computed value
}

func ExampleStore_Increment() {
	manager := cache.NewManager()
	manager.RegisterDriver("memory", func() cache.Store {
		return memory.New()
	})

	store, _ := manager.Default()
	ctx := context.Background()

	// Increment counter
	count1, _ := store.Increment(ctx, "visits", 1)
	fmt.Printf("Visit count: %d\n", count1)

	count2, _ := store.Increment(ctx, "visits", 5)
	fmt.Printf("Visit count: %d\n", count2)

	// Output:
	// Visit count: 1
	// Visit count: 6
}

func ExampleStore_Pull() {
	manager := cache.NewManager()
	manager.RegisterDriver("memory", func() cache.Store {
		return memory.New()
	})

	store, _ := manager.Default()
	ctx := context.Background()

	// Store a value
	_ = store.Put(ctx, "token", "abc123", 1*time.Hour)

	// Pull retrieves and deletes in one operation
	var token string
	_ = store.Pull(ctx, "token", &token)
	fmt.Printf("Token: %s\n", token)

	// Second pull will fail (already deleted)
	err := store.Pull(ctx, "token", &token)
	fmt.Printf("Error: %v\n", err)

	// Output:
	// Token: abc123
	// Error: cache: key not found
}

func ExampleStore_Add() {
	manager := cache.NewManager()
	manager.RegisterDriver("memory", func() cache.Store {
		return memory.New()
	})

	store, _ := manager.Default()
	ctx := context.Background()

	// Add only succeeds if key doesn't exist
	added1, _ := store.Add(ctx, "config", "v1", 1*time.Hour)
	fmt.Printf("First add: %v\n", added1)

	// Second add fails (key exists)
	added2, _ := store.Add(ctx, "config", "v2", 1*time.Hour)
	fmt.Printf("Second add: %v\n", added2)

	// Value is still v1
	var value string
	_ = store.Get(ctx, "config", &value)
	fmt.Printf("Value: %s\n", value)

	// Output:
	// First add: true
	// Second add: false
	// Value: v1
}

func ExampleStore_Tags() {
	manager := cache.NewManager()
	manager.RegisterDriver("memory", func() cache.Store {
		return memory.New()
	})

	store, _ := manager.Default()
	ctx := context.Background()

	// Create tagged cache for users and posts
	usersCache := store.Tags("users")
	postsCache := store.Tags("posts")

	// Store user data
	_ = usersCache.Put(ctx, "user:1", "Alice", 1*time.Hour)
	_ = usersCache.Put(ctx, "user:2", "Bob", 1*time.Hour)

	// Store post data
	_ = postsCache.Put(ctx, "post:1", "First Post", 1*time.Hour)

	// Get user data
	var user string
	_ = usersCache.Get(ctx, "user:1", &user)
	fmt.Printf("User: %s\n", user)

	// Flush all user cache (but posts remain)
	_ = usersCache.FlushTags(ctx)

	// User data is gone
	err := usersCache.Get(ctx, "user:1", &user)
	fmt.Printf("After flush: %v\n", err)

	// Posts are still there
	var post string
	_ = postsCache.Get(ctx, "post:1", &post)
	fmt.Printf("Post: %s\n", post)

	// Output:
	// User: Alice
	// After flush: cache: key not found
	// Post: First Post
}

func ExampleStore_Lock() {
	manager := cache.NewManager()
	manager.RegisterDriver("memory", func() cache.Store {
		return memory.New()
	})

	store, _ := manager.Default()
	ctx := context.Background()

	// Create a lock with 30 second TTL
	lock := store.Lock("process-job", 30*time.Second)

	// Execute code while holding the lock
	err := lock.Get(ctx, func() error {
		fmt.Println("Processing job...")
		// Do work here
		return nil
	})

	if err != nil {
		fmt.Printf("Failed to acquire lock: %v\n", err)
	}

	// Output:
	// Processing job...
}

func ExampleLock_Block() {
	manager := cache.NewManager()
	manager.RegisterDriver("memory", func() cache.Store {
		return memory.New()
	})

	store, _ := manager.Default()
	ctx := context.Background()

	// Create a lock
	lock := store.Lock("critical-section", 10*time.Second)

	// Block waits up to 5 seconds to acquire the lock
	err := lock.Block(ctx, 5*time.Second, func() error {
		fmt.Println("Acquired lock after waiting")
		// Do work here
		return nil
	})

	if err != nil {
		fmt.Printf("Timeout or error: %v\n", err)
		return
	}

	// Output:
	// Acquired lock after waiting
}
