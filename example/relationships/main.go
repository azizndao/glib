// Package main demonstrates all relationship types in glib ORM
package main

import (
	"fmt"
	"log"

	"github.com/azizndao/glib/config"
	"github.com/azizndao/glib/database"
	"github.com/azizndao/glib/foundation"
	"github.com/azizndao/glib/orm"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================================================
// Models
// ============================================================================

// User model with HasOne (Profile), HasMany (Posts), and ManyToMany (Roles)
type User struct {
	orm.Model
	Name    string
	Email   string
	Profile *Profile `gorm:"foreignKey:UserID"`
	Posts   []Post   `gorm:"foreignKey:UserID"`
	Roles   []Role   `gorm:"many2many:user_roles;"`
}

// Profile model demonstrating HasOne relationship
type Profile struct {
	orm.Model
	UserID uuid.UUID `gorm:"type:char(36);index"`
	Bio    string
	Avatar string
	User   *User `gorm:"foreignKey:UserID"`
}

// Post model demonstrating BelongsTo (User) and HasMany (Comments)
type Post struct {
	orm.Model
	UserID    uuid.UUID `gorm:"type:char(36);index"`
	Title     string
	Body      string
	Published bool
	User      *User     `gorm:"foreignKey:UserID"`
	Comments  []Comment `gorm:"foreignKey:PostID"`
	Tags      []Tag     `gorm:"many2many:post_tags;"`
}

// Comment model demonstrating BelongsTo relationship
type Comment struct {
	orm.Model
	PostID uuid.UUID `gorm:"type:char(36);index"`
	Body   string
	Post   *Post `gorm:"foreignKey:PostID"`
}

// Tag model for ManyToMany with Posts
type Tag struct {
	orm.Model
	Name  string
	Posts []Post `gorm:"many2many:post_tags;"`
}

// Role model for ManyToMany with Users
type Role struct {
	orm.Model
	Name  string
	Users []User `gorm:"many2many:user_roles;"`
}

func main() {
	fmt.Println("=== glib ORM Relationships Example ===\n")

	// Setup application
	app := foundation.New("/app")
	cfg := config.New()
	cfg.Set("database.default", "sqlite")
	cfg.Set("database.connections.sqlite.driver", "sqlite")
	cfg.Set("database.connections.sqlite.database", "/tmp/glib_relationships.db")
	app.SetConfig(cfg)

	app.Register(&database.ServiceProvider{})
	if err := app.Bootstrap(); err != nil {
		log.Fatalf("Failed to bootstrap: %v", err)
	}

	dbManager := database.NewManager(app.Config(), app.Logger())
	conn, err := dbManager.DB()
	if err != nil {
		log.Fatalf("Failed to get database: %v", err)
	}
	db := conn.DB()

	// Auto-migrate all models
	fmt.Println("1. Migrating database schema...")
	err = db.AutoMigrate(&User{}, &Profile{}, &Post{}, &Comment{}, &Tag{}, &Role{})
	if err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}
	fmt.Println("   ✓ Schema migrated\n")

	// Demonstrate HasOne Relationship
	demonstrateHasOne(db)

	// Demonstrate HasMany Relationship
	demonstrateHasMany(db)

	// Demonstrate BelongsTo Relationship
	demonstrateBelongsTo(db)

	// Demonstrate ManyToMany Relationship
	demonstrateManyToMany(db)

	// Demonstrate Nested Eager Loading
	demonstrateNestedEagerLoading(db)

	// Demonstrate PreloadRelations Helper
	demonstratePreloadHelper(db)

	// Demonstrate Association Helpers
	demonstrateAssociationHelpers(db)

	fmt.Println("\n=== All Relationship Types Demonstrated ===")
}

func demonstrateHasOne(db *gorm.DB) {
	fmt.Println("2. HasOne Relationship (User -> Profile)")
	fmt.Println("   Creating user with profile...")

	user := User{
		Name:  "John Doe",
		Email: "john@example.com",
		Profile: &Profile{
			Bio:    "Software Engineer at TechCorp",
			Avatar: "avatar1.jpg",
		},
	}

	if err := db.Create(&user).Error; err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Printf("   ✓ User created: %s\n", user.Name)
	fmt.Printf("   ✓ Profile created with Bio: %s\n", user.Profile.Bio)

	// Load user with profile
	var loadedUser User
	db.Preload("Profile").First(&loadedUser, "id = ?", user.ID)
	fmt.Printf("   ✓ Loaded user with profile: %s has bio '%s'\n", loadedUser.Name, loadedUser.Profile.Bio)
	fmt.Println()
}

func demonstrateHasMany(db *gorm.DB) {
	fmt.Println("3. HasMany Relationship (User -> Posts)")
	fmt.Println("   Creating user with multiple posts...")

	user := User{
		Name:  "Jane Smith",
		Email: "jane@example.com",
		Posts: []Post{
			{Title: "Getting Started with Go", Body: "Go is awesome!", Published: true},
			{Title: "glib Framework Guide", Body: "Building web apps with glib", Published: true},
			{Title: "Draft Post", Body: "Work in progress", Published: false},
		},
	}

	if err := db.Create(&user).Error; err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Printf("   ✓ User created: %s\n", user.Name)
	fmt.Printf("   ✓ Created %d posts\n", len(user.Posts))

	// Load user with all posts
	var loadedUser User
	db.Preload("Posts").First(&loadedUser, "id = ?", user.ID)
	fmt.Printf("   ✓ Loaded user with %d posts\n", len(loadedUser.Posts))

	// Load user with only published posts
	var userWithPublished User
	db.Preload("Posts", "published = ?", true).First(&userWithPublished, "id = ?", user.ID)
	fmt.Printf("   ✓ Loaded user with %d published posts (filtered)\n", len(userWithPublished.Posts))
	fmt.Println()
}

