# Quickstart - Simple TODO REST API

A simple, educational REST API built with Glib framework. Perfect for getting started!

## 🎯 What You'll Learn

- ✅ Setting up a Glib HTTP server
- ✅ Creating REST API endpoints (GET, POST, PUT, DELETE)
- ✅ Request validation with struct tags
- ✅ Error handling with structured responses
- ✅ Route grouping and organization
- ✅ Basic middleware usage
- ✅ In-memory data storage

## 🚀 Quick Start

```bash
# Run the server
go run main.go

# Or use hot reload (requires air)
air
```

The server will start on `http://localhost:8080`

## 📝 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/todos` | List all todos |
| GET | `/api/todos/{id}` | Get a specific todo |
| POST | `/api/todos` | Create a new todo |
| PUT | `/api/todos/{id}` | Update a todo |
| DELETE | `/api/todos/{id}` | Delete a todo |

## 🧪 Try It Out

Use the `test.http` file with VS Code REST Client extension or curl:

```bash
# Health check
curl http://localhost:8080/health

# Create a todo
curl -X POST http://localhost:8080/api/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Learn Glib framework"}'

# List all todos
curl http://localhost:8080/api/todos

# Update a todo
curl -X PUT http://localhost:8080/api/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"title": "Master Glib framework", "completed": true}'

# Delete a todo
curl -X DELETE http://localhost:8080/api/todos/1
```

## 📚 Code Structure

```
quickstart/
├── main.go           # Complete application (~270 lines)
│   ├── Models        # Todo, CreateTodoRequest, UpdateTodoRequest
│   ├── TodoStore     # In-memory storage with thread-safe operations
│   ├── Handlers      # listTodos, getTodo, createTodo, updateTodo, deleteTodo
│   ├── Middleware    # loggingMiddleware
│   └── main()        # Server setup and routing
├── go.mod
├── .env.example
├── test.http         # API requests for testing
└── .air.toml         # Hot reload configuration
```

## 🔍 Key Concepts Explained

### 1. Request Validation

```go
type CreateTodoRequest struct {
    Title string `json:"title" validate:"required,min=3,max=100"`
}

func createTodo(store *TodoStore) glib.HandleFunc {
    return func(c *glib.Ctx) error {
        var req CreateTodoRequest
        if err := c.ValidateBody(&req); err != nil {
            return err // Returns 400 with validation errors
        }
        // ...
    }
}
```

The `validate` tag automatically validates incoming requests. If validation fails, Glib returns a structured error response.

### 2. Error Handling

```go
// Returns 404 Not Found
return errors.NotFound("Todo not found", nil)

// Returns 400 Bad Request
return errors.BadRequest("Invalid todo ID", err)

// Returns 204 No Content (success)
return c.NoContent()
```

Glib's error package provides semantic error types that automatically set the correct HTTP status codes.

### 3. Route Grouping

```go
router.Route("/api", func(api glib.Router) {
    api.Get("/todos", listTodos(store))
    api.Post("/todos", createTodo(store))
    // All routes are prefixed with /api
})
```

Grouping keeps your routes organized and makes it easy to apply middleware to specific route groups.

### 4. Middleware

```go
func loggingMiddleware(next glib.HandleFunc) glib.HandleFunc {
    return func(c *glib.Ctx) error {
        start := time.Now()
        err := next(c)
        duration := time.Since(start)
        c.Logger().Info("Request completed", "duration", duration)
        return err
    }
}

router.Use(loggingMiddleware)
```

Middleware wraps handlers to add cross-cutting concerns like logging, authentication, etc.

### 5. Thread-Safe In-Memory Storage

```go
type TodoStore struct {
    mu      sync.RWMutex
    todos   []Todo
    nextID  int
}

func (s *TodoStore) GetAll() []Todo {
    s.mu.RLock()
    defer s.mu.RUnlock()
    // Read operations use RLock for concurrent reads
}

func (s *TodoStore) Create(title string) *Todo {
    s.mu.Lock()
    defer s.mu.Unlock()
    // Write operations use Lock for exclusive access
}
```

The store uses `sync.RWMutex` to ensure thread-safe concurrent access.

## 🎓 Learning Path

1. **Start here** - Understand the basic structure
2. **Experiment** - Modify the code, add new fields to Todo
3. **Try validation** - Add new validation rules, test error responses
4. **Add features** - Try adding filtering, sorting, or search
5. **Next step** - Move to the [fullstack example](../fullstack/) to learn about databases and authentication

## 💡 Exercises

Try these to deepen your understanding:

1. **Add filtering**: Filter todos by completion status (`/api/todos?completed=true`)
2. **Add pagination**: Limit results (`/api/todos?page=1&limit=10`)
3. **Add sorting**: Sort by date or title (`/api/todos?sort=created_at`)
4. **Add due dates**: Add a `DueDate` field to todos
5. **Add categories**: Group todos by category
6. **Add search**: Search todos by title

## 📖 Next Steps

- Ready for database integration? Check out the [fullstack example](../fullstack/)
- Want to learn more about HTTP features? Read [http/README.md](../../http/README.md)
- Need validation help? See [validation docs](../../validation/README.md)

## 🐛 Common Issues

**"Port 8080 already in use"**
- Change the port in `.env`: `APP_PORT=3000`

**"Validation not working"**
- Make sure your request has `Content-Type: application/json` header

**"Cannot find package"**
- Run `go mod tidy` to download dependencies

---

**Questions?** Open an issue on [GitHub](https://github.com/azizndao/glib)
