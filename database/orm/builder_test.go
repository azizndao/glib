package orm

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&TestUser{}, &TestPost{})
	assert.NoError(t, err)

	return db
}

func Test_Create(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	user := &TestUser{
		Name:   "Alice",
		Email:  "alice@example.com",
		Age:    28,
		Active: true,
	}

	err := G[TestUser](db).Create(ctx, user)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.False(t, user.CreatedAt.IsZero())
}

func Test_CreateInBatches(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	users := []TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true},
		{Name: "Bob", Email: "bob@example.com", Age: 35, Active: true},
		{Name: "Charlie", Email: "charlie@example.com", Age: 42, Active: false},
	}

	err := G[TestUser](db).CreateInBatches(ctx, &users, 2)
	assert.NoError(t, err)

	for _, user := range users {
		assert.NotEqual(t, uuid.Nil, user.ID)
	}
}

func Test_First(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create test data
	user1 := &TestUser{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true}
	err := G[TestUser](db).Create(ctx, user1)
	assert.NoError(t, err)

	// Test First
	found, err := G[TestUser](db).Where("email = ?", "alice@example.com").First(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", found.Name)
	assert.Equal(t, user1.ID, found.ID)
}

func Test_Find(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create test data
	users := []TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true},
		{Name: "Bob", Email: "bob@example.com", Age: 35, Active: true},
		{Name: "Charlie", Email: "charlie@example.com", Age: 42, Active: false},
	}
	err := G[TestUser](db).CreateInBatches(ctx, &users, 10)
	assert.NoError(t, err)

	// Test Find all
	allUsers, err := G[TestUser](db).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, allUsers, 3)

	// Test Find with condition
	activeUsers, err := G[TestUser](db).Where("active = ?", true).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, activeUsers, 2)
}

func Test_Update(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create test data
	user := &TestUser{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true}
	err := G[TestUser](db).Create(ctx, user)
	assert.NoError(t, err)

	// Test Update single column
	rowsAffected, err := G[TestUser](db).Where("id = ?", user.ID).Update(ctx, "age", 29)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowsAffected)

	// Verify update
	updated, err := G[TestUser](db).Where("id = ?", user.ID).First(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 29, updated.Age)
}

func Test_Updates(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create test data
	user := &TestUser{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true}
	err := G[TestUser](db).Create(ctx, user)
	assert.NoError(t, err)

	// Test Updates multiple columns (Note: GORM doesn't update zero values by default)
	updatedUser := TestUser{Age: 30, Name: "Alice Updated"}
	rowsAffected, err := G[TestUser](db).Where("id = ?", user.ID).Updates(ctx, updatedUser)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowsAffected)

	// Verify updates
	found, err := G[TestUser](db).Where("id = ?", user.ID).First(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 30, found.Age)
	assert.Equal(t, "Alice Updated", found.Name)
}

func Test_Delete(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create test data
	user := &TestUser{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true}
	err := G[TestUser](db).Create(ctx, user)
	assert.NoError(t, err)

	// Test Delete (soft delete)
	rowsAffected, err := G[TestUser](db).Where("id = ?", user.ID).Delete(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowsAffected)

	// Verify soft delete - should not find
	_, err = G[TestUser](db).Where("id = ?", user.ID).First(ctx)
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)

	// Verify record exists in database with deleted_at set
	var deletedUser TestUser
	err = db.Unscoped().Where("id = ?", user.ID).First(&deletedUser).Error
	assert.NoError(t, err)
	assert.True(t, deletedUser.DeletedAt.Valid)
}

func Test_Count(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create test data
	users := []TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true},
		{Name: "Bob", Email: "bob@example.com", Age: 35, Active: true},
		{Name: "Charlie", Email: "charlie@example.com", Age: 42, Active: false},
	}
	err := G[TestUser](db).CreateInBatches(ctx, &users, 10)
	assert.NoError(t, err)

	// Test Count all
	count, err := G[TestUser](db).Count(ctx, "*")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Test Count with condition
	activeCount, err := G[TestUser](db).Where("active = ?", true).Count(ctx, "*")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), activeCount)
}

func Test_Scopes(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create test data
	users := []TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true},
		{Name: "Bob", Email: "bob@example.com", Age: 35, Active: true},
		{Name: "Charlie", Email: "charlie@example.com", Age: 42, Active: false},
	}
	err := G[TestUser](db).CreateInBatches(ctx, &users, 10)
	assert.NoError(t, err)

	// Create a custom scope
	activeScope := func(stmt *gorm.Statement) {
		stmt.DB = stmt.DB.Where("active = ?", true)
	}

	// Test Scopes
	activeUsers, err := G[TestUser](db).Scopes(activeScope).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, activeUsers, 2)
}

func Test_OrderAndLimit(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create test data
	users := []TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true},
		{Name: "Bob", Email: "bob@example.com", Age: 35, Active: true},
		{Name: "Charlie", Email: "charlie@example.com", Age: 42, Active: false},
	}
	err := G[TestUser](db).CreateInBatches(ctx, &users, 10)
	assert.NoError(t, err)

	// Test Order and Limit
	oldestTwo, err := G[TestUser](db).Order("age DESC").Limit(2).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, oldestTwo, 2)
	assert.Equal(t, "Charlie", oldestTwo[0].Name) // age 42
	assert.Equal(t, "Bob", oldestTwo[1].Name)     // age 35
}