func demonstrateBelongsTo(db *gorm.DB) {
	fmt.Println("4. BelongsTo Relationship (Post -> User)")
	fmt.Println("   Loading post with its author...")

	// Get any post
	var post Post
	db.First(&post)

	// Load post with user
	db.Preload("User").First(&post, "id = ?", post.ID)
	if post.User != nil {
		fmt.Printf("   ✓ Post '%s' belongs to user '%s'\n", post.Title, post.User.Name)
	}
	fmt.Println()
}

func demonstrateManyToMany(db *gorm.DB) {
	fmt.Println("5. ManyToMany Relationship (Users <-> Roles)")
	fmt.Println("   Creating roles and assigning to users...")

	// Create roles
	admin := Role{Name: "Admin"}
	editor := Role{Name: "Editor"}
	viewer := Role{Name: "Viewer"}

	db.Create(&admin)
	db.Create(&editor)
	db.Create(&viewer)

	fmt.Printf("   ✓ Created roles: %s, %s, %s\n", admin.Name, editor.Name, viewer.Name)

	// Create user with roles
	user := User{
		Name:  "Alice Johnson",
		Email: "alice@example.com",
		Roles: []Role{admin, editor},
	}

	db.Create(&user)
	fmt.Printf("   ✓ User '%s' created with %d roles\n", user.Name, len(user.Roles))

	// Load user with roles
	var loadedUser User
	db.Preload("Roles").First(&loadedUser, "id = ?", user.ID)
	fmt.Printf("   ✓ Loaded user with roles: ")
	for i, role := range loadedUser.Roles {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(role.Name)
	}
	fmt.Println()
	fmt.Println()
}

func demonstrateNestedEagerLoading(db *gorm.DB) {
	fmt.Println("6. Nested Eager Loading (User -> Posts -> Comments)")
	fmt.Println("   Creating user with posts and comments...")

	user := User{
		Name:  "Bob Brown",
		Email: "bob@example.com",
		Posts: []Post{
			{
				Title: "My First Post",
				Body:  "Hello World!",
				Comments: []Comment{
					{Body: "Great post!"},
					{Body: "Thanks for sharing!"},
				},
			},
		},
	}

	db.Create(&user)
	fmt.Printf("   ✓ User created with posts and comments\n")

	// Load user with nested relationships
	var loadedUser User
	db.Preload("Posts.Comments").First(&loadedUser, "id = ?", user.ID)
	fmt.Printf("   ✓ Loaded user '%s'\n", loadedUser.Name)
	for _, post := range loadedUser.Posts {
		fmt.Printf("     - Post: '%s' (%d comments)\n", post.Title, len(post.Comments))
	}
	fmt.Println()
}

func demonstratePreloadHelper(db *gorm.DB) {
	fmt.Println("7. PreloadRelations Helper")
	fmt.Println("   Loading user with multiple relations at once...")

	// Get any user with profile and posts
	var user User
	db.First(&user)

	// Load with multiple relations using helper
	query := orm.PreloadRelations(db, "Profile", "Posts", "Roles")
	var loadedUser User
	query.First(&loadedUser, "id = ?", user.ID)

	fmt.Printf("   ✓ User: %s\n", loadedUser.Name)
	if loadedUser.Profile != nil {
		fmt.Printf("   ✓ Profile: %s\n", loadedUser.Profile.Bio)
	}
	fmt.Printf("   ✓ Posts: %d\n", len(loadedUser.Posts))
	fmt.Printf("   ✓ Roles: %d\n", len(loadedUser.Roles))
	fmt.Println()
}

func demonstrateAssociationHelpers(db *gorm.DB) {
	fmt.Println("8. Association Helpers")
	fmt.Println("   Managing Many-to-Many relationships...")

	// Get user and role
	var user User
	db.Preload("Roles").First(&user, "email = ?", "alice@example.com")

	viewer := Role{Name: "PowerUser"}
	db.Create(&viewer)

	fmt.Printf("   User '%s' initially has %d roles\n", user.Name, len(user.Roles))

	// Append a role
	orm.Association(db, &user, "Roles").Append(&viewer)
	count := orm.Association(db, &user, "Roles").Count()
	fmt.Printf("   ✓ After Append: %d roles\n", count)

	// Delete a role
	if len(user.Roles) > 0 {
		orm.Association(db, &user, "Roles").Delete(&user.Roles[0])
		count = orm.Association(db, &user, "Roles").Count()
		fmt.Printf("   ✓ After Delete: %d roles\n", count)
	}

	// Count roles without loading them
	finalCount := orm.Association(db, &user, "Roles").Count()
	fmt.Printf("   ✓ Final role count: %d\n", finalCount)
	fmt.Println()
}
