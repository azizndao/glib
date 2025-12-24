// Package main demonstrates a fullstack blog application with Glib framework.
// This example shows how to build a complete application with database, authentication,
// and relationships.
//
// What you'll learn:
// - Foundation module (Application, ServiceProviders)
// - Database integration with GORM
// - Model relationships (User -> Posts -> Comments)
// - JWT authentication
// - Middleware usage
// - Request validation
// - Pagination
// - Soft deletes
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"glib/example/fullstack/middleware"
	"glib/example/fullstack/models"

	"github.com/azizndao/glib"
	"github.com/azizndao/glib/common/config"
	"github.com/azizndao/glib/common/container"
	"github.com/azizndao/glib/common/errors"
	"github.com/azizndao/glib/database"
	"github.com/azizndao/glib/database/orm"
	"github.com/azizndao/glib/foundation"
	"github.com/azizndao/glib/validation"
	"github.com/google/uuid"
)

// =============================================================================
// Request/Response Types
// =============================================================================

type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

type CreatePostRequest struct {
	Title   string `json:"title" validate:"required,min=5,max=200"`
	Content string `json:"content" validate:"required,min=10"`
}

type UpdatePostRequest struct {
	Title     *string `json:"title" validate:"omitempty,min=5,max=200"`
	Content   *string `json:"content" validate:"omitempty,min=10"`
	Published *bool   `json:"published"`
}

type CreateCommentRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

// =============================================================================
// Handlers
// =============================================================================

// Auth handlers
func register(app *foundation.Application) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		var req RegisterRequest
		if err := c.ValidateBody(&req); err != nil {
			return err
		}

		dbManager, _ := container.Resolve[*database.Manager](app.Container())
		conn, _ := dbManager.DB()
		db := conn.DB()

		// Check if user exists
		var existingUser models.User
		if err := db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
			return errors.Conflict("Email already registered", nil)
		}

		// Hash password
		hash := sha256.Sum256([]byte(req.Password))
		hashedPassword := hex.EncodeToString(hash[:])

		// Create user
		user := models.User{
			Name:     req.Name,
			Email:    req.Email,
			Password: hashedPassword,
		}

		if err := db.Create(&user).Error; err != nil {
			return errors.InternalServerError("Failed to create user", err)
		}

		// Generate token
		token, err := middleware.CreateToken(user.ID, user.Email)
		if err != nil {
			return errors.InternalServerError("Failed to generate token", err)
		}

		return c.Status(201).JSON(LoginResponse{
			Token: token,
			User:  &user,
		})
	}
}

func login(app *foundation.Application) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		var req LoginRequest
		if err := c.ValidateBody(&req); err != nil {
			return err
		}

		dbManager, _ := container.Resolve[*database.Manager](app.Container())
		conn, _ := dbManager.DB()
		db := conn.DB()

		// Find user
		var user models.User
		if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
			return errors.Unauthorized("Invalid credentials", nil)
		}

		// Verify password
		hash := sha256.Sum256([]byte(req.Password))
		hashedPassword := hex.EncodeToString(hash[:])

		if user.Password != hashedPassword {
			return errors.Unauthorized("Invalid credentials", nil)
		}

		// Generate token
		token, err := middleware.CreateToken(user.ID, user.Email)
		if err != nil {
			return errors.InternalServerError("Failed to generate token", err)
		}

		return c.JSON(LoginResponse{
			Token: token,
			User:  &user,
		})
	}
}

// Post handlers
func listPosts(app *foundation.Application) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		dbManager, _ := container.Resolve[*database.Manager](app.Container())
		conn, _ := dbManager.DB()
		db := conn.DB()
		ctx := c.Request.Context()

		// Get query parameters
		page := 1
		perPage := 10

		// Build query
		chain := orm.G[models.Post](db).
			Where("published = ?", true).
			Order("created_at DESC")

		// Paginate
		paginator, err := orm.Paginate(ctx, chain, page, perPage)
		if err != nil {
			return errors.InternalServerError("Failed to fetch posts", err)
		}

		return c.JSON(paginator)
	}
}

func getPost(app *foundation.Application) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		idStr := c.PathValue("id")
		postID, err := uuid.Parse(idStr)
		if err != nil {
			return errors.BadRequest("Invalid post ID", err)
		}

		dbManager, _ := container.Resolve[*database.Manager](app.Container())
		conn, _ := dbManager.DB()
		db := conn.DB()
		_ = c.Request.Context()

		// Load post with user and comments (using standard GORM for Preload)
		var post models.Post
		err = db.Preload("User").Preload("Comments.User").Where("id = ?", postID).First(&post).Error
		if err != nil {
			return errors.NotFound("Post not found", err)
		}

		return c.JSON(post)
	}
}

func createPost(app *foundation.Application) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		var req CreatePostRequest
		if err := c.ValidateBody(&req); err != nil {
			return err
		}

		userID, err := middleware.GetUserID(c)
		if err != nil {
			return err
		}

		dbManager, _ := container.Resolve[*database.Manager](app.Container())
		conn, _ := dbManager.DB()
		db := conn.DB()

		post := models.Post{
			Title:   req.Title,
			Content: req.Content,
			UserID:  userID,
		}

		if err := db.Create(&post).Error; err != nil {
			return errors.InternalServerError("Failed to create post", err)
		}

		// Load user relation
		db.Preload("User").First(&post, "id = ?", post.ID)

		return c.Status(201).JSON(post)
	}
}

