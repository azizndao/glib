package main

import (
	"context"
	"fmt"
	"log"

	"github.com/azizndao/glib/database/orm"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User model demonstrates basic ORM Model usage
type User struct {
	orm.Model
	Name   string
	Email  string `gorm:"uniqueIndex"`
	Age    int
	Active bool `gorm:"default:true"`
	Posts  []Post
}

// Post model demonstrates relationships and scopes
type Post struct {
	orm.Model
	Title     string
	Content   string
	Published bool `gorm:"default:false"`
	UserID    uuid.UUID
	User      User
	Tags      []Tag `gorm:"many2many:post_tags;"`
}

// Tag model demonstrates many-to-many relationships
type Tag struct {
	orm.Model
	Name  string `gorm:"uniqueIndex"`
	Posts []Post `gorm:"many2many:post_tags;"`
}

func main() {
	fmt.Println("=== glib ORM Example (Generics API) ===")
	ctx := context.Background()

	// 1. Setup database connection
	fmt.Println("1. Setting up database connection...")
	db, err := gorm.Open(sqlite.Open("orm_example.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate models
	err = db.AutoMigrate(&User{}, &Post{}, &Tag{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	fmt.Println("✓ Database connected and migrated")

	// Clean up previous data
	db.Exec("DELETE FROM post_tags")
	db.Exec("DELETE FROM posts")
	db.Exec("DELETE FROM tags")
	db.Exec("DELETE FROM users")

	// 2. Create records using Generics API
	fmt.Println("2. Creating users...")
	users := []User{
		{Name: "Alice Johnson", Email: "alice@example.com", Age: 28, Active: true},
		{Name: "Bob Smith", Email: "bob@example.com", Age: 35, Active: true},
		{Name: "Charlie Brown", Email: "charlie@example.com", Age: 42, Active: false},
	}

	for i := range users {
		if err := orm.G[User](db).Create(ctx, &users[i]); err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}
		fmt.Printf("  ✓ Created user: %s (ID: %s)\n", users[i].Name, users[i].ID)
	}
	fmt.Println()

	// 3. Create posts
	fmt.Println("3. Creating posts...")
	posts := []Post{
		{Title: "Getting Started with Go", Content: "Go is a great language...", Published: true, UserID: users[0].ID},
		{Title: "Advanced Go Patterns", Content: "Let's explore patterns...", Published: true, UserID: users[0].ID},
		{Title: "Database Design", Content: "Database design tips...", Published: false, UserID: users[1].ID},
		{Title: "Web Development", Content: "Building web apps...", Published: true, UserID: users[1].ID},
	}

	for i := range posts {
		if err := orm.G[Post](db).Create(ctx, &posts[i]); err != nil {
			log.Fatalf("Failed to create post: %v", err)
		}
		fmt.Printf("  ✓ Created post: %s (ID: %s)\n", posts[i].Title, posts[i].ID)
	}
	fmt.Println()

	// 4. Query with Generics API - Basic Where
	fmt.Println("4. Querying with Generics API...")
	activeUsers, err := orm.G[User](db).Where("active = ?", true).Order("name ASC").Find(ctx)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	fmt.Printf("  Found %d active users:\n", len(activeUsers))
	for _, user := range activeUsers {
		fmt.Printf("    - %s (age: %d)\n", user.Name, user.Age)
	}
	fmt.Println()

	// 5. Query with Scopes
	fmt.Println("5. Using query scopes...")
	// Apply scope to DB before wrapping with G[T]
	scopedDB := orm.WhereColumn("published", true)(db)
	publishedPosts, err := orm.G[Post](scopedDB).Order("created_at DESC").Find(ctx)
	if err != nil {
		log.Fatalf("Scopes query failed: %v", err)
	}
	fmt.Printf("  Found %d published posts:\n", len(publishedPosts))
	for _, post := range publishedPosts {
		fmt.Printf("    - %s\n", post.Title)
	}
	fmt.Println()

	// 6. Combining multiple scopes
	fmt.Println("6. Combining multiple scopes...")
	scopedDB2 := orm.WhereColumn("published", true)(db)
	scopedDB2 = orm.BelongsTo("user_id", users[0].ID)(scopedDB2)
	alicePosts, err := orm.G[Post](scopedDB2).Find(ctx)
	if err != nil {
		log.Fatalf("Combined scopes query failed: %v", err)
	}
	fmt.Printf("  Alice's published posts: %d\n", len(alicePosts))
	for _, post := range alicePosts {
		fmt.Printf("    - %s\n", post.Title)
	}
	fmt.Println()

	// 7. Using First (record not found returns error)
	fmt.Println("7. Using First...")
	user, err := orm.G[User](db).Where("email = ?", "bob@example.com").First(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Println("  User not found")
		} else {
			log.Fatalf("First failed: %v", err)
		}
	} else {
		fmt.Printf("  Found user: %s (ID: %s)\n", user.Name, user.ID)
	}
	fmt.Println()

	// 8. Count and Exists
	fmt.Println("8. Using Count and Exists...")
	count, err := orm.G[Post](db).Where("published = ?", true).Count(ctx, "*")
	if err != nil {
		log.Fatalf("Count failed: %v", err)
	}
	fmt.Printf("  Total published posts: %d\n", count)

	exists, err := orm.Exists(ctx, orm.G[User](db).Where("email = ?", "nonexistent@example.com"))
	if err != nil {
		log.Fatalf("Exists failed: %v", err)
	}
	fmt.Printf("  User with nonexistent email exists: %v\n", exists)
	fmt.Println()

	// 9. Update operations
	fmt.Println("9. Updating records...")
	rows, err := orm.G[User](db).Where("name = ?", "Charlie Brown").Update(ctx, "active", true)
	if err != nil {
		log.Fatalf("Update failed: %v", err)
	}
	fmt.Printf("  ✓ Activated Charlie's account (%d rows affected)\n", rows)

	rows, err = orm.G[Post](db).Where("title = ?", "Database Design").Updates(ctx, Post{
		Published: true,
		Title:     "Database Design Best Practices",
	})
	if err != nil {
		log.Fatalf("Updates failed: %v", err)
	}
	fmt.Printf("  ✓ Updated Database Design post (%d rows affected)\n", rows)
	fmt.Println()

	// 10. Pagination
	fmt.Println("10. Paginating results...")
	paginatedChain := orm.G[User](db).Order("name ASC")
	paginator, err := orm.Paginate(ctx, paginatedChain, 1, 2)
	if err != nil {
		log.Fatalf("Pagination failed: %v", err)
	}
	fmt.Printf("  Page %d of %d (Total: %d)\n", paginator.CurrentPage, paginator.LastPage, paginator.Total)
	fmt.Printf("  Users on this page:\n")
	for _, user := range paginator.Data {
		fmt.Printf("    - %s\n", user.Name)
	}
	fmt.Println()

	// 11. Search scope
	fmt.Println("11. Using search scope...")
	searchDB := orm.Search([]string{"title", "content"}, "Go")(db)
	searchResults, err := orm.G[Post](searchDB).Find(ctx)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}
	fmt.Printf("  Posts matching 'Go': %d\n", len(searchResults))
	for _, post := range searchResults {
		fmt.Printf("    - %s\n", post.Title)
	}
	fmt.Println()

	// 12. Soft delete
	fmt.Println("12. Soft delete demonstration...")
	postToDelete := posts[2]

	rows, err = orm.G[Post](db).Where("id = ?", postToDelete.ID).Delete(ctx)
	if err != nil {
		log.Fatalf("Delete failed: %v", err)
	}
	fmt.Printf("  ✓ Soft deleted post: %s (%d rows affected)\n", postToDelete.Title, rows)

	// Verify it's not in normal queries
	visiblePosts, _ := orm.G[Post](db).Find(ctx)
	fmt.Printf("  Visible posts after soft delete: %d\n", len(visiblePosts))

	// But still exists with Unscoped
	var allPostsSlice []Post
	db.Unscoped().Find(&allPostsSlice)
	fmt.Printf("  Total posts including soft deleted: %d\n", len(allPostsSlice))
	fmt.Println()

	// 13. Custom scope
	fmt.Println("13. Custom scope example...")
	seniorUsers := func(db *gorm.DB) *gorm.DB {
		return db.Where("age >= ?", 35)
	}

	seniorDB := seniorUsers(db)
	seniors, err := orm.G[User](seniorDB).Order("age DESC").Find(ctx)
	if err != nil {
		log.Fatalf("Custom scope failed: %v", err)
	}
	fmt.Printf("  Senior users (age >= 35): %d\n", len(seniors))
	for _, user := range seniors {
		fmt.Printf("    - %s (age: %d)\n", user.Name, user.Age)
	}
	fmt.Println()

	// 14. Chunk processing
	fmt.Println("14. Chunk processing...")
	processedCount := 0
	err = orm.Chunk(ctx, orm.G[User](db).Order("name ASC"), 2, func(batch []User) error {
		processedCount += len(batch)
		fmt.Printf("  Processing batch: %d users\n", len(batch))
		return nil
	})
	if err != nil {
		log.Fatalf("Chunk processing failed: %v", err)
	}
	fmt.Printf("  Total processed: %d users\n", processedCount)
	fmt.Println()

	// 15. Model methods
	fmt.Println("15. Using Model methods...")
	newUser := User{Name: "Diana Prince", Email: "diana@example.com", Age: 30}
	if newUser.IsNew() {
		fmt.Printf("  User is new (ID: %s)\n", newUser.ID)
	}

	db.Create(&newUser)
	if !newUser.IsNew() {
		fmt.Printf("  ✓ User saved with ID: %s\n", newUser.ID)
	}
	fmt.Printf("  Created at: %s\n", newUser.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()

	// 16. DoesntExist helper
	fmt.Println("16. Using DoesntExist helper...")
	doesntExist, err := orm.DoesntExist(ctx, orm.G[User](db).Where("email = ?", "nonexistent@example.com"))
	if err != nil {
		log.Fatalf("DoesntExist failed: %v", err)
	}
	fmt.Printf("  User with nonexistent email doesn't exist: %v\n", doesntExist)
	fmt.Println()

	fmt.Println("=== ORM Example Complete ===")
	fmt.Println("\nKey features demonstrated:")
	fmt.Println("  ✓ Model definition with UUID primary keys and soft deletes")
	fmt.Println("  ✓ Type-safe Generics API (orm.G[T])")
	fmt.Println("  ✓ Query scopes (WhereColumn, Search, BelongsTo, etc.)")
	fmt.Println("  ✓ CRUD operations (Create, Find, Update, Delete)")
	fmt.Println("  ✓ Pagination with orm.Paginate")
	fmt.Println("  ✓ Chunk processing with orm.Chunk")
	fmt.Println("  ✓ Soft deletes and Unscoped")
	fmt.Println("  ✓ Custom scopes")
	fmt.Println("  ✓ Helper functions (Exists, DoesntExist)")
	fmt.Println("  ✓ Model helper methods (IsNew, IsDeleted)")
	fmt.Println("  ✓ Context-aware queries")
}
