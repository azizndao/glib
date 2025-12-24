// Package main demonstrates a simple REST API with Glib framework.
// This is a TODO list API with in-memory storage - perfect for learning the basics.
//
// What you'll learn:
// - Setting up a Glib HTTP server
// - Creating REST endpoints (GET, POST, PUT, DELETE)
// - Request validation
// - Error handling
// - Route grouping
// - Basic middleware
package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/azizndao/glib"
	"github.com/azizndao/glib/common/errors"
	"github.com/azizndao/glib/validation"
)

// =============================================================================
// Models
// =============================================================================

// Todo represents a task in our TODO list
type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateTodoRequest defines the structure for creating a new TODO
type CreateTodoRequest struct {
	Title string `json:"title" validate:"required,min=3,max=100"`
}

// UpdateTodoRequest defines the structure for updating a TODO
type UpdateTodoRequest struct {
	Title     *string `json:"title" validate:"omitempty,min=3,max=100"`
	Completed *bool   `json:"completed"`
}

// =============================================================================
// In-Memory Storage
// =============================================================================

// TodoStore manages TODO items in memory with thread-safe operations
type TodoStore struct {
	mu     sync.RWMutex
	todos  []Todo
	nextID int
}

// NewTodoStore creates a new in-memory TODO store
func NewTodoStore() *TodoStore {
	return &TodoStore{
		todos:  make([]Todo, 0),
		nextID: 1,
	}
}

// GetAll returns all todos
func (s *TodoStore) GetAll() []Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]Todo, len(s.todos))
	copy(result, s.todos)
	return result
}

// GetByID returns a single todo by ID
func (s *TodoStore) GetByID(id int) (*Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.todos {
		if s.todos[i].ID == id {
			todo := s.todos[i]
			return &todo, nil
		}
	}
	return nil, errors.NotFound("Todo not found", nil)
}

// Create adds a new todo
func (s *TodoStore) Create(title string) *Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo := Todo{
		ID:        s.nextID,
		Title:     title,
		Completed: false,
		CreatedAt: time.Now(),
	}
	s.nextID++
	s.todos = append(s.todos, todo)
	return &todo
}

// Update modifies an existing todo
func (s *TodoStore) Update(id int, title *string, completed *bool) (*Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.todos {
		if s.todos[i].ID == id {
			if title != nil {
				s.todos[i].Title = *title
			}
			if completed != nil {
				s.todos[i].Completed = *completed
			}
			todo := s.todos[i]
			return &todo, nil
		}
	}
	return nil, errors.NotFound("Todo not found", nil)
}

// Delete removes a todo
func (s *TodoStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.todos {
		if s.todos[i].ID == id {
			s.todos = append(s.todos[:i], s.todos[i+1:]...)
			return nil
		}
	}
	return errors.NotFound("Todo not found", nil)
}

// =============================================================================
// Handlers
// =============================================================================

// listTodos returns all todos
// GET /todos
func listTodos(store *TodoStore) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		todos := store.GetAll()
		return c.JSON(map[string]interface{}{
			"todos": todos,
			"count": len(todos),
		})
	}
}

// getTodo returns a single todo by ID
// GET /todos/{id}
func getTodo(store *TodoStore) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		idStr := c.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return errors.BadRequest("Invalid todo ID", err)
		}

		todo, err := store.GetByID(id)
		if err != nil {
			return err
		}

		return c.JSON(todo)
	}
}

// createTodo creates a new todo
// POST /todos
func createTodo(store *TodoStore) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		var req CreateTodoRequest

		// ValidateBody parses JSON and validates it
		if err := c.ValidateBody(&req); err != nil {
			return err
		}

		todo := store.Create(req.Title)
		return c.Status(201).JSON(todo)
	}
}

// updateTodo updates an existing todo
// PUT /todos/{id}
func updateTodo(store *TodoStore) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		idStr := c.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return errors.BadRequest("Invalid todo ID", err)
		}

		var req UpdateTodoRequest
		if err := c.ValidateBody(&req); err != nil {
			return err
		}

		todo, err := store.Update(id, req.Title, req.Completed)
		if err != nil {
			return err
		}

		return c.JSON(todo)
	}
}

// deleteTodo deletes a todo
// DELETE /todos/{id}
func deleteTodo(store *TodoStore) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		idStr := c.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return errors.BadRequest("Invalid todo ID", err)
		}

		if err := store.Delete(id); err != nil {
			return err
		}

		return c.NoContent()
	}
}

// =============================================================================
// Middleware
// =============================================================================

// loggingMiddleware logs each request
func loggingMiddleware(next glib.HandleFunc) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		start := time.Now()

		c.Logger().Info("Request started",
			"method", c.Method(),
			"path", c.Path(),
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

// =============================================================================
// Main
// =============================================================================

func main() {
	// Create validator for request validation
	validator := validation.New(validation.Config{
		DefaultLocale:     "en",
		UseJSONFieldNames: true,
	})

	// Create HTTP server
	server := glib.New(glib.Config{
		Validator: validator,
	})

	// Create in-memory todo store
	store := NewTodoStore()

	// Get router
	router := server.Router()

	// Apply global middleware
	router.Use(loggingMiddleware)

	// Health check endpoint
	router.Get("/health", func(c *glib.Ctx) error {
		return c.JSON(map[string]interface{}{
			"status": "healthy",
			"time":   time.Now().Unix(),
		})
	})

	// API routes group
	router.Route("/api", func(api glib.Router) {
		// Todo endpoints
		api.Get("/todos", listTodos(store))
		api.Get("/todos/{id}", getTodo(store))
		api.Post("/todos", createTodo(store))
		api.Put("/todos/{id}", updateTodo(store))
		api.Delete("/todos/{id}", deleteTodo(store))
	})

	// Start server
	fmt.Println("🚀 Server starting...")
	fmt.Printf("📍 Listening on %s\n", server.Address())
	fmt.Println("📝 API endpoints:")
	fmt.Println("   GET    /health")
	fmt.Println("   GET    /api/todos")
	fmt.Println("   GET    /api/todos/{id}")
	fmt.Println("   POST   /api/todos")
	fmt.Println("   PUT    /api/todos/{id}")
	fmt.Println("   DELETE /api/todos/{id}")
	fmt.Println()

	if err := server.ListenWithGracefulShutdown(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
