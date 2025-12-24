// Package orm provides soft delete helpers.
package orm

import (
	"context"

	"gorm.io/gorm"
)

// WithTrashed includes soft-deleted records in the query.
// Use this when you want to retrieve all records including deleted ones.
//
// Example:
//
//	// Get all users including soft-deleted
//	db = orm.WithTrashed(db)
//	users, err := orm.G[User](db).Find(ctx)
//
// Or apply directly:
//
//	users, err := orm.G[User](orm.WithTrashed(db)).Find(ctx)
func WithTrashed(db *gorm.DB) *gorm.DB {
	return db.Unscoped()
}

// OnlyTrashed returns only soft-deleted records.
// This creates a scope that filters for records where deleted_at IS NOT NULL.
//
// Example:
//
//	// Get only soft-deleted users
//	db = orm.OnlyTrashed(db)
//	trashedUsers, err := orm.G[User](db).Find(ctx)
//
// Or apply directly:
//
//	trashedUsers, err := orm.G[User](orm.OnlyTrashed(db)).Find(ctx)
func OnlyTrashed(db *gorm.DB) *gorm.DB {
	return db.Unscoped().Where("deleted_at IS NOT NULL")
}

// ForceDelete permanently deletes records, bypassing soft delete.
// This is a hard delete that cannot be undone.
//
// Example:
//
//	// Permanently delete user
//	err := orm.ForceDelete[User](ctx, db, userID)
//
//	// Or with a query
//	rowsAffected, err := orm.ForceDeleteWhere[User](ctx, db, "email = ?", "spam@example.com")
func ForceDelete[T any](ctx context.Context, db *gorm.DB, id any) error {
	return db.WithContext(ctx).Unscoped().Delete(new(T), id).Error
}

// ForceDeleteWhere permanently deletes records matching the condition.
//
// Example:
//
//	// Permanently delete all inactive users
//	rows, err := orm.ForceDeleteWhere[User](ctx, db, "active = ?", false)
func ForceDeleteWhere[T any](ctx context.Context, db *gorm.DB, query any, args ...any) (int64, error) {
	result := db.WithContext(ctx).Unscoped().Where(query, args...).Delete(new(T))
	return result.RowsAffected, result.Error
}

// Restore un-deletes soft-deleted records by setting deleted_at to NULL.
//
// Example:
//
//	// Restore a single user
//	err := orm.Restore[User](ctx, db, userID)
//
//	// Restore multiple users
//	err := orm.Restore[User](ctx, db, userID1, userID2, userID3)
func Restore[T any](ctx context.Context, db *gorm.DB, ids ...any) error {
	if len(ids) == 0 {
		return nil
	}

	// For single ID
	if len(ids) == 1 {
		return db.WithContext(ctx).
			Model(new(T)).
			Unscoped().
			Where("id = ?", ids[0]).
			Update("deleted_at", nil).Error
	}

	// For multiple IDs
	return db.WithContext(ctx).
		Model(new(T)).
		Unscoped().
		Where("id IN ?", ids).
		Update("deleted_at", nil).Error
}

// RestoreWhere un-deletes soft-deleted records matching the condition.
//
// Example:
//
//	// Restore all users with specific email domain
//	rows, err := orm.RestoreWhere[User](ctx, db, "email LIKE ?", "%@company.com")
func RestoreWhere[T any](ctx context.Context, db *gorm.DB, query any, args ...any) (int64, error) {
	result := db.WithContext(ctx).
		Model(new(T)).
		Unscoped().
		Where("deleted_at IS NOT NULL").
		Where(query, args...).
		Update("deleted_at", nil)
	return result.RowsAffected, result.Error
}

// IsTrashed checks if a record is soft deleted.
// This is a convenience function that checks the DeletedAt field.
//
// Example:
//
//	user, _ := orm.G[User](db).Where("id = ?", userID).First(ctx)
//	if orm.IsTrashed(user) {
//	    fmt.Println("User is soft deleted")
//	}
//
// Note: This requires the model to have a DeletedAt field.
// For models using orm.Model, you can also use model.IsDeleted().
func IsTrashed(model any) bool {
	// Try to get DeletedAt via type assertion
	type deletable interface {
		IsDeleted() bool
	}

	if d, ok := model.(deletable); ok {
		return d.IsDeleted()
	}

	return false
}
