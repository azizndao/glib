package services

import (
	"fmt"
	"glib/demo/models"
	"log"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// @Provider singleton
func NewDatabase() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("demo.db"), &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// Auto migrate schemas
	if err := db.AutoMigrate(
		&models.User{},
		&models.Post{},
		&models.Comment{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database connected and migrated successfully")

	// Seed initial data if empty
	seedDatabase(db)

	return db, nil
}

func seedDatabase(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Count(&count)
	if count > 0 {
		return // Already seeded
	}

	log.Println("Seeding database with initial data...")

	// Create users with realistic UUIDs
	users := []models.User{
		{
			ID:        uuid.MustParse("a1b2c3d4-e5f6-4a5b-8c9d-0e1f2a3b4c5d"),
			Email:     "john@example.com",
			Username:  "john_doe",
			FirstName: "John",
			LastName:  "Doe",
			Bio:       "Software engineer passionate about Go and web development",
			Active:    true,
		},
		{
			ID:        uuid.MustParse("b2c3d4e5-f6a7-4b5c-9d0e-1f2a3b4c5d6e"),
			Email:     "jane@example.com",
			Username:  "jane_smith",
			FirstName: "Jane",
			LastName:  "Smith",
			Bio:       "Full-stack developer and tech blogger",
			Active:    true,
		},
		{
			ID:        uuid.MustParse("c3d4e5f6-a7b8-4c5d-0e1f-2a3b4c5d6e7f"),
			Email:     "bob@example.com",
			Username:  "bob_wilson",
			FirstName: "Bob",
			LastName:  "Wilson",
			Bio:       "DevOps engineer and cloud enthusiast",
			Active:    true,
		},
	}

	for i := range users {
		if err := db.Create(&users[i]).Error; err != nil {
			log.Printf("Error seeding user: %v", err)
		}
	}

	// Create posts
	posts := []models.Post{
		{
			ID:        uuid.MustParse("d4e5f6a7-b8c9-4d5e-1f2a-3b4c5d6e7f8a"),
			Title:     "Getting Started with Glib Framework",
			Body:      "Glib is a code-generation-first web framework for Go that uses annotations to generate HTTP routing, dependency injection, and request/response handling code.",
			Slug:      "getting-started-with-glib",
			Published: true,
			AuthorID:  users[0].ID,
			Tags:      "golang,web,framework",
		},
		{
			ID:        uuid.MustParse("e5f6a7b8-c9d0-4e5f-2a3b-4c5d6e7f8a9b"),
			Title:     "Understanding GORM Relationships",
			Body:      "GORM provides powerful features for handling database relationships including has-one, has-many, many-to-many, and polymorphic associations.",
			Slug:      "understanding-gorm-relationships",
			Published: true,
			AuthorID:  users[1].ID,
			Tags:      "golang,database,gorm",
		},
		{
			ID:        uuid.MustParse("f6a7b8c9-d0e1-4f5a-3b4c-5d6e7f8a9b0c"),
			Title:     "Building RESTful APIs with Go",
			Body:      "Learn how to build production-ready RESTful APIs using Go's standard library and modern frameworks with proper error handling and validation.",
			Slug:      "building-restful-apis-go",
			Published: true,
			AuthorID:  users[0].ID,
			Tags:      "golang,api,rest",
		},
		{
			ID:        uuid.MustParse("a7b8c9d0-e1f2-4a5b-4c5d-6e7f8a9b0c1d"),
			Title:     "Docker and Kubernetes for Go Applications",
			Body:      "A comprehensive guide to containerizing Go applications and deploying them to Kubernetes clusters with best practices and real-world examples.",
			Slug:      "docker-kubernetes-go",
			Published: true,
			AuthorID:  users[2].ID,
			Tags:      "golang,docker,kubernetes,devops",
		},
		{
			ID:        uuid.MustParse("b8c9d0e1-f2a3-4b5c-5d6e-7f8a9b0c1d2e"),
			Title:     "Advanced Go Concurrency Patterns",
			Body:      "Explore advanced concurrency patterns in Go including worker pools, pipelines, fan-out/fan-in, and context-based cancellation.",
			Slug:      "advanced-go-concurrency",
			Published: false,
			AuthorID:  users[1].ID,
			Tags:      "golang,concurrency,patterns",
		},
	}

	for i := range posts {
		if err := db.Create(&posts[i]).Error; err != nil {
			log.Printf("Error seeding post: %v", err)
		}
	}

	// Create comments
	comments := []models.Comment{
		{
			ID:      uuid.MustParse("c9d0e1f2-a3b4-4c5d-6e7f-8a9b0c1d2e3f"),
			PostID:  posts[0].ID,
			UserID:  users[1].ID,
			Content: "Great introduction to Glib! Can't wait to try it in my next project.",
		},
		{
			ID:      uuid.MustParse("d0e1f2a3-b4c5-4d6e-7f8a-9b0c1d2e3f4a"),
			PostID:  posts[0].ID,
			UserID:  users[2].ID,
			Content: "This looks really promising. How does it compare to Gin or Echo?",
		},
		{
			ID:      uuid.MustParse("e1f2a3b4-c5d6-4e7f-8a9b-0c1d2e3f4a5b"),
			PostID:  posts[1].ID,
			UserID:  users[0].ID,
			Content: "Very helpful explanation. The examples really clarify the concepts.",
		},
		{
			ID:      uuid.MustParse("f2a3b4c5-d6e7-4f8a-9b0c-1d2e3f4a5b6c"),
			PostID:  posts[2].ID,
			UserID:  users[1].ID,
			Content: "Excellent guide! Saved me hours of debugging my API.",
		},
		{
			ID:      uuid.MustParse("a3b4c5d6-e7f8-4a9b-0c1d-2e3f4a5b6c7d"),
			PostID:  posts[3].ID,
			UserID:  users[0].ID,
			Content: "Just what I needed for our Kubernetes migration. Thanks!",
		},
	}

	for i := range comments {
		if err := db.Create(&comments[i]).Error; err != nil {
			log.Printf("Error seeding comment: %v", err)
		}
	}

	log.Println("Database seeded successfully!")
}
