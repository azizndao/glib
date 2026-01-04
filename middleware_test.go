package glib

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponse_Header(t *testing.T) {
	resp := Response{}

	// Should initialize headers on first access
	resp.Header().Set("X-Custom", "value")

	if got := resp.Header().Get("X-Custom"); got != "value" {
		t.Errorf("expected 'value', got '%s'", got)
	}
}

func TestResponse_HeaderMultiple(t *testing.T) {
	resp := Response{}

	resp.Header().Add("X-Multi", "value1")
	resp.Header().Add("X-Multi", "value2")

	values := resp.Header().Values("X-Multi")
	if len(values) != 2 {
		t.Errorf("expected 2 values, got %d", len(values))
	}
}

func TestRequest_NewRequest(t *testing.T) {
	httpReq := httptest.NewRequest("GET", "/test?foo=bar", nil)
	httpReq.Header.Set("Authorization", "Bearer token")

	req := NewRequest(httpReq)

	if req.Method() != "GET" {
		t.Errorf("expected GET, got %s", req.Method())
	}

	if req.Path() != "/test" {
		t.Errorf("expected /test, got %s", req.Path())
	}

	if req.Query("foo") != "bar" {
		t.Errorf("expected bar, got %s", req.Query("foo"))
	}

	if req.Header("Authorization") != "Bearer token" {
		t.Errorf("expected 'Bearer token', got '%s'", req.Header("Authorization"))
	}
}

func TestRequest_WithValue(t *testing.T) {
	httpReq := httptest.NewRequest("GET", "/", nil)
	req := NewRequest(httpReq)

	type key string
	const userKey key = "user"

	req2 := req.WithValue(userKey, "alice")

	// Original should be unchanged (immutable)
	if req.Value(userKey) != nil {
		t.Error("original request should not have value")
	}

	// New request should have value
	if req2.Value(userKey) != "alice" {
		t.Errorf("expected 'alice', got '%v'", req2.Value(userKey))
	}
}

func TestRequest_WithValues(t *testing.T) {
	httpReq := httptest.NewRequest("GET", "/", nil)
	req := NewRequest(httpReq)

	type key string
	const (
		userKey  key = "user"
		emailKey key = "email"
		roleKey  key = "role"
	)

	req2 := req.WithValues(map[any]any{
		userKey:  "alice",
		emailKey: "alice@example.com",
		roleKey:  "admin",
	})

	if req2.Value(userKey) != "alice" {
		t.Errorf("expected 'alice', got '%v'", req2.Value(userKey))
	}

	if req2.Value(emailKey) != "alice@example.com" {
		t.Errorf("expected 'alice@example.com', got '%v'", req2.Value(emailKey))
	}

	if req2.Value(roleKey) != "admin" {
		t.Errorf("expected 'admin', got '%v'", req2.Value(roleKey))
	}
}

func TestRequest_HTTPRequest(t *testing.T) {
	httpReq := httptest.NewRequest("GET", "/", nil)
	req := NewRequest(httpReq)

	type key string
	const userKey key = "user"

	req = req.WithValue(userKey, "bob")

	// HTTPRequest should return request with updated context
	updatedHTTPReq := req.HTTPRequest()

	if updatedHTTPReq.Context().Value(userKey) != "bob" {
		t.Error("HTTPRequest should have updated context")
	}
}

func TestRequest_QuerySlice(t *testing.T) {
	httpReq := httptest.NewRequest("GET", "/test?tags=go&tags=web&tags=api", nil)
	req := NewRequest(httpReq)

	tags := req.QuerySlice("tags")
	if len(tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tags))
	}

	// Empty slice for missing param
	empty := req.QuerySlice("missing")
	if len(empty) != 0 {
		t.Errorf("expected empty slice, got %d items", len(empty))
	}
}

func TestRequest_RemoteAddr(t *testing.T) {
	httpReq := httptest.NewRequest("GET", "/", nil)
	httpReq.RemoteAddr = "192.168.1.100:12345"

	req := NewRequest(httpReq)

	if req.RemoteAddr() != "192.168.1.100:12345" {
		t.Errorf("expected '192.168.1.100:12345', got '%s'", req.RemoteAddr())
	}
}

func TestMiddlewareIntegration(t *testing.T) {
	// Simulate a glib-style middleware
	authMiddleware := func(req Request, next Next) Response {
		token := req.Header("Authorization")
		if token == "" {
			return Response{
				Err:        http.ErrNotSupported, // Mock error
				HTTPStatus: http.StatusUnauthorized,
			}
		}

		// Add user to context
		req = req.WithValue("user", "alice")

		// Call next
		resp := next(req)

		// Add response header
		resp.Header().Set("X-Auth", "passed")

		return resp
	}

	// Test: No auth header (should return error)
	t.Run("no auth header", func(t *testing.T) {
		httpReq := httptest.NewRequest("GET", "/", nil)
		req := NewRequest(httpReq)

		resp := authMiddleware(req, func(req Request) Response {
			t.Fatal("next should not be called")
			return Response{}
		})

		if resp.Err == nil {
			t.Error("expected error response")
		}

		if resp.HTTPStatus != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", resp.HTTPStatus)
		}
	})

	// Test: Valid auth header
	t.Run("valid auth header", func(t *testing.T) {
		httpReq := httptest.NewRequest("GET", "/", nil)
		httpReq.Header.Set("Authorization", "Bearer valid-token")
		req := NewRequest(httpReq)

		nextCalled := false
		resp := authMiddleware(req, func(req Request) Response {
			nextCalled = true

			// Verify context was updated
			if req.Value("user") != "alice" {
				t.Error("expected user in context")
			}

			return Response{Payload: map[string]string{"status": "ok"}}
		})

		if !nextCalled {
			t.Error("next should have been called")
		}

		if resp.Err != nil {
			t.Errorf("unexpected error: %v", resp.Err)
		}

		// Verify response header was added
		if resp.Header().Get("X-Auth") != "passed" {
			t.Error("expected X-Auth header")
		}
	})
}
