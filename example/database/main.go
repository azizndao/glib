// Package main demonstrates the database layer of the glib framework.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/azizndao/glib/config"
	"github.com/azizndao/glib/database"
	"github.com/azizndao/glib/foundation"
)

// User model example
type User struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name
func (User) TableName() string {
	return "users"
}

func main() {
	fmt.Println("=== glib Database Layer Example ===\n")

	// 1. Create Application
	fmt.Println("1. Setting up application...")
	app := foundation.New("/app")

	// Configure database
	cfg := config.New()
	cfg.Set("database.default", "sqlite")
	cfg.Set("database.connections.sqlite.driver", "sqlite")
	cfg.Set("database.connections.sqlite.database", "/tmp/glib_example.db")
	app.SetConfig(cfg)

	fmt.Println("   ✓ Application configured")
	fmt.Println()

	// 2. Register Database Provider
	fmt.Println("2. Registering database provider...")
	app.Register(&database.ServiceProvider{})

	if err := app.Bootstrap(); err != nil {
		log.Fatalf("Failed to bootstrap: %v", err)
	}
	fmt.Println("   ✓ Database provider registered")
	fmt.Println()

	// 3. Create Database Manager directly (bypassing container for now)
	fmt.Println("3. Creating database manager...")
	dbManager := database.NewManager(app.Config(), app.Logger())
	fmt.Println("   ✓ Database manager created")
	fmt.Println()

	// 4. Get Database Connection
	fmt.Println("4. Connecting to database...")
	db, err := dbManager.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("   ✓ Connected to SQLite database")
	fmt.Println()

	// 5. Auto Migrate
	fmt.Println("5. Running auto migration...")
	if err := db.AutoMigrate(&User{}); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}
	fmt.Println("   ✓ Users table created")
	fmt.Println()

	// 6. Create Records
	fmt.Println("6. Creating user records...")

	users := []User{
		{Name: "John Doe", Email: "john@example.com", Age: 30},
		{Name: "Jane Smith", Email: "jane@example.com", Age: 25},
		{Name: "Bob Johnson", Email: "bob@example.com", Age: 35},
	}

	result := db.Create(&users)
	if result.Error != nil {
		log.Fatalf("Failed to create users: %v", result.Error)
	}
	fmt.Printf("   ✓ Created %d users\n", len(users))
	fmt.Println()

	// 7. Query Records
	fmt.Println("7. Querying records...")

	// Find all users
	var allUsers []User
	db.Find(&allUsers)
	fmt.Printf("   Total users: %d\n", len(allUsers))

	// Find first user
	var firstUser User
	db.First(&firstUser)
	fmt.Printf("   First user: %s (%s)\n", firstUser.Name, firstUser.Email)

	// Find with WHERE clause
	var adults []User
	db.Where("age >= ?", 30).Find(&adults)
	fmt.Printf("   Adults (30+): %d\n", len(adults))
	fmt.Println()

	// 8. Update Records
	fmt.Println("8. Updating records...")

	db.Model(&User{}).Where("email = ?", "john@example.com").Update("age", 31)

	var updatedUser User
	db.Where("email = ?", "john@example.com").First(&updatedUser)
	fmt.Printf("   Updated John's age to: %d\n", updatedUser.Age)
	fmt.Println()

	// 9. Count Records
	fmt.Println("9. Counting records...")

	var count int64
	db.Model(&User{}).Count(&count)
	fmt.Printf("   Total users in database: %d\n", count)
	fmt.Println()

	// 10. Transactions
	fmt.Println("10. Demonstrating transactions...")

	err = db.Transaction(func(tx *database.Connection) error {
		// Create user in transaction
		newUser := User{Name: "Alice Wonder", Email: "alice@example.com", Age: 28}
		if result := tx.Create(&newUser); result.Error != nil {
			return result.Error
		}

		fmt.Println("   ✓ Created user in transaction")

		// Update another user in same transaction
		if result := tx.Model(&User{}).Where("email = ?", "jane@example.com").Update("age", 26); result.Error != nil {
			return result.Error
		}

		fmt.Println("   ✓ Updated user in transaction")

		return nil // Commit
	})

	if err != nil {
		log.Fatalf("Transaction failed: %v", err)
	}
	fmt.Println("   ✓ Transaction committed successfully")
	fmt.Println()

	// 11. Raw Queries
	fmt.Println("11. Executing raw queries...")

	var usersBySQL []User
	db.Raw("SELECT * FROM users WHERE age >= ? ORDER BY age DESC", 30).Scan(&usersBySQL)
	fmt.Printf("   Found %d users using raw SQL\n", len(usersBySQL))
	for _, u := range usersBySQL {
		fmt.Printf("      - %s (age %d)\n", u.Name, u.Age)
	}
	fmt.Println()

	// 12. Cleanup
	fmt.Println("12. Cleaning up...")

	// Delete all users
	db.Where("1 = 1").Delete(&User{})

	var finalCount int64
	db.Model(&User{}).Count(&finalCount)
	fmt.Printf("   Users remaining: %d\n", finalCount)

	// Close connections
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
	fmt.Println("   ✓ Database connections closed")
	fmt.Println()

	fmt.Println("=== Database Layer Example Complete ===")
}