func Test_Paginate(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create test data
	users := []TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true},
		{Name: "Bob", Email: "bob@example.com", Age: 35, Active: true},
		{Name: "Charlie", Email: "charlie@example.com", Age: 42, Active: false},
		{Name: "Diana", Email: "diana@example.com", Age: 25, Active: true},
		{Name: "Eve", Email: "eve@example.com", Age: 31, Active: true},
	}
	err := G[TestUser](db).CreateInBatches(ctx, &users, 10)
	assert.NoError(t, err)

	// Test Paginate - first page
	builder := G[TestUser](db).Order("name ASC")
	paginator, err := Paginate(ctx, builder, 1, 2)
	assert.NoError(t, err)
	assert.Len(t, paginator.Data, 2)
	assert.Equal(t, int64(5), paginator.Total)
	assert.Equal(t, 1, paginator.CurrentPage)
	assert.Equal(t, 3, paginator.LastPage)
	assert.True(t, paginator.HasMorePages())
	assert.False(t, paginator.IsEmpty())

	// Test Paginate - second page
	paginator, err = Paginate(ctx, builder, 2, 2)
	assert.NoError(t, err)
	assert.Len(t, paginator.Data, 2)
	assert.Equal(t, 2, paginator.CurrentPage)
}

func Test_Chunk(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create test data
	users := []TestUser{
		{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true},
		{Name: "Bob", Email: "bob@example.com", Age: 35, Active: true},
		{Name: "Charlie", Email: "charlie@example.com", Age: 42, Active: false},
		{Name: "Diana", Email: "diana@example.com", Age: 25, Active: true},
		{Name: "Eve", Email: "eve@example.com", Age: 31, Active: true},
	}
	err := G[TestUser](db).CreateInBatches(ctx, &users, 10)
	assert.NoError(t, err)

	// Test Chunk
	var processedCount int
	builder := G[TestUser](db).Order("name ASC")
	err = Chunk(ctx, builder, 2, func(batch []TestUser) error {
		processedCount += len(batch)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 5, processedCount)
}

func Test_Exists(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Initially should not exist
	exists, err := Exists(ctx, G[TestUser](db).Where("email = ?", "alice@example.com"))
	assert.NoError(t, err)
	assert.False(t, exists)

	// Create user
	user := &TestUser{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true}
	err = G[TestUser](db).Create(ctx, user)
	assert.NoError(t, err)

	// Now should exist
	exists, err = Exists(ctx, G[TestUser](db).Where("email = ?", "alice@example.com"))
	assert.NoError(t, err)
	assert.True(t, exists)

	// Test DoesntExist
	doesntExist, err := DoesntExist(ctx, G[TestUser](db).Where("email = ?", "nonexistent@example.com"))
	assert.NoError(t, err)
	assert.True(t, doesntExist)
}

func Test_WithUUIDPrimaryKey(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create user - UUID should be auto-generated
	user := &TestUser{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true}
	err := G[TestUser](db).Create(ctx, user)
	assert.NoError(t, err)

	// Verify UUID was generated
	assert.NotEqual(t, uuid.Nil, user.ID)

	// Verify we can query by UUID
	found, err := G[TestUser](db).Where("id = ?", user.ID).First(ctx)
	assert.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "Alice", found.Name)
}

func Test_Preload(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create user with posts
	user := &TestUser{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true}
	err := G[TestUser](db).Create(ctx, user)
	assert.NoError(t, err)

	posts := []TestPost{
		{Title: "Post 1", Content: "Content 1", Published: true, UserID: user.ID},
		{Title: "Post 2", Content: "Content 2", Published: false, UserID: user.ID},
		{Title: "Post 3", Content: "Content 3", Published: true, UserID: user.ID},
	}
	err = G[TestPost](db).CreateInBatches(ctx, &posts, 10)
	assert.NoError(t, err)

	// Note: Preload syntax varies, this is a simplified test
	// In real usage, you'd define relationships in the model and use Preload
	publishedPosts, err := G[TestPost](db).Where("user_id = ? AND published = ?", user.ID, true).Find(ctx)
	assert.NoError(t, err)
	assert.Len(t, publishedPosts, 2)
}

func Test_ErrRecordNotFound(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Try to find non-existent record
	_, err := G[TestUser](db).Where("email = ?", "nonexistent@example.com").First(ctx)
	assert.Error(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func Test_TimestampsAutoManaged(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Create user
	user := &TestUser{Name: "Alice", Email: "alice@example.com", Age: 28, Active: true}
	err := G[TestUser](db).Create(ctx, user)
	assert.NoError(t, err)

	// Verify timestamps
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())

	createdAt := user.CreatedAt

	// Sleep to ensure UpdatedAt changes
	time.Sleep(time.Millisecond * 10)

	// Update user
	_, err = G[TestUser](db).Where("id = ?", user.ID).Update(ctx, "age", 29)
	assert.NoError(t, err)

	// Verify UpdatedAt changed
	updated, err := G[TestUser](db).Where("id = ?", user.ID).First(ctx)
	assert.NoError(t, err)
	assert.True(t, updated.UpdatedAt.After(createdAt))
}
