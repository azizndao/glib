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
	"log"
	"time"

	"glib/example/fullstack/controllers"
	"glib/example/fullstack/middleware"
	"glib/example/fullstack/models"

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

	// Create controllers once - dependencies resolved at startup, not per-request
	authCtrl := controllers.NewAuthController(app)
	postCtrl := controllers.NewPostController(app)

	// Public routes
	router.Post("/auth/register", authCtrl.Register)
	router.Post("/auth/login", authCtrl.Login)
	router.Get("/posts", postCtrl.Index)
	router.Get("/posts/{id}", postCtrl.Show)

	// Protected routes (require authentication)
	router.Route("/api", func(api glib.Router) {
		api.Use(middleware.Auth)

		// Option 1: Manual route registration (explicit, clear)
		api.Post("/posts", postCtrl.Store)
		api.Put("/posts/{id}", postCtrl.Update)
		api.Patch("/posts/{id}", postCtrl.Update)
		api.Delete("/posts/{id}", postCtrl.Destroy)
		api.Post("/posts/{id}/comments", postCtrl.AddComment)

		// Option 2: Automatic resource routing (Laravel-style)
		// Uncomment to use APIResource instead of manual routes above:
		// api.APIResource("posts", func(app *foundation.Application) glib.Controller {
		//     return controllers.NewPostController(app)
		// }, glib.ResourceOptions{
		//     Except: []string{"Index", "Show"}, // Public routes already registered
		// })
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
