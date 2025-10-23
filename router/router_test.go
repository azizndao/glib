package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/azizndao/glib/errors"
	"github.com/azizndao/glib/slog"
	"github.com/azizndao/glib/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRouter creates a router for testing
func setupTestRouter() Router {
	logger := slog.DiscardLogger()
	validator := validation.New(validation.DefaultValidatorConfig())
	return New(logger, validator)
}

func TestNew(t *testing.T) {
	logger := slog.DiscardLogger()
	validator := validation.New(validation.DefaultValidatorConfig())

	t.Run("with default options", func(t *testing.T) {
		r := New(logger, validator)
		assert.NotNil(t, r)
	})

	t.Run("with custom options", func(t *testing.T) {
		opts := RouterConfig{
			AutoHEAD:              false,
			TrailingSlashRedirect: false,
		}
		r := New(logger, validator, opts)
		assert.NotNil(t, r)
	})
}

func TestRouter_HTTPMethods(t *testing.T) {
	tests := []struct {
		name   string
		method string
		setup  func(r Router, pattern string, handler HandleFunc)
	}{
		{
			name:   "GET",
			method: http.MethodGet,
			setup:  func(r Router, pattern string, handler HandleFunc) { r.Get(pattern, handler) },
		},
		{
			name:   "POST",
			method: http.MethodPost,
			setup:  func(r Router, pattern string, handler HandleFunc) { r.Post(pattern, handler) },
		},
		{
			name:   "PUT",
			method: http.MethodPut,
			setup:  func(r Router, pattern string, handler HandleFunc) { r.Put(pattern, handler) },
		},
		{
			name:   "PATCH",
			method: http.MethodPatch,
			setup:  func(r Router, pattern string, handler HandleFunc) { r.Patch(pattern, handler) },
		},
		{
			name:   "DELETE",
			method: http.MethodDelete,
			setup:  func(r Router, pattern string, handler HandleFunc) { r.Delete(pattern, handler) },
		},
		{
			name:   "OPTIONS",
			method: http.MethodOptions,
			setup:  func(r Router, pattern string, handler HandleFunc) { r.Options(pattern, handler) },
		},
		{
			name:   "HEAD",
			method: http.MethodHead,
			setup:  func(r Router, pattern string, handler HandleFunc) { r.Head(pattern, handler) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			called := false

			handler := func(c *Ctx) error {
				called = true
				assert.Equal(t, tt.method, c.Method())
				return c.JSON(map[string]string{"method": tt.method})
			}

			tt.setup(r, "/test", handler)

			req := httptest.NewRequest(tt.method, "/test", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.True(t, called, "handler should be called")
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestRouter_PathParameters(t *testing.T) {
	r := setupTestRouter()

	t.Run("single parameter", func(t *testing.T) {
		r.Get("/users/{id}", func(c *Ctx) error {
			id := c.PathValue("id")
			return c.JSON(map[string]string{"id": id})
		})

		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "123", resp["id"])
	})

	t.Run("multiple parameters", func(t *testing.T) {
		r.Get("/repos/{owner}/{repo}/issues/{id}", func(c *Ctx) error {
			return c.JSON(map[string]string{
				"owner": c.PathValue("owner"),
				"repo":  c.PathValue("repo"),
				"id":    c.PathValue("id"),
			})
		})

		req := httptest.NewRequest("GET", "/repos/azizndao/glib/issues/42", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "azizndao", resp["owner"])
		assert.Equal(t, "glib", resp["repo"])
		assert.Equal(t, "42", resp["id"])
	})

	t.Run("parameter with regex constraint", func(t *testing.T) {
		// Chi supports regex constraints: {id:[0-9]+}
		r.Get("/items/{id:[0-9]+}", func(c *Ctx) error {
			return c.JSON(map[string]string{"id": c.PathValue("id")})
		})

		// Valid numeric ID
		req := httptest.NewRequest("GET", "/items/123", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Invalid non-numeric ID should 404
		req = httptest.NewRequest("GET", "/items/abc", nil)
		w = httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestRouter_Middleware(t *testing.T) {
	t.Run("router-level middleware", func(t *testing.T) {
		r := setupTestRouter()
		var calls []string

		middleware1 := func(next HandleFunc) HandleFunc {
			return func(c *Ctx) error {
				calls = append(calls, "mw1-before")
				err := next(c)
				calls = append(calls, "mw1-after")
				return err
			}
		}

		middleware2 := func(next HandleFunc) HandleFunc {
			return func(c *Ctx) error {
				calls = append(calls, "mw2-before")
				err := next(c)
				calls = append(calls, "mw2-after")
				return err
			}
		}

		r.Use(middleware1, middleware2)

		r.Get("/test", func(c *Ctx) error {
			calls = append(calls, "handler")
			return c.JSON(map[string]string{"ok": "true"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
		assert.Equal(t, expected, calls)
	})

	t.Run("route-specific middleware", func(t *testing.T) {
		r := setupTestRouter()
		var calls []string

		routeMiddleware := func(next HandleFunc) HandleFunc {
			return func(c *Ctx) error {
				calls = append(calls, "route-mw")
				return next(c)
			}
		}

		r.With(routeMiddleware).Get("/test", func(c *Ctx) error {
			calls = append(calls, "handler")
			return c.JSON(map[string]string{"ok": "true"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Contains(t, calls, "route-mw")
		assert.Contains(t, calls, "handler")
	})

	t.Run("middleware can short-circuit", func(t *testing.T) {
		r := setupTestRouter()
		handlerCalled := false

		authMiddleware := func(next HandleFunc) HandleFunc {
			return func(c *Ctx) error {
				token := c.Get("Authorization")
				if token == "" {
					return errors.Unauthorized("Missing token", nil)
				}
				return next(c)
			}
		}

		r.With(authMiddleware).Get("/protected", func(c *Ctx) error {
			handlerCalled = true
			return c.JSON(map[string]string{"data": "secret"})
		})

		// Request without token
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.False(t, handlerCalled, "handler should not be called")
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestRouter_SubRouter(t *testing.T) {
	r := setupTestRouter()

	t.Run("basic sub-router", func(t *testing.T) {
		r.Route("/api", func(api Router) {
			api.Get("/users", func(c *Ctx) error {
				return c.JSON(map[string]string{"route": "api-users"})
			})
		})

		req := httptest.NewRequest("GET", "/api/users", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "api-users", resp["route"])
	})

	t.Run("sub-router with middleware", func(t *testing.T) {
		var middlewareCalled bool

		authMiddleware := func(next HandleFunc) HandleFunc {
			return func(c *Ctx) error {
				middlewareCalled = true
				return next(c)
			}
		}

		r.Route("/v1", func(api Router) {
			api.Use(authMiddleware)
			api.Get("/protected", func(c *Ctx) error {
				return c.JSON(map[string]string{"protected": "data"})
			})
		})

		req := httptest.NewRequest("GET", "/v1/protected", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.True(t, middlewareCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("nested sub-routers", func(t *testing.T) {
		// Create a fresh router for this test to avoid mount conflicts
		freshRouter := setupTestRouter()

		freshRouter.Route("/api", func(api Router) {
			api.Route("/v1", func(v1 Router) {
				v1.Get("/users", func(c *Ctx) error {
					return c.JSON(map[string]string{"version": "v1"})
				})
			})
		})

		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()

		freshRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRouter_Group(t *testing.T) {
	r := setupTestRouter()
	var middlewareCalled bool

	authMiddleware := func(next HandleFunc) HandleFunc {
		return func(c *Ctx) error {
			middlewareCalled = true
			return next(c)
		}
	}

	// Create group with middleware (no prefix)
	r.Group(func(protected Router) {
		protected.Use(authMiddleware)
		protected.Get("/admin/dashboard", func(c *Ctx) error {
			return c.JSON(map[string]string{"page": "dashboard"})
		})
	})

	req := httptest.NewRequest("GET", "/admin/dashboard", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.True(t, middlewareCalled)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRouter_ErrorHandling(t *testing.T) {
	r := setupTestRouter()

	t.Run("returns ApiError", func(t *testing.T) {
		r.Get("/error", func(c *Ctx) error {
			return errors.BadRequest("Invalid input", nil)
		})

		req := httptest.NewRequest("GET", "/error", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(http.StatusBadRequest), resp["code"])
	})

	t.Run("returns generic error", func(t *testing.T) {
		r.Get("/panic-error", func(c *Ctx) error {
			return errors.New("something went wrong")
		})

		req := httptest.NewRequest("GET", "/panic-error", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("404 not found", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/nonexistent", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(http.StatusNotFound), resp["code"])
	})

	t.Run("405 method not allowed", func(t *testing.T) {
		r.Get("/only-get", func(c *Ctx) error {
			return c.JSON(map[string]string{"ok": "true"})
		})

		// Try POST on GET-only route
		req := httptest.NewRequest("POST", "/only-get", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestRouter_AutoHEAD(t *testing.T) {
	t.Run("explicit HEAD route", func(t *testing.T) {
		r := setupTestRouter()
		headCalled := false

		r.Get("/test", func(c *Ctx) error {
			return c.JSON(map[string]string{"method": "GET"})
		})

		r.Head("/test", func(c *Ctx) error {
			headCalled = true
			return c.JSON(map[string]string{"method": "HEAD"})
		})

		// Send HEAD request
		req := httptest.NewRequest("HEAD", "/test", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.True(t, headCalled)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GET route without HEAD returns 405", func(t *testing.T) {
		r := setupTestRouter()

		r.Get("/test", func(c *Ctx) error {
			return c.JSON(map[string]string{"data": "value"})
		})

		// Send HEAD request to GET-only route
		req := httptest.NewRequest("HEAD", "/test", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		// Chi doesn't auto-generate HEAD routes, so this should return 405
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestRouter_ContextIntegration(t *testing.T) {
	r := setupTestRouter()

	t.Run("Ctx methods work correctly", func(t *testing.T) {
		r.Post("/test", func(c *Ctx) error {
			// Test various Ctx methods
			assert.Equal(t, "POST", c.Method())
			assert.Equal(t, "/test", c.Path())
			assert.NotNil(t, c.Logger())

			// Test response methods
			return c.Status(201).JSON(map[string]string{"created": "true"})
		})

		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Ctx with request body", func(t *testing.T) {
		r.Post("/json", func(c *Ctx) error {
			var body map[string]string
			if err := c.ParseBody(&body); err != nil {
				return err
			}
			return c.JSON(body)
		})

		payload := map[string]string{"name": "test"}
		jsonData, _ := json.Marshal(payload)

		req := httptest.NewRequest("POST", "/json", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "test", resp["name"])
	})

	t.Run("Ctx with query parameters", func(t *testing.T) {
		r.Get("/search", func(c *Ctx) error {
			query := c.Query("q")
			page := c.QueryIntDefault("page", 1)
			return c.JSON(map[string]any{
				"query": query,
				"page":  page,
			})
		})

		req := httptest.NewRequest("GET", "/search?q=test&page=2", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "test", resp["query"])
		assert.Equal(t, float64(2), resp["page"])
	})
}

func TestRouter_WildcardRoutes(t *testing.T) {
	r := setupTestRouter()

	t.Run("wildcard route", func(t *testing.T) {
		r.Get("/files/*", func(c *Ctx) error {
			// Chi provides the wildcard match via URL path
			path := strings.TrimPrefix(c.Path(), "/files/")
			return c.JSON(map[string]string{"path": path})
		})

		req := httptest.NewRequest("GET", "/files/docs/readme.md", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRouter_Route(t *testing.T) {
	r := setupTestRouter()

	// Create a standard http.Handler
	standardHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("standard handler"))
	})

	r.Mount("/external", standardHandler)

	req := httptest.NewRequest("GET", "/external/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "standard handler")
}

func TestRouter_ComplexScenario(t *testing.T) {
	r := setupTestRouter()
	var calls []string

	// Global middleware
	loggingMiddleware := func(next HandleFunc) HandleFunc {
		return func(c *Ctx) error {
			calls = append(calls, "logging")
			return next(c)
		}
	}

	r.Use(loggingMiddleware)

	// Public routes
	r.Get("/public", func(c *Ctx) error {
		calls = append(calls, "public-handler")
		return c.JSON(map[string]string{"public": "data"})
	})

	// Protected API routes
	authMiddleware := func(next HandleFunc) HandleFunc {
		return func(c *Ctx) error {
			calls = append(calls, "auth")
			token := c.Get("Authorization")
			if token != "valid-token" {
				return errors.Unauthorized("Invalid token", nil)
			}
			return next(c)
		}
	}

	r.Route("/api", func(api Router) {
		api.Use(authMiddleware)

		// API v1
		api.Route("/v1", func(v1 Router) {
			v1.Get("/users/{id}", func(c *Ctx) error {
				calls = append(calls, "v1-users-handler")
				return c.JSON(map[string]string{
					"version": "v1",
					"id":      c.PathValue("id"),
				})
			})
		})
	})

	// Test public route
	t.Run("public route", func(t *testing.T) {
		calls = []string{}
		req := httptest.NewRequest("GET", "/public", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, []string{"logging", "public-handler"}, calls)
	})

	// Test protected route without auth
	t.Run("protected route without auth", func(t *testing.T) {
		calls = []string{}
		req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, calls, "logging")
		assert.Contains(t, calls, "auth")
		assert.NotContains(t, calls, "v1-users-handler")
	})

	// Test protected route with auth
	t.Run("protected route with auth", func(t *testing.T) {
		calls = []string{}
		req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
		req.Header.Set("Authorization", "valid-token")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, []string{"logging", "auth", "v1-users-handler"}, calls)

		var resp map[string]string
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "v1", resp["version"])
		assert.Equal(t, "123", resp["id"])
	})
}
