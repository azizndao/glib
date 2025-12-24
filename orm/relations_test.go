package orm_test

import (
	"testing"

	"github.com/azizndao/glib/orm"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test models for relationships

type User struct {
	orm.Model
	Name    string
	Email   string
	Profile *Profile `gorm:"foreignKey:UserID"`
	Posts   []Post   `gorm:"foreignKey:UserID"`
	Roles   []Role   `gorm:"many2many:user_roles;"`
}

type Profile struct {
	orm.Model
	UserID uuid.UUID `gorm:"type:char(36)"`
	Bio    string
	Avatar string
	User   *User `gorm:"foreignKey:UserID"`
}

type Post struct {
	orm.Model
	UserID    uuid.UUID `gorm:"type:char(36)"`
	Title     string
	Body      string
	Published bool
	User      *User     `gorm:"foreignKey:UserID"`
	Comments  []Comment `gorm:"foreignKey:PostID"`
	Tags      []Tag     `gorm:"many2many:post_tags;"`
}

type Comment struct {
	orm.Model
	PostID uuid.UUID `gorm:"type:char(36)"`
	Body   string
	Post   *Post `gorm:"foreignKey:PostID"`
}

type Tag struct {
	orm.Model
	Name  string
	Posts []Post `gorm:"many2many:post_tags;"`
}

type Role struct {
	orm.Model
	Name  string
	Users []User `gorm:"many2many:user_roles;"`
}

// setupTestDB creates an in-memory SQLite database for testing
func setupRelationsTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto migrate all models
	err = db.AutoMigrate(&User{}, &Profile{}, &Post{}, &Comment{}, &Tag{}, &Role{})
	require.NoError(t, err)

	return db
}

// TestHasOneRelationship tests HasOne (1:1) relationship
func TestHasOneRelationship(t *testing.T) {
	db := setupRelationsTestDB(t)

	// Create user with profile
	user := User{
		Name:  "John Doe",
		Email: "john@example.com",
		Profile: &Profile{
			Bio:    "Software Developer",
			Avatar: "avatar.jpg",
		},
	}

	err := db.Create(&user).Error
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, user.ID)
	assert.NotEqual(t, uuid.Nil, user.Profile.ID)
	assert.Equal(t, user.ID, user.Profile.UserID)

	// Load user with profile using Preload
	var loadedUser User
	err = db.Preload("Profile").Where("id = ?", user.ID).First(&loadedUser).Error
	require.NoError(t, err)
	assert.NotNil(t, loadedUser.Profile)
	assert.Equal(t, "Software Developer", loadedUser.Profile.Bio)
}

// TestHasManyRelationship tests HasMany (1:N) relationship
func TestHasManyRelationship(t *testing.T) {
	db := setupRelationsTestDB(t)

	// Create user with posts
	user := User{
		Name:  "Jane Doe",
		Email: "jane@example.com",
		Posts: []Post{
			{Title: "First Post", Body: "Content 1", Published: true},
			{Title: "Second Post", Body: "Content 2", Published: false},
			{Title: "Third Post", Body: "Content 3", Published: true},
		},
	}

	err := db.Create(&user).Error
	require.NoError(t, err)
	assert.Len(t, user.Posts, 3)

	// Load user with all posts
	var loadedUser User
	err = db.Preload("Posts").Where("id = ?", user.ID).First(&loadedUser).Error
	require.NoError(t, err)
	assert.Len(t, loadedUser.Posts, 3)

	// Load user with conditional preload (only published posts)
	var userWithPublished User
	err = db.Preload("Posts", "published = ?", true).
		Where("id = ?", user.ID).
		First(&userWithPublished).Error
	require.NoError(t, err)
	assert.Len(t, userWithPublished.Posts, 2)
	for _, post := range userWithPublished.Posts {
		assert.True(t, post.Published)
	}

	// Load user with custom preload query
	var userWithLimited User
	err = db.Preload("Posts", func(db *gorm.DB) *gorm.DB {
		return db.Where("published = ?", true).Order("created_at DESC").Limit(1)
	}).Where("id = ?", user.ID).First(&userWithLimited).Error
	require.NoError(t, err)
	assert.Len(t, userWithLimited.Posts, 1)
}

// TestBelongsToRelationship tests BelongsTo (inverse of HasOne/HasMany)
func TestBelongsToRelationship(t *testing.T) {
	db := setupRelationsTestDB(t)

	// Create user and post separately
	user := User{
		Name:  "Bob Smith",
		Email: "bob@example.com",
	}
	err := db.Create(&user).Error
	require.NoError(t, err)

	post := Post{
		UserID: user.ID,
		Title:  "My Post",
		Body:   "Post content",
	}
	err = db.Create(&post).Error
	require.NoError(t, err)

	// Load post with user
	var loadedPost Post
	err = db.Preload("User").Where("id = ?", post.ID).First(&loadedPost).Error
	require.NoError(t, err)
	assert.NotNil(t, loadedPost.User)
	assert.Equal(t, user.Name, loadedPost.User.Name)
	assert.Equal(t, user.Email, loadedPost.User.Email)
}

