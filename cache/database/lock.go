package database

import (
	"context"
	"fmt"
	"time"

	"github.com/azizndao/glib/cache"
	"github.com/azizndao/glib/cache/internal"
	"gorm.io/gorm"
)

// CacheLock represents a lock entry in the database.
type CacheLock struct {
	Name       string    `gorm:"primaryKey;size:255"`
	Owner      string    `gorm:"size:255;index"`
	Expiration time.Time `gorm:"index"`
	CreatedAt  time.Time
}

// TableName specifies the table name for locks.
func (CacheLock) TableName() string {
	return "cache_locks"
}

// IsExpired checks if the lock has expired.
func (l *CacheLock) IsExpired() bool {
	return time.Now().After(l.Expiration)
}

// DatabaseLock implements distributed locking with database.
type DatabaseLock struct {
	db    *gorm.DB
	name  string
	owner string
	ttl   time.Duration
}

// NewDatabaseLock creates a new database-based distributed lock.
func NewDatabaseLock(db *gorm.DB, name string, ttl time.Duration) *DatabaseLock {
	return &DatabaseLock{
		db:    db,
		name:  name,
		owner: internal.GenerateOwnerID(),
		ttl:   ttl,
	}
}

// Acquire attempts to acquire the lock.
func (l *DatabaseLock) Acquire(ctx context.Context) (bool, error) {
	now := time.Now()
	expiration := now.Add(l.ttl)

	// Try to insert the lock
	lock := CacheLock{
		Name:       l.name,
		Owner:      l.owner,
		Expiration: expiration,
		CreatedAt:  now,
	}

	err := l.db.WithContext(ctx).Create(&lock).Error
	if err != nil {
		// Lock might already exist, check if it's expired
		var existing CacheLock
		err := l.db.WithContext(ctx).
			Where("name = ?", l.name).
			First(&existing).Error

		if err != nil {
			return false, fmt.Errorf("database lock check: %w", err)
		}

		// If expired, update it with our ownership
		if existing.IsExpired() {
			result := l.db.WithContext(ctx).
				Model(&CacheLock{}).
				Where("name = ? AND expiration < ?", l.name, now).
				Updates(map[string]interface{}{
					"owner":      l.owner,
					"expiration": expiration,
					"created_at": now,
				})

			if result.Error != nil {
				return false, fmt.Errorf("database lock update: %w", result.Error)
			}

			return result.RowsAffected > 0, nil
		}

		return false, nil // Lock is held by someone else
	}

	return true, nil
}

// Release releases the lock only if owned by this instance.
func (l *DatabaseLock) Release(ctx context.Context) error {
	result := l.db.WithContext(ctx).
		Where("name = ? AND owner = ?", l.name, l.owner).
		Delete(&CacheLock{})

	if result.Error != nil {
		return fmt.Errorf("database lock release: %w", result.Error)
	}

	return nil
}

// ForceRelease forcefully releases the lock regardless of owner.
func (l *DatabaseLock) ForceRelease(ctx context.Context) error {
	err := l.db.WithContext(ctx).
		Where("name = ?", l.name).
		Delete(&CacheLock{}).Error

	if err != nil {
		return fmt.Errorf("database lock force release: %w", err)
	}

	return nil
}

// Owner returns the lock owner identifier.
func (l *DatabaseLock) Owner() string {
	return l.owner
}

// Get executes callback while holding the lock.
func (l *DatabaseLock) Get(ctx context.Context, callback func() error) error {
	acquired, err := l.Acquire(ctx)
	if err != nil {
		return err
	}
	if !acquired {
		return cache.ErrLockNotAcquired
	}

	defer l.Release(ctx)

	return callback()
}

// Block waits to acquire lock then executes callback.
func (l *DatabaseLock) Block(ctx context.Context, waitTime time.Duration, callback func() error) error {
	deadline := time.Now().Add(waitTime)

	for {
		acquired, err := l.Acquire(ctx)
		if err != nil {
			return err
		}

		if acquired {
			defer l.Release(ctx)
			return callback()
		}

		if time.Now().After(deadline) {
			return cache.ErrLockTimeout
		}

		// Wait a bit before retrying
		select {
		case <-time.After(100 * time.Millisecond):
			// Continue loop
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
