// Package main demonstrates comprehensive usage of glib.Server with all features
// This example covers:
// - Server creation and configuration
// - All HTTP methods (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
// - Route parameters and query parameters
// - Request body parsing and validation with i18n
// - Response types (JSON, HTML, File, Redirect, NoContent)
// - Error handling
// - Route grouping and sub-routing
// - Custom middleware
// - Context methods (cookies, headers, IP, user agent)
// - File uploads
// - Timeout middleware
// - Chi middleware integration
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/azizndao/glib"
	"github.com/azizndao/glib/errors"
	"github.com/azizndao/glib/router"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/locales/es"
	"github.com/go-playground/locales/fr"
	est "github.com/go-playground/validator/v10/translations/es"
	frt "github.com/go-playground/validator/v10/translations/fr"
	"github.com/joho/godotenv"
)

// Request/Response Types
type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=100"`
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Age      int    `json:"age" validate:"required,gte=18,lte=120"`
}

type UpdateUserRequest struct {
	Name string `json:"name" validate:"omitempty,min=2,max=100"`
	Age  int    `json:"age" validate:"omitempty,gte=18,lte=120"`
}

type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

type CreateProductRequest struct {
	Name        string  `json:"name" validate:"required,min=3,max=100"`
	Description string  `json:"description" validate:"required,min=10,max=500"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	Stock       int     `json:"stock" validate:"required,gte=0"`
	SKU         string  `json:"sku" validate:"required,alphanum,len=8"`
}

func main() {
	// Load environment variables from .env file
	godotenv.Load()

	// Create server with multi-language validation support
	serverConfig := glib.Config{
		Locales: []glib.LocaleConfig{
			glib.Locale(fr.New(), frt.RegisterDefaultTranslations),
			glib.Locale(es.New(), est.RegisterDefaultTranslations),
		},
	}

	server := glib.New(serverConfig)
	r := server.Router()

	// ====================
	// GLOBAL MIDDLEWARE
	// ====================
	// Use custom middleware globally
	r.Use(loggingMiddleware)

	// Use Chi middleware directly
	r.UseChiMiddleware(chimiddleware.Heartbeat("/ping"))

	// ====================
	// BASIC ROUTES - All HTTP Methods
	// ====================
	r.Get("/", homeHandler)
	r.Get("/health", healthCheckHandler)

	// ====================
	// USER ROUTES - CRUD Operations
	// ====================
	r.Route("/users", func(users router.Router) {
		users.Get("/", listUsersHandler)            // GET /users
		users.Post("/", createUserHandler)          // POST /users
		users.Get("/{id}", getUserHandler)          // GET /users/{id}
		users.Put("/{id}", updateUserHandler)       // PUT /users/{id}
		users.Patch("/{id}", patchUserHandler)      // PATCH /users/{id}
		users.Delete("/{id}", deleteUserHandler)    // DELETE /users/{id}
		users.Head("/{id}", checkUserExistsHandler) // HEAD /users/{id}
		users.Options("/{id}", userOptionsHandler)  // OPTIONS /users/{id}
	})

	// ====================
	// PRODUCT ROUTES with Route Group
	// ====================
	productsGroup := r.Route("/products", func(products router.Router) {
		// Apply middleware only to product routes
		products.Use(authMiddleware)

		products.Get("/", listProductsHandler)
		products.Post("/", createProductHandler)
		products.Get("/{id}", getProductHandler)
		products.Put("/{id}", updateProductHandler)
		products.Delete("/{id}", deleteProductHandler)
	})

	// Add more routes to the products group after creation
	productsGroup.Get("/search", searchProductsHandler)

	// ====================
	// CONTEXT FEATURES
	// ====================
	r.Route("/context", func(ctx router.Router) {
		ctx.Get("/request-info", requestInfoHandler)
		ctx.Get("/headers", headersHandler)
		ctx.Get("/cookies", cookiesHandler)
		ctx.Post("/set-cookie", setCookieHandler)
		ctx.Get("/clear-cookie", clearCookieHandler)
		ctx.Get("/query-params", queryParamsHandler)
		ctx.Get("/ip-info", ipInfoHandler)
	})

	// ====================
	// FILE OPERATIONS
	// ====================
	r.Route("/files", func(files router.Router) {
		files.Post("/upload", fileUploadHandler)
		files.Get("/download/{filename}", fileDownloadHandler)
		files.Get("/serve", serveFileHandler)
	})

	// ====================
	// RESPONSE TYPES
	// ====================
	r.Route("/responses", func(resp router.Router) {
		resp.Get("/json", jsonResponseHandler)
		resp.Get("/text", textResponseHandler)
		resp.Get("/html", htmlResponseHandler)
		resp.Get("/redirect", redirectHandler)
		resp.Delete("/no-content", noContentHandler)
		resp.Get("/custom-status", customStatusHandler)
	})

	// ====================
	// ERROR HANDLING
	// ====================
	r.Route("/errors", func(errs router.Router) {
		errs.Get("/bad-request", badRequestHandler)
		errs.Get("/unauthorized", unauthorizedHandler)
		errs.Get("/forbidden", forbiddenHandler)
		errs.Get("/not-found", notFoundErrorHandler)
		errs.Get("/conflict", conflictHandler)
		errs.Get("/internal", internalErrorHandler)
		errs.Get("/custom", customErrorHandler)
	})

	// ====================
	// MIDDLEWARE EXAMPLES
	// ====================
	r.Route("/middleware", func(mw router.Router) {
		// Route with inline middleware using With()
		mw.With(timingMiddleware).Get("/timed", timedHandler)

		// Slow endpoint examples (without timeout middleware as it's not implemented)
		slowGroup := mw.Route("/timeout", func(slow router.Router) {
			slow.Get("/fast", fastHandler)
			slow.Get("/slow", slowHandler)
		})
		_ = slowGroup

		// Request ID from Chi middleware
		mw.Get("/request-id", requestIDHandler)
	})

	// ====================
	// ADVANCED FEATURES
	// ====================
	r.Route("/advanced", func(adv router.Router) {
		// Content negotiation
		adv.Get("/negotiate", contentNegotiationHandler)

		// Context values
		adv.Use(setContextValueMiddleware)
		adv.Get("/context-value", getContextValueHandler)

		// Multiple middleware chain
		adv.With(authMiddleware, loggingMiddleware, timingMiddleware).
			Get("/protected", protectedHandler)
	})

	// ====================
	// VALIDATION EXAMPLES
	// ====================
	r.Route("/validation", func(val router.Router) {
		val.Post("/user", validateUserHandler)
		val.Post("/product", validateProductHandler)
		val.Post("/french", validateFrenchHandler)   // Test with Accept-Language: fr
		val.Post("/spanish", validateSpanishHandler) // Test with Accept-Language: es
	})

	// ====================
	// NESTED SUB-ROUTING
	// ====================
	r.Route("/api", func(api router.Router) {
		api.Get("/version", func(c *router.Ctx) error {
			return c.JSON(map[string]string{"version": "1.0.0"})
		})

		// Nested v1 routes
		api.Route("/v1", func(v1 router.Router) {
			v1.Get("/status", func(c *router.Ctx) error {
				return c.JSON(map[string]string{"api": "v1", "status": "active"})
			})

			// Deeply nested admin routes
			v1.Route("/admin", func(admin router.Router) {
				admin.Use(authMiddleware)
				admin.Get("/stats", adminStatsHandler)
				admin.Get("/users", adminUsersHandler)
			})
		})

		// v2 routes
		api.Route("/v2", func(v2 router.Router) {
			v2.Get("/status", func(c *router.Ctx) error {
				return c.JSON(map[string]string{"api": "v2", "status": "beta"})
			})
		})
	})

	// ====================
	// CUSTOM ERROR HANDLERS
	// ====================
	r.NotFound(func(c *router.Ctx) error {
		return errors.NotFound(map[string]string{
			"message": "Custom 404: Route not found",
			"path":    c.Path(),
			"method":  c.Method(),
		}, nil)
	})

	r.MethodNotAllowed(func(c *router.Ctx) error {
		return errors.MethodNotAllowed(map[string]string{
			"message": "Custom 405: Method not allowed",
			"path":    c.Path(),
			"method":  c.Method(),
		}, nil)
	})

	// ====================
	// START SERVER
	// ====================
	server.Logger().Info("Server starting with comprehensive examples")
	server.Logger().Info(fmt.Sprintf("Visit http://%s for examples", server.Address()))

	if err := server.ListenWithGracefulShutdown(); err != nil {
		log.Fatal(err)
	}
}

// ====================
// HANDLERS
// ====================

func homeHandler(c *router.Ctx) error {
	return c.JSON(map[string]interface{}{
		"message": "Welcome to glib comprehensive example",
		"endpoints": map[string]string{
			"health":     "/health",
			"users":      "/users",
			"products":   "/products",
			"context":    "/context/*",
			"files":      "/files/*",
			"responses":  "/responses/*",
			"errors":     "/errors/*",
			"middleware": "/middleware/*",
			"advanced":   "/advanced/*",
			"validation": "/validation/*",
			"api":        "/api/*",
		},
	})
}

func healthCheckHandler(c *router.Ctx) error {
	return c.JSON(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"uptime":    "running",
	})
}

// User Handlers
func listUsersHandler(c *router.Ctx) error {
	// Query parameters with defaults
	page := c.QueryIntDefault("page", 1)
	limit := c.QueryIntDefault("limit", 10)
	search := c.QueryDefault("search", "")
	active := c.QueryBool("active")

	users := []User{
		{ID: 1, Email: "user1@example.com", Name: "John Doe", CreatedAt: time.Now()},
		{ID: 2, Email: "user2@example.com", Name: "Jane Smith", CreatedAt: time.Now()},
	}

	return c.JSON(map[string]interface{}{
		"users":  users,
		"page":   page,
		"limit":  limit,
		"search": search,
		"active": active,
		"total":  len(users),
	})
}

func getUserHandler(c *router.Ctx) error {
	id := c.PathValue("id")

	user := User{
		ID:        1,
		Email:     "user@example.com",
		Name:      "John Doe",
		CreatedAt: time.Now(),
	}

	return c.JSON(map[string]interface{}{
		"user": user,
		"id":   id,
	})
}

func createUserHandler(c *router.Ctx) error {
	var req CreateUserRequest

	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	user := User{
		ID:        1,
		Email:     req.Email,
		Name:      req.Name,
		CreatedAt: time.Now(),
	}

	return c.Status(201).JSON(map[string]interface{}{
		"message": "User created successfully",
		"user":    user,
	})
}

func updateUserHandler(c *router.Ctx) error {
	id := c.PathValue("id")
	var req UpdateUserRequest

	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	return c.JSON(map[string]interface{}{
		"message": "User updated successfully",
		"id":      id,
		"data":    req,
	})
}

func patchUserHandler(c *router.Ctx) error {
	id := c.PathValue("id")
	var req UpdateUserRequest

	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	return c.JSON(map[string]interface{}{
		"message": "User partially updated",
		"id":      id,
		"data":    req,
	})
}

func deleteUserHandler(c *router.Ctx) error {
	id := c.PathValue("id")

	return c.JSON(map[string]interface{}{
		"message": "User deleted successfully",
		"id":      id,
	})
}

func checkUserExistsHandler(c *router.Ctx) error {
	id := c.PathValue("id")
	c.Logger().Info("HEAD request for user", "id", id)
	// HEAD requests should not return a body
	return c.NoContent()
}

func userOptionsHandler(c *router.Ctx) error {
	c.Set("Allow", "GET, PUT, PATCH, DELETE, HEAD, OPTIONS")
	return c.NoContent()
}

// Product Handlers
func listProductsHandler(c *router.Ctx) error {
	products := []Product{
		{ID: 1, Name: "Product 1", Price: 29.99, Stock: 100},
		{ID: 2, Name: "Product 2", Price: 49.99, Stock: 50},
	}

	return c.JSON(map[string]interface{}{
		"products": products,
		"total":    len(products),
	})
}

func getProductHandler(c *router.Ctx) error {
	id := c.PathValue("id")

	product := Product{
		ID:          1,
		Name:        "Sample Product",
		Description: "A great product",
		Price:       29.99,
		Stock:       100,
	}

	return c.JSON(map[string]interface{}{
		"product": product,
		"id":      id,
	})
}

func createProductHandler(c *router.Ctx) error {
	var req CreateProductRequest

	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	product := Product{
		ID:          1,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
	}

	return c.Status(201).JSON(map[string]interface{}{
		"message": "Product created successfully",
		"product": product,
	})
}

func updateProductHandler(c *router.Ctx) error {
	id := c.PathValue("id")
	var req CreateProductRequest

	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	return c.JSON(map[string]interface{}{
		"message": "Product updated successfully",
		"id":      id,
		"data":    req,
	})
}

func deleteProductHandler(c *router.Ctx) error {
	id := c.PathValue("id")

	return c.JSON(map[string]interface{}{
		"message": "Product deleted successfully",
		"id":      id,
	})
}

func searchProductsHandler(c *router.Ctx) error {
	query := c.Query("q")
	minPrice := c.QueryFloatDefault("min_price", 0)
	maxPrice := c.QueryFloatDefault("max_price", 1000)

	return c.JSON(map[string]interface{}{
		"query":     query,
		"min_price": minPrice,
		"max_price": maxPrice,
		"results":   []Product{},
	})
}

// Context Handlers
func requestInfoHandler(c *router.Ctx) error {
	return c.JSON(map[string]interface{}{
		"method":       c.Method(),
		"path":         c.Path(),
		"host":         c.Host(),
		"scheme":       c.Scheme(),
		"base_url":     c.BaseURL(),
		"is_secure":    c.IsSecure(),
		"ip":           c.IP(),
		"user_agent":   c.UserAgent(),
		"content_type": c.ContentType(),
	})
}

func headersHandler(c *router.Ctx) error {
	headers := c.GetHeaders()

	return c.JSON(map[string]interface{}{
		"headers":       headers,
		"authorization": c.Authorization(),
		"accept":        c.Get("Accept"),
	})
}

func cookiesHandler(c *router.Ctx) error {
	sessionCookie, err := c.GetCookie("session")

	cookies := map[string]interface{}{
		"session_exists": err == nil,
	}

	if err == nil {
		cookies["session_value"] = sessionCookie.Value
	}

	return c.JSON(cookies)
}

func setCookieHandler(c *router.Ctx) error {
	cookie := &http.Cookie{
		Name:     "session",
		Value:    "abc123xyz",
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		Secure:   c.IsSecure(),
		SameSite: http.SameSiteLaxMode,
	}

	c.SetCookie(cookie)

	return c.JSON(map[string]string{
		"message": "Cookie set successfully",
	})
}

func clearCookieHandler(c *router.Ctx) error {
	c.ClearCookie("session")

	return c.JSON(map[string]string{
		"message": "Cookie cleared successfully",
	})
}

func queryParamsHandler(c *router.Ctx) error {
	// Demonstrate various query parameter methods
	name := c.Query("name")
	age, _ := c.QueryInt("age")
	score, _ := c.QueryFloat("score")
	active := c.QueryBool("active")
	tags := c.QueryAll("tags") // Multiple values: ?tags=go&tags=web&tags=api

	return c.JSON(map[string]interface{}{
		"name":   name,
		"age":    age,
		"score":  score,
		"active": active,
		"tags":   tags,
	})
}

func ipInfoHandler(c *router.Ctx) error {
	return c.JSON(map[string]interface{}{
		"ip":              c.IP(),
		"remote_addr":     c.Request.RemoteAddr,
		"x_forwarded_for": c.Get("X-Forwarded-For"),
		"x_real_ip":       c.Get("X-Real-IP"),
	})
}

// File Handlers
func fileUploadHandler(c *router.Ctx) error {
	file, header, err := c.FormFile("file")
	if err != nil {
		return errors.BadRequest("Failed to read file", err)
	}
	defer file.Close()

	return c.JSON(map[string]interface{}{
		"message":  "File uploaded successfully",
		"filename": header.Filename,
		"size":     header.Size,
		"headers":  header.Header,
	})
}

func fileDownloadHandler(c *router.Ctx) error {
	filename := c.PathValue("filename")

	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Set("Content-Type", "application/octet-stream")

	return c.SendString(fmt.Sprintf("Content of %s", filename))
}

func serveFileHandler(c *router.Ctx) error {
	// In a real app, you would serve an actual file
	// return c.File("/path/to/file.pdf")

	return c.JSON(map[string]string{
		"message": "Use c.File(path) to serve actual files",
	})
}

// Response Type Handlers
func jsonResponseHandler(c *router.Ctx) error {
	return c.JSON(map[string]interface{}{
		"type":    "json",
		"message": "This is a JSON response",
		"data": map[string]interface{}{
			"nested": "value",
			"array":  []int{1, 2, 3},
		},
	})
}

func textResponseHandler(c *router.Ctx) error {
	return c.SendString("This is a plain text response")
}

func htmlResponseHandler(c *router.Ctx) error {
	html := []byte(`
		<!DOCTYPE html>
		<html>
		<head><title>glib Example</title></head>
		<body>
			<h1>Hello from glib!</h1>
			<p>This is an HTML response.</p>
		</body>
		</html>
	`)

	return c.HTML(html)
}

func redirectHandler(c *router.Ctx) error {
	return c.Redirect(302, "/")
}

func noContentHandler(c *router.Ctx) error {
	return c.NoContent()
}

func customStatusHandler(c *router.Ctx) error {
	return c.Status(418).JSON(map[string]string{
		"message": "I'm a teapot",
		"code":    "418",
	})
}

// Error Handlers
func badRequestHandler(c *router.Ctx) error {
	return errors.BadRequest(map[string]string{
		"message": "This is a bad request error",
		"field":   "example",
		"issue":   "invalid format",
	}, nil)
}

func unauthorizedHandler(c *router.Ctx) error {
	return errors.Unauthorized("You are not authenticated", nil)
}

func forbiddenHandler(c *router.Ctx) error {
	return errors.Forbidden("You don't have permission to access this resource", nil)
}

func notFoundErrorHandler(c *router.Ctx) error {
	return errors.NotFound(map[string]string{
		"message":  "Resource not found",
		"resource": "user",
		"id":       "123",
	}, nil)
}

func conflictHandler(c *router.Ctx) error {
	return errors.Conflict(map[string]string{
		"message": "Resource already exists",
		"email":   "user@example.com",
	}, nil)
}

func internalErrorHandler(c *router.Ctx) error {
	return errors.InternalServerError("Something went wrong", fmt.Errorf("database connection failed"))
}

func customErrorHandler(c *router.Ctx) error {
	return errors.NewApi(422, map[string]interface{}{
		"message": "Unprocessable Entity: Validation failed",
		"errors":  []string{"Invalid data format", "Missing required fields"},
	}, nil)
}

// Middleware Handlers
func timedHandler(c *router.Ctx) error {
	return c.JSON(map[string]string{
		"message": "This request was timed by middleware",
	})
}

func fastHandler(c *router.Ctx) error {
	time.Sleep(500 * time.Millisecond)
	return c.JSON(map[string]string{"message": "Fast response"})
}

func slowHandler(c *router.Ctx) error {
	time.Sleep(3 * time.Second)
	return c.JSON(map[string]string{"message": "Slow response (not actually timing out without timeout middleware)"})
}

func requestIDHandler(c *router.Ctx) error {
	// Request ID would be available from Chi middleware if configured
	requestID := c.Get("X-Request-Id")
	if requestID == "" {
		requestID = "not-configured"
	}
	return c.JSON(map[string]string{
		"request_id": requestID,
		"message":    "Request ID can be added by Chi middleware",
	})
}

// Advanced Handlers
func contentNegotiationHandler(c *router.Ctx) error {
	data := map[string]string{
		"message": "Content negotiation example",
		"format":  "auto-detected",
	}

	if c.AcceptsJSON() {
		return c.JSON(data)
	} else if c.AcceptsHTML() {
		html := []byte("<h1>Content Negotiation</h1><p>Detected HTML preference</p>")
		return c.HTML(html)
	}

	return c.SendString("Plain text response")
}

func getContextValueHandler(c *router.Ctx) error {
	// Retrieve value set by middleware
	userID := c.GetValue("user_id")

	return c.JSON(map[string]interface{}{
		"user_id": userID,
		"message": "Value retrieved from context",
	})
}

func protectedHandler(c *router.Ctx) error {
	return c.JSON(map[string]string{
		"message": "This route is protected by multiple middleware",
		"user_id": c.GetValue("user_id").(string),
	})
}

// Validation Handlers
func validateUserHandler(c *router.Ctx) error {
	var req CreateUserRequest

	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	return c.Status(201).JSON(map[string]interface{}{
		"message": "User validation passed",
		"data":    req,
	})
}

func validateProductHandler(c *router.Ctx) error {
	var req CreateProductRequest

	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	return c.Status(201).JSON(map[string]interface{}{
		"message": "Product validation passed",
		"data":    req,
	})
}

func validateFrenchHandler(c *router.Ctx) error {
	// Send request with header: Accept-Language: fr
	var req CreateUserRequest

	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	return c.JSON(map[string]interface{}{
		"message": "Validation avec locale française",
		"data":    req,
	})
}

func validateSpanishHandler(c *router.Ctx) error {
	// Send request with header: Accept-Language: es
	var req CreateUserRequest

	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	return c.JSON(map[string]interface{}{
		"message": "Validación con configuración regional española",
		"data":    req,
	})
}

// Admin Handlers
func adminStatsHandler(c *router.Ctx) error {
	return c.JSON(map[string]interface{}{
		"total_users":    1234,
		"total_products": 567,
		"total_orders":   890,
	})
}

func adminUsersHandler(c *router.Ctx) error {
	return c.JSON(map[string]interface{}{
		"message": "Admin users list",
		"users":   []string{"admin1", "admin2"},
	})
}

// ====================
// MIDDLEWARE
// ====================

func loggingMiddleware(next router.HandleFunc) router.HandleFunc {
	return func(c *router.Ctx) error {
		start := time.Now()

		c.Logger().Info("Request started",
			"method", c.Method(),
			"path", c.Path(),
			"ip", c.IP(),
		)

		err := next(c)

		duration := time.Since(start)
		c.Logger().Info("Request completed",
			"method", c.Method(),
			"path", c.Path(),
			"duration", duration.String(),
		)

		return err
	}
}

func authMiddleware(next router.HandleFunc) router.HandleFunc {
	return func(c *router.Ctx) error {
		authHeader := c.Authorization()

		if authHeader == "" {
			return errors.Unauthorized("Missing authorization header", nil)
		}

		// In a real app, validate the token here
		// For demo purposes, we'll just check if it exists
		if authHeader != "Bearer valid-token" {
			return errors.Unauthorized("Invalid authorization token", nil)
		}

		// Set user context
		c.SetValue("user_id", "12345")

		return next(c)
	}
}

func timingMiddleware(next router.HandleFunc) router.HandleFunc {
	return func(c *router.Ctx) error {
		start := time.Now()

		err := next(c)

		duration := time.Since(start)
		c.Set("X-Response-Time", duration.String())

		return err
	}
}

func setContextValueMiddleware(next router.HandleFunc) router.HandleFunc {
	return func(c *router.Ctx) error {
		c.SetValue("user_id", "demo-user-123")
		c.SetValue("request_time", time.Now())

		return next(c)
	}
}
