// Package main demonstrates a fullstack blog application with Glib framework.
// This example shows how to build a complete application with database, authentication,
// and relationships using Laravel-style controllers.
//
// What you'll learn:
// - Foundation module (Application, ServiceProviders)
// - Controller pattern with dependency injection
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
	"fmt"
	"glib/example/fullstack/controllers"
	"glib/example/fullstack/middleware"
	"glib/example/fullstack/models"
	"log"
	"time"

	"github.com/azizndao/glib"
	"github.com/azizndao/glib/common/config"
	"github.com/azizndao/glib/common/container"
	"github.com/azizndao/glib/database"
	"github.com/azizndao/glib/foundation"
	"github.com/azizndao/glib/validation"
)

func main() {
	fmt.Println("🚀 Starting Fullstack Blog Application with Controllers...")

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

	// 7. Set Application for Controller Dependency Injection
	server.SetApplicationDirect(app)
	fmt.Println("✓ Application configured for controllers")

	// 8. Setup Routes with Controllers
	router := server.Router()

	// Auth routes (public)
	authCtrl := controllers.NewAuth(app)
	router.Post("/auth/register", authCtrl.Register)
	router.Post("/auth/login", authCtrl.Login)

	// Public post routes (no authentication)
	postCtrl := controllers.NewPost(app)
	router.Get("/posts", postCtrl.Index)
	router.Get("/posts/{id}", postCtrl.Show)

	// Protected routes (require authentication)
	router.Route("/api", func(api glib.Router) {
		api.Use(middleware.Auth)

		// Register remaining post routes (Store, Update, Destroy)
		api.APIResource("/posts", postCtrl, glib.ResourceOptions{
			Except: []string{"Index", "Show"}, // Already registered as public
		})
	})

	// 9. Print Routes
	fmt.Println("📍 Server endpoints:")
	fmt.Println("   POST /auth/register         - Create account")
	fmt.Println("   POST /auth/login            - Login")
	fmt.Println("   GET  /posts                 - List posts (public)")
	fmt.Println("   GET  /posts/{id}            - Get post (public)")
	fmt.Println("   POST /api/posts             - Create post (auth) [Controller]")
	fmt.Println("   PUT  /api/posts/{id}        - Update post (auth) [Controller]")
	fmt.Println("   PATCH /api/posts/{id}       - Update post (auth) [Controller]")
	fmt.Println("   DELETE /api/posts/{id}      - Delete post (auth) [Controller]")
	fmt.Println("   POST /api/posts/{id}/comments - Add comment (auth)")
	fmt.Printf("\n📍 Listening on %s\n", server.Address())
	fmt.Println("✓ Controllers loaded with dependency injection")
	fmt.Println()

	// 10. Start Server
	if err := server.ListenWithGracefulShutdown(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

	// 11. Graceful Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}
