package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"time"
)

// BaseStore provides default implementations for common cache operations.
// Drivers can embed this to avoid implementing every method.
type BaseStore struct {
	Store
}

// Has checks if an item exists in cache.
func (s *BaseStore) Has(ctx context.Context, key string) (bool, error) {
	var dummy any
	err := s.Get(ctx, key, &dummy)
	if err == ErrCacheMiss {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Missing checks if an item doesn't exist in cache.
func (s *BaseStore) Missing(ctx context.Context, key string) (bool, error) {
	has, err := s.Has(ctx, key)
	return !has, err
}

// Remember implements the cache-aside pattern.
// It gets the value from cache, or executes the callback and stores the result.
func (s *BaseStore) Remember(ctx context.Context, key string, dest any, ttl time.Duration, callback func() (any, error)) error {
	// Try to get from cache
	err := s.Get(ctx, key, dest)
	if err == nil {
		return nil // Cache hit
	}

	if err != ErrCacheMiss {
		return err // Real error
	}

	// Cache miss - execute callback
	value, err := callback()
	if err != nil {
		return err
	}

	// Store in cache
	if err := s.Put(ctx, key, value, ttl); err != nil {
		return err
	}

	// Copy value to dest
	return copyValue(value, dest)
}

// RememberForever is like Remember but stores the value without expiration.
func (s *BaseStore) RememberForever(ctx context.Context, key string, dest any, callback func() (any, error)) error {
	err := s.Get(ctx, key, dest)
	if err == nil {
		return nil
	}

	if err != ErrCacheMiss {
		return err
	}

	value, err := callback()
	if err != nil {
		return err
	}

	if err := s.Forever(ctx, key, value); err != nil {
		return err
	}

	return copyValue(value, dest)
}

// Pull retrieves an item and then deletes it.
func (s *BaseStore) Pull(ctx context.Context, key string, dest any) error {
	err := s.Get(ctx, key, dest)
	if err != nil {
		return err
	}

	// Delete the key (ignore error since we already got the value)
	_ = s.Forget(ctx, key)

	return nil
}

// Add stores an item only if it doesn't already exist.
func (s *BaseStore) Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	has, err := s.Has(ctx, key)
	if err != nil {
		return false, err
	}

	if has {
		return false, nil
	}

	return true, s.Put(ctx, key, value, ttl)
}

// copyValue copies a value to destination using JSON marshaling.
// This is a simple implementation that works for most types.
func copyValue(src, dest any) error {
	// If src is already the same type as dest, try direct assignment
	srcVal := reflect.ValueOf(src)
	destVal := reflect.ValueOf(dest)

	if destVal.Kind() != reflect.Ptr {
		return nil // Can't set non-pointer
	}

	destElem := destVal.Elem()

	// Try direct assignment if types match
	if srcVal.Type().AssignableTo(destElem.Type()) {
		destElem.Set(srcVal)
		return nil
	}

	// Fall back to JSON marshaling
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}
