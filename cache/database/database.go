package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/azizndao/glib/cache"
	"github.com/azizndao/glib/cache/internal"
	"gorm.io/gorm"
)

// CacheEntry represents a cache entry in the database.
type CacheEntry struct {
	Key        string    `gorm:"primaryKey;size:255"`
	Value      []byte    `gorm:"type:blob"`
	Expiration time.Time `gorm:"index"`
	CreatedAt  time.Time
}

// TableName specifies the table name for cache entries.
func (CacheEntry) TableName() string {
	return "cache_entries"
}

// IsExpired checks if the cache entry has expired.
func (e *CacheEntry) IsExpired() bool {
	return !e.Expiration.IsZero() && time.Now().After(e.Expiration)
}

// Database implements a GORM-backed cache store.
type Database struct {
	db     *gorm.DB
	prefix string
}

// NewDatabase creates a new database cache store.
func New(db *gorm.DB, prefix string) *Database {
	if prefix != "" && prefix[len(prefix)-1] != ':' {
		prefix = prefix + ":"
	}
	return &Database{
		db:     db,
		prefix: prefix,
	}
}

// prefixKey adds the prefix to a cache key.
func (d *Database) prefixKey(key string) string {
	return d.prefix + key
}

// Get retrieves an item from cache.
func (d *Database) Get(ctx context.Context, key string, dest any) error {
	var entry CacheEntry
	err := d.db.WithContext(ctx).
		Where("key = ?", d.prefixKey(key)).
		First(&entry).Error

	if err == gorm.ErrRecordNotFound {
		return cache.ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("database get: %w", err)
	}

	// Check expiration
	if entry.IsExpired() {
		// Delete expired entry
		d.db.WithContext(ctx).Delete(&entry)
		return cache.ErrCacheMiss
	}

	return json.Unmarshal(entry.Value, dest)
}

// Put stores an item in cache with TTL.
func (d *Database) Put(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("database marshal: %w", err)
	}

	entry := CacheEntry{
		Key:        d.prefixKey(key),
		Value:      data,
		Expiration: time.Now().Add(ttl),
		CreatedAt:  time.Now(),
	}

	// Use upsert to handle both insert and update
	return d.db.WithContext(ctx).
		Save(&entry).Error
}

// Forever stores an item without expiration.
func (d *Database) Forever(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("database marshal: %w", err)
	}

	entry := CacheEntry{
		Key:        d.prefixKey(key),
		Value:      data,
		Expiration: time.Time{}, // Zero time means no expiration
		CreatedAt:  time.Now(),
	}

	return d.db.WithContext(ctx).
		Save(&entry).Error
}

// Has checks if item exists in cache.
func (d *Database) Has(ctx context.Context, key string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&CacheEntry{}).
		Where("key = ? AND (expiration IS NULL OR expiration = ? OR expiration > ?)",
			d.prefixKey(key), time.Time{}, time.Now()).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("database has: %w", err)
	}

	return count > 0, nil
}

// Missing checks if item doesn't exist.
func (d *Database) Missing(ctx context.Context, key string) (bool, error) {
	has, err := d.Has(ctx, key)
	return !has, err
}

// Increment increments a numeric value.
func (d *Database) Increment(ctx context.Context, key string, value int64) (int64, error) {
	return d.modifyCounter(ctx, key, value)
}

// Decrement decrements a numeric value.
func (d *Database) Decrement(ctx context.Context, key string, value int64) (int64, error) {
	return d.modifyCounter(ctx, key, -value)
}

// modifyCounter modifies a counter atomically.
func (d *Database) modifyCounter(ctx context.Context, key string, delta int64) (int64, error) {
	// Use a transaction for atomicity
	var result int64
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var entry CacheEntry
		err := tx.Where("key = ?", d.prefixKey(key)).
			First(&entry).Error

		var current int64
		if err == nil {
			// Entry exists, parse current value
			if err := json.Unmarshal(entry.Value, &current); err != nil {
				return fmt.Errorf("invalid counter value: %w", err)
			}
		} else if err != gorm.ErrRecordNotFound {
			return err
		}
		// If not found, current remains 0

		// Apply delta
		current += delta
		result = current

		// Marshal new value
		data, err := json.Marshal(current)
		if err != nil {
			return err
		}

		// Update or create entry
		entry.Key = d.prefixKey(key)
		entry.Value = data
		entry.CreatedAt = time.Now()

		return tx.Save(&entry).Error
	})

	return result, err
}

// Forget removes an item from cache.
func (d *Database) Forget(ctx context.Context, key string) error {
	return d.db.WithContext(ctx).
		Where("key = ?", d.prefixKey(key)).
		Delete(&CacheEntry{}).Error
}

// Flush clears all items from cache.
func (d *Database) Flush(ctx context.Context) error {
	query := d.db.WithContext(ctx)

	// If there's a prefix, only delete keys with that prefix
	if d.prefix != "" {
		query = query.Where("key LIKE ?", d.prefix+"%")
	}

	return query.Delete(&CacheEntry{}).Error
}

// Remember gets value or stores callback result (cache-aside pattern).
func (d *Database) Remember(ctx context.Context, key string, dest any, ttl time.Duration, callback func() (any, error)) error {
	// Try to get from cache
	err := d.Get(ctx, key, dest)
	if err == nil {
		return nil // Cache hit
	}

	if err != cache.ErrCacheMiss {
		return err // Real error
	}

	// Cache miss - execute callback
	value, err := callback()
	if err != nil {
		return err
	}

	// Store in cache
	if err := d.Put(ctx, key, value, ttl); err != nil {
		return err
	}

	// Copy value to dest
	return internal.CopyValue(value, dest)
}

// RememberForever gets value or stores callback result forever.
func (d *Database) RememberForever(ctx context.Context, key string, dest any, callback func() (any, error)) error {
	err := d.Get(ctx, key, dest)
	if err == nil {
		return nil
	}

	if err != cache.ErrCacheMiss {
		return err
	}

	value, err := callback()
	if err != nil {
		return err
	}

	if err := d.Forever(ctx, key, value); err != nil {
		return err
	}

	return internal.CopyValue(value, dest)
}

// Pull retrieves and deletes an item.
func (d *Database) Pull(ctx context.Context, key string, dest any) error {
	err := d.Get(ctx, key, dest)
	if err != nil {
		return err
	}

	return d.Forget(ctx, key)
}

// Add stores item only if it doesn't exist.
func (d *Database) Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	has, err := d.Has(ctx, key)
	if err != nil {
		return false, err
	}

	if has {
		return false, nil
	}

	return true, d.Put(ctx, key, value, ttl)
}

// Tags returns a tagged cache instance (not implemented for database).
func (d *Database) Tags(names ...string) cache.TaggedStore {
	// Database tagged cache is more complex and would require a junction table
	// For now, return a simple wrapper that treats tags as key prefixes
	panic("database tagged cache not yet implemented")
}

// Lock creates a cache lock using database.
func (d *Database) Lock(name string, ttl time.Duration) cache.Lock {
	return NewDatabaseLock(d.db, d.prefixKey("lock:"+name), ttl)
}

// CleanupExpired removes all expired cache entries.
// This should be called periodically (e.g., via a cron job).
func (d *Database) CleanupExpired(ctx context.Context) error {
	return d.db.WithContext(ctx).
		Where("expiration != ? AND expiration < ?", time.Time{}, time.Now()).
		Delete(&CacheEntry{}).Error
}