// TestManyToManyRelationship tests ManyToMany (N:M) relationship
func TestManyToManyRelationship(t *testing.T) {
	db := setupRelationsTestDB(t)

	// Create roles
	admin := Role{Name: "admin"}
	editor := Role{Name: "editor"}
	viewer := Role{Name: "viewer"}

	err := db.Create(&admin).Error
	require.NoError(t, err)
	err = db.Create(&editor).Error
	require.NoError(t, err)
	err = db.Create(&viewer).Error
	require.NoError(t, err)

	// Create user with roles
	user := User{
		Name:  "Alice Johnson",
		Email: "alice@example.com",
		Roles: []Role{admin, editor},
	}

	err = db.Create(&user).Error
	require.NoError(t, err)
	assert.Len(t, user.Roles, 2)

	// Load user with roles
	var loadedUser User
	err = db.Preload("Roles").Where("id = ?", user.ID).First(&loadedUser).Error
	require.NoError(t, err)
	assert.Len(t, loadedUser.Roles, 2)

	// Test Association helpers
	t.Run("Append", func(t *testing.T) {
		// Append a new role
		err := orm.Association(db, &user, "Roles").Append(&viewer)
		require.NoError(t, err)

		// Verify
		count := orm.Association(db, &user, "Roles").Count()
		assert.Equal(t, int64(3), count)
	})

	t.Run("Delete", func(t *testing.T) {
		// Remove a role
		err := orm.Association(db, &user, "Roles").Delete(&editor)
		require.NoError(t, err)

		// Verify
		count := orm.Association(db, &user, "Roles").Count()
		assert.Equal(t, int64(2), count)
	})

	t.Run("Replace", func(t *testing.T) {
		// Replace all roles
		err := orm.Association(db, &user, "Roles").Replace(&admin)
		require.NoError(t, err)

		// Verify
		count := orm.Association(db, &user, "Roles").Count()
		assert.Equal(t, int64(1), count)

		// Load and check
		var roles []Role
		err = orm.Association(db, &user, "Roles").Find(&roles)
		require.NoError(t, err)
		assert.Len(t, roles, 1)
		assert.Equal(t, "admin", roles[0].Name)
	})

	t.Run("Clear", func(t *testing.T) {
		// Clear all roles
		err := orm.Association(db, &user, "Roles").Clear()
		require.NoError(t, err)

		// Verify
		count := orm.Association(db, &user, "Roles").Count()
		assert.Equal(t, int64(0), count)
	})
}

// TestNestedEagerLoading tests loading nested relationships
func TestNestedEagerLoading(t *testing.T) {
	db := setupRelationsTestDB(t)

	// Create user with posts and comments
	user := User{
		Name:  "Charlie Brown",
		Email: "charlie@example.com",
		Posts: []Post{
			{
				Title: "Post with Comments",
				Body:  "Some content",
				Comments: []Comment{
					{Body: "First comment"},
					{Body: "Second comment"},
				},
			},
		},
	}

	err := db.Create(&user).Error
	require.NoError(t, err)

	// Load user with nested relationships
	var loadedUser User
	err = db.Preload("Posts.Comments").
		Where("id = ?", user.ID).
		First(&loadedUser).Error
	require.NoError(t, err)
	assert.Len(t, loadedUser.Posts, 1)
	assert.Len(t, loadedUser.Posts[0].Comments, 2)
}

// TestPreloadRelationsHelper tests the PreloadRelations helper function
func TestPreloadRelationsHelper(t *testing.T) {
	db := setupRelationsTestDB(t)

	// Create test data
	user := User{
		Name:  "David Lee",
		Email: "david@example.com",
		Profile: &Profile{
			Bio: "Engineer",
		},
		Posts: []Post{
			{Title: "Post 1", Published: true},
			{Title: "Post 2", Published: false},
		},
		Roles: []Role{
			{Name: "user"},
		},
	}

	err := db.Create(&user).Error
	require.NoError(t, err)

	// Load with multiple relations using helper
	query := orm.PreloadRelations(db, "Profile", "Posts", "Roles")
	var loadedUser User
	err = query.Where("id = ?", user.ID).First(&loadedUser).Error
	require.NoError(t, err)

	assert.NotNil(t, loadedUser.Profile)
	assert.Len(t, loadedUser.Posts, 2)
	assert.Len(t, loadedUser.Roles, 1)
}

// TestAssociationCount tests counting associated records
func TestAssociationCount(t *testing.T) {
	db := setupRelationsTestDB(t)

	// Create user with multiple posts
	user := User{
		Name:  "Emma Wilson",
		Email: "emma@example.com",
		Posts: []Post{
			{Title: "Post 1"},
			{Title: "Post 2"},
			{Title: "Post 3"},
		},
	}

	err := db.Create(&user).Error
	require.NoError(t, err)

	// Count posts without loading them
	count := orm.Association(db, &user, "Posts").Count()
	assert.Equal(t, int64(3), count)
}

// TestAssociationFind tests finding associated records
func TestAssociationFind(t *testing.T) {
	db := setupRelationsTestDB(t)

	// Create user with roles
	admin := Role{Name: "admin"}
	editor := Role{Name: "editor"}

	user := User{
		Name:  "Frank Miller",
		Email: "frank@example.com",
		Roles: []Role{admin, editor},
	}

	err := db.Create(&user).Error
	require.NoError(t, err)

	// Find associated roles
	var roles []Role
	err = orm.Association(db, &user, "Roles").Find(&roles)
	require.NoError(t, err)
	assert.Len(t, roles, 2)
}

// BenchmarkEagerLoadingVsLazy benchmarks eager loading vs lazy loading (N+1)
func BenchmarkEagerLoadingVsLazy(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = db.AutoMigrate(&User{}, &Post{})

	// Create test data
	for range 10 {
		user := User{
			Name:  "User",
			Email: "user@example.com",
			Posts: []Post{
				{Title: "Post 1"},
				{Title: "Post 2"},
				{Title: "Post 3"},
			},
		}
		_ = db.Create(&user)
	}

	b.Run("EagerLoading", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var users []User
			_ = db.Preload("Posts").Find(&users).Error
		}
	})

	b.Run("LazyLoading_N+1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var users []User
			_ = db.Find(&users).Error
			for _, user := range users {
				var posts []Post
				_ = db.Where("user_id = ?", user.ID).Find(&posts).Error
			}
		}
	})
}
