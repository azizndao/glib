package glib

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteResponseWithMetadata(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		data               any
		expectedStatus     int
		expectedHeaders    map[string]string
		expectedHeaderMiss []string
	}{
		{
			name:   "simple struct without metadata",
			method: "GET",
			data: struct {
				Name string `json:"name"`
			}{Name: "test"},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "struct with Location header",
			method: "POST",
			data: struct {
				ID       int    `json:"id"`
				Location string `header:"Location"`
			}{
				ID:       123,
				Location: "/api/posts/123",
			},
			expectedStatus: http.StatusCreated,
			expectedHeaders: map[string]string{
				"Location": "/api/posts/123",
			},
		},
		{
			name:   "struct with custom status code",
			method: "POST",
			data: struct {
				Message string `json:"message"`
				Status  int    `response:"httpstatus"`
			}{
				Message: "accepted",
				Status:  http.StatusAccepted,
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:   "struct with header and status",
			method: "POST",
			data: struct {
				ID       string `json:"id"`
				Location string `header:"Location"`
				Status   int    `response:"httpstatus"`
			}{
				ID:       "abc123",
				Location: "/api/items/abc123",
				Status:   http.StatusCreated,
			},
			expectedStatus: http.StatusCreated,
			expectedHeaders: map[string]string{
				"Location": "/api/items/abc123",
			},
		},
		{
			name:   "struct with omitempty header (value present)",
			method: "GET",
			data: struct {
				Data string `json:"data"`
				ETag string `header:"ETag,omitempty"`
			}{
				Data: "test",
				ETag: "abc123",
			},
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"ETag": "abc123",
			},
		},
		{
			name:   "struct with omitempty header (value empty)",
			method: "GET",
			data: struct {
				Data string `json:"data"`
				ETag string `header:"ETag,omitempty"`
			}{
				Data: "test",
				ETag: "",
			},
			expectedStatus:     http.StatusOK,
			expectedHeaderMiss: []string{"ETag"},
		},
		{
			name:   "pointer to struct",
			method: "POST",
			data: &struct {
				Name     string `json:"name"`
				Location string `header:"Location"`
			}{
				Name:     "test",
				Location: "/test",
			},
			expectedStatus: http.StatusCreated,
			expectedHeaders: map[string]string{
				"Location": "/test",
			},
		},
		{
			name:           "nil pointer",
			method:         "GET",
			data:           (*struct{ Name string })(nil),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "primitive type (no metadata)",
			method:         "GET",
			data:           "simple string",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "DELETE method defaults to 204 No Content",
			method:         "DELETE",
			data:           struct{ Message string }{Message: "deleted"},
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteResponseWithMetadata(w, tt.method, tt.data)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check expected headers
			for key, expectedValue := range tt.expectedHeaders {
				actualValue := w.Header().Get(key)
				if actualValue != expectedValue {
					t.Errorf("expected header %s=%q, got %q", key, expectedValue, actualValue)
				}
			}

			// Check headers that should not be present
			for _, key := range tt.expectedHeaderMiss {
				if actualValue := w.Header().Get(key); actualValue != "" {
					t.Errorf("expected header %s to be absent, but got %q", key, actualValue)
				}
			}
		})
	}
}

func TestGetDefaultStatusCode(t *testing.T) {
	tests := []struct {
		method       string
		expectedCode int
	}{
		{"GET", http.StatusOK},
		{"POST", http.StatusCreated},
		{"PUT", http.StatusOK},
		{"PATCH", http.StatusOK},
		{"DELETE", http.StatusNoContent},
		{"HEAD", http.StatusOK},
		{"OPTIONS", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			code := getDefaultStatusCode(tt.method)
			if code != tt.expectedCode {
				t.Errorf("expected %d for %s, got %d", tt.expectedCode, tt.method, code)
			}
		})
	}
}
