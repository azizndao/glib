# Comprehensive glib Example

This example demonstrates **all features** of the glib web framework in a single application. It serves as both a tutorial and a reference implementation.

## Features Covered

### 🚀 Core Features
- Server creation and configuration
- Environment-based configuration
- Graceful shutdown handling
- Multi-language validation (English, French, Spanish)

### 🛣️ Routing
- All HTTP methods: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`, `TRACE`, `CONNECT`
- Route parameters (e.g., `/users/{id}`)
- Query parameters with type conversion
- Route grouping with `Route()`
- Nested sub-routing (e.g., `/api/v1/admin/stats`)
- Custom 404 and 405 handlers

### 📝 Request Handling
- JSON body parsing with validation
- Form data handling
- File uploads (multipart/form-data)
- Query parameter parsing (string, int, float, bool, arrays)
- Path parameter extraction
- Header access
- Cookie management (get, set, clear)

### 📤 Response Types
- JSON responses
- Plain text responses
- HTML responses
- File serving
- HTTP redirects
- No content (204) responses
- Custom status codes

### 🎯 Context Methods
- Request information (method, path, IP, user agent)
- Header manipulation
- Cookie operations
- Content type detection
- Content negotiation (Accept header)
- IP address extraction (with proxy support)
- Secure connection detection
- Base URL and scheme access

### ✅ Validation
- Struct validation with tags
- Internationalization (i18n) support
- Automatic locale detection from `Accept-Language` header
- Custom validation error messages
- Multiple validation rules per field

### ⚠️ Error Handling
- Built-in error helpers:
  - `BadRequest` (400)
  - `Unauthorized` (401)
  - `Forbidden` (403)
  - `NotFound` (404)
  - `Conflict` (409)
  - `InternalServerError` (500)
  - Custom errors with `errors.New()`
- Structured error responses
- Error data attachment

### 🔧 Middleware
- Custom middleware creation
- Global middleware with `Use()`
- Route-specific middleware with `With()`
- Group middleware
- Chi middleware integration with `UseChiMiddleware()`
- Built-in middleware:
  - Request ID
  - Timeout
  - Logging
  - Recovery
  - Compression
  - CORS
  - Rate limiting

### 🏗️ Advanced Patterns
- Middleware chaining
- Context value storage and retrieval
- Authentication middleware example
- Timing middleware example
- Content negotiation
- Protected routes

## Project Structure

```
comprehensive/
├── main.go           # Complete example with all features
├── go.mod            # Go module definition
├── .env.example      # Example environment configuration
├── test.http         # HTTP requests for testing all endpoints
└── README.md         # This file
```

## Getting Started

### Prerequisites

- Go 1.22 or higher
- Basic understanding of HTTP and REST APIs

### Installation

1. Clone the repository and navigate to this example:

```bash
cd glib/example/comprehensive
```

2. Install dependencies:

```bash
go mod download
```

3. Copy the example environment file:

```bash
cp .env.example .env
```

4. Run the server:

```bash
go run main.go
```

The server will start on `http://localhost:8080` (or the port specified in `.env`).

### Testing the Endpoints

Use the provided `test.http` file with the REST Client extension in VS Code, or use curl:

```bash
# Home endpoint
curl http://localhost:8080/

# Health check
curl http://localhost:8080/health

# Create user with validation
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"securePass123","name":"John Doe","age":25}'

# Get user
curl http://localhost:8080/users/123

# Protected endpoint (requires auth)
curl http://localhost:8080/products \
  -H "Authorization: Bearer valid-token"
```

## Environment Configuration

All server and middleware settings can be configured via environment variables. See `.env.example` for all available options:

- **Server**: `HOST`, `PORT`, timeouts
- **Logging**: `LOG_LEVEL`, `LOG_FORMAT`, `IS_DEBUG`
- **Middleware**: Enable/disable CORS, compression, rate limiting
- **CORS**: Origins, methods, headers, credentials
- **Rate Limiting**: Max requests, time window
- **Body Limit**: Maximum request body size

## API Endpoints Reference

### Basic Routes
- `GET /` - Home page with endpoint list
- `GET /health` - Health check
- `GET /ping` - Heartbeat (Chi middleware)

### User CRUD
- `GET /users` - List users (with pagination)
- `POST /users` - Create user (with validation)
- `GET /users/{id}` - Get user by ID
- `PUT /users/{id}` - Update user (full)
- `PATCH /users/{id}` - Update user (partial)
- `DELETE /users/{id}` - Delete user
- `HEAD /users/{id}` - Check if user exists
- `OPTIONS /users/{id}` - Get allowed methods

### Products (Auth Required)
- `GET /products` - List products
- `POST /products` - Create product
- `GET /products/{id}` - Get product by ID
- `PUT /products/{id}` - Update product
- `DELETE /products/{id}` - Delete product
- `GET /products/search` - Search products

### Context Features
- `GET /context/request-info` - Request metadata
- `GET /context/headers` - All headers
- `GET /context/cookies` - Cookie operations
- `POST /context/set-cookie` - Set cookie
- `GET /context/clear-cookie` - Clear cookie
- `GET /context/query-params` - Query parameter examples
- `GET /context/ip-info` - IP address information

### File Operations
- `POST /files/upload` - Upload file
- `GET /files/download/{filename}` - Download file
- `GET /files/serve` - Serve static file

