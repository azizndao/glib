package glib

// Controller is the base interface for all controllers.
// Controllers are instantiated via constructor functions that receive the application instance.
type Controller any

// ResourceController provides standard RESTful CRUD operations.
// Implements the full resource pattern with all 7 methods.
//
// Example:
//
//	type PostController struct {
//	    db *gorm.DB
//	}
//
//	// Constructor function returns the interface type for compatibility
//	func NewPostController(app *foundation.Application) glib.ResourceController {
//	    dbManager, _ := container.Resolve[*database.Manager](app.Container())
//	    conn, _ := dbManager.DB()
//	    return &PostController{db: conn.DB()}
//	}
//
//	func (ctrl *PostController) Index(c *glib.Ctx) error {
//	    var posts []models.Post
//	    ctrl.db.Find(&posts)
//	    return c.JSON(posts)
//	}
//	// ... implement other 6 methods: Create, Store, Show, Edit, Update, Destroy
//
// Usage:
//
//	router.Resource("posts", controllers.NewPostController)
type ResourceController interface {
	// Index lists all resources
	// GET /resource
	Index(c *Ctx) error

	// Create shows form for creating new resource (optional for APIs)
	// GET /resource/create
	Create(c *Ctx) error

	// Store creates a new resource
	// POST /resource
	Store(c *Ctx) error

	// Show displays a specific resource
	// GET /resource/{id}
	Show(c *Ctx) error

	// Edit shows form for editing resource (optional for APIs)
	// GET /resource/{id}/edit
	Edit(c *Ctx) error

	// Update modifies an existing resource
	// PUT/PATCH /resource/{id}
	Update(c *Ctx) error

	// Destroy deletes a resource
	// DELETE /resource/{id}
	Destroy(c *Ctx) error
}

// APIResourceController provides RESTful CRUD operations without form methods.
// This is ideal for API-only applications that don't need Create/Edit form views.
//
// Example:
//
//	type UserController struct {
//	    db *gorm.DB
//	}
//
//	// Constructor function returns the interface type for compatibility
//	func NewUserController(app *foundation.Application) glib.APIResourceController {
//	    dbManager, _ := container.Resolve[*database.Manager](app.Container())
//	    conn, _ := dbManager.DB()
//	    return &UserController{db: conn.DB()}
//	}
//
//	func (ctrl *UserController) Index(c *glib.Ctx) error {
//	    var users []models.User
//	    ctrl.db.Find(&users)
//	    return c.JSON(users)
//	}
//	// ... implement other 4 methods: Store, Show, Update, Destroy
//
// Usage:
//
//	router.APIResource("users", controllers.NewUserController)
type APIResourceController interface {
	// Index lists all resources
	// GET /resource
	Index(c *Ctx) error

	// Store creates a new resource
	// POST /resource
	Store(c *Ctx) error

	// Show displays a specific resource
	// GET /resource/{id}
	Show(c *Ctx) error

	// Update modifies an existing resource
	// PUT/PATCH /resource/{id}
	Update(c *Ctx) error

	// Destroy deletes a resource
	// DELETE /resource/{id}
	Destroy(c *Ctx) error
}

// InvokableController represents a single-action controller.
// Use this for controllers that perform only one specific action.
//
// Example:
//
//	type PublishPostController struct {
//	    db *gorm.DB
//	}
//
//	// Constructor function returns the interface type for compatibility
//	func NewPublishPostController(app *foundation.Application) glib.InvokableController {
//	    dbManager, _ := container.Resolve[*database.Manager](app.Container())
//	    conn, _ := dbManager.DB()
//	    return &PublishPostController{db: conn.DB()}
//	}
//
//	func (ctrl *PublishPostController) Invoke(c *glib.Ctx) error {
//	    id := c.PathValue("id")
//	    // Publish post logic
//	    return c.JSON(map[string]string{"status": "published"})
//	}
//
// Usage:
//
//	router.InvokableController("POST", "/posts/{id}/publish", controllers.NewPublishPostController)
type InvokableController interface {
	// Invoke handles the single action
	Invoke(c *Ctx) error
}