func updatePost(app *foundation.Application) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		idStr := c.PathValue("id")
		postID, err := uuid.Parse(idStr)
		if err != nil {
			return errors.BadRequest("Invalid post ID", err)
		}

		var req UpdatePostRequest
		if err := c.ValidateBody(&req); err != nil {
			return err
		}

		userID, err := middleware.GetUserID(c)
		if err != nil {
			return err
		}

		dbManager, _ := container.Resolve[*database.Manager](app.Container())
		conn, _ := dbManager.DB()
		db := conn.DB()

		// Find post
		var post models.Post
		if err := db.Where("id = ? AND user_id = ?", postID, userID).First(&post).Error; err != nil {
			return errors.NotFound("Post not found or unauthorized", err)
		}

		// Update fields
		updates := make(map[string]interface{})
		if req.Title != nil {
			updates["title"] = *req.Title
		}
		if req.Content != nil {
			updates["content"] = *req.Content
		}
		if req.Published != nil {
			updates["published"] = *req.Published
		}

		if err := db.Model(&post).Updates(updates).Error; err != nil {
			return errors.InternalServerError("Failed to update post", err)
		}

		// Reload post
		db.Preload("User").First(&post, "id = ?", post.ID)

		return c.JSON(post)
	}
}

func deletePost(app *foundation.Application) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		idStr := c.PathValue("id")
		postID, err := uuid.Parse(idStr)
		if err != nil {
			return errors.BadRequest("Invalid post ID", err)
		}

		userID, err := middleware.GetUserID(c)
		if err != nil {
			return err
		}

		dbManager, _ := container.Resolve[*database.Manager](app.Container())
		conn, _ := dbManager.DB()
		db := conn.DB()

		// Soft delete (only owner can delete)
		result := db.Where("id = ? AND user_id = ?", postID, userID).Delete(&models.Post{})
		if result.Error != nil {
			return errors.InternalServerError("Failed to delete post", result.Error)
		}

		if result.RowsAffected == 0 {
			return errors.NotFound("Post not found or unauthorized", nil)
		}

		return c.NoContent()
	}
}

// Comment handlers
func createComment(app *foundation.Application) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		idStr := c.PathValue("id")
		postID, err := uuid.Parse(idStr)
		if err != nil {
			return errors.BadRequest("Invalid post ID", err)
		}

		var req CreateCommentRequest
		if err := c.ValidateBody(&req); err != nil {
			return err
		}

		userID, err := middleware.GetUserID(c)
		if err != nil {
			return err
		}

		dbManager, _ := container.Resolve[*database.Manager](app.Container())
		conn, _ := dbManager.DB()
		db := conn.DB()

		// Verify post exists
		var post models.Post
		if err := db.Where("id = ?", postID).First(&post).Error; err != nil {
			return errors.NotFound("Post not found", err)
		}

		comment := models.Comment{
			Content: req.Content,
			PostID:  postID,
			UserID:  userID,
		}

		if err := db.Create(&comment).Error; err != nil {
			return errors.InternalServerError("Failed to create comment", err)
		}

		// Load relations
		db.Preload("User").First(&comment, "id = ?", comment.ID)

		return c.Status(201).JSON(comment)
	}
}

// =============================================================================
// Main
// =============================================================================

func main() {
	fmt.Println("🚀 Starting Fullstack Blog Application...")

	// 1. Create Foundation Application
	app := foundation.New(".")

	// 2. Configure Application
	cfg := config.New()
	cfg.Set("database.default", "sqlite")
	cfg.Set("database.connections.sqlite.driver", "sqlite")
	cfg.Set("database.connections.sqlite.database", "blog.db")
	app.SetConfig(cfg)

	// 3. Register Service Providers
	app.Register(&database.ServiceProvider{})

	// 4. Bootstrap Application (registers & boots all providers)
	if err := app.Bootstrap(); err != nil {
		log.Fatalf("Failed to bootstrap application: %v", err)
	}

	// 5. Run Migrations
	dbManager, _ := container.Resolve[*database.Manager](app.Container())
	conn, _ := dbManager.DB()
	if err := conn.AutoMigrate(&models.User{}, &models.Post{}, &models.Comment{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	fmt.Println("✓ Database migrated")

	// 6. Create HTTP Server
	validator := validation.New(validation.Config{
		DefaultLocale:     "en",
		UseJSONFieldNames: true,
	})

	server := glib.New(glib.Config{
		Validator: validator,
	})

	// 7. Setup Routes
	router := server.Router()

	// Public routes
	router.Post("/auth/register", register(app))
	router.Post("/auth/login", login(app))
	router.Get("/posts", listPosts(app))
	router.Get("/posts/{id}", getPost(app))

	// Protected routes (require authentication)
	router.Route("/api", func(api glib.Router) {
		api.Use(middleware.Auth)

		// Post management
		api.Post("/posts", createPost(app))
		api.Put("/posts/{id}", updatePost(app))
		api.Delete("/posts/{id}", deletePost(app))

		// Comments
		api.Post("/posts/{id}/comments", createComment(app))
	})

	// 8. Start Server
	fmt.Println("📍 Server endpoints:")
	fmt.Println("   POST /auth/register    - Create account")
	fmt.Println("   POST /auth/login       - Login")
	fmt.Println("   GET  /posts            - List posts (public)")
	fmt.Println("   GET  /posts/{id}       - Get post (public)")
	fmt.Println("   POST /api/posts        - Create post (auth)")
	fmt.Println("   PUT  /api/posts/{id}   - Update post (auth)")
	fmt.Println("   DELETE /api/posts/{id} - Delete post (auth)")
	fmt.Println("   POST /api/posts/{id}/comments - Add comment (auth)")
	fmt.Printf("\n📍 Listening on %s\n\n", server.Address())

	if err := server.ListenWithGracefulShutdown(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

	// 9. Graceful Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}