### Response Types
- `GET /responses/json` - JSON response
- `GET /responses/text` - Plain text
- `GET /responses/html` - HTML response
- `GET /responses/redirect` - HTTP redirect
- `DELETE /responses/no-content` - 204 No Content
- `GET /responses/custom-status` - Custom status code

### Error Examples
- `GET /errors/bad-request` - 400 error
- `GET /errors/unauthorized` - 401 error
- `GET /errors/forbidden` - 403 error
- `GET /errors/not-found` - 404 error
- `GET /errors/conflict` - 409 error
- `GET /errors/internal` - 500 error
- `GET /errors/custom` - Custom error

### Middleware Examples
- `GET /middleware/timed` - With timing middleware
- `GET /middleware/timeout/fast` - Fast response (< 2s)
- `GET /middleware/timeout/slow` - Slow response (timeout)
- `GET /middleware/request-id` - Request ID demo

### Advanced Features
- `GET /advanced/negotiate` - Content negotiation
- `GET /advanced/context-value` - Context values
- `GET /advanced/protected` - Multiple middleware

### Validation Examples
- `POST /validation/user` - User validation
- `POST /validation/product` - Product validation
- `POST /validation/french` - French locale (Accept-Language: fr)
- `POST /validation/spanish` - Spanish locale (Accept-Language: es)

### Nested Routing
- `GET /api/version` - API version
- `GET /api/v1/status` - v1 status
- `GET /api/v1/admin/stats` - Admin stats (auth required)
- `GET /api/v1/admin/users` - Admin users (auth required)
- `GET /api/v2/status` - v2 status

## Code Examples

### Creating a Server

```go
serverConfig := glib.Config{
    Locales: []glib.LocaleConfig{
        glib.Locale(fr.New(), frt.RegisterDefaultTranslations),
        glib.Locale(es.New(), est.RegisterDefaultTranslations),
    },
}

server := glib.New(serverConfig)
r := server.Router()
```

### Defining Routes

```go
// Simple route
r.Get("/hello", func(c *router.Ctx) error {
    return c.JSON(map[string]string{"message": "Hello World"})
})

// Route with path parameter
r.Get("/users/{id}", func(c *router.Ctx) error {
    id := c.PathValue("id")
    return c.JSON(map[string]string{"id": id})
})

// Route group
r.Route("/api", func(api router.Router) {
    api.Get("/version", versionHandler)
    api.Post("/users", createUserHandler)
})
```

### Request Validation

```go
type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Name     string `json:"name" validate:"required,min=2"`
    Age      int    `json:"age" validate:"required,gte=18"`
}

func createUserHandler(c *router.Ctx) error {
    var req CreateUserRequest
    
    if err := c.ValidateBody(&req); err != nil {
        return err // Automatically returns validation errors
    }
    
    // Process valid request...
    return c.Status(201).JSON(req)
}
```

### Custom Middleware

```go
func loggingMiddleware(next router.HandleFunc) router.HandleFunc {
    return func(c *router.Ctx) error {
        start := time.Now()
        
        c.Logger().Info("Request started", "path", c.Path())
        
        err := next(c)
        
        duration := time.Since(start)
        c.Logger().Info("Request completed", "duration", duration)
        
        return err
    }
}

// Use globally
r.Use(loggingMiddleware)

// Use on specific route
r.With(loggingMiddleware).Get("/timed", handler)
```

### Error Handling

```go
func handler(c *router.Ctx) error {
    // Return typed errors
    if unauthorized {
        return errors.Unauthorized("Authentication required", nil)
    }
    
    if notFound {
        return errors.NotFound("User not found", map[string]string{
            "user_id": "123",
        })
    }
    
    // Custom error
    return errors.New(422, "Unprocessable Entity", "Validation failed", data)
}
```

### Context Operations

```go
func handler(c *router.Ctx) error {
    // Request data
    method := c.Method()
    path := c.Path()
    ip := c.IP()
    userAgent := c.UserAgent()
    
    // Headers
    authHeader := c.Authorization()
    customHeader := c.Get("X-Custom-Header")
    
    // Query parameters
    page := c.QueryIntDefault("page", 1)
    search := c.QueryDefault("search", "")
    
    // Cookies
    cookie, _ := c.GetCookie("session")
    
    // Set response
    c.Set("X-Custom-Response", "value")
    c.SetCookie(&http.Cookie{Name: "token", Value: "abc"})
    
    return c.JSON(data)
}
```

## Key Takeaways

1. **Environment-Driven**: Configure everything via environment variables
2. **Type-Safe**: Leverage Go's type system for validation and error handling
3. **Middleware-First**: Chain middleware for cross-cutting concerns
4. **Error Handling**: Use typed errors for consistent API responses
5. **Validation**: Built-in i18n validation with locale detection
6. **Context-Rich**: Full access to request/response through the Ctx object
7. **Flexible Routing**: Group, nest, and organize routes logically
8. **Production-Ready**: Includes logging, graceful shutdown, and error recovery

## Next Steps

- Explore the `test.http` file for complete request examples
- Modify `.env` to test different configurations
- Add your own routes and middleware
- Check out the `basic` and `sub_routing` examples for simpler use cases

## Resources

- [glib Repository](https://github.com/azizndao/glib)
- [Chi Router Documentation](https://github.com/go-chi/chi)
- [Go Validator Documentation](https://github.com/go-playground/validator)

## License

This example is part of the glib framework and follows the same license.