package orm_test

import (
	"context"
	"testing"

	"github.com/azizndao/glib/database/orm"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestModel for soft delete testing
type TestUser struct {
	orm.Model
	Name  string
	Email string
}

func setupSoftDeleteTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&TestUser{})
	assert.NoError(t, err)

	return db
}

func TestWithTrashed(t *testing.T) {
	db := setupSoftDeleteTestDB(t)
	ctx := context.Background()

	// Create a user
	user := &TestUser{Name: "John", Email: "john@example.com"}
	err := orm.G[TestUser](db).Create(ctx, user)
	assert.NoError(t, err)

	// Soft delete the user
	_, err = orm.G[TestUser](db).Where("id = ?", user.ID).Delete(ctx)
	assert.NoError(t, err)

	// Normal query should not find deleted user
	users, err := orm.G[TestUser](db).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, 0)

	// WithTrashed should find the user
	users, err = orm.G[TestUser](orm.WithTrashed(db)).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "John", users[0].Name)
}

func TestOnlyTrashed(t *testing.T) {
	db := setupSoftDeleteTestDB(t)
	ctx := context.Background()

	// Create two users
	user1 := &TestUser{Name: "John", Email: "john@example.com"}
	user2 := &TestUser{Name: "Jane", Email: "jane@example.com"}
	err := orm.G[TestUser](db).Create(ctx, user1)
	assert.NoError(t, err)
	err = orm.G[TestUser](db).Create(ctx, user2)
	assert.NoError(t, err)

	// Delete only user1
	_, err = orm.G[TestUser](db).Where("id = ?", user1.ID).Delete(ctx)
	assert.NoError(t, err)

	// OnlyTrashed should find only user1
	users, err := orm.G[TestUser](orm.OnlyTrashed(db)).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "John", users[0].Name)

	// Normal query should find only user2
	users, err = orm.G[TestUser](db).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "Jane", users[0].Name)
}

func TestForceDelete(t *testing.T) {
	db := setupSoftDeleteTestDB(t)
	ctx := context.Background()

	// Create a user
	user := &TestUser{Name: "John", Email: "john@example.com"}
	err := orm.G[TestUser](db).Create(ctx, user)
	assert.NoError(t, err)
	userID := user.ID

	// Force delete the user
	err = orm.ForceDelete[TestUser](ctx, db, userID)
	assert.NoError(t, err)

	// WithTrashed should also not find the user (permanently deleted)
	users, err := orm.G[TestUser](orm.WithTrashed(db)).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, 0)
}

func TestForceDeleteWhere(t *testing.T) {
	db := setupSoftDeleteTestDB(t)
	ctx := context.Background()

	// Create users
	users := []*TestUser{
		{Name: "John", Email: "john@example.com"},
		{Name: "Jane", Email: "jane@example.com"},
		{Name: "Bob", Email: "bob@test.com"},
	}

	for _, u := range users {
		err := orm.G[TestUser](db).Create(ctx, u)
		assert.NoError(t, err)
	}

	// Force delete all @example.com users
	rows, err := orm.ForceDeleteWhere[TestUser](ctx, db, "email LIKE ?", "%@example.com")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), rows)

	// Should only have Bob left
	remaining, err := orm.G[TestUser](db).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, remaining, 1)
	assert.Equal(t, "Bob", remaining[0].Name)
}

func TestRestore(t *testing.T) {
	db := setupSoftDeleteTestDB(t)
	ctx := context.Background()

	// Create a user
	user := &TestUser{Name: "John", Email: "john@example.com"}
	err := orm.G[TestUser](db).Create(ctx, user)
	assert.NoError(t, err)
	userID := user.ID

	// Soft delete the user
	_, err = orm.G[TestUser](db).Where("id = ?", userID).Delete(ctx)
	assert.NoError(t, err)

	// Verify it's deleted
	users, err := orm.G[TestUser](db).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, 0)

	// Restore the user
	err = orm.Restore[TestUser](ctx, db, userID)
	assert.NoError(t, err)

	// Verify it's restored
	users, err = orm.G[TestUser](db).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "John", users[0].Name)
	assert.False(t, users[0].IsDeleted())
}

func TestRestoreMultiple(t *testing.T) {
	db := setupSoftDeleteTestDB(t)
	ctx := context.Background()

	// Create users
	user1 := &TestUser{Name: "John", Email: "john@example.com"}
	user2 := &TestUser{Name: "Jane", Email: "jane@example.com"}
	err := orm.G[TestUser](db).Create(ctx, user1)
	assert.NoError(t, err)
	err = orm.G[TestUser](db).Create(ctx, user2)
	assert.NoError(t, err)

	// Delete both
	_, err = orm.G[TestUser](db).Where("id IN ?", []uuid.UUID{user1.ID, user2.ID}).Delete(ctx)
	assert.NoError(t, err)

	// Restore both
	err = orm.Restore[TestUser](ctx, db, user1.ID, user2.ID)
	assert.NoError(t, err)

	// Verify both are restored
	users, err := orm.G[TestUser](db).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, 2)
}

func TestRestoreWhere(t *testing.T) {
	db := setupSoftDeleteTestDB(t)
	ctx := context.Background()

	// Create users
	users := []*TestUser{
		{Name: "John", Email: "john@example.com"},
		{Name: "Jane", Email: "jane@example.com"},
		{Name: "Bob", Email: "bob@test.com"},
	}

	for _, u := range users {
		err := orm.G[TestUser](db).Create(ctx, u)
		assert.NoError(t, err)
	}

	// Delete all users
	_, err := orm.G[TestUser](db).Delete(ctx)
	assert.NoError(t, err)

	// Restore only @example.com users
	rows, err := orm.RestoreWhere[TestUser](ctx, db, "email LIKE ?", "%@example.com")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), rows)

	// Should have John and Jane restored
	restored, err := orm.G[TestUser](db).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, restored, 2)

	// Bob should still be deleted
	trashed, err := orm.G[TestUser](orm.OnlyTrashed(db)).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, trashed, 1)
	assert.Equal(t, "Bob", trashed[0].Name)
}

func TestIsTrashed(t *testing.T) {
	db := setupSoftDeleteTestDB(t)
	ctx := context.Background()

	// Create a user
	user := &TestUser{Name: "John", Email: "john@example.com"}
	err := orm.G[TestUser](db).Create(ctx, user)
	assert.NoError(t, err)

	// Should not be trashed initially
	assert.False(t, orm.IsTrashed(user))
	assert.False(t, user.IsDeleted())

	// Soft delete
	_, err = orm.G[TestUser](db).Where("id = ?", user.ID).Delete(ctx)
	assert.NoError(t, err)

	// Fetch the deleted user with WithTrashed
	deletedUser, err := orm.G[TestUser](orm.WithTrashed(db)).Where("id = ?", user.ID).First(ctx)
	assert.NoError(t, err)

	// Should be trashed now
	assert.True(t, orm.IsTrashed(&deletedUser))
	assert.True(t, deletedUser.IsDeleted())
}
